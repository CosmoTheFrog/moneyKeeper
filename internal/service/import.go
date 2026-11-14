package service

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"moneykeeper/internal/models"
	"moneykeeper/internal/repo"
	"moneykeeper/pkg/apptime"
	"moneykeeper/pkg/money"
)

const maxImportRows = 500

// importAccount возвращает счёт для импорта. В простом режиме выписка любой валюты идёт на основной счёт (других счетов
// пользователь не видит), поэтому такой счёт принимает любые валюты. В режиме с картами счёт остаётся как есть: только
// его валюта.
func (s *Service) importAccount(ctx context.Context, u models.User, accountID int64) (models.Account, error) {
	acc, err := s.accounts.Get(ctx, u.ActiveBudgetID, accountID)
	if err != nil {
		return models.Account{}, err
	}
	b, err := s.budgets.Get(ctx, u.ActiveBudgetID)
	if err != nil {
		return models.Account{}, err
	}
	if b.InterfaceMode == models.ModeSimple && acc.IsDefault {
		acc.MultiCurrency = true
	}
	return acc, nil
}

// WithImportRules включает запоминание выбранных при импорте категорий: продавец или MCC и категория. Без него
// выбор приходится повторять при каждом импорте.
func (s *Service) WithImportRules(r *repo.ImportRules) *Service {
	s.importRules = r
	return s
}

// ruleKeys даёт ключи запомненных соответствий для строки: сначала продавец, потом название из файла.
func ruleKeys(r models.ImportRow) []string {
	var keys []string
	if d := strings.ToLower(collapse(r.Description)); d != "" {
		keys = append(keys, "d:"+d)
	}
	if c := strings.ToLower(collapse(r.SourceCategory)); c != "" {
		keys = append(keys, "s:"+c)
	}
	return keys
}

// applyRules подставляет запомненную категорию расходам, у которых категории ещё нет. Подставленная категория
// считается предложенной (Suggested): пользователь видит её и может сменить. Категории, которых больше нет в списке
// (скрыта или удалена), игнорируются.
func (s *Service) applyRules(ctx context.Context, u models.User, rows []models.ImportRow, usable map[string]int64) error {
	if s.importRules == nil || len(rows) == 0 {
		return nil
	}
	rules, err := s.importRules.All(ctx, u.ActiveBudgetID)
	if err != nil {
		return err
	}
	ok := make(map[int64]bool, len(usable))
	for _, id := range usable {
		ok[id] = true
	}
	for i, r := range rows {
		if r.AmountMinor >= 0 || r.CategoryID != nil {
			continue
		}
		for _, k := range ruleKeys(r) {
			if id, found := rules[k]; found && ok[id] {
				rows[i].CategoryID, rows[i].Suggested = &id, true
				break
			}
		}
	}
	return nil
}

// currencySigns: выписки часто пишут знак вместо кода валюты.
var currencySigns = map[string]string{"₾": "GEL", "$": "USD", "€": "EUR", "₽": "RUB", "RUR": "RUB"}

// markDuplicates помечает строки, для которых на счёте уже есть запись с той же датой, суммой и валютой. Одна
// существующая запись закрывает одну строку: две одинаковые покупки за день остаются двумя.
func (s *Service) markDuplicates(ctx context.Context, u models.User, accountID int64, rows []models.ImportRow) error {
	if len(rows) == 0 {
		return nil
	}
	from, to := rows[0].Date, rows[0].Date
	for _, r := range rows {
		if r.Date.Before(from) {
			from = r.Date
		}
		if to.Before(r.Date) {
			to = r.Date
		}
	}
	existing, err := s.transactions.OnAccount(ctx, u.ActiveBudgetID, accountID, from, to)
	if err != nil {
		return err
	}
	type key struct {
		day    apptime.Day
		amount int64
		cur    money.Currency
	}
	seen := make(map[key]int, len(existing))
	for _, t := range existing {
		seen[key{t.OccurredOn, t.AmountMinor, t.Currency}]++
	}
	for i, r := range rows {
		k := key{r.Date, r.AmountMinor, r.Currency}
		if seen[k] > 0 {
			seen[k]--
			rows[i].Duplicate = true
		}
	}
	return nil
}

// CommitImport сохраняет подтверждённые строки одной транзакцией БД: отрицательные расходом (категория обязательна),
// положительные доходом. Строка с ExternalID (перевод из сети) пропускается, если он уже внесён. Выбранные категории
// запоминаются по продавцу и названию из файла. Возвращает номера сохранённых записей: по ним импорт можно отменить.
func (s *Service) CommitImport(ctx context.Context, u models.User, accountID int64, rows []models.ImportRow) ([]int64, error) {
	if len(rows) == 0 || len(rows) > maxImportRows {
		return nil, fmt.Errorf("rows count: %w", models.ErrInvalidImport)
	}
	var ids []int64
	err := s.db.InTx(ctx, func(ctx context.Context) error {
		ids = nil
		learned := map[string]int64{}
		acc, err := s.importAccount(ctx, u, accountID)
		if err != nil {
			return err
		}
		// Основной счёт простого режима принимает любые валюты: флаг сохраняется, когда выписка в чужой валюте.
		if stored, err := s.accounts.Get(ctx, u.ActiveBudgetID, accountID); err != nil {
			return err
		} else if acc.MultiCurrency && !stored.MultiCurrency {
			for _, r := range rows {
				if r.Currency != acc.Currency {
					if err := s.accounts.EnableMultiCurrency(ctx, u.ActiveBudgetID, acc.ID); err != nil {
						return err
					}
					break
				}
			}
		}
		for _, r := range rows {
			abs := r.AmountMinor
			if abs < 0 {
				abs = -abs
			}
			if abs == 0 || abs > maxAmountMinor || !acc.Allows(r.Currency) {
				return fmt.Errorf("row amount or currency: %w", models.ErrInvalidImport)
			}
			note, err := validNote(r.Description)
			if err != nil {
				return err
			}
			if r.ExternalID != "" {
				if s.wallets == nil || len(r.ExternalID) > maxExternalIDLen {
					return fmt.Errorf("external id: %w", models.ErrInvalidImport)
				}
				fresh, err := s.wallets.MarkImported(ctx, acc.ID, r.ExternalID)
				if err != nil {
					return err
				}
				if !fresh {
					continue
				}
			}
			t := models.Transaction{
				BudgetID: u.ActiveBudgetID, AccountID: acc.ID, Kind: models.KindIncome,
				AmountMinor: r.AmountMinor, Currency: r.Currency, OccurredOn: r.Date, Note: note, CreatedBy: u.ID,
			}
			if r.AmountMinor < 0 {
				cat, err := s.usableCategory(ctx, u, r.CategoryID)
				if err != nil {
					return err
				}
				t.Kind, t.CategoryID = models.KindExpense, &cat.ID
			} else if r.CategoryID != nil {
				return fmt.Errorf("income with category: %w", models.ErrInvalidImport)
			}
			saved, err := s.transactions.Insert(ctx, t)
			if err != nil {
				return err
			}
			ids = append(ids, saved.ID)
			if t.CategoryID != nil && r.ExternalID == "" {
				for _, k := range ruleKeys(r) {
					learned[k] = *t.CategoryID
				}
			}
		}
		if s.importRules != nil && len(learned) > 0 {
			return s.importRules.Save(ctx, u.ActiveBudgetID, learned)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return ids, nil
}

// UndoImport удаляет записи, только что созданные импортом (расходы и доходы), одной транзакцией БД. Чужие записи и
// записи другого вида отклоняются целиком.
func (s *Service) UndoImport(ctx context.Context, u models.User, ids []int64) error {
	if len(ids) == 0 || len(ids) > maxImportRows {
		return fmt.Errorf("ids count: %w", models.ErrInvalidImport)
	}
	return s.db.InTx(ctx, func(ctx context.Context) error {
		for _, id := range ids {
			t, err := s.transactions.Get(ctx, u.ActiveBudgetID, id)
			if err != nil {
				return err
			}
			if t.Kind != models.KindExpense && t.Kind != models.KindIncome {
				return fmt.Errorf("transaction %d: %w", id, models.ErrInvalidImport)
			}
			if err := s.transactions.Delete(ctx, u.ActiveBudgetID, id, u.ID); err != nil {
				return err
			}
		}
		return nil
	})
}

// parseStatementAmount читает сумму из ячейки выписки. Знак отбрасывается (направление задаёт вызывающий), пробелы и
// апострофы это разделители тысяч. Последний «.» или «,» десятичный, если после него не больше units цифр, иначе
// все разделители считаются тысячными.
func parseStatementAmount(raw string, units int) (int64, error) {
	var digits strings.Builder
	lastSep, afterSep := -1, 0
	for _, r := range raw {
		switch {
		case r >= '0' && r <= '9':
			digits.WriteRune(r)
			if lastSep >= 0 {
				afterSep++
			}
		case r == '.' || r == ',':
			lastSep, afterSep = digits.Len(), 0
		case strings.ContainsRune("-+−   '’–", r):
		default:
			return 0, fmt.Errorf("amount %q: %w", raw, models.ErrInvalidImport)
		}
	}
	all := digits.String()
	whole, frac := all, ""
	if lastSep >= 0 && afterSep <= units {
		whole, frac = all[:lastSep], all[lastSep:]
	}
	if whole == "" || len(whole) > 12 || len(frac) > units {
		return 0, fmt.Errorf("amount %q: %w", raw, models.ErrInvalidImport)
	}
	frac += strings.Repeat("0", units-len(frac))

	var total int64
	for _, r := range whole + frac {
		total = total*10 + int64(r-'0')
	}
	if total == 0 {
		return 0, fmt.Errorf("amount %q: %w", raw, models.ErrInvalidImport)
	}
	return total, nil
}

func truncateRunes(s string, max int) string {
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	return string([]rune(s)[:max])
}
