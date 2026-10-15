//go:build integration

package service

import (
	"context"
	"errors"
	"testing"

	"moneykeeper/internal/models"
)

func (l ledger) wallet(t *testing.T, name, main string, opening int64) models.Account {
	t.Helper()
	a, err := l.s.CreateAccount(context.Background(), l.u, name, "wallet", main, true, opening)
	if err != nil {
		t.Fatalf("create wallet %s: %v", name, err)
	}
	if !a.OffBudget {
		t.Fatalf("wallet %s must be off-budget: %+v", name, a)
	}
	return a
}

func (l ledger) balanceOf(t *testing.T, accountID int64, cur string) int64 {
	t.Helper()
	for _, b := range l.account(t, accountID).Balances {
		if string(b.Currency) == cur {
			return b.BalanceMinor
		}
	}
	return 0
}

func TestTransferAcrossTheBudgetBoundary(t *testing.T) {
	ctx := context.Background()
	l := newLedger(t)
	l.setBase(t, "GEL")
	food := l.category(t, "Продукты")
	w := l.wallet(t, "Binance", "USDT", 0)
	w2 := l.wallet(t, "Ledger", "USDT", 0)

	send := func(from, to int64, minor int64, cat *int64) error {
		_, err := l.s.CreateTransaction(ctx, l.u, models.NewTransaction{Kind: models.KindTransfer, AccountID: from, ToAccountID: &to, AmountMinor: minor, Currency: "GEL", CategoryID: cat})
		return err
	}

	// Из бюджета на кошелёк нужна категория, внутри бюджета и между кошельками её быть не должно.
	if err := send(l.main.ID, w.ID, 20000, nil); !errors.Is(err, models.ErrInvalidTransaction) {
		t.Errorf("into a wallet without category: err = %v, want ErrInvalidTransaction", err)
	}
	if err := send(l.main.ID, l.cash.ID, 1000, &food); !errors.Is(err, models.ErrInvalidTransaction) {
		t.Errorf("inside the budget with category: err = %v, want ErrInvalidTransaction", err)
	}
	if err := send(l.main.ID, w.ID, 20000, &food); err != nil {
		t.Fatalf("into a wallet: %v", err)
	}
	if err := send(w.ID, w2.ID, 5000, nil); err != nil {
		t.Fatalf("between wallets: %v", err)
	}
	if err := send(w.ID, l.main.ID, 3000, nil); err != nil {
		t.Fatalf("out of a wallet: %v", err)
	}

	// Расход конверта 200 ₾ и приход 30 ₾ при выводе; перевод между кошельками бюджет не трогает.
	b := l.budget(t, "2026-10")
	row := budgetRows(b.Rows).of(food)
	if row.SpentMinor != 20000 || b.IncomeMinor != 100000+3000 {
		t.Errorf("budget: food spent %d income %d, want 20000 and 103000", row.SpentMinor, b.IncomeMinor)
	}
	// Остаток на кошельке отдельно от бюджета: 200 − 50 − 30.
	if got := l.balanceOf(t, w.ID, "GEL"); got != 12000 {
		t.Errorf("wallet balance = %d, want 12000", got)
	}
}

func TestOffBudgetOpeningBalanceIsNotIncome(t *testing.T) {
	l := newLedger(t)
	l.setBase(t, "GEL")
	l.wallet(t, "Ledger", "BTC", 100000000)
	if b := l.budget(t, "2026-10"); b.IncomeMinor != 100000 || b.ToBudgetMinor != 100000 {
		t.Errorf("budget = income %d to-budget %d, want the 1000 ₾ of the main account only", b.IncomeMinor, b.ToBudgetMinor)
	}
}

func TestExchangeBetweenBudgetAccounts(t *testing.T) {
	ctx := context.Background()
	l := newLedger(t)
	l.setBase(t, "GEL")
	food := l.category(t, "Продукты")

	// 10,00 € ушли с евро-счёта, на лари-счёт пришло 31,50 ₾, комиссия банка 0,50 €.
	rows, err := l.s.CreateExchange(ctx, l.u, models.NewExchange{
		FromAccountID: l.eur.ID, ToAccountID: l.main.ID, OutMinor: 1000, OutCurrency: "EUR", InMinor: 3150, InCurrency: "GEL",
		FeeMinor: 50, FeeCategoryID: &food, Note: "для поездки",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 || rows[0].Kind != models.KindExchange || rows[0].AmountMinor != -1000 || rows[1].AmountMinor != 3150 ||
		rows[2].Kind != models.KindExpense || rows[2].AmountMinor != -50 || rows[2].CategoryID == nil {
		t.Fatalf("rows = %+v", rows)
	}
	if rows[0].TransferID == nil || *rows[0].TransferID != *rows[1].TransferID || *rows[0].TransferID != *rows[2].TransferID {
		t.Errorf("rows must share one group: %+v", rows)
	}
	if got := l.balanceOf(t, l.eur.ID, "EUR"); got != -1050 {
		t.Errorf("EUR balance = %d, want -1050", got)
	}
	if got := l.balanceOf(t, l.main.ID, "GEL"); got != 100000+3150 {
		t.Errorf("GEL balance = %d, want 103150", got)
	}

	// В бюджете только комиссия: 0,50 € по курсу 3,1594 это 1,58 ₾. Сам обмен между счетами бюджета не считается.
	b := l.budget(t, "2026-10")
	if r := budgetRows(b.Rows).of(food); r.SpentMinor != 158 || b.IncomeMinor != 100000 {
		t.Errorf("budget: food spent %d income %d, want 158 and 100000", r.SpentMinor, b.IncomeMinor)
	}

	// Дата и заметка правятся у всей группы, суммы нет.
	d := day(t, "2026-10-01")
	note := "новая заметка"
	if _, err := l.s.UpdateTransaction(ctx, l.u, rows[0].ID, models.TransactionPatch{OccurredOn: d, Note: &note}); err != nil {
		t.Fatal(err)
	}
	all, _ := l.s.ListTransactions(ctx, l.u, 0, 0, "")
	moved := 0
	for _, tx := range all {
		if tx.TransferID != nil && *tx.TransferID == *rows[0].TransferID && tx.OccurredOn == *d && tx.Note == note {
			moved++
		}
	}
	if moved != 3 {
		t.Errorf("%d of 3 rows moved with the exchange", moved)
	}
	amount := int64(500)
	if _, err := l.s.UpdateTransaction(ctx, l.u, rows[0].ID, models.TransactionPatch{AmountMinor: &amount}); !errors.Is(err, models.ErrInvalidTransaction) {
		t.Errorf("amount of an exchange: err = %v, want ErrInvalidTransaction", err)
	}

	// Удаление любой записи убирает всю группу вместе с комиссией.
	if err := l.s.DeleteTransaction(ctx, l.u, rows[1].ID); err != nil {
		t.Fatal(err)
	}
	if got := l.balanceOf(t, l.eur.ID, "EUR"); got != 0 {
		t.Errorf("EUR balance after delete = %d, want 0", got)
	}
	if got := l.balanceOf(t, l.main.ID, "GEL"); got != 100000 {
		t.Errorf("GEL balance after delete = %d, want 100000", got)
	}
}

func TestExchangeCoinPurchaseFromBudgetAndFromWallet(t *testing.T) {
	ctx := context.Background()
	l := newLedger(t)
	l.setBase(t, "GEL")
	food := l.category(t, "Продукты")
	usdt := l.wallet(t, "Binance", "USDT", 100000)

	// С карты на биржу USDT: деньги уходят из бюджета, нужна категория.
	buy := models.NewExchange{FromAccountID: l.main.ID, ToAccountID: usdt.ID, OutMinor: 27300, OutCurrency: "GEL", InMinor: 10000, InCurrency: "USDT"}
	if _, err := l.s.CreateExchange(ctx, l.u, buy); !errors.Is(err, models.ErrInvalidTransaction) {
		t.Errorf("leaving the budget without category: err = %v, want ErrInvalidTransaction", err)
	}
	buy.CategoryID = &food
	if _, err := l.s.CreateExchange(ctx, l.u, buy); err != nil {
		t.Fatal(err)
	}

	// На бирже USDT в BTC: оба счёта вне бюджета, категории нет, комиссия биржи только уменьшает портфель.
	coin := models.NewExchange{FromAccountID: usdt.ID, ToAccountID: usdt.ID, OutMinor: 50000, OutCurrency: "USDT", InMinor: 600000, InCurrency: "BTC", FeeMinor: 50}
	rows, err := l.s.CreateExchange(ctx, l.u, coin)
	if err != nil {
		t.Fatalf("coin purchase on the exchange: %v", err)
	}
	if len(rows) != 3 || rows[2].Kind != models.KindFee || rows[2].CategoryID != nil || rows[2].AmountMinor != -50 {
		t.Errorf("rows = %+v", rows)
	}
	if got := l.balanceOf(t, usdt.ID, "USDT"); got != 100000+10000-50000-50 {
		t.Errorf("USDT = %d, want 59950", got)
	}
	if got := l.balanceOf(t, usdt.ID, "BTC"); got != 600000 {
		t.Errorf("BTC = %d, want 600000", got)
	}

	// Бюджет увидел только первую покупку: 273 ₾ в конверт «Продукты».
	if r := budgetRows(l.budget(t, "2026-10").Rows).of(food); r.SpentMinor != 27300 {
		t.Errorf("food spent %d, want 27300", r.SpentMinor)
	}
}

func TestExchangeValidation(t *testing.T) {
	ctx := context.Background()
	l := newLedger(t)
	l.setBase(t, "GEL")
	food := l.category(t, "Продукты")
	w := l.wallet(t, "Binance", "USDT", 0)

	ok := models.NewExchange{FromAccountID: l.eur.ID, ToAccountID: l.main.ID, OutMinor: 1000, OutCurrency: "EUR", InMinor: 3000, InCurrency: "GEL"}
	with := func(f func(*models.NewExchange)) models.NewExchange { e := ok; f(&e); return e }

	tests := []struct {
		name string
		in   models.NewExchange
		want error
	}{
		{"same currency is a transfer", with(func(e *models.NewExchange) { e.InCurrency, e.ToAccountID = "EUR", l.eur.ID }), models.ErrInvalidTransaction},
		{"zero amount out", with(func(e *models.NewExchange) { e.OutMinor = 0 }), models.ErrInvalidTransaction},
		{"negative amount in", with(func(e *models.NewExchange) { e.InMinor = -1 }), models.ErrInvalidTransaction},
		{"too large", with(func(e *models.NewExchange) { e.OutMinor = maxAmountMinor + 1 }), models.ErrInvalidTransaction},
		{"negative fee", with(func(e *models.NewExchange) { e.FeeMinor = -1 }), models.ErrInvalidTransaction},
		{"fee on a budget account without category", with(func(e *models.NewExchange) { e.FeeMinor = 10 }), models.ErrInvalidTransaction},
		{"fee category on a wallet", with(func(e *models.NewExchange) {
			e.FromAccountID, e.OutCurrency, e.FeeMinor, e.FeeCategoryID = w.ID, "USDT", 10, &food
		}), models.ErrInvalidTransaction},
		{"category without leaving the budget", with(func(e *models.NewExchange) { e.CategoryID = &food }), models.ErrInvalidTransaction},
		{"plain account takes only its currency", with(func(e *models.NewExchange) { e.OutCurrency = "USD" }), models.ErrCurrencyMismatch},
		{"plain target takes only its currency", with(func(e *models.NewExchange) { e.InCurrency = "RUB" }), models.ErrCurrencyMismatch},
		{"unknown account", with(func(e *models.NewExchange) { e.ToAccountID = 999999 }), models.ErrNotFound},
		{"unknown currency", with(func(e *models.NewExchange) { e.InCurrency = "XXX" }), nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := l.s.CreateExchange(ctx, l.u, tt.in)
			if err == nil {
				t.Fatal("want an error")
			}
			if tt.want != nil && !errors.Is(err, tt.want) {
				t.Errorf("err = %v, want %v", err, tt.want)
			}
		})
	}
	// Ничего из отклонённого не записалось.
	if got := l.balanceOf(t, l.eur.ID, "EUR"); got != 0 {
		t.Errorf("EUR balance = %d after rejected exchanges", got)
	}
}

func TestWalletTransferWithNetworkFee(t *testing.T) {
	ctx := context.Background()
	l := newLedger(t)
	a := l.wallet(t, "Binance", "BTC", 100000000)
	b := l.wallet(t, "Ledger", "BTC", 0)

	send := func(amount, fee int64, from, to int64) ([]models.Transaction, error) {
		return l.s.CreateTransaction(ctx, l.u, models.NewTransaction{Kind: models.KindTransfer, AccountID: from, ToAccountID: &to, AmountMinor: amount, FeeMinor: fee})
	}
	rows, err := send(50000000, 10000, a.ID, b.ID)
	if err != nil {
		t.Fatal(err)
	}
	if rows[0].AmountMinor != -50000000 || rows[1].AmountMinor != 49990000 {
		t.Fatalf("legs = %d and %d", rows[0].AmountMinor, rows[1].AmountMinor)
	}
	if got := l.balanceOf(t, a.ID, "BTC"); got != 50000000 {
		t.Errorf("sender = %d, want 50000000", got)
	}
	if got := l.balanceOf(t, b.ID, "BTC"); got != 49990000 {
		t.Errorf("receiver = %d, want 49990000", got)
	}

	// Сумму перевода с комиссией не пересчитать одним числом; дата и заметка правятся.
	amount := int64(1)
	if _, err := l.s.UpdateTransaction(ctx, l.u, rows[0].ID, models.TransactionPatch{AmountMinor: &amount}); !errors.Is(err, models.ErrInvalidTransaction) {
		t.Errorf("amount of a transfer with a fee: err = %v, want ErrInvalidTransaction", err)
	}
	note := "холодное хранение"
	if _, err := l.s.UpdateTransaction(ctx, l.u, rows[0].ID, models.TransactionPatch{Note: &note}); err != nil {
		t.Errorf("note of a transfer with a fee: %v", err)
	}

	for name, fee := range map[string]int64{"negative": -1, "equal to the amount": 100, "bigger than the amount": 101} {
		if _, err := send(100, fee, a.ID, b.ID); !errors.Is(err, models.ErrInvalidTransaction) {
			t.Errorf("fee %s: err = %v, want ErrInvalidTransaction", name, err)
		}
	}
	// Комиссия сети только между кошельками вне бюджета.
	if _, err := send(100, 1, l.main.ID, l.cash.ID); !errors.Is(err, models.ErrInvalidTransaction) {
		t.Errorf("fee inside the budget: err = %v, want ErrInvalidTransaction", err)
	}
}

func TestPortfolioFromExchanges(t *testing.T) {
	ctx := context.Background()
	l := newLedger(t)
	l.setBase(t, "GEL")
	w := l.wallet(t, "Binance", "USDT", 100000)

	trade := func(outMinor int64, out string, inMinor int64, in string) {
		t.Helper()
		if _, err := l.s.CreateExchange(ctx, l.u, models.NewExchange{FromAccountID: w.ID, ToAccountID: w.ID, OutMinor: outMinor, OutCurrency: out, InMinor: inMinor, InCurrency: in}); err != nil {
			t.Fatal(err)
		}
	}
	// 80 USDT = 218,40 ₾ за 0,001 BTC, потом продана половина за 45 USDT = 122,85 ₾.
	trade(8000, "USDT", 100000, "BTC")
	trade(50000, "BTC", 4500, "USDT")

	p, err := l.s.Portfolio(ctx, l.u, "")
	if err != nil {
		t.Fatal(err)
	}
	if p.Currency != "GEL" || len(p.Items) != 1 || p.MissingRates != 0 {
		t.Fatalf("portfolio = %+v", p)
	}
	it := p.Items[0]
	// Осталось 0,0005 BTC стоимостью 125 ₾ при цене покупки 109,20 ₾ (половина от 218,40).
	if it.Currency != "BTC" || it.QuantityMinor != 50000 || it.ValueMinor != 12500 || it.CostMinor != 10920 || it.ProfitMinor != 1580 {
		t.Errorf("item = %+v", it)
	}
	if it.ProfitBP == nil || *it.ProfitBP != 1447 || it.RealizedMinor != 12285-10920 {
		t.Errorf("profit bp %v realized %d, want 1447 and 1365", it.ProfitBP, it.RealizedMinor)
	}
	if it.AvgPrice != "218400" {
		t.Errorf("avg price = %q, want 218400", it.AvgPrice)
	}

	// В другой валюте те же позиции по сегодняшнему курсу: 125 ₾ это 45,79 USDT.
	inUSDT, err := l.s.Portfolio(ctx, l.u, "USDT")
	if err != nil || inUSDT.Currency != "USDT" || inUSDT.Items[0].ValueMinor != 4579 {
		t.Errorf("in USDT = %+v, %v", inUSDT, err)
	}
	if _, err := l.s.Portfolio(ctx, l.u, "XXX"); err == nil {
		t.Error("unknown currency must fail")
	}

	// Без монет портфель пуст.
	empty := newLedger(t)
	if p, err := empty.s.Portfolio(ctx, empty.u, ""); err != nil || len(p.Items) != 0 {
		t.Errorf("empty portfolio = %+v, %v", p, err)
	}
}

func TestPiggyBank(t *testing.T) {
	ctx := context.Background()
	l := newLedger(t)
	l.setBase(t, "GEL")

	target := int64(250000)
	until := day(t, "2027-05-01")
	piggy, err := l.s.CreatePiggy(ctx, l.u, "Отпуск", "GEL", &target, until)
	if err != nil {
		t.Fatal(err)
	}
	if piggy.Kind != models.AccountPiggy || !piggy.OffBudget || piggy.IsDefault || piggy.CategoryID == nil ||
		piggy.TargetMinor == nil || *piggy.TargetMinor != 250000 || piggy.TargetDate == nil || *piggy.TargetDate != *until {
		t.Fatalf("piggy = %+v", piggy)
	}
	// Конверт копилки появился в группе «Цели накопления» и переносит остаток.
	cats, _ := l.s.ListCategories(ctx, l.u)
	var envelope models.Category
	for _, c := range cats {
		if c.ID == *piggy.CategoryID {
			envelope = c
		}
	}
	if envelope.Name != "Отпуск" || envelope.GroupID != l.group(t, models.GroupGoal) || !envelope.Rollover {
		t.Errorf("envelope = %+v", envelope)
	}
	// Основным счётом копилка не становится.
	if l.account(t, l.main.ID).IsDefault != true {
		t.Error("the main account lost its default flag")
	}

	// Пополнение: перевод со счёта бюджета в конверт копилки. Деньги уходят из конверта и сразу видны в копилке.
	if _, err := l.s.CreateTransaction(ctx, l.u, models.NewTransaction{Kind: models.KindTransfer, AccountID: l.main.ID, ToAccountID: &piggy.ID, AmountMinor: 30000, CategoryID: piggy.CategoryID}); err != nil {
		t.Fatal(err)
	}
	if got := l.balanceOf(t, piggy.ID, "GEL"); got != 30000 {
		t.Errorf("piggy = %d, want 30000", got)
	}
	if got := l.balanceOf(t, l.main.ID, "GEL"); got != 70000 {
		t.Errorf("source = %d, want 70000", got)
	}
	if r := budgetRows(l.budget(t, "2026-10").Rows).of(*piggy.CategoryID); r.SpentMinor != 30000 {
		t.Errorf("envelope spent %d, want 30000", r.SpentMinor)
	}

	// Копилка в другой валюте и с тем же названием использует уже созданный конверт.
	usdt, err := l.s.CreatePiggy(ctx, l.u, "отпуск", "USDT", nil, nil)
	if err != nil || usdt.CategoryID == nil || *usdt.CategoryID != *piggy.CategoryID || usdt.TargetMinor != nil {
		t.Errorf("second piggy = %+v, %v", usdt, err)
	}

	bad := map[string]func() (models.Account, error){
		"empty name":  func() (models.Account, error) { return l.s.CreatePiggy(ctx, l.u, "  ", "GEL", nil, nil) },
		"zero target": func() (models.Account, error) { z := int64(0); return l.s.CreatePiggy(ctx, l.u, "Х", "GEL", &z, nil) },
		"target too big": func() (models.Account, error) {
			z := int64(maxAmountMinor + 1)
			return l.s.CreatePiggy(ctx, l.u, "Х", "GEL", &z, nil)
		},
		"unknown currency": func() (models.Account, error) { return l.s.CreatePiggy(ctx, l.u, "Х", "XXX", nil, nil) },
	}
	for name, f := range bad {
		if _, err := f(); err == nil {
			t.Errorf("%s must fail", name)
		}
	}
	// Копилку нельзя завести обычным путём: без конверта она не пополняется.
	if _, err := l.s.CreateAccount(ctx, l.u, "Х", "piggy", "GEL", false, 0); !errors.Is(err, models.ErrInvalidAccount) {
		t.Errorf("piggy via CreateAccount: err = %v, want ErrInvalidAccount", err)
	}
}

func TestArchiveAccount(t *testing.T) {
	ctx := context.Background()
	l := newLedger(t)
	l.setBase(t, "GEL")
	piggy, err := l.s.CreatePiggy(ctx, l.u, "Отпуск", "GEL", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := l.s.CreateTransaction(ctx, l.u, models.NewTransaction{Kind: models.KindTransfer, AccountID: l.main.ID, ToAccountID: &piggy.ID, AmountMinor: 1000, CategoryID: piggy.CategoryID}); err != nil {
		t.Fatal(err)
	}

	if err := l.s.ArchiveAccount(ctx, l.u, piggy.ID); !errors.Is(err, models.ErrInvalidAccount) {
		t.Errorf("piggy with money: err = %v, want ErrInvalidAccount", err)
	}
	if err := l.s.ArchiveAccount(ctx, l.u, l.main.ID); !errors.Is(err, models.ErrInvalidAccount) {
		t.Errorf("default account: err = %v, want ErrInvalidAccount", err)
	}
	if err := l.s.ArchiveAccount(ctx, l.u, 999999); !errors.Is(err, models.ErrNotFound) {
		t.Errorf("unknown account: err = %v, want ErrNotFound", err)
	}

	// Деньги вернули на счёт: копилка пуста и закрывается, история остаётся.
	if _, err := l.s.CreateTransaction(ctx, l.u, models.NewTransaction{Kind: models.KindTransfer, AccountID: piggy.ID, ToAccountID: &l.main.ID, AmountMinor: 1000}); err != nil {
		t.Fatal(err)
	}
	if err := l.s.ArchiveAccount(ctx, l.u, piggy.ID); err != nil {
		t.Fatal(err)
	}
	if err := l.s.ArchiveAccount(ctx, l.u, piggy.ID); err != nil {
		t.Errorf("archiving twice: %v", err)
	}
	if !l.account(t, piggy.ID).Archived {
		t.Error("the piggy must be marked archived")
	}
	if l.account(t, l.main.ID).Archived {
		t.Error("the main account must stay active")
	}
	txs, _ := l.s.ListTransactions(ctx, l.u, 0, 0, "")
	moved := 0
	for _, tx := range txs {
		if tx.AccountID == piggy.ID {
			moved++
		}
	}
	if moved != 2 {
		t.Errorf("%d records of the archived piggy are left in the history, want 2", moved)
	}

	other, err := l.s.Resolve(ctx, models.TelegramUser{ID: 7, FirstName: "Anna"}, "Asia/Tbilisi")
	if err != nil {
		t.Fatal(err)
	}
	if err := l.s.ArchiveAccount(ctx, other, l.cash.ID); !errors.Is(err, models.ErrNotFound) {
		t.Errorf("foreign account: err = %v, want ErrNotFound", err)
	}
}
