package service

import (
	"testing"

	"moneykeeper/internal/models"
	"moneykeeper/pkg/apptime"
)

func month(t *testing.T, s string) apptime.Month {
	t.Helper()
	m, err := apptime.ParseMonth(s)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

type cell struct {
	category int64
	month    string
	minor    int64
}

func input(t *testing.T, at string, cats []models.Category, budgeted, spent []cell, income map[string]int64) budgetInput {
	t.Helper()
	in := budgetInput{
		month: month(t, at), categories: cats,
		budgeted: map[categoryMonth]int64{}, spent: map[categoryMonth]int64{}, income: map[apptime.Month]int64{},
	}
	for _, c := range budgeted {
		in.budgeted[categoryMonth{c.category, month(t, c.month)}] = c.minor
	}
	for _, c := range spent {
		in.spent[categoryMonth{c.category, month(t, c.month)}] = c.minor
	}
	for m, v := range income {
		in.income[month(t, m)] = v
	}
	return in
}

func TestBuildBudget(t *testing.T) {
	food := models.Category{ID: 1, Name: "Продукты"}
	rainy := models.Category{ID: 2, Name: "Ремонт", Rollover: true}
	cats := []models.Category{food, rainy}

	tests := []struct {
		name     string
		at       string
		budgeted []cell
		spent    []cell
		income   map[string]int64
		want     []models.BudgetRow // категории в порядке food, rainy
		toBudget int64
		budget   int64
		incomeM  int64
	}{
		{
			name: "empty budget has nothing",
			at:   "2026-10",
			want: []models.BudgetRow{{CategoryID: 1}, {CategoryID: 2}},
		},
		{
			name:     "income not yet budgeted is left to budget",
			at:       "2026-10",
			income:   map[string]int64{"2026-10": 100000},
			want:     []models.BudgetRow{{CategoryID: 1}, {CategoryID: 2}},
			toBudget: 100000, incomeM: 100000,
		},
		{
			name:     "budgeted and spent in the same month",
			at:       "2026-10",
			income:   map[string]int64{"2026-10": 100000},
			budgeted: []cell{{1, "2026-10", 60000}, {2, "2026-10", 10000}},
			spent:    []cell{{1, "2026-10", 45000}},
			want: []models.BudgetRow{
				{CategoryID: 1, BudgetedMinor: 60000, SpentMinor: 45000, BalanceMinor: 15000},
				{CategoryID: 2, BudgetedMinor: 10000, BalanceMinor: 10000},
			},
			toBudget: 30000, budget: 70000, incomeM: 100000,
		},
		{
			name:     "fully allocated income leaves zero",
			at:       "2026-10",
			income:   map[string]int64{"2026-10": 100000},
			budgeted: []cell{{1, "2026-10", 100000}},
			want:     []models.BudgetRow{{CategoryID: 1, BudgetedMinor: 100000, BalanceMinor: 100000}, {CategoryID: 2}},
			toBudget: 0, budget: 100000, incomeM: 100000,
		},
		{
			name:     "over-allocation makes to-budget negative",
			at:       "2026-10",
			income:   map[string]int64{"2026-10": 50000},
			budgeted: []cell{{1, "2026-10", 80000}},
			want:     []models.BudgetRow{{CategoryID: 1, BudgetedMinor: 80000, BalanceMinor: 80000}, {CategoryID: 2}},
			toBudget: -30000, budget: 80000, incomeM: 50000,
		},
		{
			name:     "leftover without rollover is carried only when positive",
			at:       "2026-10",
			budgeted: []cell{{1, "2026-09", 50000}},
			spent:    []cell{{1, "2026-09", 20000}},
			income:   map[string]int64{"2026-09": 50000},
			// В сентябре остался плюс 30 000, он переходит в октябрь.
			want:     []models.BudgetRow{{CategoryID: 1, BalanceMinor: 30000}, {CategoryID: 2}},
			toBudget: 0, incomeM: 0,
		},
		{
			name:     "overspending without rollover is forgotten next month",
			at:       "2026-10",
			budgeted: []cell{{1, "2026-09", 10000}, {1, "2026-10", 5000}},
			spent:    []cell{{1, "2026-09", 25000}},
			income:   map[string]int64{"2026-09": 30000},
			// Перерасход сентября 15 000 в октябре не вычитается: перенос только плюса.
			want:     []models.BudgetRow{{CategoryID: 1, BudgetedMinor: 5000, BalanceMinor: 5000}, {CategoryID: 2}},
			toBudget: 15000, budget: 5000,
		},
		{
			name:     "rollover category carries leftover and overspending",
			at:       "2026-10",
			budgeted: []cell{{2, "2026-08", 40000}, {2, "2026-09", 10000}, {2, "2026-10", 10000}},
			spent:    []cell{{2, "2026-09", 70000}},
			income:   map[string]int64{"2026-08": 60000},
			// Август +40 000, сентябрь 40 000 + 10 000 − 70 000 = −20 000, октябрь −20 000 + 10 000.
			want:     []models.BudgetRow{{CategoryID: 1}, {CategoryID: 2, BudgetedMinor: 10000, BalanceMinor: -10000}},
			toBudget: 0, budget: 10000,
		},
		{
			name:     "months with a gap keep carrying",
			at:       "2026-12",
			budgeted: []cell{{2, "2026-09", 30000}},
			income:   map[string]int64{"2026-09": 30000},
			want:     []models.BudgetRow{{CategoryID: 1}, {CategoryID: 2, BalanceMinor: 30000}},
			toBudget: 0,
		},
		{
			name:     "year boundary",
			at:       "2027-01",
			budgeted: []cell{{2, "2026-12", 5000}, {2, "2027-01", 1000}},
			spent:    []cell{{2, "2026-12", 2000}},
			income:   map[string]int64{"2026-12": 10000},
			want:     []models.BudgetRow{{CategoryID: 1}, {CategoryID: 2, BudgetedMinor: 1000, BalanceMinor: 4000}},
			toBudget: 4000, budget: 1000,
		},
		{
			name:     "future months do not touch the month asked",
			at:       "2026-09",
			budgeted: []cell{{1, "2026-09", 1000}},
			income:   map[string]int64{"2026-09": 1000},
			want:     []models.BudgetRow{{CategoryID: 1, BudgetedMinor: 1000, BalanceMinor: 1000}, {CategoryID: 2}},
			toBudget: 0, budget: 1000, incomeM: 1000,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildBudget(input(t, tt.at, cats, tt.budgeted, tt.spent, tt.income))
			if len(got.Rows) != len(tt.want) {
				t.Fatalf("rows = %+v, want %+v", got.Rows, tt.want)
			}
			for i, w := range tt.want {
				if got.Rows[i] != w {
					t.Errorf("row %d = %+v, want %+v", i, got.Rows[i], w)
				}
			}
			if got.ToBudgetMinor != tt.toBudget || got.BudgetedMinor != tt.budget || got.IncomeMinor != tt.incomeM {
				t.Errorf("to-budget %d budgeted %d income %d, want %d %d %d", got.ToBudgetMinor, got.BudgetedMinor, got.IncomeMinor, tt.toBudget, tt.budget, tt.incomeM)
			}
			if got.Month != month(t, tt.at) {
				t.Errorf("month = %s, want %s", got.Month, tt.at)
			}
		})
	}
}
