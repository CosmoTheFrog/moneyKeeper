package bot

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"

	"moneykeeper/internal/models"
	"moneykeeper/internal/telegram"
	"moneykeeper/pkg/money"
)

type fakeService struct {
	user      *models.User
	budget    models.Budget
	cats      []models.Category
	recent    []models.Transaction
	created   []models.NewTransaction
	deleted   []int64
	nextTxID  int64
	noAccount bool
}

func (f *fakeService) FindUser(context.Context, int64) (models.User, error) {
	if f.user == nil {
		return models.User{}, models.ErrNotFound
	}
	return *f.user, nil
}
func (f *fakeService) Budget(context.Context, models.User) (models.Budget, error) {
	return f.budget, nil
}
func (f *fakeService) ListAccounts(context.Context, models.User, string) ([]models.Account, error) {
	if f.noAccount {
		return nil, nil
	}
	return []models.Account{{ID: 7, Currency: money.GEL, IsDefault: true}}, nil
}
func (f *fakeService) ListCategories(context.Context, models.User) ([]models.Category, error) {
	return f.cats, nil
}
func (f *fakeService) ListTransactions(context.Context, models.User, int, int, string) ([]models.Transaction, error) {
	return f.recent, nil
}
func (f *fakeService) CreateTransaction(_ context.Context, _ models.User, in models.NewTransaction) ([]models.Transaction, error) {
	f.created = append(f.created, in)
	f.nextTxID++
	return []models.Transaction{{ID: f.nextTxID}}, nil
}
func (f *fakeService) DeleteTransaction(_ context.Context, _ models.User, id int64) error {
	f.deleted = append(f.deleted, id)
	return nil
}

type sent struct {
	text    string
	buttons [][]telegram.Button
	edited  bool
}

type fakeAPI struct {
	out      []sent
	answered []string
}

func (f *fakeAPI) Updates(context.Context, int64) ([]telegram.Update, error) { return nil, nil }
func (f *fakeAPI) Send(_ context.Context, _ int64, text string) error {
	f.out = append(f.out, sent{text: text})
	return nil
}
func (f *fakeAPI) SendButtons(_ context.Context, _ int64, text string, rows [][]telegram.Button) (int64, error) {
	f.out = append(f.out, sent{text: text, buttons: rows})
	return 99, nil
}
func (f *fakeAPI) EditText(_ context.Context, _, _ int64, text string, rows [][]telegram.Button) error {
	f.out = append(f.out, sent{text: text, buttons: rows, edited: true})
	return nil
}
func (f *fakeAPI) AnswerCallback(_ context.Context, id string) error {
	f.answered = append(f.answered, id)
	return nil
}

func (f *fakeAPI) last(t *testing.T) sent {
	t.Helper()
	if len(f.out) == 0 {
		t.Fatal("no messages sent")
	}
	return f.out[len(f.out)-1]
}

func setup() (*Bot, *fakeService, *fakeAPI) {
	base := money.GEL
	svc := &fakeService{
		user:   &models.User{ID: 1},
		budget: models.Budget{BaseCurrency: &base, InterfaceMode: models.ModeSimple},
		cats:   []models.Category{{ID: 10, Name: "Продукты"}, {ID: 11, Name: "Кафе"}, {ID: 12, Name: "Скрытая", Hidden: true}},
	}
	api := &fakeAPI{}
	return New(svc, api, slog.New(slog.NewTextHandler(io.Discard, nil))), svc, api
}

func msg(text string) telegram.Update {
	return telegram.Update{Message: &telegram.Message{ID: 5, From: &telegram.User{ID: 42}, Chat: telegram.Chat{ID: 42, Type: "private"}, Text: text}}
}

func press(from int64, data string) telegram.Update {
	return telegram.Update{Callback: &telegram.CallbackQuery{ID: "cb", From: telegram.User{ID: from}, Data: data, Message: &telegram.Message{ID: 99, Chat: telegram.Chat{ID: from, Type: "private"}}}}
}

func TestUnknownOrUnfinishedUserGetsAHint(t *testing.T) {
	b, svc, api := setup()
	svc.user = nil
	b.Handle(context.Background(), msg("кофе 5"))
	if !strings.Contains(api.last(t).text, "приложение") || len(svc.created) != 0 {
		t.Errorf("unknown user: %+v created %v", api.out, svc.created)
	}

	b, svc, api = setup()
	svc.budget.InterfaceMode = ""
	b.Handle(context.Background(), msg("кофе 5"))
	if len(svc.created) != 0 || len(api.out) != 1 {
		t.Errorf("setup not finished: created %v, out %v", svc.created, api.out)
	}
}

func TestKnownNoteIsRecordedAtOnceAndCanBeUndone(t *testing.T) {
	b, svc, api := setup()
	cat := int64(11)
	svc.recent = []models.Transaction{{Kind: models.KindExpense, CategoryID: &cat, Note: "Кофе "}}

	b.Handle(context.Background(), msg("кофе 5"))
	if len(svc.created) != 1 {
		t.Fatalf("created = %v", svc.created)
	}
	c := svc.created[0]
	if c.Kind != models.KindExpense || c.AmountMinor != 500 || *c.CategoryID != 11 || c.AccountID != 7 || c.Note != "кофе" {
		t.Errorf("created = %+v", c)
	}
	got := api.last(t)
	if !strings.Contains(got.text, "5,00") || !strings.Contains(got.text, "Кафе") || got.buttons[0][0].Data != "u:1" {
		t.Errorf("reply = %+v", got)
	}

	b.Handle(context.Background(), press(42, "u:1"))
	if len(svc.deleted) != 1 || svc.deleted[0] != 1 || !api.last(t).edited {
		t.Errorf("undo: deleted %v, last %+v", svc.deleted, api.last(t))
	}
}

func TestNewNoteAsksForCategoryOnce(t *testing.T) {
	b, svc, api := setup()
	cat := int64(10)
	svc.recent = []models.Transaction{{Kind: models.KindExpense, CategoryID: &cat, Note: "хлеб"}}

	b.Handle(context.Background(), msg("такси 12 EUR"))
	if len(svc.created) != 0 {
		t.Fatalf("recorded before the category was chosen: %v", svc.created)
	}
	rows := api.last(t).buttons
	var data []string
	for _, r := range rows {
		for _, btn := range r {
			data = append(data, btn.Data)
		}
	}
	// Первой идёт категория из истории, скрытая не предлагается.
	if len(data) != 2 || !strings.HasPrefix(data[0], "c:") || !strings.HasSuffix(data[0], ":10") || strings.Contains(strings.Join(data, ","), ":12") {
		t.Fatalf("buttons = %v", data)
	}

	b.Handle(context.Background(), press(42, data[1]))
	if len(svc.created) != 1 || svc.created[0].Currency != "EUR" || svc.created[0].AmountMinor != 1200 || *svc.created[0].CategoryID != 11 {
		t.Fatalf("created = %+v", svc.created)
	}
	// Повторное нажатие и нажатие другим человеком ничего не записывают.
	b.Handle(context.Background(), press(42, data[1]))
	b.Handle(context.Background(), press(43, data[0]))
	if len(svc.created) != 1 {
		t.Errorf("created again: %v", svc.created)
	}
}

func TestIncomeAndNoAmount(t *testing.T) {
	b, svc, api := setup()
	b.Handle(context.Background(), msg("+1000 зарплата"))
	if len(svc.created) != 1 || svc.created[0].Kind != models.KindIncome || svc.created[0].AmountMinor != 100000 || svc.created[0].CategoryID != nil {
		t.Errorf("income = %+v", svc.created)
	}
	b.Handle(context.Background(), msg("просто текст"))
	if len(svc.created) != 1 || !strings.Contains(api.last(t).text, "сумм") {
		t.Errorf("no amount: created %v, last %+v", svc.created, api.last(t))
	}
}

func TestGroupChatsAreIgnored(t *testing.T) {
	b, svc, api := setup()
	u := msg("кофе 5")
	u.Message.Chat.Type = "group"
	b.Handle(context.Background(), u)
	if len(svc.created) != 0 || len(api.out) != 0 {
		t.Errorf("group message handled: %v %v", svc.created, api.out)
	}
}
