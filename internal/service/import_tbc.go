package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"moneykeeper/internal/models"
	"moneykeeper/pkg/apptime"
	"moneykeeper/pkg/money"
	"moneykeeper/pkg/pdftext"
)

// Покупка по карте в выписке TBC: «POS - MERCHANT, 34.00 GEL, Oct 31 2026 12:00AM, VISA 34.00 -4.16».
// Первое число в описании это сумма в валюте покупки, сумма после типа карты это списание в валюте счёта.
var (
	tbcDated = regexp.MustCompile(`^\s*(\d{2}/\d{2}/\d{4})\s+(.*)$`)
	// Покупка без суммы списания рядом с картой: описание переносится на вторую строку, а суммы лежат в своих колонках в
	// конце записи («18.00 94.00»: списано и остаток). Выписка на грузинском выглядит так.
	tbcPOSLoose = regexp.MustCompile(`^POS(?: wallet)? - (.+?), \d[\d,]*\.\d{2} [A-Z]{3}, ([A-Z][a-z]{2} \d{1,2} \d{4})\b`)
	tbcPair     = regexp.MustCompile(`(\d[\d,]*\.\d{2}) (-?\d[\d,]*\.\d{2})(?: |$)`)
	// В записи выписки бывают две даты подряд (операции и списания): нужна первая, вторая мешает разбору описания.
	tbcSecondDate = regexp.MustCompile(`^\d{2}/\d{2}/\d{4}\s+`)
	tbcDatedAny   = regexp.MustCompile(`(?m)^\s*\d{2}/\d{2}/\d{4}\s+\S`)
	tbcPOS        = regexp.MustCompile(`^POS(?: wallet)? - (.+?), \d[\d,]*\.\d{2} [A-Z]{3}, ([A-Z][a-z]{2} \d{1,2} \d{4})\b.*?\b(?:VISA|MASTERCARD|MC)\s+(\d[\d,]*\.\d{2})`)
	tbcMCC        = regexp.MustCompile(`MCC: (\d{4})`)
	tbcLayout     = "Jan 2 2006"
	tbcDateFmt    = "02/01/2006"
)

// mccCategories сопоставляет MCC-код названию стартовой категории. Кода нет в таблице или категории с таким именем
// нет в бюджете: пользователь выбирает категорию сам, строки с одним MCC идут одним выбором.
var mccCategories = map[string]string{
	"5411": "Продукты", "5422": "Продукты", "5441": "Продукты", "5451": "Продукты", "5462": "Продукты", "5499": "Продукты",
	"5812": "Питание вне дома", "5813": "Питание вне дома", "5814": "Питание вне дома",
	"4111": "Транспорт и бензин", "4121": "Транспорт и бензин", "4131": "Транспорт и бензин", "5541": "Транспорт и бензин", "5542": "Транспорт и бензин",
	"5651": "Одежда и обувь", "5691": "Одежда и обувь", "5699": "Одежда и обувь", "5661": "Одежда и обувь",
	"7832": "Развлечения", "7922": "Развлечения", "7991": "Развлечения", "7996": "Развлечения", "7997": "Развлечения",
	"4814": "Подписки", "4899": "Подписки", "5968": "Подписки",
	"4900": "Коммунальные услуги",
	"7230": "Личный уход", "7298": "Личный уход",
}

// PreviewFile определяет тип файла по началу. PDF: выписка TBC разбирается на сервере, любая другая отправляется на
// разбор модели (нужно согласие пользователя). Остальное это xlsx-отчёт прежнего бота.
func (s *Service) PreviewFile(ctx context.Context, u models.User, accountID int64, data []byte, consent bool) (models.ImportPreview, error) {
	if len(data) >= 4 && string(data[:4]) == "%PDF" {
		return s.previewPDF(ctx, u, accountID, data, consent)
	}
	return s.PreviewTrackerExport(ctx, u, accountID, data)
}

// previewPDF достаёт текст PDF. Выписка TBC узнаётся по строкам «дата POS - ...» и разбирается без внешних сервисов,
// остальные банки идут тем же путём, что csv и txt: текст уходит модели, только с согласием.
func (s *Service) previewPDF(ctx context.Context, u models.User, accountID int64, pdf []byte, consent bool) (models.ImportPreview, error) {
	text, err := pdftext.Extract(ctx, pdf)
	if err != nil {
		if errors.Is(err, pdftext.ErrUnavailable) {
			return models.ImportPreview{}, fmt.Errorf("%w: %w", models.ErrImportUnavailable, err)
		}
		return models.ImportPreview{}, fmt.Errorf("%w: %w", models.ErrInvalidImport, err)
	}
	if looksLikeTBC(text) {
		p, err := s.previewTBCText(ctx, u, accountID, text)
		// Выписка TBC на другом языке или с другой раскладкой колонок: разбор по шаблону ничего не нашёл. Строки не
		// теряем, а отдаём тот же текст модели (с согласием), как выписку любого другого банка.
		if err != nil || len(p.Rows) > 0 || strings.TrimSpace(text) == "" {
			return p, err
		}
	}
	if looksLikeTBank(text) {
		return s.previewParsed(ctx, u, accountID, func(acc models.Account, _ map[string]int64) models.ImportPreview {
			return parseTBank(text, acc)
		})
	}
	// Без текстового слоя (скан) модели нечего читать.
	if strings.TrimSpace(text) == "" {
		return models.ImportPreview{}, fmt.Errorf("pdf has no text layer: %w", models.ErrInvalidImport)
	}
	return s.PreviewImport(ctx, u, accountID, statementText(text), consent)
}

// looksLikeTBC: в выписке TBC строки начинаются с даты «31/10/2026» и содержат покупки «POS - ...».
func looksLikeTBC(text string) bool {
	return strings.Contains(text, "POS - ") && tbcDatedAny.MatchString(text)
}

// previewTBCText разбирает текст выписки TBC. Берутся только покупки по карте, остальные операции пока не
// поддерживаются и считаются в Skipped. Служебные проводки без описания (перекладка между счетами карты) отбрасываются
// молча: иначе каждая покупка была бы посчитана дважды.
func (s *Service) previewTBCText(ctx context.Context, u models.User, accountID int64, text string) (models.ImportPreview, error) {
	return s.previewParsed(ctx, u, accountID, func(acc models.Account, byName map[string]int64) models.ImportPreview {
		return parseTBC(text, acc, byName)
	})
}

// previewParsed прогоняет разбор выписки по шаблону через общие шаги: предел строк, запомненные категории, отметка
// уже внесённых строк. parse получает счёт и категории пользователя (название в нижнем регистре, id).
func (s *Service) previewParsed(ctx context.Context, u models.User, accountID int64, parse func(models.Account, map[string]int64) models.ImportPreview) (models.ImportPreview, error) {
	acc, err := s.accounts.Get(ctx, u.ActiveBudgetID, accountID)
	if err != nil {
		return models.ImportPreview{}, err
	}
	cats, err := s.categories.List(ctx, u.ActiveBudgetID)
	if err != nil {
		return models.ImportPreview{}, err
	}
	byName := make(map[string]int64, len(cats))
	for _, c := range cats {
		if !c.Hidden {
			byName[strings.ToLower(c.Name)] = c.ID
		}
	}

	out := parse(acc, byName)
	if len(out.Rows) > maxImportRows {
		out.Skipped += len(out.Rows) - maxImportRows
		out.Rows = out.Rows[:maxImportRows]
	}
	if err := s.applyRules(ctx, u, out.Rows, byName); err != nil {
		return models.ImportPreview{}, err
	}
	if err := s.markDuplicates(ctx, u, acc.ID, out.Rows); err != nil {
		return models.ImportPreview{}, err
	}
	return out, nil
}

// parseTBC собирает записи из текста выписки. Запись начинается со строки с датой, её продолжение (в том числе MCC)
// идёт до следующей строки с датой.
func parseTBC(text string, acc models.Account, byName map[string]int64) models.ImportPreview {
	type entry struct{ head, tail string }
	var (
		entries []entry
		cur     *entry
	)
	for _, ln := range strings.Split(text, "\n") {
		if m := tbcDated.FindStringSubmatch(ln); m != nil {
			entries = append(entries, entry{head: m[1] + " " + collapse(m[2])})
			cur = &entries[len(entries)-1]
			continue
		}
		// Продолжение записи: описание, MCC, суммы. Колонки выписки могут ложиться на любую из этих строк.
		if cur != nil {
			cur.tail += " " + collapse(ln)
		}
	}

	var out models.ImportPreview
	var balance *int64 // остаток после предыдущей записи: по нему видно, списание это или зачисление
	for _, e := range entries {
		rest := strings.TrimSpace(tbcSecondDate.ReplaceAllString(strings.TrimSpace(e.head[len("02/01/2006"):]), ""))
		paid, after, hasPair := entryAmounts(rest+" "+e.tail, acc.Currency)
		credit := hasPair && balance != nil && *balance+paid == after && *balance-paid != after
		if hasPair {
			balance = &after
		}
		if isTBCTransfer(rest) {
			continue
		}
		if credit {
			out.Skipped++ // зачисление (возврат покупки): в расходы не идёт
			continue
		}
		row, ok := tbcRow(e.head[:len("02/01/2006")], rest, e.tail, acc, byName)
		if !ok {
			out.Skipped++
			continue
		}
		out.Rows = append(out.Rows, row)
	}
	return out
}

// entryAmounts достаёт из записи последнюю пару «сумма остаток», в минорных единицах валюты счёта. Остаток может быть
// отрицательным (овердрафт), сумма в паре без знака.
func entryAmounts(text string, cur money.Currency) (paid, balance int64, ok bool) {
	all := tbcPair.FindAllStringSubmatch(text, -1)
	if len(all) == 0 {
		return 0, 0, false
	}
	m := all[len(all)-1]
	units := money.MinorUnits(cur)
	paid, err := parseStatementAmount(m[1], units)
	if err != nil {
		return 0, 0, false
	}
	balance, err = parseStatementAmount(m[2], units)
	if err != nil {
		if strings.TrimLeft(m[2], "-") == "0.00" {
			return paid, 0, true // остаток ноль: parseStatementAmount нулевую сумму не принимает
		}
		return 0, 0, false
	}
	if strings.HasPrefix(m[2], "-") {
		balance = -balance
	}
	return paid, balance, true
}

// isTBCTransfer: у служебной проводки описания нет, остаётся только «, TBCBGE22, IBAN суммы».
func isTBCTransfer(rest string) bool {
	return strings.HasPrefix(rest, ",") || strings.HasPrefix(rest, "TBCBGE22")
}

func tbcRow(stmtDate, rest, tail string, acc models.Account, byName map[string]int64) (models.ImportRow, bool) {
	m := tbcPOS.FindStringSubmatch(rest)
	if m == nil {
		// Сумма списания не рядом с картой: покупку узнаём по описанию, сумму берём из колонок в конце записи.
		loose := tbcPOSLoose.FindStringSubmatch(rest)
		paid, _, ok := entryAmounts(rest+" "+tail, acc.Currency)
		if loose == nil || !ok || paid > maxAmountMinor {
			return models.ImportRow{}, false
		}
		return tbcPurchase(stmtDate, loose[1], loose[2], paid, tail, acc, byName)
	}
	minor, err := parseStatementAmount(m[3], money.MinorUnits(acc.Currency))
	if err != nil || minor > maxAmountMinor {
		return models.ImportRow{}, false
	}
	return tbcPurchase(stmtDate, m[1], m[2], minor, tail, acc, byName)
}

// tbcPurchase собирает строку импорта из покупки: продавец, дата покупки из описания (проводка по счёту идёт позже),
// сумма списания в валюте счёта и категория по MCC.
func tbcPurchase(stmtDate, merchant, purchaseDate string, minor int64, tail string, acc models.Account, byName map[string]int64) (models.ImportRow, bool) {
	day, err := apptime.ParseDay(mustISO(stmtDate))
	if err != nil {
		return models.ImportRow{}, false
	}
	if t, err := time.Parse(tbcLayout, purchaseDate); err == nil {
		day = apptime.FromDate(t)
	}
	row := models.ImportRow{
		Date: day, AmountMinor: -minor, Currency: acc.Currency,
		Description: truncateRunes(strings.TrimSpace(merchant), maxNoteRunes),
	}
	if mcc := tbcMCC.FindStringSubmatch(tail); mcc != nil {
		if id, ok := byName[strings.ToLower(mccCategories[mcc[1]])]; ok {
			row.CategoryID = &id
		} else {
			row.SourceCategory = "MCC " + mcc[1]
		}
	}
	if row.CategoryID == nil && row.SourceCategory == "" {
		row.SourceCategory = "MCC"
	}
	return row, true
}

// mustISO переводит 02/11/2026 в 2026-11-02. Формат уже проверен регулярным выражением.
func mustISO(d string) string {
	t, err := time.Parse(tbcDateFmt, d)
	if err != nil {
		return ""
	}
	return t.Format("2006-01-02")
}

func collapse(s string) string { return strings.Join(strings.Fields(s), " ") }
