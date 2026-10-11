package models

import (
	"time"

	"moneykeeper/pkg/apptime"
	"moneykeeper/pkg/money"
)

type CategoryGroup string

const (
	GroupCritical CategoryGroup = "critical"
	GroupDebt     CategoryGroup = "debt"
	GroupDaily    CategoryGroup = "daily"
	GroupRainy    CategoryGroup = "rainy"
	GroupGoal     CategoryGroup = "goal"
)

type Category struct {
	ID          int64
	BudgetID    int64
	Group       CategoryGroup
	Name        string
	Description string
	Hidden      bool
}

type NewCategory struct {
	Group       CategoryGroup
	Name        string
	Description string
}

type CategoryPatch struct {
	Name        *string
	Description *string
	Hidden      *bool
}

type TransactionKind string

const (
	KindExpense  TransactionKind = "expense"
	KindIncome   TransactionKind = "income"
	KindTransfer TransactionKind = "transfer"
	// KindOpening это начальный остаток счёта: доход в «К распределению» на дату создания счёта.
	KindOpening TransactionKind = "opening"
)

// Transaction хранит сумму со знаком: расход и уходящая нога перевода отрицательные.
type Transaction struct {
	ID          int64
	BudgetID    int64
	AccountID   int64
	CategoryID  *int64
	Kind        TransactionKind
	AmountMinor int64
	Currency    money.Currency
	OccurredOn  apptime.Day
	Note        string
	TransferID  *int64
	CreatedBy   int64
	CreatedAt   time.Time
}

// NewTransaction описывает ввод пользователя. AmountMinor всегда положительная.
type NewTransaction struct {
	Kind        TransactionKind
	AccountID   int64
	ToAccountID *int64
	CategoryID  *int64
	AmountMinor int64
	OccurredOn  *apptime.Day
	Note        string
}

// TransactionPatch меняет только заданные поля. AmountMinor положительная.
type TransactionPatch struct {
	AccountID   *int64
	CategoryID  *int64
	AmountMinor *int64
	OccurredOn  *apptime.Day
	Note        *string
}

type EventAction string

const (
	EventCreate EventAction = "create"
	EventUpdate EventAction = "update"
	EventDelete EventAction = "delete"
)

// TransactionSnapshot это состояние записи в момент события истории.
type TransactionSnapshot struct {
	AccountID   int64       `json:"account_id"`
	CategoryID  *int64      `json:"category_id"`
	AmountMinor int64       `json:"amount_minor"`
	OccurredOn  apptime.Day `json:"occurred_on"`
	Note        string      `json:"note"`
}

type TransactionEvent struct {
	ID            int64
	TransactionID int64
	Action        EventAction
	Before        *TransactionSnapshot
	After         *TransactionSnapshot
	At            time.Time
}
