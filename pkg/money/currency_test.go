package money

import (
	"errors"
	"slices"
	"testing"
)

func TestRegistry(t *testing.T) {
	if len(All) != 25 || All[0] != GEL || All[5] != "BTC" {
		t.Fatalf("All = %v", All)
	}
	if !slices.Equal(Core, []Currency{EUR, USD, RUB, USDT}) {
		t.Errorf("Core = %v", Core)
	}
	seen := map[Currency]bool{}
	for _, c := range All {
		if seen[c] {
			t.Errorf("%s twice", c)
		}
		seen[c] = true
	}
	for _, in := range Infos() {
		// Предел суммы 10^12 минорных единиц не должен превращаться в мизер: хотя бы 10 000 целых единиц.
		whole := int64(1_000_000_000_000)
		for range in.Units {
			whole /= 10
		}
		if whole < 10_000 {
			t.Errorf("%s: %d decimals leave only %d whole units under the amount cap", in.Code, in.Units, whole)
		}
	}
}

func TestMinorUnitsAndCoins(t *testing.T) {
	tests := []struct {
		c     Currency
		units int
		coin  bool
	}{
		{GEL, 2, false}, {USDT, 2, false}, {"BTC", 8, true}, {"SHIB", 0, true}, {"DOGE", 4, true}, {"XXX", 2, false},
	}
	for _, tt := range tests {
		if got := MinorUnits(tt.c); got != tt.units {
			t.Errorf("MinorUnits(%s) = %d, want %d", tt.c, got, tt.units)
		}
		if got := IsCoin(tt.c); got != tt.coin {
			t.Errorf("IsCoin(%s) = %v, want %v", tt.c, got, tt.coin)
		}
	}
}

func TestParse(t *testing.T) {
	if c, err := Parse("ATOM"); err != nil || c != "ATOM" {
		t.Errorf("Parse(ATOM) = %q, %v", c, err)
	}
	if _, err := Parse("XXX"); !errors.Is(err, ErrUnknownCurrency) {
		t.Errorf("Parse(XXX) err = %v", err)
	}
	if _, err := ParseBase("BTC"); !errors.Is(err, ErrUnknownCurrency) {
		t.Errorf("a coin must not be a base currency, err = %v", err)
	}
}
