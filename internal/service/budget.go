package service

import (
	"context"
	"fmt"

	"moneykeeper/internal/models"
	"moneykeeper/pkg/apptime"
	"moneykeeper/pkg/money"
)

type categoryMonth struct {
	category int64
	month    apptime.Month
}

// budgetInput это всё, что нужно для расчёта бюджета, уже в базовой валюте. spent положительные суммы расходов.
type budgetInput struct {
	month      apptime.Month
	categories []models.Category
	budgeted   map[categoryMonth]int64
	spent      map[categoryMonth]int64
	income     map[apptime.Month]int64
}

// buildBudget считает бюджет за месяц. Остаток категории = заложено − потрачено + перенос с прошлого месяца: у категории
// с переносом это весь прошлый остаток (в том числе перерасход), без него только плюс. «К распределению» накопительное:
// все доходы до конца месяца минус всё заложенное до него включительно.
func buildBudget(in budgetInput) models.BudgetMonth {
	first := in.month
	for k := range in.budgeted {
		if k.month.Before(first) {
			first = k.month
		}
	}
	for k := range in.spent {
		if k.month.Before(first) {
			first = k.month
		}
	}
	for m := range in.income {
		if m.Before(first) {
			first = m
		}
	}

	var totalIncome, totalBudgeted int64
	balance := make(map[int64]int64, len(in.categories))
	out := models.BudgetMonth{Month: in.month}
	for m := first; !in.month.Before(m); m = m.Next() {
		totalIncome += in.income[m]
		var monthBudgeted int64
		for _, c := range in.categories {
			k := categoryMonth{c.ID, m}
			carry := balance[c.ID]
			if !c.Rollover {
				carry = max(carry, 0)
			}
			budgeted, spent := in.budgeted[k], in.spent[k]
			balance[c.ID] = budgeted - spent + carry
			monthBudgeted += budgeted
			if m == in.month {
				out.Rows = append(out.Rows, models.BudgetRow{CategoryID: c.ID, BudgetedMinor: budgeted, SpentMinor: spent, BalanceMinor: balance[c.ID]})
			}
		}
		totalBudgeted += monthBudgeted
		if m == in.month {
			out.IncomeMinor, out.BudgetedMinor = in.income[m], monthBudgeted
		}
	}
	out.ToBudgetMinor = totalIncome - totalBudgeted
	return out
}

// BudgetMonth собирает бюджет месяца: траты и доходы пересчитываются в базовую валюту по курсу своего дня.
func (s *Service) BudgetMonth(ctx context.Context, u models.User, month apptime.Month) (models.BudgetMonth, error) {
	base, ok, err := s.baseCurrency(ctx, u)
	if err != nil || !ok {
		return models.BudgetMonth{Month: month}, err
	}
	cats, err := s.categories.List(ctx, u.ActiveBudgetID)
	if err != nil {
		return models.BudgetMonth{}, err
	}
	amounts, err := s.plan.Budgeted(ctx, u.ActiveBudgetID, month)
	if err != nil {
		return models.BudgetMonth{}, err
	}
	flows, err := s.plan.Flows(ctx, u.ActiveBudgetID, month)
	if err != nil {
		return models.BudgetMonth{}, err
	}

	days := make([]apptime.Day, 0, len(flows))
	currencies := make([]money.Currency, 0, len(flows)+1)
	for _, f := range flows {
		days = append(days, s.rateDay(u, f.Day))
		currencies = append(currencies, f.Currency)
	}
	book := s.rates(ctx, days, onlyCoins(append(currencies, base)))

	in := budgetInput{
		month: month, categories: cats,
		budgeted: map[categoryMonth]int64{}, spent: map[categoryMonth]int64{}, income: map[apptime.Month]int64{},
	}
	for _, a := range amounts {
		in.budgeted[categoryMonth{a.CategoryID, a.Month}] = a.Minor
	}
	missing := 0
	for _, f := range flows {
		v, ok := book.convert(f.AmountMinor, f.Currency, base, s.rateDay(u, f.Day))
		if !ok {
			missing++
			continue
		}
		if f.CategoryID != nil {
			in.spent[categoryMonth{*f.CategoryID, apptime.MonthOf(f.Day)}] -= v // расходы хранятся со знаком минус
		} else {
			in.income[apptime.MonthOf(f.Day)] += v
		}
	}

	out := buildBudget(in)
	out.Currency, out.MissingRates = base, missing
	return out, nil
}

// SetBudgeted задаёт, сколько заложено на категорию в месяце.
func (s *Service) SetBudgeted(ctx context.Context, u models.User, categoryID int64, month apptime.Month, minor int64) error {
	if minor < 0 || minor > maxAmountMinor {
		return fmt.Errorf("budgeted amount: %w", models.ErrInvalidBudget)
	}
	return s.plan.SetBudgeted(ctx, u.ActiveBudgetID, categoryID, month, minor)
}
