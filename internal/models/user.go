package models

import (
	"moneykeeper/pkg/apptime"
	"moneykeeper/pkg/money"
)

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
	// AccountWallet (кошелёк, биржа) и AccountPiggy (копилка) лежат вне бюджета: их остатки не входят в «К распределению».
	AccountWallet AccountKind = "wallet"
	AccountPiggy  AccountKind = "piggy"
)

// CurrencyBalance это остаток и расходы счёта в одной валюте. Считаются из транзакций, нигде не хранятся.
// SpentMinor это сумма расходов за всё время (положительная), переводы и доходы в неё не входят.
//
// BaseBalanceMinor это остаток в базовой валюте бюджета по курсу сегодняшнего дня, BaseSpentMinor расходы
// в ней по курсу дня каждой записи. Оба nil, пока базовая валюта не выбрана или нет нужного курса.
type CurrencyBalance struct {
	Currency         money.Currency
	BalanceMinor     int64
	SpentMinor       int64
	BaseBalanceMinor *int64
	BaseSpentMinor   *int64
}

// DailySpend это расходы счёта в одной валюте за один день, положительные.
type DailySpend struct {
	AccountID  int64
	Currency   money.Currency
	Day        apptime.Day
	SpentMinor int64
}

// Account.Currency это основная валюта счёта. У обычного счёта она единственная, у счёта с
// MultiCurrency записи возможны в любой валюте, остатки считаются отдельно по каждой.
// В Balances основная валюта идёт первой, затем остальные по коду.
type Account struct {
	ID            int64
	BudgetID      int64
	Name          string
	Kind          AccountKind
	Currency      money.Currency
	MultiCurrency bool
	IsDefault     bool
	// OffBudget: счёт вне бюджета (кошелёк, копилка). Считается из вида счёта.
	OffBudget bool
	// Копилка: цель (сумма и дата) и конверт в группе «Цели накопления», из которого она пополняется.
	TargetMinor *int64
	TargetDate  *apptime.Day
	CategoryID  *int64
	Balances    []CurrencyBalance
}

// Allows говорит, можно ли вести на счёте записи в валюте c.
func (a Account) Allows(c money.Currency) bool { return a.MultiCurrency || a.Currency == c }

func (a Account) Balance(c money.Currency) int64 {
	for _, b := range a.Balances {
		if b.Currency == c {
			return b.BalanceMinor
		}
	}
	return 0
}

func (a Account) Spent(c money.Currency) int64 {
	for _, b := range a.Balances {
		if b.Currency == c {
			return b.SpentMinor
		}
	}
	return 0
}

// NewAccount.OpeningMinor — начальный остаток в основной валюте, он записывается доходом на дату создания счёта.
type NewAccount struct {
	Name          string
	Kind          AccountKind
	Currency      money.Currency
	MultiCurrency bool
	OpeningMinor  int64
	// Цель копилки: сумма и дата, оба поля необязательны. CategoryID это конверт копилки.
	TargetMinor *int64
	TargetDate  *apptime.Day
	CategoryID  *int64
}
