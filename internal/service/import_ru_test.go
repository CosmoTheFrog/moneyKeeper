package service

import (
	"testing"

	"moneykeeper/internal/models"
	"moneykeeper/pkg/money"
)

// Раскладка выписки Альфа-Банка; названия, номера и суммы выдуманы.
const alfaText = `Выписка по счету
Операции по счету
Дата проводки Код операции Описание   Сумма
в валюте счета
14.08.2026   CRD_AAAAAA   Операция по карте: 555949++++++0000, на сумму: 313.14 RUR, дата совершения   -313,14 RUR
операции: 11.08.26, место совершения операции: 29157616\RU\SANKT
PETERBU\SHOP ONE 25 MCC5411
16.08.2026   C171608261189302   Перевод C171608261189302 через Систему быстрых платежей от +70000000000.   15 000,00 RUR
Без НДС.
17.08.2026   C421708260297508   Перевод через Систему быстрых платежей на +7 (900) 000-00-   -1 000,00 RUR
00. Без НДС.
18.08.2026   A01   Перевод между своими счетами   -500,00 RUR
Страница 1 из 1
Уполномоченное лицо
(подпись сотрудника АО «АЛЬФА-БАНК»)
`

func TestParseAlfa(t *testing.T) {
	groceries := int64(12)
	if !looksLikeAlfa(alfaText) || looksLikeAlfa(tbankText) || looksLikeAlfa(tbcText) {
		t.Fatal("looksLikeAlfa is wrong")
	}
	p := parseAlfa(alfaText, models.Account{ID: 1, Currency: money.RUB}, map[string]int64{"продукты": groceries})
	if len(p.Rows) != 3 || p.Skipped != 0 {
		t.Fatalf("rows = %d skipped = %d, want 3 and 0 (the own-accounts transfer is not an expense)", len(p.Rows), p.Skipped)
	}
	card, income, transfer := p.Rows[0], p.Rows[1], p.Rows[2]
	if card.AmountMinor != -31314 || card.Date.String() != "2026-08-11" || card.Description != "SHOP ONE 25" || card.CategoryID == nil || *card.CategoryID != groceries {
		t.Errorf("card purchase = %d %s %q cat=%v", card.AmountMinor, card.Date, card.Description, card.CategoryID)
	}
	if income.AmountMinor != 1500000 || income.Date.String() != "2026-08-16" || income.CategoryID != nil {
		t.Errorf("income = %d %s", income.AmountMinor, income.Date)
	}
	if transfer.AmountMinor != -100000 || transfer.SourceCategory != "" {
		t.Errorf("transfer out = %d source=%q, want -100000 and no MCC", transfer.AmountMinor, transfer.SourceCategory)
	}
}

// Раскладка выписки Яндекс Банка: описание слева, две даты и две суммы со знаком «–» (длинное тире), время на второй
// строке описания.
const yandexText = `Выписка по договору
АО «Яндекс Банк»
Описание операции Дата и время   Дата   Карта Сумма в валюте   Сумма в валюте
Внутрибанковский перевод с   02.08.2026   02.08.2026 +9 000,00 ₽ +9 000,00 ₽
42301810680000000000, Виктор П.   в 14:00
Оплата СБП QR (SHOP ONE) 02.08.2026   02.08.2026 –1 114,76 ₽ –1 114,76 ₽
в 20:55
Перевод между счетами одного   17.08.2026   17.08.2026 –245 000,00 ₽ –245 000,00 ₽
клиента   в 09:26
Продолжение на следующей странице
Страница 1 из 1
Исходящий остаток за 13.11.2026   25,63 ₽
Всего расходных операций   –246 114,76 ₽
`

func TestParseYandex(t *testing.T) {
	if !looksLikeYandex(yandexText) || looksLikeYandex(alfaText) {
		t.Fatal("looksLikeYandex is wrong")
	}
	p := parseYandex(yandexText, models.Account{ID: 1, Currency: money.RUB}, nil)
	if len(p.Rows) != 2 || p.Skipped != 0 {
		t.Fatalf("rows = %d skipped = %d, want 2 and 0", len(p.Rows), p.Skipped)
	}
	want := []struct {
		minor int64
		date  string
		desc  string
	}{
		{900000, "2026-08-02", "Внутрибанковский перевод с 42301810680000000000, Виктор П."},
		{-111476, "2026-08-02", "Оплата СБП QR (SHOP ONE)"},
	}
	for i, w := range want {
		if r := p.Rows[i]; r.AmountMinor != w.minor || r.Date.String() != w.date || r.Description != w.desc || r.Currency != money.RUB {
			t.Errorf("row %d = %d %s %q %s, want %d %s %q RUB", i, r.AmountMinor, r.Date, r.Description, r.Currency, w.minor, w.date, w.desc)
		}
	}
}

func TestParseRUOtherCurrency(t *testing.T) {
	p := parseYandex(yandexText, models.Account{ID: 1, Currency: money.GEL}, nil)
	if len(p.Rows) != 0 || p.Skipped != 3 || len(p.OtherCurrencies) != 1 || p.OtherCurrencies[0] != "RUB" {
		t.Errorf("rows = %d skipped = %d other = %v, want 0, 3, [RUB]", len(p.Rows), p.Skipped, p.OtherCurrencies)
	}
}
