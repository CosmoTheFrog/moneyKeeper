//go:build integration

package service

import (
	"context"
	"errors"
	"testing"

	"moneykeeper/internal/models"
)

func (l ledger) budget(t *testing.T, at string) models.BudgetMonth {
	t.Helper()
	b, err := l.s.BudgetMonth(context.Background(), l.u, month(t, at))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func (b budgetRows) of(id int64) models.BudgetRow {
	for _, r := range b {
		if r.CategoryID == id {
			return r
		}
	}
	return models.BudgetRow{}
}

type budgetRows []models.BudgetRow

func TestBudgetMonthFromLedger(t *testing.T) {
	ctx := context.Background()
	l := newLedger(t)
	l.setBase(t, "GEL")
	food := l.category(t, "Продукты")

	// Начальный остаток главного счёта 1000 ₾ это доход октября (счета заведены «сегодня», 12 октября).
	b := l.budget(t, "2026-10")
	if b.Currency != "GEL" || b.IncomeMinor != 100000 || b.ToBudgetMinor != 100000 || b.BudgetedMinor != 0 {
		t.Fatalf("empty month = %+v", b)
	}

	set := func(at string, minor int64) {
		t.Helper()
		if err := l.s.SetBudgeted(ctx, l.u, food, month(t, at), minor); err != nil {
			t.Fatal(err)
		}
	}
	set("2026-10", 30000)
	set("2026-09", 10000) // сентябрь без трат: плюс 10 000 переходит в октябрь, хотя перенос у «Продуктов» выключен

	// Расход 10,00 € по курсу дня это 31,59 ₾, он списывается с категории.
	if _, err := l.s.CreateTransaction(ctx, l.u, models.NewTransaction{Kind: models.KindExpense, AccountID: l.eur.ID, CategoryID: &food, AmountMinor: 1000}); err != nil {
		t.Fatal(err)
	}
	// Перевод между счетами в бюджет не входит.
	if _, err := l.s.CreateTransaction(ctx, l.u, models.NewTransaction{Kind: models.KindTransfer, AccountID: l.main.ID, ToAccountID: &l.cash.ID, AmountMinor: 5000}); err != nil {
		t.Fatal(err)
	}

	b = l.budget(t, "2026-10")
	row := budgetRows(b.Rows).of(food)
	if row.BudgetedMinor != 30000 || row.SpentMinor != 3159 || row.BalanceMinor != 30000-3159+10000 {
		t.Errorf("food row = %+v", row)
	}
	// Доходы 100 000 минус заложено 30 000 и 10 000.
	if b.IncomeMinor != 100000 || b.BudgetedMinor != 30000 || b.ToBudgetMinor != 60000 || b.MissingRates != 0 {
		t.Errorf("month = %+v", b)
	}

	// Сентябрь: заложено 10 000, ничего не потрачено, дохода в сентябре ещё не было.
	sep := l.budget(t, "2026-09")
	if r := budgetRows(sep.Rows).of(food); r.BudgetedMinor != 10000 || r.BalanceMinor != 10000 || sep.ToBudgetMinor != -10000 {
		t.Errorf("september = %+v", sep)
	}

	// Сумму можно снять, задав ноль.
	set("2026-10", 0)
	if r := budgetRows(l.budget(t, "2026-10").Rows).of(food); r.BudgetedMinor != 0 {
		t.Errorf("after clearing = %+v", r)
	}
}

func TestBudgetInChosenBaseCurrencyAndMissingRates(t *testing.T) {
	ctx := context.Background()
	l := newLedger(t)
	l.setBase(t, "GEL")
	food := l.category(t, "Продукты")
	l.fx().Err = errors.New("no network")

	if _, err := l.s.CreateTransaction(ctx, l.u, models.NewTransaction{Kind: models.KindExpense, AccountID: l.eur.ID, CategoryID: &food, AmountMinor: 1000}); err != nil {
		t.Fatal(err)
	}
	b := l.budget(t, "2026-10")
	if b.MissingRates != 1 || budgetRows(b.Rows).of(food).SpentMinor != 0 {
		t.Errorf("without rates: missing %d, row %+v", b.MissingRates, budgetRows(b.Rows).of(food))
	}

	l.fx().Err = nil
	b = l.budget(t, "2026-10")
	if b.MissingRates != 0 || budgetRows(b.Rows).of(food).SpentMinor != 3159 {
		t.Errorf("after recovery: missing %d, row %+v", b.MissingRates, budgetRows(b.Rows).of(food))
	}
}

func TestSetBudgetedRejectsBadInput(t *testing.T) {
	ctx := context.Background()
	l := newLedger(t)
	l.setBase(t, "GEL")
	food := l.category(t, "Продукты")
	other, err := l.s.Resolve(ctx, models.TelegramUser{ID: 7, FirstName: "Anna"}, "Asia/Tbilisi")
	if err != nil {
		t.Fatal(err)
	}

	for name, minor := range map[string]int64{"negative": -1, "too large": maxAmountMinor + 1} {
		if err := l.s.SetBudgeted(ctx, l.u, food, month(t, "2026-10"), minor); !errors.Is(err, models.ErrInvalidBudget) {
			t.Errorf("%s: err = %v, want ErrInvalidBudget", name, err)
		}
	}
	if err := l.s.SetBudgeted(ctx, other, food, month(t, "2026-10"), 100); !errors.Is(err, models.ErrNotFound) {
		t.Errorf("foreign category: err = %v, want ErrNotFound", err)
	}
	if err := l.s.SetBudgeted(ctx, l.u, 999999, month(t, "2026-10"), 100); !errors.Is(err, models.ErrNotFound) {
		t.Errorf("unknown category: err = %v, want ErrNotFound", err)
	}
}

func TestRolloverDefaults(t *testing.T) {
	ctx := context.Background()
	l := newLedger(t)
	cats, err := l.s.ListCategories(ctx, l.u)
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]models.Category{}
	for _, c := range cats {
		byName[c.Name] = c
	}
	if !byName["Сбережения"].Rollover || byName["Продукты"].Rollover || byName["Аренда и ипотека"].Rollover {
		t.Errorf("seeded rollover: savings %v food %v rent %v", byName["Сбережения"].Rollover, byName["Продукты"].Rollover, byName["Аренда и ипотека"].Rollover)
	}

	// Новая категория в накопительной группе переносит остаток сразу, в своей группе нет.
	goal, err := l.s.CreateCategory(ctx, l.u, "Отпуск", l.group(t, models.GroupGoal))
	if err != nil || !goal.Rollover {
		t.Errorf("goal category = %+v, %v", goal, err)
	}
	own, err := l.s.CreateCategoryGroup(ctx, l.u, "Путешествия")
	if err != nil {
		t.Fatal(err)
	}
	trip, err := l.s.CreateCategory(ctx, l.u, "Авиабилеты", own.ID)
	if err != nil || trip.Rollover {
		t.Errorf("category in own group = %+v, %v", trip, err)
	}

	// Флаг переключается.
	on := true
	updated, err := l.s.UpdateCategory(ctx, l.u, trip.ID, models.CategoryPatch{Rollover: &on})
	if err != nil || !updated.Rollover {
		t.Errorf("after enabling = %+v, %v", updated, err)
	}
}
