package service

import (
	"regexp"
	"slices"
	"strings"

	"moneykeeper/internal/models"
	"moneykeeper/pkg/apptime"
	"moneykeeper/pkg/money"
)

// Справка о движении средств Т-Банка: строка операции начинается с двух дат, затем сумма в валюте операции и сумма в
// валюте карты со знаком, описание и четыре цифры карты. Время и продолжение описания лежат на следующих строках.
var (
	tbankRow    = regexp.MustCompile(`^(\d{2}\.\d{2}\.\d{4})\s+\d{2}\.\d{2}\.\d{4}\s+[+-]\s?[\d  ]+\.\d{2}\s+\S{1,3}\s+([+-])\s?([\d  ]+\.\d{2})\s+(\S{1,3})\s+(.*)$`)
	tbankTimes  = regexp.MustCompile(`^\d{2}:\d{2}\s+\d{2}:\d{2}\s*`)
	tbankCard   = regexp.MustCompile(`\s+(?:\d{4}|—)$`)
	tbankPrefix = regexp.MustCompile(`^Оплата в\s+`)
	tbankStop   = []string{"Пополнения:", "Расходы:", "С уважением", "Дата и время", "операции списания"}
)

// looksLikeTBank: заголовок справки и колонки с суммами.
func looksLikeTBank(text string) bool {
	return strings.Contains(text, "Справка о движении средств") && tbankRow.MatchString(firstTBankRow(text))
}

func firstTBankRow(text string) string {
	for _, ln := range strings.Split(text, "\n") {
		if tbankRow.MatchString(strings.TrimSpace(ln)) {
			return strings.TrimSpace(ln)
		}
	}
	return ""
}

// parseTBank собирает операции справки: расход со знаком минус, зачисление со знаком плюс. Сумма берётся в валюте
// карты. Операции в валюте, которую счёт не принимает, считаются в Skipped и валюта попадает в OtherCurrencies.
// Категорий в справке нет: пользователь выбирает их в превью, выбор запоминается по описанию.
func parseTBank(text string, acc models.Account) models.ImportPreview {
	type entry struct {
		date, sign, amount, currency, desc string
	}
	var (
		entries []entry
		cur     *entry
	)
	for _, ln := range strings.Split(text, "\n") {
		ln = strings.TrimSpace(ln)
		if m := tbankRow.FindStringSubmatch(ln); m != nil {
			entries = append(entries, entry{date: m[1], sign: m[2], amount: m[3], currency: m[4], desc: tbankCard.ReplaceAllString(collapse(m[5]), "")})
			cur = &entries[len(entries)-1]
			continue
		}
		if cur == nil || hasAnyPrefix(ln, tbankStop) {
			cur = nil // итоги, колонтитулы и подпись к операциям не относятся
			continue
		}
		if more := collapse(tbankTimes.ReplaceAllString(ln, "")); more != "" {
			cur.desc += " " + more
		}
	}

	var out models.ImportPreview
	for _, e := range entries {
		code := e.currency
		if c, ok := currencySigns[code]; ok {
			code = c
		}
		curr, err := money.Parse(strings.ToUpper(code))
		if err != nil || !acc.Allows(curr) {
			out.Skipped++
			if err == nil && !slices.Contains(out.OtherCurrencies, string(curr)) {
				out.OtherCurrencies = append(out.OtherCurrencies, string(curr))
			}
			continue
		}
		minor, err := parseStatementAmount(e.amount, money.MinorUnits(curr))
		day, derr := apptime.ParseDay(dotDateISO(e.date))
		if err != nil || derr != nil || minor > maxAmountMinor {
			out.Skipped++
			continue
		}
		if e.sign == "-" {
			minor = -minor
		}
		desc := tbankPrefix.ReplaceAllString(strings.TrimSpace(e.desc), "")
		out.Rows = append(out.Rows, models.ImportRow{Date: day, AmountMinor: minor, Currency: curr, Description: truncateRunes(desc, maxNoteRunes)})
	}
	return out
}

// dotDateISO переводит 20.09.2026 в 2026-09-20.
func dotDateISO(d string) string {
	if len(d) != 10 {
		return ""
	}
	return d[6:] + "-" + d[3:5] + "-" + d[:2]
}

func hasAnyPrefix(s string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}
