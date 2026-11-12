// Package bot принимает сообщения пользователей в чате и записывает из них траты: «кофе 5» это расход, «+1000 зарплата»
// доход. Бот работает через long polling, пользователь определяется по Telegram-аккаунту, суммы и описания в журнал
// не пишутся.
package bot

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	"moneykeeper/internal/i18n"
	"moneykeeper/internal/models"
	"moneykeeper/internal/telegram"
	"moneykeeper/pkg/money"
	"moneykeeper/pkg/quickinput"
)

const (
	recentLimit   = 200
	choicesLimit  = 8
	pendingTTL    = time.Hour
	retryInterval = 5 * time.Second
)

type Service interface {
	FindUser(ctx context.Context, telegramID int64) (models.User, error)
	Budget(ctx context.Context, u models.User) (models.Budget, error)
	ListAccounts(ctx context.Context, u models.User, in string) ([]models.Account, error)
	ListCategories(ctx context.Context, u models.User) ([]models.Category, error)
	ListTransactions(ctx context.Context, u models.User, limit, offset int, in string) ([]models.Transaction, error)
	CreateTransaction(ctx context.Context, u models.User, in models.NewTransaction) ([]models.Transaction, error)
	DeleteTransaction(ctx context.Context, u models.User, id int64) error
}

type API interface {
	Updates(ctx context.Context, offset int64) ([]telegram.Update, error)
	Send(ctx context.Context, chatID int64, text string) error
	SendButtons(ctx context.Context, chatID int64, text string, rows [][]telegram.Button) (int64, error)
	EditText(ctx context.Context, chatID, messageID int64, text string, rows [][]telegram.Button) error
	AnswerCallback(ctx context.Context, callbackID string) error
}

// pending это расход, которому пользователь ещё выбирает категорию. Хранится в памяти: после перезапуска выбор
// устаревает, и трату нужно прислать заново.
type pending struct {
	telegramID  int64
	amountMinor int64
	currency    string
	note        string
	expires     time.Time
}

type Bot struct {
	svc     Service
	api     API
	log     *slog.Logger
	now     func() time.Time
	mu      sync.Mutex
	pending map[string]pending
}

func New(svc Service, api API, log *slog.Logger) *Bot {
	return &Bot{svc: svc, api: api, log: log, now: time.Now, pending: map[string]pending{}}
}

// Run принимает сообщения, пока не закрыт ctx. Ошибка опроса не останавливает бота: он повторяет через несколько секунд.
func (b *Bot) Run(ctx context.Context) {
	var offset int64
	for ctx.Err() == nil {
		updates, err := b.api.Updates(ctx, offset)
		if err != nil {
			if ctx.Err() == nil {
				b.log.Warn("poll updates", "err", err)
				select {
				case <-ctx.Done():
				case <-time.After(retryInterval):
				}
			}
			continue
		}
		for _, u := range updates {
			offset = u.ID + 1
			b.Handle(ctx, u)
		}
	}
}

// Handle разбирает одно событие. Сообщения вне личного чата игнорируются.
func (b *Bot) Handle(ctx context.Context, u telegram.Update) {
	switch {
	case u.Message != nil && u.Message.From != nil && u.Message.Chat.Type == "private" && u.Message.Text != "":
		b.message(ctx, *u.Message)
	case u.Callback != nil:
		b.callback(ctx, *u.Callback)
	}
}

func (b *Bot) reply(ctx context.Context, chatID int64, text string) {
	if err := b.api.Send(ctx, chatID, text); err != nil {
		b.log.Warn("send reply", "err", err)
	}
}

func (b *Bot) message(ctx context.Context, m telegram.Message) {
	chat := m.Chat.ID
	text := strings.TrimSpace(m.Text)
	if text == "/start" || text == "/help" {
		b.reply(ctx, chat, i18n.BotHelp())
		return
	}
	u, acc, ok := b.account(ctx, m.From.ID, chat)
	if !ok {
		return
	}

	income := strings.HasPrefix(text, "+")
	res, err := quickinput.Parse(strings.TrimPrefix(text, "+"), acc.Currency)
	if err != nil {
		b.reply(ctx, chat, i18n.BotNoAmount())
		return
	}
	cur := string(res.Currency)
	shown := res.Currency
	if shown == "" {
		shown = acc.Currency
	}
	amount := money.Format(res.AmountMinor, shown)

	if income {
		if _, err := b.svc.CreateTransaction(ctx, u, models.NewTransaction{Kind: models.KindIncome, AccountID: acc.ID, AmountMinor: res.AmountMinor, Currency: cur, Note: res.Note}); err != nil {
			b.failed(ctx, chat, err)
			return
		}
		b.reply(ctx, chat, i18n.BotSavedIncome(amount, res.Note))
		return
	}

	cats, err := b.svc.ListCategories(ctx, u)
	if err != nil {
		b.failed(ctx, chat, err)
		return
	}
	recent, err := b.svc.ListTransactions(ctx, u, recentLimit, 0, "")
	if err != nil {
		b.failed(ctx, chat, err)
		return
	}
	names := map[int64]string{}
	for _, c := range cats {
		if !c.Hidden {
			names[c.ID] = c.Name
		}
	}
	if id, found := guessCategory(recent, res.Note); found {
		if _, ok := names[id]; ok {
			b.saveExpense(ctx, u, acc, chat, 0, pending{amountMinor: res.AmountMinor, currency: cur, note: res.Note}, id, names[id], amount)
			return
		}
	}

	key := b.remember(pending{telegramID: m.From.ID, amountMinor: res.AmountMinor, currency: cur, note: res.Note})
	var row []telegram.Button
	var rows [][]telegram.Button
	for _, id := range choices(recent, cats, names) {
		row = append(row, telegram.Button{Text: names[id], Data: fmt.Sprintf("c:%s:%d", key, id)})
		if len(row) == 2 {
			rows, row = append(rows, row), nil
		}
	}
	if len(row) > 0 {
		rows = append(rows, row)
	}
	if _, err := b.api.SendButtons(ctx, chat, i18n.BotPick(amount, res.Note), rows); err != nil {
		b.log.Warn("send categories", "err", err)
	}
}

// account находит пользователя и его основной счёт. Если настройка в приложении не пройдена, отвечает подсказкой.
func (b *Bot) account(ctx context.Context, telegramID, chat int64) (models.User, models.Account, bool) {
	u, err := b.svc.FindUser(ctx, telegramID)
	if errors.Is(err, models.ErrNotFound) {
		b.reply(ctx, chat, i18n.BotUnknownUser())
		return models.User{}, models.Account{}, false
	}
	if err != nil {
		b.failed(ctx, chat, err)
		return models.User{}, models.Account{}, false
	}
	budget, err := b.svc.Budget(ctx, u)
	if err != nil {
		b.failed(ctx, chat, err)
		return models.User{}, models.Account{}, false
	}
	if budget.BaseCurrency == nil || budget.InterfaceMode == "" {
		b.reply(ctx, chat, i18n.BotUnknownUser())
		return models.User{}, models.Account{}, false
	}
	accs, err := b.svc.ListAccounts(ctx, u, "")
	if err != nil {
		b.failed(ctx, chat, err)
		return models.User{}, models.Account{}, false
	}
	for _, a := range accs {
		if a.IsDefault && !a.Archived {
			return u, a, true
		}
	}
	b.reply(ctx, chat, i18n.BotUnknownUser())
	return models.User{}, models.Account{}, false
}

func (b *Bot) failed(ctx context.Context, chat int64, err error) {
	if errors.Is(err, models.ErrInvalidTransaction) || errors.Is(err, models.ErrCurrencyMismatch) {
		b.reply(ctx, chat, i18n.BotCurrencyMismatch())
		return
	}
	b.log.Error("bot request", "err", err)
	b.reply(ctx, chat, i18n.BotFailed())
}

// saveExpense записывает расход и показывает результат с кнопкой «Отменить». messageID это сообщение с выбором
// категории, которое нужно заменить; 0 значит отправить новое.
func (b *Bot) saveExpense(ctx context.Context, u models.User, acc models.Account, chat, messageID int64, p pending, category int64, name, amount string) {
	txs, err := b.svc.CreateTransaction(ctx, u, models.NewTransaction{
		Kind: models.KindExpense, AccountID: acc.ID, CategoryID: &category, AmountMinor: p.amountMinor, Currency: p.currency, Note: p.note,
	})
	if err != nil || len(txs) == 0 {
		if err == nil {
			err = errors.New("no transaction created")
		}
		b.failed(ctx, chat, err)
		return
	}
	text := i18n.BotSavedExpense(amount, name, p.note)
	undo := [][]telegram.Button{{{Text: i18n.BotUndoButton(), Data: fmt.Sprintf("u:%d", txs[0].ID)}}}
	if messageID != 0 {
		if err := b.api.EditText(ctx, chat, messageID, text, undo); err != nil {
			b.log.Warn("edit message", "err", err)
		}
		return
	}
	if _, err := b.api.SendButtons(ctx, chat, text, undo); err != nil {
		b.log.Warn("send saved", "err", err)
	}
}

func (b *Bot) callback(ctx context.Context, cb telegram.CallbackQuery) {
	if err := b.api.AnswerCallback(ctx, cb.ID); err != nil {
		b.log.Warn("answer callback", "err", err)
	}
	if cb.Message == nil {
		return
	}
	chat, msg := cb.Message.Chat.ID, cb.Message.ID
	parts := strings.Split(cb.Data, ":")
	u, acc, ok := b.account(ctx, cb.From.ID, chat)
	if !ok {
		return
	}

	switch {
	case len(parts) == 3 && parts[0] == "c":
		p, found := b.take(parts[1], cb.From.ID)
		if !found {
			b.reply(ctx, chat, i18n.BotExpired())
			return
		}
		var id int64
		if _, err := fmt.Sscan(parts[2], &id); err != nil {
			return
		}
		cats, err := b.svc.ListCategories(ctx, u)
		if err != nil {
			b.failed(ctx, chat, err)
			return
		}
		for _, c := range cats {
			if c.ID == id && !c.Hidden {
				shown := money.Currency(p.currency)
				if shown == "" {
					shown = acc.Currency
				}
				b.saveExpense(ctx, u, acc, chat, msg, p, id, c.Name, money.Format(p.amountMinor, shown))
				return
			}
		}
	case len(parts) == 2 && parts[0] == "u":
		var id int64
		if _, err := fmt.Sscan(parts[1], &id); err != nil {
			return
		}
		if err := b.svc.DeleteTransaction(ctx, u, id); err != nil && !errors.Is(err, models.ErrNotFound) {
			b.failed(ctx, chat, err)
			return
		}
		if err := b.api.EditText(ctx, chat, msg, i18n.BotUndone(), nil); err != nil {
			b.log.Warn("edit message", "err", err)
		}
	}
}

func (b *Bot) remember(p pending) string {
	raw := make([]byte, 4)
	_, _ = rand.Read(raw)
	key := hex.EncodeToString(raw)
	now := b.now()
	p.expires = now.Add(pendingTTL)
	b.mu.Lock()
	defer b.mu.Unlock()
	for k, v := range b.pending {
		if now.After(v.expires) {
			delete(b.pending, k)
		}
	}
	b.pending[key] = p
	return key
}

// take забирает выбор: повторное нажатие и чужое нажатие ничего не записывают.
func (b *Bot) take(key string, telegramID int64) (pending, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	p, ok := b.pending[key]
	if !ok || p.telegramID != telegramID || b.now().After(p.expires) {
		return pending{}, false
	}
	delete(b.pending, key)
	return p, true
}

// guessCategory берёт категорию последнего расхода с такой же заметкой (без учёта регистра и лишних пробелов).
// Транзакции идут от новых к старым.
func guessCategory(txs []models.Transaction, note string) (int64, bool) {
	key := strings.ToLower(strings.Join(strings.Fields(note), " "))
	if key == "" {
		return 0, false
	}
	for _, t := range txs {
		if t.Kind == models.KindExpense && t.CategoryID != nil && strings.ToLower(strings.Join(strings.Fields(t.Note), " ")) == key {
			return *t.CategoryID, true
		}
	}
	return 0, false
}

// choices даёт категории для кнопок: сначала самые частые в последних расходах, остальные места занимают первые
// категории списка. Скрытые категории не предлагаются.
func choices(txs []models.Transaction, cats []models.Category, names map[int64]string) []int64 {
	type stat struct{ count, first int }
	seen := map[int64]*stat{}
	index := 0
	for _, t := range txs {
		if t.Kind != models.KindExpense || t.CategoryID == nil {
			continue
		}
		if _, ok := names[*t.CategoryID]; !ok {
			continue
		}
		index++
		if s, ok := seen[*t.CategoryID]; ok {
			s.count++
		} else {
			seen[*t.CategoryID] = &stat{1, index}
		}
	}
	out := make([]int64, 0, choicesLimit)
	for id := range seen {
		out = append(out, id)
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := seen[out[i]], seen[out[j]]
		return a.count > b.count || (a.count == b.count && a.first < b.first)
	})
	if len(out) > choicesLimit {
		out = out[:choicesLimit]
	}
	used := map[int64]bool{}
	for _, id := range out {
		used[id] = true
	}
	for _, c := range cats {
		if len(out) >= choicesLimit {
			break
		}
		if _, ok := names[c.ID]; ok && !used[c.ID] {
			out = append(out, c.ID)
		}
	}
	return out
}
