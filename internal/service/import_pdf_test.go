package service

import (
	"strings"
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

func TestStatementText(t *testing.T) {
	in := "Account GE29NB0000000000000001   holder\n05.11.2026      Glovo        -23.40     card 4315710000008037\n"
	got := statementText(in)
	for _, leaked := range []string{"GE29NB0000000000000001", "4315710000008037"} {
		if strings.Contains(got, leaked) {
			t.Errorf("%q leaked into %q", leaked, got)
		}
	}
	// Дата, продавец и сумма остаются, колонки сжаты.
	for _, kept := range []string{"[IBAN]", "[NUM]", "05.11.2026 Glovo -23.40"} {
		if !strings.Contains(got, kept) {
			t.Errorf("%q missing in %q", kept, got)
		}
	}
	if strings.Contains(got, "  ") {
		t.Errorf("columns not collapsed: %q", got)
	}
}
