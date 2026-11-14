//go:build integration

package service

import (
	"context"
	"errors"
	"testing"

	"moneykeeper/internal/models"
	"moneykeeper/internal/repo"
	"moneykeeper/pkg/apptime"
)

const csvStatement = "date;amount;description\n2026-10-03;-12,50;Glovo\n2026-10-03;-12,50;Glovo\n2026-10-04;1 000,00;Salary"

func TestPreviewAndCommitImport(t *testing.T) {
	l := newLedger(t)
	ctx := context.Background()
	cafe := l.category(t, "Питание вне дома")

	// Одна из двух одинаковых покупок уже внесена вручную.
	day3, _ := apptime.ParseDay("2026-10-03")
	if _, err := l.s.CreateTransaction(ctx, l.u, models.NewTransaction{
		Kind: models.KindExpense, AccountID: l.cash.ID, CategoryID: &cafe, AmountMinor: 1250, OccurredOn: &day3,
	}); err != nil {
		t.Fatal(err)
	}

	t.Run("not a statement", func(t *testing.T) {
		_, err := l.s.PreviewFile(ctx, l.u, l.cash.ID, []byte("hello;world\n1;2"))
		if !errors.Is(err, models.ErrInvalidImport) {
			t.Errorf("err = %v, want ErrInvalidImport", err)
		}
	})

	p, err := l.s.PreviewFile(ctx, l.u, l.cash.ID, []byte(csvStatement))
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Rows) != 3 || p.Skipped != 0 {
		t.Fatalf("rows = %d skipped = %d, want 3 and 0", len(p.Rows), p.Skipped)
	}
	if !p.Rows[0].Duplicate || p.Rows[1].Duplicate {
		t.Errorf("duplicates = %v %v, want one of two identical purchases marked", p.Rows[0].Duplicate, p.Rows[1].Duplicate)
	}
	if p.Rows[2].AmountMinor != 100000 {
		t.Errorf("income = %d, want 100000", p.Rows[2].AmountMinor)
	}

	// Категорию расходу выбирает пользователь в превью.
	p.Rows[1].CategoryID = &cafe
	before := l.balance(t, l.cash.ID)
	ids, err := l.s.CommitImport(ctx, l.u, l.cash.ID, p.Rows[1:])
	if err != nil || len(ids) != 2 {
		t.Fatalf("commit = %v, %v", ids, err)
	}
	if got, want := l.balance(t, l.cash.ID), before-1250+100000; got != want {
		t.Errorf("balance = %d, want %d", got, want)
	}

	t.Run("expense without category is rejected as a whole", func(t *testing.T) {
		rows := []models.ImportRow{p.Rows[2], {Date: p.Rows[0].Date, AmountMinor: -500, Currency: p.Rows[0].Currency}}
		before := l.balance(t, l.cash.ID)
		if _, err := l.s.CommitImport(ctx, l.u, l.cash.ID, rows); err == nil {
			t.Fatal("want error")
		}
		if got := l.balance(t, l.cash.ID); got != before {
			t.Errorf("balance changed to %d after a rejected import, want %d", got, before)
		}
	})
}

func TestImportRulesAndUndo(t *testing.T) {
	ctx := context.Background()
	s, db := newService(t)
	s.WithImportRules(repo.NewImportRules(db))
	u, err := s.Resolve(ctx, tgIgor, "Asia/Tbilisi")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.SetBaseCurrency(ctx, u, "GEL"); err != nil {
		t.Fatal(err)
	}
	acc, err := s.CreateAccount(ctx, u, "Cash", "cash", "GEL", false, 0)
	if err != nil {
		t.Fatal(err)
	}
	cats, _ := s.ListCategories(ctx, u)
	food := cats[6].ID // «Продукты»
	day, _ := apptime.ParseDay("2026-10-03")
	row := func(desc, source string, cat *int64) models.ImportRow {
		return models.ImportRow{Date: day, AmountMinor: -500, Currency: acc.Currency, Description: desc, SourceCategory: source, CategoryID: cat}
	}

	// Выбранная категория запоминается по продавцу и по коду MCC.
	ids, err := s.CommitImport(ctx, u, acc.ID, []models.ImportRow{row("Test Market", "MCC 5411", &food)})
	if err != nil || len(ids) != 1 {
		t.Fatalf("commit = %v, %v", ids, err)
	}
	rows := []models.ImportRow{row("  TEST   market ", "MCC 1", nil), row("Other", "MCC 5411", nil), row("Unknown", "MCC 9", nil)}
	if err := s.applyRules(ctx, u, rows, map[string]int64{"продукты": food}); err != nil {
		t.Fatal(err)
	}
	for i, wantSet := range []bool{true, true, false} {
		got := rows[i].CategoryID != nil && *rows[i].CategoryID == food && rows[i].Suggested
		if got != wantSet {
			t.Errorf("row %d: category applied = %v, want %v", i, got, wantSet)
		}
	}
	// Скрытая или пропавшая из списка категория не подставляется.
	rows = []models.ImportRow{row("Test Market", "", nil)}
	if err := s.applyRules(ctx, u, rows, map[string]int64{}); err != nil || rows[0].CategoryID != nil {
		t.Errorf("unusable category applied: %+v, %v", rows[0], err)
	}

	// Отмена удаляет записи импорта; запись другого вида отклоняется целиком.
	if err := s.UndoImport(ctx, u, ids); err != nil {
		t.Fatal(err)
	}
	if txs, _ := s.ListTransactions(ctx, u, 50, 0, ""); len(txs) != 0 {
		t.Errorf("transactions after undo = %d, want 0", len(txs))
	}
	if err := s.UndoImport(ctx, u, []int64{999}); !errors.Is(err, models.ErrNotFound) {
		t.Errorf("unknown id: err = %v, want ErrNotFound", err)
	}
}

func TestPreviewImportOtherCurrency(t *testing.T) {
	l := newLedger(t)
	ctx := context.Background()
	// Выписка в евро и долларах, а счёт в лари: пользователь должен узнать, какие валюты не подошли.
	csv := "date;amount;currency;description\n2026-10-03;-12,50;EUR;Glovo\n2026-10-04;-7,00;€;Bolt\n2026-10-05;-9,00;USD;Uber"
	p, err := l.s.PreviewFile(ctx, l.u, l.cash.ID, []byte(csv))
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Rows) != 0 || p.Skipped != 3 {
		t.Fatalf("rows = %d skipped = %d, want 0 and 3", len(p.Rows), p.Skipped)
	}
	if len(p.OtherCurrencies) != 2 || p.OtherCurrencies[0] != "EUR" || p.OtherCurrencies[1] != "USD" {
		t.Errorf("other currencies = %v, want [EUR USD]", p.OtherCurrencies)
	}
}
