package apptime

import (
	"errors"
	"testing"
	"time"
)

func TestParseDay(t *testing.T) {
	tests := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"2026-10-11", "2026-10-11", false},
		{"2024-02-29", "2024-02-29", false},
		{"2026-02-29", "", true},
		{"2026-13-01", "", true},
		{"11.10.2026", "", true},
		{"", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			d, err := ParseDay(tt.in)
			if (err != nil) != tt.wantErr || (err != nil && !errors.Is(err, ErrInvalidDay)) {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && d.String() != tt.want {
				t.Errorf("got %s, want %s", d, tt.want)
			}
		})
	}
}

func TestToday(t *testing.T) {
	// 2026-10-11 21:30 UTC: в Тбилиси (UTC+4) уже 12 октября, в Нью-Йорке (UTC-4) ещё 11.
	now := time.Date(2026, 10, 11, 21, 30, 0, 0, time.UTC)
	tests := []struct {
		zone string
		want string
	}{
		{"UTC", "2026-10-11"},
		{"Asia/Tbilisi", "2026-10-12"},
		{"America/New_York", "2026-10-11"},
	}
	for _, tt := range tests {
		t.Run(tt.zone, func(t *testing.T) {
			loc, err := time.LoadLocation(tt.zone)
			if err != nil {
				t.Fatal(err)
			}
			if got := Today(loc, now).String(); got != tt.want {
				t.Errorf("got %s, want %s", got, tt.want)
			}
		})
	}
}

func TestDateRoundTrip(t *testing.T) {
	d, err := ParseDay("2026-10-11")
	if err != nil {
		t.Fatal(err)
	}
	if got := FromDate(d.Date()); got != d {
		t.Errorf("round trip = %v, want %v", got, d)
	}
}
