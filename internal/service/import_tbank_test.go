package service

import (
	"strings"
	"testing"

	"moneykeeper/internal/models"
	"moneykeeper/pkg/money"
)

// Текст повторяет раскладку справки Т-Банка; названия и суммы выдуманы.
const tbankText = `АКЦИОНЕРНОЕ ОБЩЕСТВО «ТБАНК»
Справка о движении средств
Движение средств за период с 01.10.2026 по 14.11.2026
Дата и время   Дата   Сумма в валюте   Сумма операции   Описание   Номер
операции   списания   операции   в валюте карты   операции   карты
05.11.2026   05.11.2026   -56.30 ₽ -56.30 ₽ Плата за оповещения об   —
22:57   22:57   операциях
04.11.2026   05.11.2026   -1 178.00 ₽   -1 178.00 ₽   Оплата в SHOP ONE   1234
20:39   20:54   CITY RUS
04.11.2026   04.11.2026   +41.00 ₽ +41.00 ₽ Кэшбэк за обычные   —
22:31   22:31   покупки
03.11.2026   03.11.2026   -10.00 $ -950.00 ₽ Оплата в ABROAD CAFE   1234
09:00   09:05   NEW YORK USA
Пополнения:   41,00 ₽
Расходы:   2 184,30 ₽
С уважением,
`

func TestParseTBank(t *testing.T) {
	if !looksLikeTBank(tbankText) || looksLikeTBank(tbcText) {
		t.Fatal("looksLikeTBank is wrong")
	}
	p := parseTBank(tbankText, models.Account{ID: 1, Currency: money.RUB})
	if len(p.Rows) != 4 || p.Skipped != 0 {
		t.Fatalf("rows = %d skipped = %d, want 4 and 0", len(p.Rows), p.Skipped)
	}
	tests := []struct {
		name      string
		i         int
		wantMinor int64
		wantDate  string
		wantDesc  string
	}{
		{"fee with wrapped description", 0, -5630, "2026-11-05", "Плата за оповещения об операциях"},
		{"thousands with a space, prefix cut, wrapped merchant", 1, -117800, "2026-11-04", "SHOP ONE CITY RUS"},
		{"cashback is income", 2, 4100, "2026-11-04", "Кэшбэк за обычные покупки"},
		{"card-currency amount of a foreign purchase", 3, -95000, "2026-11-03", "ABROAD CAFE NEW YORK USA"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := p.Rows[tt.i]
			if r.AmountMinor != tt.wantMinor || r.Date.String() != tt.wantDate || r.Description != tt.wantDesc || r.Currency != money.RUB {
				t.Errorf("row = %d %s %q %s, want %d %s %q RUB", r.AmountMinor, r.Date, r.Description, r.Currency, tt.wantMinor, tt.wantDate, tt.wantDesc)
			}
		})
	}
}

func TestParseTBankOtherCurrency(t *testing.T) {
	p := parseTBank(tbankText, models.Account{ID: 1, Currency: money.GEL})
	if len(p.Rows) != 0 || p.Skipped != 4 || len(p.OtherCurrencies) != 1 || p.OtherCurrencies[0] != "RUB" {
		t.Errorf("rows = %d skipped = %d other = %v, want 0, 4, [RUB]", len(p.Rows), p.Skipped, p.OtherCurrencies)
	}
}

// Выписка TBC на грузинском: две даты в записи, перенос описания, суммы и остаток в колонках; возврат покупки
// (остаток вырос) в расходы не попадает; покупка в валюте берёт сумму списания со счёта.
const tbcGeorgianText = `საწყისი ნაშთი 100.00 GEL
26/10/2026   POS wallet - shop one, 19.60 GEL, Oct 27 2026   თიბისი ბანკის VISA ბარათებით TBCBGE22, GE88TB0000002501034003   19.60 80.40
12:00AM, სასურსათო მაღაზიები, MCC: 5411, VISA,   ბანკის ტერმინალებ
431571******8037
28/10/2026 29/10/2026   POS - PLUMBING, 28.00 GEL, Oct 28 2026 12:00AM, საოჯახო   TBCBGE22, GE88TB0000002501034003   28.00 52.40
ხარჯი, MCC: 5039, VISA, 431571******8037
30/10/2026   POS - ABROAD, 10.00 USD, Oct 29 2026 12:00AM, MCC: 5812, VISA,   TBCBGE22   27.10 25.30
31/10/2026   POS - SHOP ONE REFUND, 5.00 GEL, Oct 29 2026 12:00AM, MCC: 5411, VISA,   TBCBGE22   5.00 30.30
31/10/2026 კონვერტაცია იგორ სიდოროვ, TBCBGE22, GE58TB7575345064400017   30.30 0.00
1 - 8
`

func TestParseTBCGeorgian(t *testing.T) {
	groceries := int64(12)
	p := parseTBC(tbcGeorgianText, models.Account{ID: 1, Currency: money.GEL}, map[string]int64{"продукты": groceries})
	if len(p.Rows) != 3 {
		t.Fatalf("rows = %d, want 3 (the refund and the conversion are not purchases)", len(p.Rows))
	}
	if p.Skipped != 2 {
		t.Errorf("skipped = %d, want 2", p.Skipped)
	}
	want := []struct {
		minor int64
		date  string
		desc  string
		cat   bool
	}{
		{-1960, "2026-10-27", "shop one", true},
		{-2800, "2026-10-28", "PLUMBING", false},
		{-2710, "2026-10-29", "ABROAD", false},
	}
	for i, w := range want {
		r := p.Rows[i]
		if r.AmountMinor != w.minor || r.Date.String() != w.date || r.Description != w.desc || (r.CategoryID != nil) != w.cat {
			t.Errorf("row %d = %d %s %q cat=%v, want %d %s %q cat=%v", i, r.AmountMinor, r.Date, r.Description, r.CategoryID != nil, w.minor, w.date, w.desc, w.cat)
		}
	}
}

func TestStatementTextSpaces(t *testing.T) {
	got := statementText("-1 178.00 ₽ x\n+5.00")
	if !strings.Contains(got, "-1 178.00 ₽ x\n+5.00") {
		t.Errorf("got %q", got)
	}
}
