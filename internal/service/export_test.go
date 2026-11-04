package service

import "testing"

func TestSpreadsheetSafe(t *testing.T) {
	tests := []struct{ in, want string }{
		{"", ""},
		{"Кофе", "Кофе"},
		{"=SUM(A1:A9)", "'=SUM(A1:A9)"},
		{"+7 999", "'+7 999"},
		{"-5", "'-5"},
		{"@cmd", "'@cmd"},
		{"a=b", "a=b"},
	}
	for _, tt := range tests {
		if got := spreadsheetSafe(tt.in); got != tt.want {
			t.Errorf("spreadsheetSafe(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
