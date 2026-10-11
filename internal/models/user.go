package models

import "moneykeeper/pkg/money"

type TelegramUser struct {
	ID        int64
	FirstName string
}

type User struct {
	ID             int64
	TelegramID     int64
	FirstName      string
	Timezone       string
	ActiveBudgetID int64
}

type Budget struct {
	ID           int64
	Name         string
	BaseCurrency *money.Currency
}

type AccountKind string

const (
	AccountCash   AccountKind = "cash"
	AccountDebit  AccountKind = "debit"
	AccountCredit AccountKind = "credit"
)

// Account.BalanceMinor считается из транзакций, в таблице счетов не хранится.
type Account struct {
	ID           int64
	BudgetID     int64
	Name         string
	Kind         AccountKind
	Currency     money.Currency
	IsDefault    bool
	BalanceMinor int64
}

// NewAccount.OpeningMinor — начальный остаток, он записывается доходом на дату создания счёта.
type NewAccount struct {
	Name         string
	Kind         AccountKind
	Currency     money.Currency
	OpeningMinor int64
}
