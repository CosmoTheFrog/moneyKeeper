//go:build integration

package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"moneykeeper/internal/models"
	"moneykeeper/pkg/apptime"
)

// В Тбилиси (UTC+4) в этот момент уже 12 октября.
var ledgerNow = time.Date(2026, 10, 11, 21, 30, 0, 0, time.UTC)

type ledger struct {
	s    *Service
	u    models.User
	main models.Account // GEL, остаток 1000,00
	cash models.Account // GEL, без остатка
	eur  models.Account // EUR, без остатка
}

func newLedger(t *testing.T) ledger {
	t.Helper()
	s, _ := newService(t)
	s.now = func() time.Time { return ledgerNow }
	ctx := context.Background()

	u, err := s.Resolve(ctx, tgIgor, "Asia/Tbilisi")
	if err != nil {
		t.Fatal(err)
	}
	mk := func(name, kind, cur string, opening int64) models.Account {
		a, err := s.CreateAccount(ctx, u, name, kind, cur, opening)
		if err != nil {
			t.Fatalf("create account %s: %v", name, err)
		}
		return a
	}
	return ledger{s: s, u: u, main: mk("TBC", "debit", "GEL", 100000), cash: mk("Наличные", "cash", "GEL", 0), eur: mk("EUR", "cash", "EUR", 0)}
}

func (l ledger) category(t *testing.T, name string) int64 {
	t.Helper()
	cats, err := l.s.ListCategories(context.Background(), l.u)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range cats {
		if c.Name == name {
			return c.ID
		}
	}
	t.Fatalf("category %q not found", name)
	return 0
}

func (l ledger) balance(t *testing.T, accountID int64) int64 {
	t.Helper()
	list, err := l.s.ListAccounts(context.Background(), l.u)
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range list {
		if a.ID == accountID {
			return a.BalanceMinor
		}
	}
	t.Fatalf("account %d not found", accountID)
	return 0
}

func (l ledger) expense(t *testing.T, accountID int64, amount int64, category string) models.Transaction {
	t.Helper()
	cat := l.category(t, category)
	txs, err := l.s.CreateTransaction(context.Background(), l.u, models.NewTransaction{
		Kind: models.KindExpense, AccountID: accountID, CategoryID: &cat, AmountMinor: amount,
	})
	if err != nil {
		t.Fatalf("create expense: %v", err)
	}
	return txs[0]
}

func TestSeedCategories(t *testing.T) {
	l := newLedger(t)
	cats, err := l.s.ListCategories(context.Background(), l.u)
	if err != nil {
		t.Fatal(err)
	}
	if len(cats) != 15 {
		t.Fatalf("got %d categories, want 15", len(cats))
	}
	byName := map[string]models.CategoryGroup{}
	for _, c := range cats {
		byName[c.Name] = c.Group
		if c.Description == "" || c.Hidden {
			t.Errorf("category %q: description %q hidden %v", c.Name, c.Description, c.Hidden)
		}
	}
	want := map[string]models.CategoryGroup{
		"Аренда и ипотека": models.GroupCritical, "Погашение долгов": models.GroupDebt,
		"Продукты": models.GroupDaily, "Сбережения": models.GroupGoal,
	}
	for name, g := range want {
		if byName[name] != g {
			t.Errorf("%q in group %q, want %q", name, byName[name], g)
		}
	}
}

func TestCategoryManagement(t *testing.T) {
	ctx := context.Background()
	l := newLedger(t)

	created, err := l.s.CreateCategory(ctx, l.u, "  Питомцы ", "daily")
	if err != nil || created.Name != "Питомцы" || created.Group != models.GroupDaily {
		t.Fatalf("create = %+v, %v", created, err)
	}
	if _, err := l.s.CreateCategory(ctx, l.u, "Питомцы", "daily"); !errors.Is(err, models.ErrInvalidCategory) {
		t.Errorf("duplicate name: err = %v, want ErrInvalidCategory", err)
	}
	if _, err := l.s.CreateCategory(ctx, l.u, "Х", "weird"); !errors.Is(err, models.ErrInvalidCategory) {
		t.Errorf("unknown group: err = %v, want ErrInvalidCategory", err)
	}
	if _, err := l.s.CreateCategory(ctx, l.u, "  ", "daily"); !errors.Is(err, models.ErrInvalidCategory) {
		t.Errorf("empty name: err = %v, want ErrInvalidCategory", err)
	}

	name, hidden, desc := "Домашние животные", true, " Корм и ветеринар "
	updated, err := l.s.UpdateCategory(ctx, l.u, created.ID, models.CategoryPatch{Name: &name, Hidden: &hidden, Description: &desc})
	if err != nil || updated.Name != name || !updated.Hidden || updated.Description != "Корм и ветеринар" {
		t.Fatalf("update = %+v, %v", updated, err)
	}

	// Скрытой категорией новую трату записать нельзя.
	_, err = l.s.CreateTransaction(ctx, l.u, models.NewTransaction{
		Kind: models.KindExpense, AccountID: l.main.ID, CategoryID: &created.ID, AmountMinor: 100,
	})
	if !errors.Is(err, models.ErrInvalidTransaction) {
		t.Errorf("hidden category: err = %v, want ErrInvalidTransaction", err)
	}
}

func TestOpeningBalance(t *testing.T) {
	ctx := context.Background()
	l := newLedger(t)

	if got := l.balance(t, l.main.ID); got != 100000 {
		t.Errorf("balance = %d, want 100000", got)
	}
	txs, err := l.s.ListTransactions(ctx, l.u, 0, 0)
	if err != nil || len(txs) != 1 {
		t.Fatalf("list = %+v, %v", txs, err)
	}
	if txs[0].Kind != models.KindOpening || txs[0].AmountMinor != 100000 || txs[0].OccurredOn.String() != "2026-10-12" {
		t.Errorf("opening = %+v, want opening 100000 on 2026-10-12 (user's today)", txs[0])
	}
	if _, err := l.s.CreateAccount(ctx, l.u, "Минус", "debit", "GEL", -1); !errors.Is(err, models.ErrInvalidAccount) {
		t.Errorf("negative opening: err = %v, want ErrInvalidAccount", err)
	}
}

func TestExpenseAndIncomeChangeBalance(t *testing.T) {
	ctx := context.Background()
	l := newLedger(t)

	exp := l.expense(t, l.main.ID, 5000, "Продукты")
	if exp.AmountMinor != -5000 || exp.Kind != models.KindExpense || exp.Currency != "GEL" || exp.OccurredOn.String() != "2026-10-12" {
		t.Errorf("expense = %+v", exp)
	}
	day, _ := apptime.ParseDay("2026-10-01")
	inc, err := l.s.CreateTransaction(ctx, l.u, models.NewTransaction{
		Kind: models.KindIncome, AccountID: l.main.ID, AmountMinor: 20000, OccurredOn: &day, Note: "  зарплата ",
	})
	if err != nil {
		t.Fatal(err)
	}
	if inc[0].AmountMinor != 20000 || inc[0].Note != "зарплата" || inc[0].CategoryID != nil {
		t.Errorf("income = %+v", inc[0])
	}
	if got := l.balance(t, l.main.ID); got != 100000-5000+20000 {
		t.Errorf("balance = %d, want %d", got, 100000-5000+20000)
	}

	// Порядок: сначала по дню (новые выше), внутри дня по номеру. Расход и остаток 12 октября, доход 1 октября.
	txs, err := l.s.ListTransactions(ctx, l.u, 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(txs) != 3 || txs[0].ID != exp.ID || txs[1].Kind != models.KindOpening || txs[2].ID != inc[0].ID {
		t.Errorf("order = %+v", txs)
	}
	page, err := l.s.ListTransactions(ctx, l.u, 1, 1)
	if err != nil || len(page) != 1 || page[0].Kind != models.KindOpening {
		t.Errorf("page = %+v, %v", page, err)
	}
}

func (l ledger) spent(t *testing.T, accountID int64) int64 {
	t.Helper()
	list, err := l.s.ListAccounts(context.Background(), l.u)
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range list {
		if a.ID == accountID {
			return a.SpentMinor
		}
	}
	t.Fatalf("account %d not found", accountID)
	return 0
}

func TestSpentCountsOnlyExpenses(t *testing.T) {
	ctx := context.Background()
	l := newLedger(t)

	if got := l.spent(t, l.main.ID); got != 0 {
		t.Fatalf("spent on a fresh account = %d, want 0 (opening balance is not a spend)", got)
	}

	first := l.expense(t, l.main.ID, 5000, "Продукты")
	l.expense(t, l.main.ID, 2500, "Одежда")
	if _, err := l.s.CreateTransaction(ctx, l.u, models.NewTransaction{Kind: models.KindIncome, AccountID: l.main.ID, AmountMinor: 20000}); err != nil {
		t.Fatal(err)
	}
	if _, err := l.s.CreateTransaction(ctx, l.u, models.NewTransaction{Kind: models.KindTransfer, AccountID: l.main.ID, ToAccountID: &l.cash.ID, AmountMinor: 3000}); err != nil {
		t.Fatal(err)
	}
	if got := l.spent(t, l.main.ID); got != 7500 {
		t.Errorf("spent = %d, want 7500 (income and transfer are not spends)", got)
	}
	if got := l.spent(t, l.cash.ID); got != 0 {
		t.Errorf("spent on the receiving account = %d, want 0", got)
	}

	// Правка суммы, перенос на другой счёт и удаление отражаются в «потрачено».
	amount := int64(6000)
	if _, err := l.s.UpdateTransaction(ctx, l.u, first.ID, models.TransactionPatch{AmountMinor: &amount}); err != nil {
		t.Fatal(err)
	}
	if got := l.spent(t, l.main.ID); got != 8500 {
		t.Errorf("spent after edit = %d, want 8500", got)
	}
	if _, err := l.s.UpdateTransaction(ctx, l.u, first.ID, models.TransactionPatch{AccountID: &l.cash.ID}); err != nil {
		t.Fatal(err)
	}
	if l.spent(t, l.main.ID) != 2500 || l.spent(t, l.cash.ID) != 6000 {
		t.Errorf("after move: main %d, cash %d, want 2500 and 6000", l.spent(t, l.main.ID), l.spent(t, l.cash.ID))
	}
	if err := l.s.DeleteTransaction(ctx, l.u, first.ID); err != nil {
		t.Fatal(err)
	}
	if got := l.spent(t, l.cash.ID); got != 0 {
		t.Errorf("spent after delete = %d, want 0", got)
	}
}

func TestCreateTransactionValidation(t *testing.T) {
	l := newLedger(t)
	food := l.category(t, "Продукты")
	other := l.cash.ID
	longNote := ""
	for range 201 {
		longNote += "я"
	}

	tests := []struct {
		name    string
		in      models.NewTransaction
		wantErr error
	}{
		{"zero amount", models.NewTransaction{Kind: models.KindExpense, AccountID: l.main.ID, CategoryID: &food}, models.ErrInvalidTransaction},
		{"negative amount", models.NewTransaction{Kind: models.KindExpense, AccountID: l.main.ID, CategoryID: &food, AmountMinor: -5}, models.ErrInvalidTransaction},
		{"too large", models.NewTransaction{Kind: models.KindExpense, AccountID: l.main.ID, CategoryID: &food, AmountMinor: maxAmountMinor + 1}, models.ErrInvalidTransaction},
		{"expense without category", models.NewTransaction{Kind: models.KindExpense, AccountID: l.main.ID, AmountMinor: 100}, models.ErrInvalidTransaction},
		{"income with category", models.NewTransaction{Kind: models.KindIncome, AccountID: l.main.ID, CategoryID: &food, AmountMinor: 100}, models.ErrInvalidTransaction},
		{"transfer without target", models.NewTransaction{Kind: models.KindTransfer, AccountID: l.main.ID, AmountMinor: 100}, models.ErrInvalidTransaction},
		{"transfer to itself", models.NewTransaction{Kind: models.KindTransfer, AccountID: l.main.ID, ToAccountID: &l.main.ID, AmountMinor: 100}, models.ErrInvalidTransaction},
		{"transfer with category", models.NewTransaction{Kind: models.KindTransfer, AccountID: l.main.ID, ToAccountID: &other, CategoryID: &food, AmountMinor: 100}, models.ErrInvalidTransaction},
		{"opening is not creatable", models.NewTransaction{Kind: models.KindOpening, AccountID: l.main.ID, AmountMinor: 100}, models.ErrInvalidTransaction},
		{"unknown kind", models.NewTransaction{Kind: "gift", AccountID: l.main.ID, AmountMinor: 100}, models.ErrInvalidTransaction},
		{"note too long", models.NewTransaction{Kind: models.KindIncome, AccountID: l.main.ID, AmountMinor: 100, Note: longNote}, models.ErrInvalidTransaction},
		{"unknown account", models.NewTransaction{Kind: models.KindIncome, AccountID: 9999, AmountMinor: 100}, models.ErrNotFound},
		{"unknown category", models.NewTransaction{Kind: models.KindExpense, AccountID: l.main.ID, CategoryID: ptr(int64(9999)), AmountMinor: 100}, models.ErrNotFound},
		{"transfer between currencies", models.NewTransaction{Kind: models.KindTransfer, AccountID: l.main.ID, ToAccountID: &l.eur.ID, AmountMinor: 100}, models.ErrCurrencyMismatch},
		{"ok", models.NewTransaction{Kind: models.KindExpense, AccountID: l.main.ID, CategoryID: &food, AmountMinor: 100}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := l.s.CreateTransaction(context.Background(), l.u, tt.in)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
		})
	}
	// Ни одна неудачная запись не оставила следов в остатке.
	if got := l.balance(t, l.main.ID); got != 100000-100 {
		t.Errorf("balance = %d, want %d (only the ok case is saved)", got, 100000-100)
	}
}

func TestTransfer(t *testing.T) {
	ctx := context.Background()
	l := newLedger(t)

	legs, err := l.s.CreateTransaction(ctx, l.u, models.NewTransaction{
		Kind: models.KindTransfer, AccountID: l.main.ID, ToAccountID: &l.cash.ID, AmountMinor: 30000, Note: "на карманные",
	})
	if err != nil || len(legs) != 2 {
		t.Fatalf("transfer = %+v, %v", legs, err)
	}
	out, in := legs[0], legs[1]
	if out.AmountMinor != -30000 || out.AccountID != l.main.ID || in.AmountMinor != 30000 || in.AccountID != l.cash.ID {
		t.Errorf("legs = %+v", legs)
	}
	if out.TransferID == nil || in.TransferID == nil || *out.TransferID != *in.TransferID || out.CategoryID != nil {
		t.Errorf("transfer ids = %v %v", out.TransferID, in.TransferID)
	}
	if got := l.balance(t, l.main.ID); got != 70000 {
		t.Errorf("main = %d, want 70000", got)
	}
	if got := l.balance(t, l.cash.ID); got != 30000 {
		t.Errorf("cash = %d, want 30000", got)
	}

	// Правка суммы меняет обе записи, счёт у перевода менять нельзя.
	amount := int64(10000)
	if _, err := l.s.UpdateTransaction(ctx, l.u, out.ID, models.TransactionPatch{AmountMinor: &amount}); err != nil {
		t.Fatal(err)
	}
	if l.balance(t, l.main.ID) != 90000 || l.balance(t, l.cash.ID) != 10000 {
		t.Errorf("after update: main %d, cash %d, want 90000 and 10000", l.balance(t, l.main.ID), l.balance(t, l.cash.ID))
	}
	if _, err := l.s.UpdateTransaction(ctx, l.u, in.ID, models.TransactionPatch{AccountID: &l.eur.ID}); !errors.Is(err, models.ErrInvalidTransaction) {
		t.Errorf("change transfer account: err = %v, want ErrInvalidTransaction", err)
	}

	// Удаление за любую ногу убирает обе.
	if err := l.s.DeleteTransaction(ctx, l.u, in.ID); err != nil {
		t.Fatal(err)
	}
	if l.balance(t, l.main.ID) != 100000 || l.balance(t, l.cash.ID) != 0 {
		t.Errorf("after delete: main %d, cash %d", l.balance(t, l.main.ID), l.balance(t, l.cash.ID))
	}
}

func TestUpdateTransaction(t *testing.T) {
	ctx := context.Background()
	l := newLedger(t)
	exp := l.expense(t, l.main.ID, 5000, "Продукты")
	transport := l.category(t, "Транспорт и бензин")
	day, _ := apptime.ParseDay("2026-09-30")
	amount, note := int64(7500), "  такси  "

	got, err := l.s.UpdateTransaction(ctx, l.u, exp.ID, models.TransactionPatch{
		AmountMinor: &amount, CategoryID: &transport, AccountID: &l.cash.ID, OccurredOn: &day, Note: &note,
	})
	if err != nil || len(got) != 1 {
		t.Fatalf("update = %+v, %v", got, err)
	}
	u := got[0]
	if u.AmountMinor != -7500 || *u.CategoryID != transport || u.AccountID != l.cash.ID || u.OccurredOn != day || u.Note != "такси" {
		t.Errorf("updated = %+v", u)
	}
	if l.balance(t, l.main.ID) != 100000 || l.balance(t, l.cash.ID) != -7500 {
		t.Errorf("balances: main %d, cash %d", l.balance(t, l.main.ID), l.balance(t, l.cash.ID))
	}

	t.Run("history keeps before and after", func(t *testing.T) {
		events, err := l.s.TransactionEvents(ctx, l.u, exp.ID)
		if err != nil || len(events) != 2 {
			t.Fatalf("events = %+v, %v", events, err)
		}
		create, update := events[0], events[1]
		if create.Action != models.EventCreate || create.Before != nil || create.After == nil || create.After.AmountMinor != -5000 {
			t.Errorf("create event = %+v", create)
		}
		if update.Action != models.EventUpdate || update.Before.AmountMinor != -5000 || update.After.AmountMinor != -7500 ||
			update.Before.AccountID != l.main.ID || update.After.AccountID != l.cash.ID ||
			update.After.OccurredOn != day || update.After.Note != "такси" {
			t.Errorf("update event = before %+v after %+v", update.Before, update.After)
		}
	})

	t.Run("rejected patches change nothing", func(t *testing.T) {
		inc, err := l.s.CreateTransaction(ctx, l.u, models.NewTransaction{Kind: models.KindIncome, AccountID: l.main.ID, AmountMinor: 100})
		if err != nil {
			t.Fatal(err)
		}
		zero := int64(0)
		tests := []struct {
			name    string
			id      int64
			patch   models.TransactionPatch
			wantErr error
		}{
			{"category on income", inc[0].ID, models.TransactionPatch{CategoryID: &transport}, models.ErrInvalidTransaction},
			{"zero amount", exp.ID, models.TransactionPatch{AmountMinor: &zero}, models.ErrInvalidTransaction},
			{"unknown account", exp.ID, models.TransactionPatch{AccountID: ptr(int64(9999))}, models.ErrNotFound},
			{"unknown transaction", 9999, models.TransactionPatch{Note: &note}, models.ErrNotFound},
			{"unknown category", exp.ID, models.TransactionPatch{CategoryID: ptr(int64(9999))}, models.ErrNotFound},
		}
		for _, tt := range tests {
			if _, err := l.s.UpdateTransaction(ctx, l.u, tt.id, tt.patch); !errors.Is(err, tt.wantErr) {
				t.Errorf("%s: err = %v, want %v", tt.name, err, tt.wantErr)
			}
		}
		events, _ := l.s.TransactionEvents(ctx, l.u, exp.ID)
		if len(events) != 2 {
			t.Errorf("events after rejected patches = %d, want 2", len(events))
		}
	})

	t.Run("switching to an account in another currency changes the record's currency", func(t *testing.T) {
		got, err := l.s.UpdateTransaction(ctx, l.u, exp.ID, models.TransactionPatch{AccountID: &l.eur.ID})
		if err != nil || len(got) != 1 {
			t.Fatalf("update = %+v, %v", got, err)
		}
		if got[0].Currency != "EUR" || got[0].AccountID != l.eur.ID || got[0].AmountMinor != -7500 {
			t.Errorf("record = %+v, want EUR on the EUR account, amount unchanged", got[0])
		}
		// Запись ушла со счёта в лари и легла на счёт в евро.
		if l.balance(t, l.cash.ID) != 0 || l.balance(t, l.eur.ID) != -7500 {
			t.Errorf("balances: cash %d, eur %d, want 0 and -7500", l.balance(t, l.cash.ID), l.balance(t, l.eur.ID))
		}
		list, _ := l.s.ListTransactions(ctx, l.u, 10, 0)
		for _, tx := range list {
			if tx.ID == exp.ID && tx.Currency != "EUR" {
				t.Errorf("listed currency = %s, want EUR", tx.Currency)
			}
		}
	})

	t.Run("a transfer or an opening balance cannot change account", func(t *testing.T) {
		legs, err := l.s.CreateTransaction(ctx, l.u, models.NewTransaction{
			Kind: models.KindTransfer, AccountID: l.main.ID, ToAccountID: &l.cash.ID, AmountMinor: 100,
		})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := l.s.UpdateTransaction(ctx, l.u, legs[0].ID, models.TransactionPatch{AccountID: &l.eur.ID}); !errors.Is(err, models.ErrInvalidTransaction) {
			t.Errorf("transfer: err = %v, want ErrInvalidTransaction", err)
		}
		all, _ := l.s.ListTransactions(ctx, l.u, 50, 0)
		for _, tx := range all {
			if tx.Kind != models.KindOpening {
				continue
			}
			if _, err := l.s.UpdateTransaction(ctx, l.u, tx.ID, models.TransactionPatch{AccountID: &l.eur.ID}); !errors.Is(err, models.ErrInvalidTransaction) {
				t.Errorf("opening: err = %v, want ErrInvalidTransaction", err)
			}
			break
		}
	})
}

func TestDeleteTransaction(t *testing.T) {
	ctx := context.Background()
	l := newLedger(t)
	exp := l.expense(t, l.main.ID, 5000, "Продукты")

	if err := l.s.DeleteTransaction(ctx, l.u, exp.ID); err != nil {
		t.Fatal(err)
	}
	if got := l.balance(t, l.main.ID); got != 100000 {
		t.Errorf("balance = %d, want 100000", got)
	}
	txs, _ := l.s.ListTransactions(ctx, l.u, 0, 0)
	if len(txs) != 1 {
		t.Errorf("list has %d records, want only the opening one", len(txs))
	}
	if err := l.s.DeleteTransaction(ctx, l.u, exp.ID); !errors.Is(err, models.ErrNotFound) {
		t.Errorf("second delete: err = %v, want ErrNotFound", err)
	}
	if _, err := l.s.UpdateTransaction(ctx, l.u, exp.ID, models.TransactionPatch{}); !errors.Is(err, models.ErrNotFound) {
		t.Errorf("update deleted: err = %v, want ErrNotFound", err)
	}

	// История удалённой записи остаётся.
	events, err := l.s.TransactionEvents(ctx, l.u, exp.ID)
	if err != nil || len(events) != 2 || events[1].Action != models.EventDelete || events[1].Before.AmountMinor != -5000 || events[1].After != nil {
		t.Errorf("events = %+v, %v", events, err)
	}
}

func TestLedgerIsScopedToBudget(t *testing.T) {
	ctx := context.Background()
	l := newLedger(t)
	exp := l.expense(t, l.main.ID, 5000, "Продукты")
	food := l.category(t, "Продукты")

	stranger, err := l.s.Resolve(ctx, models.TelegramUser{ID: 7, FirstName: "Anna"}, "UTC")
	if err != nil {
		t.Fatal(err)
	}
	note := "чужая"
	checks := map[string]error{}
	_, checks["update"] = l.s.UpdateTransaction(ctx, stranger, exp.ID, models.TransactionPatch{Note: &note})
	checks["delete"] = l.s.DeleteTransaction(ctx, stranger, exp.ID)
	_, checks["events"] = l.s.TransactionEvents(ctx, stranger, exp.ID)
	_, checks["create on foreign account"] = l.s.CreateTransaction(ctx, stranger, models.NewTransaction{
		Kind: models.KindIncome, AccountID: l.main.ID, AmountMinor: 100,
	})
	_, checks["foreign category"] = l.s.UpdateCategory(ctx, stranger, food, models.CategoryPatch{Hidden: ptr(true)})
	for name, err := range checks {
		if !errors.Is(err, models.ErrNotFound) {
			t.Errorf("%s: err = %v, want ErrNotFound", name, err)
		}
	}
	if txs, _ := l.s.ListTransactions(ctx, stranger, 0, 0); len(txs) != 0 {
		t.Errorf("stranger sees %d transactions", len(txs))
	}
	if l.balance(t, l.main.ID) != 95000 {
		t.Errorf("owner's balance changed: %d", l.balance(t, l.main.ID))
	}
}

func ptr[T any](v T) *T { return &v }
