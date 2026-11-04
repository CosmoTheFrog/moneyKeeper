package models

import (
	"moneykeeper/pkg/apptime"
	"moneykeeper/pkg/money"
)

// DailyReminder: пора напомнить пользователю внести траты за день.
type DailyReminder struct {
	UserID     int64
	TelegramID int64
	Day        apptime.Day
}

// Пороги предупреждения о лимите категории, проценты заложенного.
const (
	LimitWarn = 80
	LimitFull = 100
)

// LimitAlert: расход категории за месяц дошёл до порога Threshold от заложенного. Уходит каждому участнику бюджета.
type LimitAlert struct {
	TelegramID    int64
	CategoryID    int64
	CategoryName  string
	Month         apptime.Month
	Threshold     int
	SpentMinor    int64
	BudgetedMinor int64
	Currency      money.Currency
}
