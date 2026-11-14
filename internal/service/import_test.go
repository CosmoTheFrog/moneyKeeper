package service

import (
	"testing"

	"moneykeeper/internal/models"
	"moneykeeper/pkg/money"
)

func TestParseStatementAmount(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		units   int
		want    int64
		wantErr bool
	}{
		{"integer", "12", 2, 1200, false},
		{"dot decimal", "12.50", 2, 1250, false},
		{"comma decimal", "12,5", 2, 1250, false},
		{"negative sign dropped", "-12,50", 2, 1250, false},
		{"unicode minus", "−12.50", 2, 1250, false},
		{"thousands comma dot decimal", "1,234.56", 2, 123456, false},
		{"thousands dot comma decimal", "1.234,56", 2, 123456, false},
		{"thousands space", "1 234,56", 2, 123456, false},
		{"thousands nbsp", "1 234,56", 2, 123456, false},
		{"thousands apostrophe", "1'234.50", 2, 123450, false},
		{"thousands only", "1,234", 2, 123400, false},
		{"no fraction currency", "1.234", 0, 1234, false},
		{"coin precision", "0.00012345", 8, 12345, false},
		{"three digits after separator are thousands", "1.234", 2, 123400, false},
		{"zero", "0.00", 2, 0, true},
		{"letters", "12 GEL", 2, 0, true},
		{"empty", "", 2, 0, true},
		{"only separator", ",", 2, 0, true},
		{"too long", "1234567890123", 2, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseStatementAmount(tt.raw, tt.units)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseStatementAmount(%q, %d) err = %v, wantErr %v", tt.raw, tt.units, err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("parseStatementAmount(%q, %d) = %d, want %d", tt.raw, tt.units, got, tt.want)
			}
		})
	}
}

func TestExcelDay(t *testing.T) {
	tests := []struct {
		in     string
		want   string
		wantOK bool
	}{
		{"46296", "2026-10-01", true},
		{"46326.0", "2026-10-31", true},
		{"2026-10-03", "2026-10-03", true},
		{"", "", false},
		{"abc", "", false},
		{"0", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, ok := excelDay(tt.in)
			if ok != tt.wantOK || (ok && got.String() != tt.want) {
				t.Errorf("excelDay(%q) = %v %v, want %v %v", tt.in, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

func TestTrackerRow(t *testing.T) {
	gel := models.Account{ID: 1, Currency: money.GEL}
	multi := models.Account{ID: 2, Currency: money.GEL, MultiCurrency: true}
	groceries := int64(3)
	byName := map[string]int64{"groceries": groceries}

	tests := []struct {
		name                               string
		acc                                models.Account
		income                             bool
		date, amount, currency, text, note string
		wantOK                             bool
		wantMinor                          int64
		wantCat                            *int64
		wantSource, wantDesc               string
	}{
		{"expense matched by name", gel, false, "46296", "1.5", "GEL", "Groceries", "milk", true, -150, &groceries, "", "milk"},
		{"expense unmatched keeps source", gel, false, "46296", "22", "GEL", "Sweets", "", true, -2200, nil, "Sweets", ""},
		{"income source goes to note", multi, true, "46300", "4770", "USD", "Salary", "", true, 477000, nil, "", "Salary"},
		{"foreign currency on single-currency account", gel, true, "46300", "4770", "USD", "Salary", "", false, 0, nil, "", ""},
		{"bad amount", gel, false, "46296", "n/a", "GEL", "Groceries", "", false, 0, nil, "", ""},
		{"bad date", gel, false, "x", "1", "GEL", "Groceries", "", false, 0, nil, "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			row, ok := trackerRow(tt.acc, tt.income, byName, tt.date, tt.amount, tt.currency, tt.text, tt.note)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if !ok {
				return
			}
			if row.AmountMinor != tt.wantMinor || row.SourceCategory != tt.wantSource || row.Description != tt.wantDesc {
				t.Errorf("row = %d %q %q, want %d %q %q", row.AmountMinor, row.SourceCategory, row.Description, tt.wantMinor, tt.wantSource, tt.wantDesc)
			}
			if (row.CategoryID == nil) != (tt.wantCat == nil) {
				t.Errorf("category = %v, want %v", row.CategoryID, tt.wantCat)
			}
		})
	}
}
