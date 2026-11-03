package service

import (
	"testing"

	"moneykeeper/internal/models"
	"moneykeeper/pkg/money"
)

// Текст повторяет раскладку pdftotext -layout для выписки TBC; названия и суммы выдуманы.
const tbcText = ` :                   GE00TB0000000000000000                          01/11/2026 - 03/11/2026         03/11/2026 22:59:40

  Date                 Description                                      Additional Information                                                  Balance
                                                                                                     Paid Out               Paid In

02/11/2026 POS - TEST PUB, 34.00 GEL, Oct 31 2026 12:00AM,   VISA      34.00                                                                -4.16
                 , , , MCC: 5812, VISA, 431571******8037 , TBCBGE22, GE00TB0000000000000001

02/11/2026                                                       , TBCBGE22, GE00TB0000000000000000                       20.53                 16.37

02/11/2026 POS wallet - test market, 9.99 GEL, Oct 31 2026 12:00AM,    VISA      9.99                                                                -9.99

         , MCC: 5411, VISA, 431571******8037 , TBCBGE22, GE00TB0000000000000001

                                                                1-2
  Date         Description                                                  Additional Information                            Balance

03/11/2026 POS - ABROAD SHOP, 10.00 USD, Nov 1 2026 12:00AM,   VISA      27.10                                          -8.49
                  , , , MCC: 5999, VISA, 431571******8037 , TBCBGE22, GE00TB0000000000000001

03/11/2026 POS - NO MCC, 1,234.50 GEL, Nov 1 2026 12:00AM,   VISA      1,234.50                                          -8.49

03/11/2026 TRANSFER TO SOMEONE   5.00                                                  0.00
`

func TestParseTBC(t *testing.T) {
	acc := models.Account{ID: 1, Currency: money.GEL}
	food, groceries := int64(11), int64(12)
	byName := map[string]int64{"питание вне дома": food, "продукты": groceries}

	p := parseTBC(tbcText, acc, byName)
	if len(p.Rows) != 4 {
		t.Fatalf("rows = %d, want 4 (service lines are dropped)", len(p.Rows))
	}
	if p.Skipped != 1 {
		t.Errorf("skipped = %d, want 1 (the transfer is not supported)", p.Skipped)
	}

	tests := []struct {
		name       string
		row        models.ImportRow
		wantMinor  int64
		wantDate   string
		wantDesc   string
		wantCat    *int64
		wantSource string
	}{
		{"pub by MCC 5812", p.Rows[0], -3400, "2026-10-31", "TEST PUB", &food, ""},
		{"wallet purchase by MCC 5411", p.Rows[1], -999, "2026-10-31", "test market", &groceries, ""},
		{"foreign currency takes the account-currency amount", p.Rows[2], -2710, "2026-11-01", "ABROAD SHOP", nil, "MCC 5999"},
		{"thousands separator and no MCC", p.Rows[3], -123450, "2026-11-01", "NO MCC", nil, "MCC"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := tt.row
			if r.AmountMinor != tt.wantMinor || r.Date.String() != tt.wantDate || r.Description != tt.wantDesc || r.SourceCategory != tt.wantSource {
				t.Errorf("row = %d %s %q %q, want %d %s %q %q", r.AmountMinor, r.Date, r.Description, r.SourceCategory,
					tt.wantMinor, tt.wantDate, tt.wantDesc, tt.wantSource)
			}
			if (r.CategoryID == nil) != (tt.wantCat == nil) || (r.CategoryID != nil && *r.CategoryID != *tt.wantCat) {
				t.Errorf("category = %v, want %v", r.CategoryID, tt.wantCat)
			}
		})
	}
}

func TestParseTBCCategoryMissingInBudget(t *testing.T) {
	// У пользователя нет категории «Питание вне дома»: выбор отдаётся ему, строки с одним MCC группируются.
	p := parseTBC(tbcText, models.Account{ID: 1, Currency: money.GEL}, map[string]int64{})
	if p.Rows[0].CategoryID != nil || p.Rows[0].SourceCategory != "MCC 5812" {
		t.Errorf("row = %+v, want no category and source MCC 5812", p.Rows[0])
	}
}
