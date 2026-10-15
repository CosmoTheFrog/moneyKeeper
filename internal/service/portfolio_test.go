package service

import (
	"testing"

	"moneykeeper/internal/models"
	"moneykeeper/pkg/apptime"
	"moneykeeper/pkg/money"
)

// ratio это цена минорной единицы валюты в минорных единицах лари: num/den.
type ratio struct{ num, den int64 }

var testRatios = map[money.Currency]ratio{
	"GEL":  {1, 1},
	"USDT": {273, 100},   // 2,73 ₾ за USDT, обе по два знака
	"SOL":  {40950, 1e6}, // 409,50 ₾ за SOL (6 знаков): 409,50·100 копеек на 10^6 единиц
	"BTC":  {1, 4},       // 250 000 ₾ за BTC (8 знаков): 250000·100 / 10^8 = 0,25 копейки за единицу
	"DOGE": {1, 10},      // 0,10 копейки за единицу
	"XXX":  {0, 1},       // курса нет, см. test valuer
}

func testValuer(amount int64, cur money.Currency, _ apptime.Day) (int64, bool) {
	if cur == "XXX" {
		return 0, false
	}
	r := testRatios[cur]
	return amount * r.num / r.den, true
}

var seq int64

func tx(day string, kind models.TransactionKind, cur money.Currency, amount int64, group int64) models.Transaction {
	seq++
	d, err := apptime.ParseDay(day)
	if err != nil {
		panic(err)
	}
	t := models.Transaction{ID: seq, Kind: kind, Currency: cur, AmountMinor: amount, OccurredOn: d}
	if group != 0 {
		t.TransferID = &group
	}
	return t
}

func TestBuildHoldings(t *testing.T) {
	const (
		d1 = "2026-10-01"
		d2 = "2026-10-05"
	)
	tests := []struct {
		name        string
		txs         []models.Transaction
		coin        money.Currency
		qty, cost   int64
		realized    int64
		wantMissing int
	}{
		{
			name: "buy with USDT",
			txs: []models.Transaction{
				tx(d1, models.KindExchange, "USDT", -30000, 1), tx(d1, models.KindExchange, "SOL", 2_000_000, 1),
			},
			coin: "SOL", qty: 2_000_000, cost: 81900, // 300 USDT · 2,73
		},
		{
			name: "partial sell fixes profit and keeps average price",
			txs: []models.Transaction{
				tx(d1, models.KindExchange, "USDT", -30000, 1), tx(d1, models.KindExchange, "SOL", 2_000_000, 1),
				tx(d2, models.KindExchange, "SOL", -1_000_000, 2), tx(d2, models.KindExchange, "USDT", 17000, 2),
			},
			// Продан 1 SOL за 170 USDT = 464,10 ₾, из цены ушла половина (409,50 ₾): прибыль 54,60 ₾.
			coin: "SOL", qty: 1_000_000, cost: 40950, realized: 5460,
		},
		{
			name: "fee in USDT raises the cost of a purchase",
			txs: []models.Transaction{
				tx(d1, models.KindExchange, "USDT", -30000, 1), tx(d1, models.KindExchange, "SOL", 2_000_000, 1), tx(d1, models.KindFee, "USDT", -100, 1),
			},
			coin: "SOL", qty: 2_000_000, cost: 81900 + 273, // комиссия 1 USDT = 2,73 ₾
		},
		{
			name: "fee reduces the proceeds of a sale",
			txs: []models.Transaction{
				tx(d1, models.KindExchange, "USDT", -30000, 1), tx(d1, models.KindExchange, "SOL", 2_000_000, 1),
				tx(d2, models.KindExchange, "SOL", -2_000_000, 2), tx(d2, models.KindExchange, "USDT", 31000, 2), tx(d2, models.KindFee, "USDT", -100, 2),
			},
			// Выручка 310 USDT = 846,30 ₾ минус комиссия 2,73 ₾ минус цена 819 ₾.
			coin: "SOL", qty: 0, cost: 0, realized: 84630 - 273 - 81900,
		},
		{
			name: "network fee on a wallet transfer lowers quantity and not cost",
			txs: []models.Transaction{
				tx(d1, models.KindExchange, "USDT", -30000, 1), tx(d1, models.KindExchange, "SOL", 2_000_000, 1),
				tx(d2, models.KindTransfer, "SOL", -1_000_000, 2), tx(d2, models.KindTransfer, "SOL", 999_995, 2),
			},
			coin: "SOL", qty: 1_999_995, cost: 81900,
		},
		{
			name: "fee in the coin itself lowers quantity",
			txs: []models.Transaction{
				tx(d1, models.KindExchange, "USDT", -30000, 1), tx(d1, models.KindExchange, "SOL", 2_000_000, 1),
				tx(d2, models.KindExchange, "SOL", -1_000_000, 2), tx(d2, models.KindExchange, "USDT", 17000, 2), tx(d2, models.KindFee, "SOL", -1000, 2),
			},
			coin: "SOL", qty: 999_000, cost: 40950, realized: 5460,
		},
		{
			name: "opening balance and income are bought at the market of their day",
			txs:  []models.Transaction{tx(d1, models.KindOpening, "BTC", 100_000_000, 0), tx(d2, models.KindIncome, "BTC", 50_000_000, 0)},
			coin: "BTC", qty: 150_000_000, cost: 25_000_000 + 12_500_000,
		},
		{
			name: "spending a coin is a sale",
			txs:  []models.Transaction{tx(d1, models.KindIncome, "DOGE", 100_000, 0), tx(d2, models.KindExpense, "DOGE", -40_000, 0)},
			coin: "DOGE", qty: 60_000, cost: 6_000, realized: 4_000 - 4_000, // продано по рынку, цена та же
		},
		{
			name: "selling more than the records show books the proceeds",
			txs:  []models.Transaction{tx(d1, models.KindExchange, "DOGE", -1_000, 1), tx(d1, models.KindExchange, "USDT", 100, 1)},
			coin: "DOGE", qty: -1_000, cost: 0, realized: 273,
		},
		{
			name: "coin to coin sells one and buys the other at the same value",
			txs: []models.Transaction{
				tx(d1, models.KindExchange, "USDT", -30000, 1), tx(d1, models.KindExchange, "SOL", 2_000_000, 1),
				tx(d2, models.KindExchange, "SOL", -2_000_000, 2), tx(d2, models.KindExchange, "BTC", 327_600, 2),
			},
			// 2 SOL стоят 819 ₾, это 0,00327600 BTC, и покупка BTC записана ценой 819 ₾.
			coin: "BTC", qty: 327_600, cost: 81900,
		},
		{
			name:        "a missing rate is counted and the cost stays unknown",
			txs:         []models.Transaction{tx(d1, models.KindExchange, "XXX", -100, 1), tx(d1, models.KindExchange, "SOL", 1_000_000, 1)},
			coin:        "SOL",
			qty:         1_000_000,
			wantMissing: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, missing := buildHoldings(tt.txs, testValuer)
			h := got[tt.coin]
			if h == nil {
				t.Fatalf("no holding for %s: %+v", tt.coin, got)
			}
			if h.qty != tt.qty || h.costGEL != tt.cost || h.realizedGEL != tt.realized || missing != tt.wantMissing {
				t.Errorf("qty %d cost %d realized %d missing %d, want %d %d %d %d", h.qty, h.costGEL, h.realizedGEL, missing, tt.qty, tt.cost, tt.realized, tt.wantMissing)
			}
		})
	}
}

func TestProportion(t *testing.T) {
	tests := []struct{ total, part, whole, want int64 }{
		{81900, 1_000_000, 2_000_000, 40950},
		{100, 1, 3, 33},
		{100, 2, 3, 67},
		{1, 1, 2, 1}, // половина округляется вверх
		{0, 5, 7, 0},
		{1_000_000_000_000, 999_999_999_999, 1_000_000_000_000, 999_999_999_999}, // без переполнения int64
	}
	for _, tt := range tests {
		if got := proportion(tt.total, tt.part, tt.whole); got != tt.want {
			t.Errorf("proportion(%d, %d, %d) = %d, want %d", tt.total, tt.part, tt.whole, got, tt.want)
		}
	}
}
