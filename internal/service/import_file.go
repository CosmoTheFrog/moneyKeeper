package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"moneykeeper/internal/models"
	"moneykeeper/pkg/apptime"
	"moneykeeper/pkg/money"
	"moneykeeper/pkg/xlsxread"
)

// Листы отчёта прежнего бота (tracker-bot): траты и доходы с колонками по названиям.
const (
	trackerExpenses = "Expenses"
	trackerIncome   = "Income entries"
)

var excelEpoch = time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC)

// PreviewTrackerExport читает xlsx-отчёт tracker-bot и возвращает строки для подтверждения, ничего не сохраняя.
// Файл остаётся на сервере. Категория подставляется при точном совпадении названия (без учёта регистра), иначе в
// SourceCategory остаётся имя из файла, и пользователь выбирает категорию сам. Строки с валютой, которую счёт не
// принимает, и нечитаемые строки считаются в Skipped.
func (s *Service) PreviewTrackerExport(ctx context.Context, u models.User, accountID int64, data []byte) (models.ImportPreview, error) {
	book, err := xlsxread.Open(data)
	if err != nil {
		return models.ImportPreview{}, fmt.Errorf("%w: %w", models.ErrInvalidImport, err)
	}
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

	var out models.ImportPreview
	found := false
	for _, sheet := range []struct {
		name    string
		income  bool
		textCol string // колонка с названием категории или источника дохода
	}{{trackerExpenses, false, "category"}, {trackerIncome, true, "source"}} {
		rows, ok, err := book.Sheet(sheet.name)
		if err != nil {
			return models.ImportPreview{}, fmt.Errorf("%w: %w", models.ErrInvalidImport, err)
		}
		if !ok || len(rows) == 0 {
			continue
		}
		found = true
		col := headerColumns(rows[0])
		for _, c := range []string{"date", "amount", "currency", sheet.textCol} {
			if _, ok := col[c]; !ok {
				return models.ImportPreview{}, fmt.Errorf("sheet %q has no %q column: %w", sheet.name, c, models.ErrInvalidImport)
			}
		}
		for _, r := range rows[1:] {
			cell := func(name string) string {
				if i, ok := col[name]; ok && i < len(r) {
					return strings.TrimSpace(r[i])
				}
				return ""
			}
			row, ok := trackerRow(acc, sheet.income, byName, cell("date"), cell("amount"), cell("currency"), cell(sheet.textCol), cell("note"))
			if !ok || len(out.Rows) >= maxImportRows {
				out.Skipped++
				continue
			}
			out.Rows = append(out.Rows, row)
		}
	}
	if !found {
		return models.ImportPreview{}, fmt.Errorf("no %q or %q sheet: %w", trackerExpenses, trackerIncome, models.ErrInvalidImport)
	}
	if err := s.markDuplicates(ctx, u, acc.ID, out.Rows); err != nil {
		return models.ImportPreview{}, err
	}
	return out, nil
}

func trackerRow(acc models.Account, income bool, byName map[string]int64, date, amount, currency, text, note string) (models.ImportRow, bool) {
	day, ok := excelDay(date)
	if !ok {
		return models.ImportRow{}, false
	}
	cur, err := money.Parse(strings.ToUpper(currency))
	if err != nil || !acc.Allows(cur) {
		return models.ImportRow{}, false
	}
	minor, err := parseStatementAmount(amount, money.MinorUnits(cur))
	if err != nil || minor > maxAmountMinor {
		return models.ImportRow{}, false
	}
	row := models.ImportRow{Date: day, AmountMinor: minor, Currency: cur}
	if income {
		// Источник дохода идёт в заметку, у дохода категории нет.
		row.Description = truncateRunes(strings.TrimSpace(text+" "+note), maxNoteRunes)
		return row, true
	}
	row.AmountMinor = -minor
	row.Description = truncateRunes(note, maxNoteRunes)
	if id, ok := byName[strings.ToLower(text)]; ok {
		row.CategoryID = &id
	} else {
		row.SourceCategory = text
	}
	return row, true
}

// excelDay переводит число Excel (дни с 30.12.1899) или строку вида 2026-10-03 в календарный день.
func excelDay(s string) (apptime.Day, bool) {
	if d, err := apptime.ParseDay(s); err == nil {
		return d, true
	}
	n, err := strconv.ParseFloat(s, 64)
	if err != nil || n < 1 || n > 100_000 {
		return apptime.Day{}, false
	}
	return apptime.FromDate(excelEpoch.AddDate(0, 0, int(n))), true
}

func headerColumns(header []string) map[string]int {
	col := make(map[string]int, len(header))
	for i, h := range header {
		col[strings.ToLower(strings.TrimSpace(h))] = i
	}
	return col
}
