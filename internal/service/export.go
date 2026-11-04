package service

import (
	"context"
	"encoding/csv"
	"io"
	"strings"

	"moneykeeper/internal/i18n"
	"moneykeeper/internal/models"
	"moneykeeper/pkg/apptime"
	"moneykeeper/pkg/money"
)

// ExportTransactions пишет в w записи бюджета за период в CSV: дата, счёт, тип, категория, сумма со знаком, валюта,
// заметка. Суммы в минорных единицах валюты записи, без пересчёта.
func (s *Service) ExportTransactions(ctx context.Context, u models.User, from, to apptime.Day, w io.Writer) error {
	accounts, err := s.accounts.List(ctx, u.ActiveBudgetID)
	if err != nil {
		return err
	}
	categories, err := s.categories.List(ctx, u.ActiveBudgetID)
	if err != nil {
		return err
	}
	txs, err := s.transactions.InRange(ctx, u.ActiveBudgetID, from, to)
	if err != nil {
		return err
	}
	accountName := make(map[int64]string, len(accounts))
	for _, a := range accounts {
		accountName[a.ID] = a.Name
	}
	categoryName := make(map[int64]string, len(categories))
	for _, c := range categories {
		categoryName[c.ID] = c.Name
	}

	out := csv.NewWriter(w)
	if err := out.Write(i18n.ExportHeader()); err != nil {
		return err
	}
	for _, t := range txs {
		category := ""
		if t.CategoryID != nil {
			category = categoryName[*t.CategoryID]
		}
		row := []string{
			t.OccurredOn.String(), spreadsheetSafe(accountName[t.AccountID]), i18n.KindName(t.Kind), spreadsheetSafe(category),
			money.Decimal(t.AmountMinor, t.Currency), string(t.Currency), spreadsheetSafe(t.Note),
		}
		if err := out.Write(row); err != nil {
			return err
		}
	}
	out.Flush()
	return out.Error()
}

// spreadsheetSafe не даёт Excel и Sheets принять текст за формулу: заметка вводится пользователем или приходит
// из выписки.
func spreadsheetSafe(s string) string {
	if s != "" && strings.IndexByte("=+-@\t\r", s[0]) >= 0 {
		return "'" + s
	}
	return s
}
