package models

import (
	"moneykeeper/pkg/apptime"
	"moneykeeper/pkg/money"
)

type AmountKind string

const (
	AmountExact  AmountKind = "exact"
	AmountApprox AmountKind = "approx"
	AmountRange  AmountKind = "range"
)

type WeekendShift string

const (
	ShiftNone    WeekendShift = "none"
	ShiftBack    WeekendShift = "back"
	ShiftForward WeekendShift = "forward"
)

// Schedule это регулярный платёж: куда (категория, счёт), сколько и когда. Повторяется раз в EveryMonths месяцев,
// начиная со StartMonth, в числа Days (0 это последний день месяца). Платёж на выходной переносится по Shift.
// Для AmountRange верхняя граница в AmountMaxMinor, для AmountApprox допуск 7,5% от AmountMinor.
type Schedule struct {
	ID             int64
	BudgetID       int64
	Name           string
	CategoryID     int64
	AccountID      *int64
	Currency       money.Currency
	AmountKind     AmountKind
	AmountMinor    int64
	AmountMaxMinor *int64
	EveryMonths    int
	Days           []int
	StartMonth     apptime.Month
	Shift          WeekendShift
	RemindBefore   int
}

type NewSchedule struct {
	Name           string
	CategoryID     int64
	AccountID      *int64
	Currency       string
	AmountKind     AmountKind
	AmountMinor    int64
	AmountMaxMinor *int64
	EveryMonths    int
	Days           []int
	Shift          WeekendShift
	RemindBefore   int
	// StartMonth: с какого месяца считать повторение; пусто значит текущий месяц пользователя.
	StartMonth *apptime.Month
}

type OccurrenceStatus string

const (
	// StatusUpcoming платёж впереди; StatusOverdue дата прошла, а траты нет; StatusPaid найдена подходящая трата;
	// StatusSkipped платёж пропущен пользователем.
	StatusUpcoming OccurrenceStatus = "upcoming"
	StatusOverdue  OccurrenceStatus = "overdue"
	StatusPaid     OccurrenceStatus = "paid"
	StatusSkipped  OccurrenceStatus = "skipped"
)

// Occurrence это один платёж расписания на конкретную дату. TransactionID задан у оплаченного.
type Occurrence struct {
	Schedule      Schedule
	DueOn         apptime.Day
	Status        OccurrenceStatus
	TransactionID *int64
}

// Reminder это напоминание, которое пора отправить пользователю.
type Reminder struct {
	TelegramID int64
	Schedule   Schedule
	DueOn      apptime.Day
	Lead       int
}
