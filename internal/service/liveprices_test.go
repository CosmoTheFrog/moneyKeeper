package service

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"math/big"
	"testing"
	"time"

	"moneykeeper/pkg/apptime"
	"moneykeeper/pkg/money"
)

type priceStub struct {
	calls int
	err   error
}

func (p *priceStub) GelPerUnit(context.Context, apptime.Day, []money.Currency) (map[money.Currency]*big.Rat, error) {
	return nil, nil
}

func (p *priceStub) FiatAvailable(context.Context, money.Currency) error { return nil }

func (p *priceStub) AssetPrice(context.Context, money.Currency, bool) (*big.Rat, error) {
	p.calls++
	return big.NewRat(100, 1), p.err
}

func TestApplyLivePrices(t *testing.T) {
	today := apptime.FromDate(time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC))
	start := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		advance   time.Duration
		err       error
		wantCalls int
		wantGEL   int64 // лари за BTC после второго вызова
	}{
		{"внутри 10 минут цена из кэша", 9 * time.Minute, nil, 1, 300},
		{"через 10 минут запрос заново", 10 * time.Minute, nil, 2, 300},
		{"ошибка источника оставляет курс дня", 11 * time.Minute, errors.New("down"), 2, 50},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub := &priceStub{}
			now := start
			s := &Service{fx: stub, log: slog.New(slog.NewTextHandler(io.Discard, nil)), now: func() time.Time { return now }}
			fresh := func() rateBook {
				return rateBook{today: {money.USDT: big.NewRat(3, 1), money.Currency("BTC"): big.NewRat(50, 1)}}
			}
			s.applyLivePrices(context.Background(), fresh(), today, []money.Currency{money.Currency("BTC")})

			now = now.Add(tt.advance)
			stub.err = tt.err
			book := fresh()
			s.applyLivePrices(context.Background(), book, today, []money.Currency{money.Currency("BTC")})

			if stub.calls != tt.wantCalls {
				t.Errorf("calls = %d, want %d", stub.calls, tt.wantCalls)
			}
			if got := new(big.Rat).Set(book[today][money.Currency("BTC")]); got.Cmp(big.NewRat(tt.wantGEL, 1)) != 0 {
				t.Errorf("BTC = %s, want %d", got.RatString(), tt.wantGEL)
			}
		})
	}
}
