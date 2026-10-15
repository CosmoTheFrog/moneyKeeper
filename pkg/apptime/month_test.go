package apptime

import (
	"errors"
	"testing"
)

func TestParseMonth(t *testing.T) {
	tests := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"2026-10", "2026-10", false},
		{"2026-01", "2026-01", false},
		{"2026-13", "", true},
		{"2026-1", "", true},
		{"2026-10-05", "", true},
		{"", "", true},
		{"abc", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			m, err := ParseMonth(tt.in)
			if (err != nil) != tt.wantErr || (err != nil && !errors.Is(err, ErrInvalidMonth)) {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && m.String() != tt.want {
				t.Errorf("got %s, want %s", m, tt.want)
			}
		})
	}
}

func TestMonthBounds(t *testing.T) {
	tests := []struct {
		month       string
		first, last string
		next        string
	}{
		{"2026-10", "2026-10-01", "2026-10-31", "2026-11"},
		{"2026-12", "2026-12-01", "2026-12-31", "2027-01"},
		{"2026-02", "2026-02-01", "2026-02-28", "2026-03"},
		{"2024-02", "2024-02-01", "2024-02-29", "2024-03"},
		{"2026-04", "2026-04-01", "2026-04-30", "2026-05"},
	}
	for _, tt := range tests {
		t.Run(tt.month, func(t *testing.T) {
			m, _ := ParseMonth(tt.month)
			if m.First().String() != tt.first || m.Last().String() != tt.last || m.Next().String() != tt.next {
				t.Errorf("first %s last %s next %s", m.First(), m.Last(), m.Next())
			}
		})
	}
}

func TestMonthOfAndOrder(t *testing.T) {
	d, _ := ParseDay("2026-10-12")
	m := MonthOf(d)
	if m.String() != "2026-10" {
		t.Fatalf("MonthOf = %s", m)
	}
	sep, _ := ParseMonth("2026-09")
	next := m.Next()
	if !sep.Before(m) || m.Before(sep) || m.Before(m) || !m.Before(next) {
		t.Error("Before order is wrong")
	}
	if (Month{}).IsZero() != true || m.IsZero() {
		t.Error("IsZero is wrong")
	}
	// Год переходит через декабрь.
	dec, _ := ParseMonth("2025-12")
	if !dec.Before(sep) {
		t.Error("2025-12 must be before 2026-09")
	}
}
