package models

import (
	"moneykeeper/pkg/apptime"
	"moneykeeper/pkg/money"
)

// MonthAmount это сумма, заложенная на категорию в месяце.
type MonthAmount struct {
	CategoryID int64
	Month      apptime.Month
	Minor      int64
}

// Flow это сумма записей одного дня в одной валюте. Расход (CategoryID задан, сумма отрицательная) или доход
// (CategoryID пуст, сумма положительная), переводы не входят. Из дневных сумм бюджет собирается по курсу каждого дня.
type Flow struct {
	CategoryID  *int64
	Currency    money.Currency
	Day         apptime.Day
	AmountMinor int64
}

// BudgetRow это строка бюджета: заложено, потрачено (положительное) и остаток категории в месяце, в базовой валюте.
type BudgetRow struct {
	CategoryID    int64
	BudgetedMinor int64
	SpentMinor    int64
	BalanceMinor  int64
}

// BudgetMonth это бюджет за месяц в базовой валюте. ToBudgetMinor накопительный: все доходы до конца месяца минус всё
// заложенное до него включительно, поэтому нераспределённое переходит дальше само. MissingRates это число дневных
// сумм, которые не удалось пересчитать из-за отсутствия курса; в итоги они не вошли.
type BudgetMonth struct {
	Month         apptime.Month
	Currency      money.Currency
	IncomeMinor   int64
	BudgetedMinor int64
	ToBudgetMinor int64
	MissingRates  int
	Rows          []BudgetRow
}
