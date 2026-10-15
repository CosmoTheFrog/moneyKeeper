package money

import "testing"

func TestFormat(t *testing.T) {
	tests := []struct {
		minor int64
		cur   Currency
		want  string
	}{
		{0, GEL, "0,00\u00a0₾"},
		{5, GEL, "0,05\u00a0₾"},
		{123450, GEL, "1\u00a0234,50\u00a0₾"},
		{100000000, USD, "1\u00a0000\u00a0000,00\u00a0$"},
		{-123450, EUR, "−1\u00a0234,50\u00a0€"},
		{150, USDT, "1,50\u00a0USDT"},
		{100000, "BTC", "0,00100000\u00a0BTC"},
		{1500000, "SHIB", "1\u00a0500\u00a0000\u00a0SHIB"},
		{999, "SHIB", "999\u00a0SHIB"},
		{12345678, "BTC", "0,12345678\u00a0BTC"},
	}
	for _, tt := range tests {
		if got := Format(tt.minor, tt.cur); got != tt.want {
			t.Errorf("Format(%d, %s) = %q, want %q", tt.minor, tt.cur, got, tt.want)
		}
	}
}
