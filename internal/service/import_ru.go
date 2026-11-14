package service

import (
	"regexp"
	"slices"
	"strings"

	"moneykeeper/internal/models"
	"moneykeeper/pkg/apptime"
	"moneykeeper/pkg/money"
)

// Выписки российских банков, где остатка после операции нет, а сумма стоит со знаком: Альфа-Банк («Выписка по счету»)
// и Яндекс Банк («Выписка по договору»). Итоги в шапке выписок сходятся с суммой строк, это проверяется на образцах.

var (
	alfaRow = regexp.MustCompile(`^(\d{2}\.\d{2}\.\d{4})\s+(\S+)\s+(.*?)\s+(-?[\d  ]+,\d{2})\s+(RUR|RUB|USD|EUR)\s*$`)
	// Операция по карте: дата покупки «11.08.26» и «место совершения операции: код\RU\ГОРОД\ПРОДАВЕЦ MCC5411».
	alfaPurchaseDate = regexp.MustCompile(`дата совершения операции:\s*(\d{2}\.\d{2}\.\d{2})`)
	alfaMerchant     = regexp.MustCompile(`место совершения операции:\s*\S*?\\[A-Z]{2}\\[^\\]*\\(.+?)\s+MCC\d{4}`)
	alfaStop         = []string{"Страница", "Дата проводки", "в валюте счета", "Операции по счету", "Уполномоченное лицо", "(подпись"}

	yandexRow  = regexp.MustCompile(`^(.*?)\s*(\d{2}\.\d{2}\.\d{4})\s+\d{2}\.\d{2}\.\d{4}\s+(?:\S+\s+)?[+–-][\d  ]+,\d{2}\s+\S{1,3}\s+([+–-])([\d  ]+,\d{2})\s+(\S{1,3})\s*$`)
	yandexTime = regexp.MustCompile(`\s*в \d{2}:\d{2}`)
	yandexStop = []string{"Продолжение на следующей", "Страница", "Описание операции", "операции", "МСК", "Договора", "Исходящий остаток", "Всего ", "С уважением"}

	// Перевод между своими счетами это не расход и не доход.
	ownTransfer = regexp.MustCompile(`(?i)между (?:своими счетами|счетами одного клиента)`)
)

func looksLikeAlfa(text string) bool {
	return strings.Contains(text, "Дата проводки") && strings.Contains(text, "Код операции") && strings.Contains(text, "АЛЬФА")
}

func looksLikeYandex(text string) bool {
	return strings.Contains(text, "Яндекс Банк") && strings.Contains(text, "Выписка по договору")
}

// ruEntry это запись выписки до разбора: дата проводки, знак, сумма и валюта в валюте счёта, описание.
type ruEntry struct {
	date, sign, amount, currency, desc string
	// day: дата покупки (ГГГГ-ММ-ДД), если она есть в описании, иначе берётся дата проводки; mcc: код продавца.
	day, mcc string
}

// ruPreview превращает записи в строки импорта. Расход со знаком минус, зачисление со знаком плюс или без знака.
// Операции в валюте, которую счёт не принимает, считаются в Skipped, валюта попадает в OtherCurrencies. Переводы между
// своими счетами пропускаются.
func ruPreview(entries []ruEntry, acc models.Account, byName map[string]int64) models.ImportPreview {
	var out models.ImportPreview
	for _, e := range entries {
		code := e.currency
		if c, ok := currencySigns[code]; ok {
			code = c
		}
		cur, err := money.Parse(strings.ToUpper(code))
		if err != nil || !acc.Allows(cur) {
			out.Skipped++
			if err == nil && !slices.Contains(out.OtherCurrencies, string(cur)) {
				out.OtherCurrencies = append(out.OtherCurrencies, string(cur))
			}
			continue
		}
		if ownTransfer.MatchString(e.desc) {
			continue
		}
		minor, err := parseStatementAmount(strings.ReplaceAll(e.amount, ",", "."), money.MinorUnits(cur))
		iso := e.day
		if iso == "" {
			iso = genericDateISO(e.date)
		}
		day, derr := apptime.ParseDay(iso)
		if err != nil || derr != nil || minor > maxAmountMinor {
			out.Skipped++
			continue
		}
		row := models.ImportRow{Date: day, AmountMinor: minor, Currency: cur, Description: truncateRunes(e.desc, maxNoteRunes)}
		if e.sign == "-" {
			row.AmountMinor = -minor
			if e.mcc != "" {
				row.CategoryID, row.SourceCategory = categoryByMCC("MCC: "+e.mcc, byName)
			}
		}
		out.Rows = append(out.Rows, row)
	}
	return out
}

// parseAlfa читает выписку Альфа-Банка: строка записи начинается с даты проводки и кода операции, сумма со знаком в её
// конце, описание переносится на следующие строки. По карте описание сводится к продавцу, дата берётся из покупки.
func parseAlfa(text string, acc models.Account, byName map[string]int64) models.ImportPreview {
	var (
		entries []ruEntry
		cur     *ruEntry
	)
	for _, ln := range strings.Split(text, "\n") {
		ln = strings.TrimSpace(ln)
		if m := alfaRow.FindStringSubmatch(ln); m != nil {
			sign := "+"
			if strings.HasPrefix(m[4], "-") {
				sign = "-"
			}
			entries = append(entries, ruEntry{date: m[1], sign: sign, amount: strings.TrimLeft(m[4], "-"), currency: m[5], desc: collapse(m[3])})
			cur = &entries[len(entries)-1]
			continue
		}
		if cur == nil || hasAnyPrefix(ln, alfaStop) {
			cur = nil
			continue
		}
		cur.desc += " " + collapse(ln)
	}
	for i := range entries {
		e := &entries[i]
		if m := alfaMerchant.FindStringSubmatch(e.desc); m != nil {
			if d := alfaPurchaseDate.FindStringSubmatch(e.desc); d != nil {
				e.day = "20" + d[1][6:] + "-" + d[1][3:5] + "-" + d[1][:2]
			}
			if mcc := tbcMCC.FindStringSubmatch(e.desc); mcc != nil {
				e.mcc = mcc[1]
			}
			e.desc = strings.TrimSpace(m[1])
		}
	}
	return ruPreview(entries, acc, byName)
}

// parseYandex читает выписку Яндекс Банка: описание слева, затем две даты и две суммы со знаком (в валюте операции и в
// валюте договора), время операции «в 14:00» на следующей строке описания. Берётся сумма в валюте договора.
func parseYandex(text string, acc models.Account, byName map[string]int64) models.ImportPreview {
	var (
		entries []ruEntry
		cur     *ruEntry
	)
	for _, ln := range strings.Split(text, "\n") {
		ln = strings.TrimSpace(ln)
		if m := yandexRow.FindStringSubmatch(ln); m != nil {
			sign := "+"
			if m[3] != "+" {
				sign = "-"
			}
			entries = append(entries, ruEntry{date: m[2], sign: sign, amount: strings.ReplaceAll(m[4], " ", ""), currency: m[5], desc: collapse(m[1])})
			cur = &entries[len(entries)-1]
			continue
		}
		if cur == nil || hasAnyPrefix(ln, yandexStop) {
			cur = nil
			continue
		}
		cur.desc += " " + collapse(ln)
	}
	for i := range entries {
		entries[i].desc = collapse(yandexTime.ReplaceAllString(entries[i].desc, ""))
	}
	return ruPreview(entries, acc, byName)
}
