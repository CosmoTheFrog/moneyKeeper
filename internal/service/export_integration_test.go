//go:build integration

package service

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"moneykeeper/internal/models"
)

func TestExportTransactions(t *testing.T) {
	ctx := context.Background()
	l := newLedger(t)
	food := l.category(t, "Продукты")
	if _, err := l.s.CreateTransaction(ctx, l.u, models.NewTransaction{
		Kind: models.KindExpense, AccountID: l.main.ID, CategoryID: &food, AmountMinor: 1250, Note: "=1+1",
	}); err != nil {
		t.Fatal(err)
	}

	var got bytes.Buffer
	if err := l.s.ExportTransactions(ctx, l.u, *day(t, "2026-10-12"), *day(t, "2026-10-12"), &got); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(got.String()), "\n")
	want := []string{
		"Дата,Счёт,Тип,Категория,Сумма,Валюта,Заметка",
		"2026-10-12,TBC,Начальный остаток,,1000.00,GEL,",
		"2026-10-12,TBC,Расход,Продукты,-12.50,GEL,'=1+1",
	}
	if strings.Join(lines, "\n") != strings.Join(want, "\n") {
		t.Errorf("csv:\n%s\nwant:\n%s", got.String(), strings.Join(want, "\n"))
	}

	got.Reset()
	if err := l.s.ExportTransactions(ctx, l.u, *day(t, "2026-09-01"), *day(t, "2026-09-30"), &got); err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(strings.TrimSpace(got.String()), "\n"); n != 0 {
		t.Errorf("empty period has %d rows, want header only", n)
	}
}
