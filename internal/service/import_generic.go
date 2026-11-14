package service

import (
	"encoding/csv"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	"moneykeeper/internal/models"
	"moneykeeper/pkg/apptime"
	"moneykeeper/pkg/money"
)

// Общий разбор выписок, для которых нет разбора под конкретный банк. Работает без внешних сервисов: PDF читается по
// остатку после каждой операции, CSV по названиям колонок.

var (
	genericDated = regexp.MustCompile(`^\s*(\d{2}[./]\d{2}[./]\d{4}|\d{4}-\d{2}-\d{2})\s+(.*)$`)
	// Остаток на начало периода: по нему видно направление самой первой операции.
	openingBalance = regexp.MustCompile(`(?i)(?:opening balance|starting balance|initial balance|начальный остаток|входящий остаток|остаток на начало|საწყისი ნაშთი)\s*:?\s*(-?\d[\d ,]*\.\d{2})`)
	// Номер страницы под таблицей («1 - 8», «1/8», «Page 1 of 8»): к последней записи страницы он не относится.
	pageFooter = regexp.MustCompile(`(?i)^(?:page\s+)?\d+\s*(?:[-/]|of)\s*\d+$`)
	nonAmount  = regexp.MustCompile(`[^\d.,+\-−\s’']`)
)

// headerCurrency: валюта счёта из шапки выписки, она стоит после остатка («Opening Balance 29.84 GEL»).
var headerCurrency = regexp.MustCompile(`(?i)(?:opening balance|closing balance|starting balance|საწყისი ნაშთი|საბოლოო ნაშთი)\s+-?\d[\d ,]*\.\d{2}\s+([A-Z]{3})\b`)

// withStatementCurrency ставит счёту валюту выписки, если шапка её называет. Счёт, который такую валюту не принимает,
// даёт ok=false: все записи выписки не подходят, и пользователю нужно показать, какая это валюта.
func withStatementCurrency(text string, acc models.Account) (models.Account, models.ImportPreview, bool) {
	if m := headerCurrency.FindStringSubmatch(text); m != nil {
		if cur, err := money.Parse(m[1]); err == nil {
			if !acc.Allows(cur) {
				skipped := 0
				for _, ln := range strings.Split(text, "\n") {
					if genericDated.MatchString(ln) {
						skipped++
					}
				}
				return acc, models.ImportPreview{Skipped: skipped, OtherCurrencies: []string{string(cur)}}, false
			}
			acc.Currency = cur
		}
	}
	return acc, models.ImportPreview{}, true
}

// minConsistent: доля операций, у которых остаток сошёлся. Меньше значит, что формат понят неверно, и строки не нужны.
const minConsistent = 0.8

// parseBalanceStatement читает PDF-выписку любого банка, где в конце записи стоят сумма и остаток после операции.
// Расход это запись, после которой остаток уменьшился на сумму, зачисление запись, после которой он вырос. Перевод между
// своими счетами и конвертацию отличить нельзя, их пользователь снимает в превью. Если остаток не сходится у слишком
// многих записей, возвращается пустой результат: формат не распознан.
func parseBalanceStatement(text string, acc models.Account, byName map[string]int64) models.ImportPreview {
	acc, mismatch, ok := withStatementCurrency(text, acc)
	if !ok {
		return mismatch
	}
	type entry struct{ date, head, tail string }
	var (
		entries []entry
		cur     *entry
	)
	for _, ln := range strings.Split(text, "\n") {
		if m := genericDated.FindStringSubmatch(ln); m != nil {
			entries = append(entries, entry{date: m[1], head: collapse(m[2])})
			cur = &entries[len(entries)-1]
			continue
		}
		if cur != nil && !pageFooter.MatchString(strings.TrimSpace(ln)) {
			cur.tail += " " + collapse(ln)
		}
	}

	var prev *int64
	if m := openingBalance.FindStringSubmatch(text); m != nil {
		if v, err := signedAmount(m[1], money.MinorUnits(acc.Currency)); err == nil {
			prev = &v
		}
	}

	var (
		out        models.ImportPreview
		withPair   int
		consistent int
	)
	for _, e := range entries {
		body := e.head + " " + e.tail
		paid, after, ok := entryAmounts(body, acc.Currency)
		if !ok {
			out.Skipped++
			continue
		}
		if prev == nil {
			// Остатка на начало нет: направление первой записи неизвестно, дальше цепочка пойдёт от её остатка.
			prev = &after
			out.Skipped++
			continue
		}
		withPair++
		direction := 0 // -1 расход, +1 зачисление, 0 не сходится
		switch {
		case *prev-paid == after:
			direction = -1
		case *prev+paid == after:
			direction = 1
		}
		prev = &after
		if direction == 0 {
			out.Skipped++
			continue
		}
		consistent++
		day, err := apptime.ParseDay(genericDateISO(e.date))
		if err != nil || paid > maxAmountMinor {
			out.Skipped++
			continue
		}
		row := models.ImportRow{Date: day, AmountMinor: int64(direction) * paid, Currency: acc.Currency, Description: genericDescription(body)}
		if direction < 0 {
			row.CategoryID, row.SourceCategory = categoryByMCC(body, byName)
		}
		out.Rows = append(out.Rows, row)
	}
	if withPair == 0 || float64(consistent) < minConsistent*float64(withPair) {
		return models.ImportPreview{}
	}
	return out
}

// genericDescription: текст записи без пары «сумма остаток» и без номеров счетов и карт.
var accountNumber = regexp.MustCompile(`\b[A-Z]{2}\d{2}[A-Z0-9]{10,30}\b|\b\d{4,6}\*+\d{4}\b`)

func genericDescription(body string) string {
	all := tbcPair.FindAllStringIndex(body, -1)
	if len(all) > 0 {
		body = body[:all[len(all)-1][0]] + " " + body[all[len(all)-1][1]:]
	}
	return truncateRunes(collapse(accountNumber.ReplaceAllString(body, "")), maxNoteRunes)
}

// genericDateISO переводит 20.09.2026 или 20/09/2026 в 2026-09-20; дата вида 2026-09-20 остаётся как есть.
func genericDateISO(d string) string {
	if len(d) != 10 {
		return ""
	}
	if d[4] == '-' {
		return d
	}
	return d[6:] + "-" + d[3:5] + "-" + d[:2]
}

// signedAmount читает сумму со знаком: минус, скобки или «−» дают отрицательное число. Ноль допустим.
func signedAmount(raw string, units int) (int64, error) {
	neg := strings.HasPrefix(strings.TrimSpace(raw), "-") || strings.HasPrefix(strings.TrimSpace(raw), "−") || strings.HasPrefix(strings.TrimSpace(raw), "(")
	digits := strings.Trim(strings.TrimLeft(strings.TrimSpace(raw), "-−+("), ")")
	var v int64
	if strings.Trim(digits, "0., ") == "" {
		return 0, nil
	}
	v, err := parseStatementAmount(digits, units)
	if err != nil {
		return 0, err
	}
	if neg {
		v = -v
	}
	return v, nil
}

// categoryByMCC подбирает категорию по коду MCC из текста записи. Нет категории в бюджете или кода: категория остаётся
// пустой, а в SourceCategory код, чтобы строки с одним MCC выбирались одним действием.
func categoryByMCC(text string, byName map[string]int64) (*int64, string) {
	mcc := tbcMCC.FindStringSubmatch(text)
	if mcc == nil {
		return nil, "MCC"
	}
	if id, ok := byName[strings.ToLower(mccCategories[mcc[1]])]; ok {
		return &id, ""
	}
	return nil, "MCC " + mcc[1]
}

// Названия колонок CSV на русском, английском и грузинском (латиницей и буквами).
var csvColumns = map[string][]string{
	"date":        {"date", "transaction date", "booking date", "posting date", "дата", "дата операции", "дата и время операции", "дата платежа", "თარიღი"},
	"amount":      {"amount", "sum", "сумма", "сумма операции", "сумма в валюте карты", "сумма платежа", "თანხა"},
	"debit":       {"debit", "paid out", "withdrawal", "withdrawals", "расход", "списание", "дебет", "გასული თანხა"},
	"credit":      {"credit", "paid in", "deposit", "deposits", "приход", "зачисление", "кредит", "შემოსული თანხა"},
	"currency":    {"currency", "валюта", "валюта операции", "ვალუტა"},
	"description": {"description", "details", "merchant", "payee", "narrative", "memo", "note", "описание", "описание операции", "назначение", "назначение платежа", "контрагент", "დანიშნულება"},
}

var dateLayouts = []string{"2006-01-02", "02.01.2006", "02/01/2006", "02-01-2006", "2006/01/02", "02.01.06"}

// parseCSV читает таблицу выписки (csv или txt с разделителем) по названиям колонок. Нужны дата и сумма: либо одна
// колонка суммы со знаком, либо отдельные колонки расхода и прихода. Если таких колонок нет, формат не распознан и
// возвращается ошибка с понятной причиной.
func parseCSV(text string, acc models.Account) (models.ImportPreview, error) {
	text = strings.TrimPrefix(text, "\ufeff")
	delim := detectDelimiter(text)
	r := csv.NewReader(strings.NewReader(text))
	r.Comma = delim
	r.FieldsPerRecord = -1
	r.LazyQuotes = true
	records, err := r.ReadAll()
	if err != nil {
		return models.ImportPreview{}, fmt.Errorf("csv: %w", models.ErrInvalidImport)
	}

	header, col := -1, map[string]int{}
	for i, rec := range records {
		if i > 30 {
			break
		}
		cols := csvHeader(rec)
		if _, ok := cols["date"]; !ok {
			continue
		}
		_, amount := cols["amount"]
		_, debit := cols["debit"]
		_, credit := cols["credit"]
		if amount || debit || credit {
			header, col = i, cols
			break
		}
	}
	if header < 0 {
		return models.ImportPreview{}, fmt.Errorf("csv has no date and amount columns: %w", models.ErrInvalidImport)
	}

	var out models.ImportPreview
	cell := func(rec []string, name string) string {
		if i, ok := col[name]; ok && i < len(rec) {
			return strings.TrimSpace(rec[i])
		}
		return ""
	}
	for _, rec := range records[header+1:] {
		if len(strings.Join(rec, "")) == 0 {
			continue
		}
		day, ok := parseCSVDate(cell(rec, "date"))
		if !ok {
			out.Skipped++
			continue
		}
		cur := acc.Currency
		if code := cell(rec, "currency"); code != "" {
			if sign, ok := currencySigns[code]; ok {
				code = sign
			}
			parsed, err := money.Parse(strings.ToUpper(code))
			if err != nil || !acc.Allows(parsed) {
				out.Skipped++
				if err == nil && !slices.Contains(out.OtherCurrencies, string(parsed)) {
					out.OtherCurrencies = append(out.OtherCurrencies, string(parsed))
				}
				continue
			}
			cur = parsed
		}
		minor, ok := csvAmount(cell(rec, "amount"), cell(rec, "debit"), cell(rec, "credit"), money.MinorUnits(cur))
		if !ok || minor == 0 || abs64(minor) > maxAmountMinor {
			out.Skipped++
			continue
		}
		out.Rows = append(out.Rows, models.ImportRow{Date: day, AmountMinor: minor, Currency: cur, Description: truncateRunes(cell(rec, "description"), maxNoteRunes)})
	}
	return out, nil
}

// csvHeader сопоставляет ячейки строки названиям колонок; возвращает номер колонки для каждого найденного названия.
func csvHeader(rec []string) map[string]int {
	cols := map[string]int{}
	for i, c := range rec {
		name := strings.ToLower(strings.TrimSpace(c))
		for key, names := range csvColumns {
			if _, taken := cols[key]; !taken && slices.Contains(names, name) {
				cols[key] = i
			}
		}
	}
	return cols
}

// csvAmount: одна колонка суммы со знаком или раздельные колонки расхода и прихода (в них суммы без знака).
func csvAmount(amount, debit, credit string, units int) (int64, bool) {
	clean := func(s string) string { return strings.TrimSpace(nonAmount.ReplaceAllString(s, "")) }
	if a := clean(amount); a != "" {
		v, err := signedAmount(a, units)
		return v, err == nil
	}
	if d := clean(debit); d != "" {
		if v, err := signedAmount(strings.TrimLeft(d, "-−"), units); err == nil && v != 0 {
			return -v, true
		}
	}
	if c := clean(credit); c != "" {
		v, err := signedAmount(c, units)
		return v, err == nil
	}
	return 0, false
}

func parseCSVDate(s string) (apptime.Day, bool) {
	// Время после даты не нужно: «20.09.2026 22:57».
	if i := strings.IndexAny(s, " T"); i > 0 {
		s = s[:i]
	}
	for _, layout := range dateLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return apptime.FromDate(t), true
		}
	}
	return apptime.Day{}, false
}

// detectDelimiter выбирает разделитель, который чаще и ровнее всего встречается в первых строках.
func detectDelimiter(text string) rune {
	lines := strings.SplitN(text, "\n", 12)
	best, bestScore := ',', 0
	for _, d := range []rune{';', '\t', ',', '|'} {
		score := 0
		for _, ln := range lines {
			if n := strings.Count(ln, string(d)); n > 0 {
				score += n
			}
		}
		if score > bestScore {
			best, bestScore = d, score
		}
	}
	return best
}

func abs64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}
