package repo

import (
	"context"
	"math/big"
	"testing"

	"moneykeeper/internal/testdb"
	"moneykeeper/pkg/apptime"
	"moneykeeper/pkg/money"
)

func TestRatesCacheKeepsEarlierStoredCurrencies(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	day, err := apptime.ParseDay("2026-10-12")
	if err != nil {
		t.Fatal(err)
	}

	// Первый процесс записал доллар, после перезапуска кэш пуст, а монету дописывает уже другой запрос.
	if err := NewRates(db).Store(ctx, day, map[money.Currency]*big.Rat{money.USD: big.NewRat(27, 10)}); err != nil {
		t.Fatalf("store usd: %v", err)
	}
	rates := NewRates(db)
	if err := rates.Store(ctx, day, map[money.Currency]*big.Rat{"BTC": big.NewRat(100000, 1)}); err != nil {
		t.Fatalf("store btc: %v", err)
	}

	for range 2 { // второй проход читает из кэша
		got, err := rates.ForDays(ctx, []apptime.Day{day})
		if err != nil {
			t.Fatalf("for days: %v", err)
		}
		if got[day][money.USD] == nil || got[day]["BTC"] == nil {
			t.Fatalf("rates of the day = %v, want both USD and BTC", got[day])
		}
	}
}
