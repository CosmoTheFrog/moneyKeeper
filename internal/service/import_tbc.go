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
	tbcDated    = regexp.MustCompile(`^\s*(\d{2}/\d{2}/\d{4})\s+(.*)$`)
	tbcDatedAny = regexp.MustCompile(`(?m)^\s*\d{2}/\d{2}/\d{4}\s+\S`)
	tbcPOS      = regexp.MustCompile(`^POS(?: wallet)? - (.+?), \d[\d,]*\.\d{2} [A-Z]{3}, ([A-Z][a-z]{2} \d{1,2} \d{4})\b.*?\b(?:VISA|MASTERCARD|MC)\s+(\d[\d,]*\.\d{2})`)
	tbcMCC      = regexp.MustCompile(`MCC: (\d{4})`)
	tbcLayout   = "Jan 2 2006"
	tbcDateFmt  = "02/01/2006"
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

	out := parseTBC(text, acc, byName)
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
		if cur != nil && strings.Contains(ln, "MCC:") {
			cur.tail += " " + collapse(ln)
		}
	}

	var out models.ImportPreview
	for _, e := range entries {
		rest := strings.TrimSpace(e.head[len("02/01/2006"):])
		if isTBCTransfer(rest) {
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

// isTBCTransfer: у служебной проводки описания нет, остаётся только «, TBCBGE22, IBAN суммы».
func isTBCTransfer(rest string) bool {
	return strings.HasPrefix(rest, ",") || strings.HasPrefix(rest, "TBCBGE22")
}

func tbcRow(stmtDate, rest, tail string, acc models.Account, byName map[string]int64) (models.ImportRow, bool) {
	m := tbcPOS.FindStringSubmatch(rest)
	if m == nil {
		return models.ImportRow{}, false
	}
	day, err := apptime.ParseDay(mustISO(stmtDate))
	if err != nil {
		return models.ImportRow{}, false
	}
	// Дата покупки стоит в описании, проводка по счёту идёт позже.
	if t, err := time.Parse(tbcLayout, m[2]); err == nil {
		day = apptime.FromDate(t)
	}
	minor, err := parseStatementAmount(m[3], money.MinorUnits(acc.Currency))
	if err != nil || minor > maxAmountMinor {
		return models.ImportRow{}, false
	}
	row := models.ImportRow{
		Date: day, AmountMinor: -minor, Currency: acc.Currency,
		Description: truncateRunes(strings.TrimSpace(m[1]), maxNoteRunes),
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
