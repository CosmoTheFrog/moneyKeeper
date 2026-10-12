package quickinput

import (
	"errors"
	"testing"

	"moneykeeper/pkg/money"
)

func TestParse(t *testing.T) {
	tests := []struct {
		in      string
		want    Result
		wantErr error
	}{
		{"кофе 5 EUR", Result{500, money.EUR, "кофе"}, nil},
		{"кофе 5eur", Result{500, money.EUR, "кофе"}, nil},
		{"кофе 5€", Result{500, money.EUR, "кофе"}, nil},
		{"кофе €5", Result{500, money.EUR, "кофе"}, nil},
		{"5.5 gel такси домой", Result{550, money.GEL, "такси домой"}, nil},
		{"такси 12,50", Result{1250, "", "такси"}, nil},
		{"такси 12", Result{1200, "", "такси"}, nil},
		{"100", Result{10000, "", ""}, nil},
		{"обед 7 ₾ с коллегами", Result{700, money.GEL, "обед с коллегами"}, nil},
		{"подарок 2 шт 15 usd", Result{1500, money.USD, "подарок 2 шт"}, nil},
		{"  кофе   5   eur  ", Result{500, money.EUR, "кофе"}, nil},
		{"кофе", Result{}, ErrNoAmount},
		{"", Result{}, ErrNoAmount},
		{"кофе 0", Result{}, ErrNoAmount},
		{"кофе 5.555", Result{}, ErrInvalidAmount},
		{"кофе 99999999999", Result{}, ErrInvalidAmount},
		{"купил 5kg", Result{}, ErrNoAmount},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := Parse(tt.in, "")
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestParseCoins(t *testing.T) {
	tests := []struct {
		in       string
		fallback money.Currency
		want     Result
		wantErr  error
	}{
		{"купил 0.00125 btc", "", Result{125000, "BTC", "купил"}, nil},
		{"0,5 ETH", "", Result{50000000, "ETH", ""}, nil},
		{"комиссия 0.5", "BTC", Result{50000000, "", "комиссия"}, nil},
		{"раздача 1500000 shib", "", Result{1500000, "SHIB", "раздача"}, nil},
		{"1.5 shib", "", Result{}, ErrInvalidAmount},
		{"5 doge", "", Result{50000, "DOGE", ""}, nil},
		{"0.123456789 btc", "", Result{}, ErrInvalidAmount},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := Parse(tt.in, tt.fallback)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}
