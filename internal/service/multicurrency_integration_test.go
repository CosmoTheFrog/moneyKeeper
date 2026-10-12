//go:build integration

package service

import (
	"context"
	"errors"
	"testing"

	"moneykeeper/internal/models"
	"moneykeeper/pkg/money"
)

func (l ledger) account(t *testing.T, id int64) models.Account {
	t.Helper()
	list, err := l.s.ListAccounts(context.Background(), l.u)
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range list {
		if a.ID == id {
			return a
		}
	}
	t.Fatalf("account %d not found", id)
	return models.Account{}
}

func newMulti(t *testing.T, l ledger, name, mainCurrency string, opening int64) models.Account {
	t.Helper()
	a, err := l.s.CreateAccount(context.Background(), l.u, name, "cash", mainCurrency, true, opening)
	if err != nil {
		t.Fatalf("create multi account: %v", err)
	}
	return a
}

func TestMultiCurrencyAccountHoldsSeveralCurrencies(t *testing.T) {
	ctx := context.Background()
	l := newLedger(t)
	wallet := newMulti(t, l, "Binance", "USDT", 100000)
	if !wallet.MultiCurrency || wallet.Currency != money.USDT {
		t.Fatalf("wallet = %+v", wallet)
	}
	food := l.category(t, "Продукты")

	// Коды вне списка поддерживаемых отклоняются даже на многовалютном счёте.
	if _, err := l.s.CreateTransaction(ctx, l.u, models.NewTransaction{Kind: models.KindIncome, AccountID: wallet.ID, AmountMinor: 500, Currency: "XXX"}); !errors.Is(err, money.ErrUnknownCurrency) {
		t.Errorf("unknown currency: err = %v, want ErrUnknownCurrency", err)
	}

	if _, err := l.s.CreateTransaction(ctx, l.u, models.NewTransaction{Kind: models.KindIncome, AccountID: wallet.ID, AmountMinor: 25000, Currency: "EUR"}); err != nil {
		t.Fatalf("income in EUR: %v", err)
	}
	if _, err := l.s.CreateTransaction(ctx, l.u, models.NewTransaction{Kind: models.KindExpense, AccountID: wallet.ID, CategoryID: &food, AmountMinor: 1000, Currency: "EUR"}); err != nil {
		t.Fatalf("expense in EUR: %v", err)
	}
	if _, err := l.s.CreateTransaction(ctx, l.u, models.NewTransaction{Kind: models.KindExpense, AccountID: wallet.ID, CategoryID: &food, AmountMinor: 300}); err != nil {
		t.Fatalf("expense in the main currency: %v", err)
	}

	got := l.account(t, wallet.ID)
	want := []models.CurrencyBalance{
		{Currency: money.USDT, BalanceMinor: 100000 - 300, SpentMinor: 300}, // основная первой
		{Currency: money.EUR, BalanceMinor: 25000 - 1000, SpentMinor: 1000},
	}
	if len(got.Balances) != len(want) || got.Balances[0] != want[0] || got.Balances[1] != want[1] {
		t.Errorf("balances = %+v, want %+v", got.Balances, want)
	}
}

func TestPlainAccountRejectsOtherCurrencies(t *testing.T) {
	ctx := context.Background()
	l := newLedger(t)
	food := l.category(t, "Продукты")

	tests := []struct {
		name string
		in   models.NewTransaction
	}{
		{"income in another currency", models.NewTransaction{Kind: models.KindIncome, AccountID: l.main.ID, AmountMinor: 100, Currency: "EUR"}},
		{"expense in another currency", models.NewTransaction{Kind: models.KindExpense, AccountID: l.main.ID, CategoryID: &food, AmountMinor: 100, Currency: "USD"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := l.s.CreateTransaction(ctx, l.u, tt.in); !errors.Is(err, models.ErrCurrencyMismatch) {
				t.Errorf("err = %v, want ErrCurrencyMismatch", err)
			}
		})
	}
	// Явно названная основная валюта допустима.
	if _, err := l.s.CreateTransaction(ctx, l.u, models.NewTransaction{Kind: models.KindIncome, AccountID: l.main.ID, AmountMinor: 100, Currency: "GEL"}); err != nil {
		t.Errorf("main currency named explicitly: %v", err)
	}
}

func TestTransferBetweenAccountsNeedsBothToAllowTheCurrency(t *testing.T) {
	ctx := context.Background()
	l := newLedger(t)
	wallet := newMulti(t, l, "Revolut", "GEL", 0)

	// Из обычного GEL-счёта в многовалютный: валюта GEL допустима обоим.
	if _, err := l.s.CreateTransaction(ctx, l.u, models.NewTransaction{Kind: models.KindTransfer, AccountID: l.main.ID, ToAccountID: &wallet.ID, AmountMinor: 5000}); err != nil {
		t.Fatalf("GEL transfer: %v", err)
	}
	// Евро между обычным GEL-счётом и многовалютным нельзя: обычный не держит EUR.
	_, err := l.s.CreateTransaction(ctx, l.u, models.NewTransaction{Kind: models.KindTransfer, AccountID: wallet.ID, ToAccountID: &l.main.ID, AmountMinor: 100, Currency: "EUR"})
	if !errors.Is(err, models.ErrCurrencyMismatch) {
		t.Errorf("EUR to a GEL-only account: err = %v, want ErrCurrencyMismatch", err)
	}
	// А между двумя многовалютными можно в любой.
	other := newMulti(t, l, "Wise", "USD", 0)
	if _, err := l.s.CreateTransaction(ctx, l.u, models.NewTransaction{Kind: models.KindTransfer, AccountID: wallet.ID, ToAccountID: &other.ID, AmountMinor: 700, Currency: "EUR"}); err != nil {
		t.Fatalf("EUR between multi accounts: %v", err)
	}
	if got := l.account(t, other.ID).Balance(money.EUR); got != 700 {
		t.Errorf("EUR on the receiving account = %d, want 700", got)
	}
}

func TestChangingCurrencyOfARecord(t *testing.T) {
	ctx := context.Background()
	l := newLedger(t)
	wallet := newMulti(t, l, "Binance", "USDT", 0)
	exp := l.expense(t, wallet.ID, 4000, "Продукты")

	eur := "EUR"
	got, err := l.s.UpdateTransaction(ctx, l.u, exp.ID, models.TransactionPatch{Currency: &eur})
	if err != nil || got[0].Currency != money.EUR || got[0].AccountID != wallet.ID {
		t.Fatalf("currency on a multi account: %+v, %v", got, err)
	}
	if a := l.account(t, wallet.ID); a.Spent(money.EUR) != 4000 || a.Spent(money.USDT) != 0 {
		t.Errorf("spent after change: EUR %d, USDT %d, want 4000 and 0", a.Spent(money.EUR), a.Spent(money.USDT))
	}

	// На обычном счёте валюту записи выбрать нельзя.
	plain := l.expense(t, l.main.ID, 100, "Продукты")
	if _, err := l.s.UpdateTransaction(ctx, l.u, plain.ID, models.TransactionPatch{Currency: &eur}); !errors.Is(err, models.ErrCurrencyMismatch) {
		t.Errorf("currency on a plain account: err = %v, want ErrCurrencyMismatch", err)
	}
	// Переезд на многовалютный счёт сохраняет валюту записи, на обычный в другой валюте меняет её.
	if _, err := l.s.UpdateTransaction(ctx, l.u, plain.ID, models.TransactionPatch{AccountID: &wallet.ID}); err != nil {
		t.Fatal(err)
	}
	if a := l.account(t, wallet.ID); a.Spent(money.GEL) != 100 {
		t.Errorf("record moved with its currency: GEL spent = %d, want 100", a.Spent(money.GEL))
	}
	if got, err := l.s.UpdateTransaction(ctx, l.u, plain.ID, models.TransactionPatch{AccountID: &l.eur.ID}); err != nil || got[0].Currency != money.EUR {
		t.Errorf("move to a plain EUR account: %+v, %v", got, err)
	}

	// У перевода и начального остатка валюта не меняется.
	legs, err := l.s.CreateTransaction(ctx, l.u, models.NewTransaction{Kind: models.KindTransfer, AccountID: l.main.ID, ToAccountID: &l.cash.ID, AmountMinor: 100})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := l.s.UpdateTransaction(ctx, l.u, legs[0].ID, models.TransactionPatch{Currency: &eur}); !errors.Is(err, models.ErrInvalidTransaction) {
		t.Errorf("transfer currency: err = %v, want ErrInvalidTransaction", err)
	}
}
