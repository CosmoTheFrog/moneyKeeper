package service

import "testing"

func TestLimitLevel(t *testing.T) {
	tests := []struct {
		name            string
		spent, budgeted int64
		want            int
	}{
		{"ничего не заложено", 500, 0, 0},
		{"траты ниже порога", 7999, 10000, 0},
		{"ровно 80%", 8000, 10000, 80},
		{"между порогами", 9999, 10000, 80},
		{"ровно 100%", 10000, 10000, 100},
		{"перерасход", 25000, 10000, 100},
		{"возврат средств", -500, 10000, 0},
		{"крупные суммы без переполнения", maxAmountMinor * 9 / 10, maxAmountMinor, 80},
	}
	for _, tt := range tests {
		if got := limitLevel(tt.spent, tt.budgeted); got != tt.want {
			t.Errorf("%s: limitLevel(%d, %d) = %d, want %d", tt.name, tt.spent, tt.budgeted, got, tt.want)
		}
	}
}
