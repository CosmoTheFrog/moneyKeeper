package money

import (
	"errors"
	"math/big"
	"testing"
)

func rat(t *testing.T, s string) *big.Rat {
	t.Helper()
	r, ok := new(big.Rat).SetString(s)
	if !ok {
		t.Fatalf("bad rat %q", s)
	}
	return r
}

func TestConvert(t *testing.T) {
	tests := []struct {
		name         string
		amount       int64
		from, to     Currency
		gelFrom, gel string
		want         int64
	}{
		{"EUR to GEL", 1000, EUR, GEL, "3.1594", "1", 3159},               // 10 € по 3,1594 = 31,594 ₾, округление до 31,59
		{"GEL to EUR", 3159, GEL, EUR, "1", "3.1594", 1000},               // 31,59 ₾ / 3,1594 = 9,9987 € → 10,00 €
		{"USD to EUR via GEL", 10000, USD, EUR, "2.7084", "3.1594", 8573}, // 100 $ = 85,73 €
		{"same currency is identity", 12345, GEL, GEL, "1", "1", 12345},
		{"negative is symmetric", -1000, EUR, GEL, "3.1594", "1", -3159},
		{"half rounds away from zero", 5, GEL, USD, "1", "2", 3}, // 0,05 ₾ / 2 = 0,025 → 0,03
		{"negative half", -5, GEL, USD, "1", "2", -3},
		{"zero amount", 0, USD, GEL, "2.7084", "1", 0},
		{"rubles are tiny", 100000, RUB, GEL, "0.0299", "1", 2990},                 // 1000 ₽ = 29,90 ₾
		{"coin with 8 decimals to lari", 100000, "BTC", GEL, "250000", "1", 25000}, // 0,001 BTC по 250 000 ₾ = 250,00 ₾`n		{"lari to coin", 25000, GEL, "BTC", "1", "250000", 100000},`n		{"coin without decimals", 1_500_000, "SHIB", GEL, "0.00003", "1", 4500}, // 1,5 млн SHIB по 0,00003 ₾ = 45,00 ₾`n		{"coin to coin", 100000, "BTC", "ETH", "250000", "12500", 2000000}, // 0,001 BTC = 0,02 ETH`n		{"big amount stays exact", 1_000_000_000_000, USD, GEL, "2.7084", "1", 2_708_400_000_000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Convert(tt.amount, tt.from, tt.to, rat(t, tt.gelFrom), rat(t, tt.gel))
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("got %d, want %d", got, tt.want)
			}
		})
	}
}

func TestConvertRejectsBadRates(t *testing.T) {
	good := rat(t, "2")
	tests := []struct {
		name     string
		from, to *big.Rat
	}{
		{"nil from", nil, good},
		{"nil to", good, nil},
		{"zero from", rat(t, "0"), good},
		{"zero to", good, rat(t, "0")},
		{"negative to", good, rat(t, "-1")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := Convert(100, USD, GEL, tt.from, tt.to); !errors.Is(err, ErrBadRate) {
				t.Errorf("err = %v, want ErrBadRate", err)
			}
		})
	}
}

func TestConvertOverflow(t *testing.T) {
	if _, err := Convert(9_000_000_000_000_000_000, USD, GEL, rat(t, "1000"), rat(t, "1")); !errors.Is(err, ErrBadRate) {
		t.Errorf("err = %v, want overflow reported as ErrBadRate", err)
	}
}
