package money

import (
	"errors"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		code    string
		want    Currency
		wantErr bool
	}{
		{"GEL", GEL, false},
		{"USDT", USDT, false},
		{"gel", "", true},
		{"", "", true},
		{"XXX", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			got, err := Parse(tt.code)
			if (err != nil) != tt.wantErr || (err != nil && !errors.Is(err, ErrUnknownCurrency)) {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseBase(t *testing.T) {
	tests := []struct {
		code    string
		wantErr bool
	}{
		{"GEL", false},
		{"RUB", false},
		{"USDT", true},
		{"XXX", true},
	}
	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			_, err := ParseBase(tt.code)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
