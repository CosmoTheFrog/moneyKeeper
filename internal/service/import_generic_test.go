package service

import (
	"errors"
	"testing"

	"moneykeeper/internal/models"
	"moneykeeper/pkg/money"
)

// Выписка неизвестного банка: суммы и остаток в конце записи, описание может переноситься. Данные выдуманы.
const balanceText = `Account statement
Opening Balance 100.00 GEL
Date Description Paid Out Paid In Balance
01/11/2026   POS - SHOP ONE, 19.60 GEL, Oct 31 2026   19.60 80.40
12:00AM, MCC: 5411, VISA, 431571******8037
02/11/2026   Salary from ACME LTD, GE00XX0000000000000001   1000.00 1080.40
03/11/2026   Transfer to savings   80.40 1000.00
Page 1 of 1
`

func TestParseBalanceStatement(t *testing.T) {
	groceries := int64(12)
	acc := models.Account{ID: 1, Currency: money.GEL}
	p := parseBalanceStatement(balanceText, acc, map[string]int64{"продукты": groceries})
	if len(p.Rows) != 3 || p.Skipped != 0 {
		t.Fatalf("rows = %d skipped = %d, want 3 and 0", len(p.Rows), p.Skipped)
	}
	tests := []struct {
		name      string
		i         int
		wantMinor int64
		wantDate  string
		wantCat   bool
		wantDesc  string
	}{
		{"debit with MCC category", 0, -1960, "2026-11-01", true, "POS - SHOP ONE, 19.60 GEL, Oct 31 2026 12:00AM, MCC: 5411, VISA,"},
		{"credit is income, account number cut", 1, 100000, "2026-11-02", false, "Salary from ACME LTD,"},
		{"debit without MCC", 2, -8040, "2026-11-03", false, "Transfer to savings"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := p.Rows[tt.i]
			if r.AmountMinor != tt.wantMinor || r.Date.String() != tt.wantDate || (r.CategoryID != nil) != tt.wantCat || r.Description != tt.wantDesc {
				t.Errorf("row = %d %s cat=%v %q, want %d %s cat=%v %q", r.AmountMinor, r.Date, r.CategoryID != nil, r.Description, tt.wantMinor, tt.wantDate, tt.wantCat, tt.wantDesc)
			}
		})
	}
}

func TestParseBalanceStatementRejectsUnknownFormat(t *testing.T) {
	acc := models.Account{ID: 1, Currency: money.GEL}
	tests := []struct {
		name string
		text string
	}{
		{"balance does not follow the amounts", "Opening Balance 100.00\n01/11/2026 Shop 10.00 500.00\n02/11/2026 Shop 10.00 700.00\n03/11/2026 Shop 10.00 900.00"},
		{"no amounts at all", "01/11/2026 Hello\n02/11/2026 World"},
		{"no dates", "Opening Balance 100.00\nShop 10.00 90.00"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if p := parseBalanceStatement(tt.text, acc, nil); len(p.Rows) != 0 {
				t.Errorf("rows = %d, want 0 (format must not be guessed)", len(p.Rows))
			}
		})
	}
}

func TestParseBalanceStatementWithoutOpeningBalance(t *testing.T) {
	// Направление первой записи неизвестно, её пропускаем; остальные определяются по цепочке остатков.
	text := "01/11/2026 Shop 10.00 90.00\n02/11/2026 Cafe 5.00 85.00\n03/11/2026 Gift 15.00 100.00"
	p := parseBalanceStatement(text, models.Account{ID: 1, Currency: money.GEL}, nil)
	if len(p.Rows) != 2 || p.Skipped != 1 || p.Rows[0].AmountMinor != -500 || p.Rows[1].AmountMinor != 1500 {
		t.Errorf("rows = %+v skipped = %d", p.Rows, p.Skipped)
	}
}

func TestParseCSV(t *testing.T) {
	acc := models.Account{ID: 1, Currency: money.GEL}
	tests := []struct {
		name      string
		text      string
		wantRows  int
		wantSkip  int
		wantMinor []int64
	}{
		{"semicolon, signed amount", "date;amount;description\n2026-10-03;-12,50;Glovo\n2026-10-04;1 000,00;Salary", 2, 0, []int64{-1250, 100000}},
		{"debit and credit columns", "Date,Description,Paid Out,Paid In\n03.10.2026,Glovo,12.50,\n04.10.2026,Salary,,1000.00", 2, 0, []int64{-1250, 100000}},
		{"russian header, time in date, BOM", "\ufeffДата операции;Сумма;Описание\n03.10.2026 22:57;-56,30;Плата\nbad;1;x", 1, 1, []int64{-5630}},
		{"junk before the header", "Statement for John\n\nDate;Amount;Details\n2026-10-03;-3.00;Cafe", 1, 0, []int64{-300}},
		{"quoted comma", "date,amount,description\n2026-10-03,-3.00,\"Cafe, Tbilisi\"", 1, 0, []int64{-300}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := parseCSV(tt.text, acc)
			if err != nil {
				t.Fatal(err)
			}
			if len(p.Rows) != tt.wantRows || p.Skipped != tt.wantSkip {
				t.Fatalf("rows = %d skipped = %d, want %d and %d", len(p.Rows), p.Skipped, tt.wantRows, tt.wantSkip)
			}
			for i, w := range tt.wantMinor {
				if p.Rows[i].AmountMinor != w {
					t.Errorf("row %d = %d, want %d", i, p.Rows[i].AmountMinor, w)
				}
			}
		})
	}
	if p, _ := parseCSV("date,amount,description\n2026-10-03,-3.00,\"Cafe, Tbilisi\"", acc); p.Rows[0].Description != "Cafe, Tbilisi" {
		t.Errorf("description = %q", p.Rows[0].Description)
	}
}

func TestParseCSVNoColumns(t *testing.T) {
	_, err := parseCSV("name,phone\nIgor,123", models.Account{ID: 1, Currency: money.GEL})
	if !errors.Is(err, models.ErrInvalidImport) {
		t.Errorf("err = %v, want ErrInvalidImport", err)
	}
}
