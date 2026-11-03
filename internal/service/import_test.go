package service

import (
	"strings"
	"testing"

	"moneykeeper/internal/models"
	"moneykeeper/pkg/apptime"
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

func TestSplitStatement(t *testing.T) {
	text := "date;amount\n" + strings.Repeat("2026-10-01;10.00\n", 10) + "2026-10-02;5.00"
	tests := []struct {
		name       string
		max        int
		wantChunks int
	}{
		{"fits in one", 10_000, 1},
		{"split", 60, 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunks := splitStatement(text, tt.max)
			if len(chunks) < 2 && tt.wantChunks > 1 || len(chunks) == 1 && tt.wantChunks != 1 {
				t.Fatalf("chunks = %d, want about %d", len(chunks), tt.wantChunks)
			}
			rows := 0
			for i, c := range chunks {
				if !strings.HasPrefix(c, "date;amount\n") {
					t.Errorf("chunk %d has no header: %q", i, c)
				}
				rows += strings.Count(c, ";") - 1 // без заголовка
			}
			if want := 11; rows != want {
				t.Errorf("rows across chunks = %d, want %d", rows, want)
			}
		})
	}
}

func TestBuildRow(t *testing.T) {
	acc := models.Account{ID: 1, Currency: money.GEL}
	chunk := "date;amount;desc\n2026-10-03;-12,50;Glovo\n2026-10-04;1 000.00;Salary"
	cafe := int64(7)
	byName := map[string]int64{"питание вне дома": cafe}

	tests := []struct {
		name                                      string
		date, raw, direction, currency, desc, cat string
		wantOK                                    bool
		wantMinor                                 int64
		wantCat                                   *int64
		wantCurrency                              money.Currency
	}{
		{"expense with category", "2026-10-03", "-12,50", "out", "", "Glovo", "Питание вне дома", true, -1250, &cafe, money.GEL},
		{"income ignores category", "2026-10-04", "1 000.00", "in", "gel", "Salary", "Питание вне дома", true, 100000, nil, money.GEL},
		{"unknown category", "2026-10-03", "-12,50", "out", "", "Glovo", "Не та", true, -1250, nil, money.GEL},
		{"amount not in statement", "2026-10-03", "-99,99", "out", "", "Glovo", "", false, 0, nil, ""},
		{"bad date", "03.10.2026", "-12,50", "out", "", "Glovo", "", false, 0, nil, ""},
		{"bad direction", "2026-10-03", "-12,50", "", "", "Glovo", "", false, 0, nil, ""},
		{"foreign currency on single-currency account", "2026-10-03", "-12,50", "out", "EUR", "Glovo", "", false, 0, nil, ""},
		{"unknown currency", "2026-10-03", "-12,50", "out", "XXX", "Glovo", "", false, 0, nil, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			row, ok := buildRow(acc, chunk, byName, tt.date, tt.raw, tt.direction, tt.currency, tt.desc, tt.cat)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if !ok {
				return
			}
			if row.AmountMinor != tt.wantMinor || row.Currency != tt.wantCurrency {
				t.Errorf("row = %d %s, want %d %s", row.AmountMinor, row.Currency, tt.wantMinor, tt.wantCurrency)
			}
			if (row.CategoryID == nil) != (tt.wantCat == nil) || row.Suggested != (tt.wantCat != nil) {
				t.Errorf("category = %v suggested = %v, want %v", row.CategoryID, row.Suggested, tt.wantCat)
			}
			if want, _ := apptime.ParseDay(tt.date); row.Date != want {
				t.Errorf("date = %v, want %v", row.Date, want)
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
