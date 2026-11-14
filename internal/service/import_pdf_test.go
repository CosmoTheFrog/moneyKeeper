package service

import (
	"testing"
)

func TestLooksLikeTBC(t *testing.T) {
	tests := []struct {
		name string
		text string
		want bool
	}{
		{"tbc layout", tbcText, true},
		{"other bank with dates", "01/11/2026   Coffee shop   -5.00 USD\n02/11/2026   Salary   +1000.00 USD", false},
		{"pos without dated lines", "POS - SHOP 5.00 GEL", false},
		{"empty", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := looksLikeTBC(tt.text); got != tt.want {
				t.Errorf("looksLikeTBC = %v, want %v", got, tt.want)
			}
		})
	}
}
