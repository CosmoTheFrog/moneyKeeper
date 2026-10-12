//go:build integration

package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"moneykeeper/internal/models"
	"moneykeeper/internal/testdb"
	"moneykeeper/pkg/apptime"
	"moneykeeper/pkg/money"
)

// «Сегодня» в тестах леджера, 2026-10-12 по Тбилиси (см. ledgerNow).
func day(t *testing.T, s string) *apptime.Day {
	t.Helper()
	d, err := apptime.ParseDay(s)
	if err != nil {
		t.Fatal(err)
	}
	return &d
}

func (l ledger) fx() *testdb.Rates {
	r, ok := l.s.fx.(*testdb.Rates)
	if !ok {
		panic("service fx is not testdb.Rates")
	}
	return r
}

func (l ledger) setBase(t *testing.T, code string) {
	t.Helper()
	if _, err := l.s.SetBaseCurrency(context.Background(), l.u, code); err != nil {
		t.Fatal(err)
	}
}

func (l ledger) listBase(t *testing.T) map[int64]*int64 {
	t.Helper()
	ts, err := l.s.ListTransactions(context.Background(), l.u, 0, 0, "")
	if err != nil {
		t.Fatal(err)
	}
	out := map[int64]*int64{}
	for _, tx := range ts {
		out[tx.ID] = tx.BaseMinor
	}
	return out
}

func TestNoConversionWithoutBaseCurrency(t *testing.T) {
	ctx := context.Background()
	l := newLedger(t)
	food := l.category(t, "Продукты")

	got, err := l.s.CreateTransaction(ctx, l.u, models.NewTransaction{Kind: models.KindExpense, AccountID: l.eur.ID, CategoryID: &food, AmountMinor: 1000})
	if err != nil {
		t.Fatal(err)
	}
	if got[0].BaseMinor != nil {
		t.Errorf("BaseMinor = %d, want none", *got[0].BaseMinor)
	}
	if n := l.fx().Calls(); n != 0 {
		t.Errorf("rates fetched %d times without a base currency", n)
	}
}

func TestTransactionsConvertedByRateOfTheirDay(t *testing.T) {
	ctx := context.Background()
	l := newLedger(t)
	food := l.category(t, "Продукты")
	l.setBase(t, "GEL")

	// 10,00 € по 3,1594 = 31,59 ₾, расход с минусом.
	eur, err := l.s.CreateTransaction(ctx, l.u, models.NewTransaction{Kind: models.KindExpense, AccountID: l.eur.ID, CategoryID: &food, AmountMinor: 1000})
	if err != nil {
		t.Fatal(err)
	}
	if b := eur[0].BaseMinor; b == nil || *b != -3159 {
		t.Fatalf("EUR expense BaseMinor = %v, want -3159", b)
	}
	// Запись в базовой валюте не пересчитывается.
	gel, err := l.s.CreateTransaction(ctx, l.u, models.NewTransaction{Kind: models.KindIncome, AccountID: l.cash.ID, AmountMinor: 5000})
	if err != nil {
		t.Fatal(err)
	}
	if b := gel[0].BaseMinor; b == nil || *b != 5000 {
		t.Errorf("GEL income BaseMinor = %v, want 5000", b)
	}

	// Курс дня берётся один раз: повторные запросы того же дня не ходят в источник.
	calls := l.fx().Calls()
	if _, err := l.s.ListTransactions(ctx, l.u, 0, 0, ""); err != nil {
		t.Fatal(err)
	}
	if n := l.fx().Calls(); n != calls {
		t.Errorf("rates fetched again for a known day: %d -> %d", calls, n)
	}

	// Запись прошлым днём берёт курс своего дня.
	if _, err := l.s.CreateTransaction(ctx, l.u, models.NewTransaction{Kind: models.KindExpense, AccountID: l.eur.ID, CategoryID: &food, AmountMinor: 1000, OccurredOn: day(t, "2026-10-01")}); err != nil {
		t.Fatal(err)
	}
	if n := l.fx().Calls(); n != calls+1 {
		t.Errorf("fetches after a past-day record = %d, want %d", n, calls+1)
	}

	// Дата в будущем считается по курсу сегодняшнего дня, будущих курсов не запрашиваем.
	calls = l.fx().Calls()
	future, err := l.s.CreateTransaction(ctx, l.u, models.NewTransaction{Kind: models.KindExpense, AccountID: l.eur.ID, CategoryID: &food, AmountMinor: 1000, OccurredOn: day(t, "2026-12-31")})
	if err != nil {
		t.Fatal(err)
	}
	if b := future[0].BaseMinor; b == nil || *b != -3159 {
		t.Errorf("future-dated BaseMinor = %v, want -3159", b)
	}
	if n := l.fx().Calls(); n != calls {
		t.Errorf("future date caused %d extra fetches", n-calls)
	}
}

func TestUnavailableRateSourceDoesNotBreakLedger(t *testing.T) {
	ctx := context.Background()
	l := newLedger(t)
	food := l.category(t, "Продукты")
	l.setBase(t, "GEL")
	l.fx().Err = errors.New("no network")

	got, err := l.s.CreateTransaction(ctx, l.u, models.NewTransaction{Kind: models.KindExpense, AccountID: l.eur.ID, CategoryID: &food, AmountMinor: 1000})
	if err != nil {
		t.Fatalf("create must work without rates: %v", err)
	}
	if got[0].BaseMinor != nil {
		t.Errorf("BaseMinor = %d, want none", *got[0].BaseMinor)
	}

	// Источник вернулся: сумма появляется без пересоздания записи.
	l.fx().Err = nil
	if b := l.listBase(t)[got[0].ID]; b == nil || *b != -3159 {
		t.Errorf("after recovery BaseMinor = %v, want -3159", b)
	}
}

func TestFetchesPerRequestAreCapped(t *testing.T) {
	ctx := context.Background()
	l := newLedger(t)
	food := l.category(t, "Продукты")
	l.setBase(t, "GEL")
	l.fx().Err = errors.New("no network") // записи создаём без курсов

	for d := 1; d <= maxRateFetches+4; d++ {
		in := models.NewTransaction{Kind: models.KindExpense, AccountID: l.eur.ID, CategoryID: &food, AmountMinor: 100}
		in.OccurredOn = day(t, "2026-09-"+twoDigits(d))
		if _, err := l.s.CreateTransaction(ctx, l.u, in); err != nil {
			t.Fatal(err)
		}
	}
	l.fx().Err = nil

	before := l.fx().Calls()
	l.listBase(t)
	if n := l.fx().Calls() - before; n != maxRateFetches {
		t.Errorf("one request fetched %d days, want the cap %d", n, maxRateFetches)
	}
}

func TestAccountTotalsInBaseCurrency(t *testing.T) {
	ctx := context.Background()
	l := newLedger(t)
	food := l.category(t, "Продукты")
	l.setBase(t, "GEL")
	wallet := newMulti(t, l, "Revolut", "EUR", 10000) // 100,00 €

	for _, d := range []string{"2026-10-01", "2026-10-02"} {
		in := models.NewTransaction{Kind: models.KindExpense, AccountID: wallet.ID, CategoryID: &food, AmountMinor: 1000, OccurredOn: day(t, d)}
		if _, err := l.s.CreateTransaction(ctx, l.u, in); err != nil {
			t.Fatal(err)
		}
	}
	// Расход в долларах на том же счёте: курсы разных валют не смешиваются.
	if _, err := l.s.CreateTransaction(ctx, l.u, models.NewTransaction{Kind: models.KindExpense, AccountID: wallet.ID, CategoryID: &food, AmountMinor: 1000, Currency: "USD"}); err != nil {
		t.Fatal(err)
	}

	got := l.account(t, wallet.ID)
	eur, usd := got.Balances[0], got.Balances[1]
	// Остаток 80,00 € по сегодняшнему курсу = 252,75 ₾ (252,752).
	if eur.BaseBalanceMinor == nil || *eur.BaseBalanceMinor != 25275 {
		t.Errorf("EUR base balance = %v, want 25275", eur.BaseBalanceMinor)
	}
	// Расходы: два дня по 10 € по 3,1594 = 2 × 31,59 ₾.
	if eur.BaseSpentMinor == nil || *eur.BaseSpentMinor != 2*3159 {
		t.Errorf("EUR base spent = %v, want %d", eur.BaseSpentMinor, 2*3159)
	}
	// -10 $ по 2,7084 = -27,08 ₾ остаток, 27,08 ₾ расход.
	if usd.BaseBalanceMinor == nil || *usd.BaseBalanceMinor != -2708 || usd.BaseSpentMinor == nil || *usd.BaseSpentMinor != 2708 {
		t.Errorf("USD base = balance %v spent %v, want -2708 and 2708", usd.BaseBalanceMinor, usd.BaseSpentMinor)
	}
}

func twoDigits(n int) string {
	if n < 10 {
		return "0" + string(rune('0'+n))
	}
	return string(rune('0'+n/10)) + string(rune('0'+n%10))
}

func TestCoinsAreConvertedAndFetchedOnlyWhenUsed(t *testing.T) {
	ctx := context.Background()
	l := newLedger(t)
	l.setBase(t, "GEL")
	food := l.category(t, "Продукты")
	wallet := newMulti(t, l, "Кошелёк", "USDT", 0)

	// Монет в записях нет: фиатный пересчёт не должен спрашивать их курс.
	if _, err := l.s.CreateTransaction(ctx, l.u, models.NewTransaction{Kind: models.KindIncome, AccountID: wallet.ID, AmountMinor: 1000}); err != nil {
		t.Fatal(err)
	}
	if got := l.fx().CoinRequests(); got != 0 {
		t.Fatalf("coin rates requested without coins in records: %d", got)
	}

	// 0,001 BTC = 100000 минорных единиц, по 250 000 ₾ это 250,00 ₾.
	got, err := l.s.CreateTransaction(ctx, l.u, models.NewTransaction{Kind: models.KindIncome, AccountID: wallet.ID, AmountMinor: 100000, Currency: "BTC"})
	if err != nil {
		t.Fatal(err)
	}
	if b := got[0].BaseMinor; b == nil || *b != 25000 {
		t.Fatalf("BTC income BaseMinor = %v, want 25000", b)
	}

	// Остаток монеты тоже пересчитывается, расход считается по курсу своего дня.
	if _, err := l.s.CreateTransaction(ctx, l.u, models.NewTransaction{Kind: models.KindExpense, AccountID: wallet.ID, CategoryID: &food, AmountMinor: 40000, Currency: "BTC"}); err != nil {
		t.Fatal(err)
	}
	acc := l.account(t, wallet.ID)
	var btc models.CurrencyBalance
	for _, b := range acc.Balances {
		if b.Currency == "BTC" {
			btc = b
		}
	}
	if btc.BalanceMinor != 60000 || btc.BaseBalanceMinor == nil || *btc.BaseBalanceMinor != 15000 || btc.BaseSpentMinor == nil || *btc.BaseSpentMinor != 10000 {
		t.Errorf("BTC balance = %+v, want 60000 worth 15000, spent worth 10000", btc)
	}

	// Курс уже сохранён: повторный запрос не ходит за ним снова.
	before := l.fx().CoinRequests()
	l.listBase(t)
	if n := l.fx().CoinRequests(); n != before {
		t.Errorf("known coin rate fetched again: %d -> %d", before, n)
	}
}

func TestConvertIntoChosenCurrency(t *testing.T) {
	ctx := context.Background()
	l := newLedger(t)
	l.setBase(t, "GEL")
	food := l.category(t, "Продукты")

	// 10,00 € = 31,59 ₾ = 11,67 $ (по 2,7084 ₾ за доллар) = 11,57 USDT (по 2,73 ₾).
	if _, err := l.s.CreateTransaction(ctx, l.u, models.NewTransaction{Kind: models.KindExpense, AccountID: l.eur.ID, CategoryID: &food, AmountMinor: 1000}); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		in   string
		want int64
	}{
		{"", -3159},
		{"GEL", -3159},
		{"USD", -1167},
		{"USDT", -1157},
		{"EUR", -1000},
		{"BTC", -12638}, // 31,594 ₾ при 250 000 ₾ за монету это 0,000126376 BTC, то есть 12638 единиц по 8 знакам
	}
	for _, tt := range tests {
		t.Run("in "+tt.in, func(t *testing.T) {
			ts, err := l.s.ListTransactions(ctx, l.u, 0, 0, tt.in)
			if err != nil {
				t.Fatal(err)
			}
			if b := ts[0].BaseMinor; b == nil || *b != tt.want {
				t.Errorf("BaseMinor = %v, want %d", b, tt.want)
			}
		})
	}

	if _, err := l.s.ListTransactions(ctx, l.u, 0, 0, "XXX"); !errors.Is(err, money.ErrUnknownCurrency) {
		t.Errorf("unknown target: err = %v, want ErrUnknownCurrency", err)
	}
}

func TestAccountsInChosenCurrency(t *testing.T) {
	ctx := context.Background()
	l := newLedger(t)
	l.setBase(t, "GEL")

	// На основном счёте 1000,00 ₾: это 369,22 $ по 2,7084.
	accs, err := l.s.ListAccounts(ctx, l.u, "USD")
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range accs {
		if a.ID != l.main.ID {
			continue
		}
		if b := a.Balances[0].BaseBalanceMinor; b == nil || *b != 36922 {
			t.Errorf("1000 GEL in USD = %v, want 36922", b)
		}
	}
}

func TestRateTable(t *testing.T) {
	ctx := context.Background()
	l := newLedger(t)

	table, err := l.s.Rates(ctx, l.u, "USD")
	if err != nil {
		t.Fatal(err)
	}
	if table.In != "USD" || table.Day.String() != "2026-10-12" {
		t.Fatalf("table = %+v", table)
	}
	got := map[money.Currency]string{}
	for _, q := range table.Quotes {
		got[q.Currency] = q.Rate
	}
	// 1 ₾ = 1/2,7084 $; евро 3,1594/2,7084; доллар к самому себе не выводится.
	if got["GEL"] != "0.369221680697" || got["EUR"] != "1.166518977994" || got["USD"] != "" {
		t.Errorf("rates = GEL %q EUR %q USD %q", got["GEL"], got["EUR"], got["USD"])
	}
	// 250 000 ₾ за монету это около 92 305 $.
	if !strings.HasPrefix(got["BTC"], "92305.") {
		t.Errorf("BTC = %q, want about 92305", got["BTC"])
	}
	// Подменный источник знает только BTC и DOGE: GEL, EUR, RUB, USDT и две монеты, доллара к самому себе нет.
	if len(table.Quotes) != 6 {
		t.Errorf("%d quotes, want 6: %v", len(table.Quotes), got)
	}

	// Без базовой валюты и без параметра таблица строится к лари.
	byDefault, err := l.s.Rates(ctx, l.u, "")
	if err != nil || byDefault.In != "GEL" {
		t.Errorf("default table = %+v, %v", byDefault, err)
	}
}
