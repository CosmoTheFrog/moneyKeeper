// Package apidto описывает формат JSON API для Mini App. Модели наружу не сериализуются.
package apidto

const (
	CodeUnauthorized       = "unauthorized"
	CodeInvalidRequest     = "invalid_request"
	CodeInvalidCurrency    = "invalid_currency"
	CodeBaseCurrencySet    = "base_currency_set"
	CodeInvalidAccount     = "invalid_account"
	CodeInvalidTimezone    = "invalid_timezone"
	CodeInvalidCategory    = "invalid_category"
	CodeInvalidTransaction = "invalid_transaction"
	CodeInvalidDate        = "invalid_date"
	CodeCurrencyMismatch   = "currency_mismatch"
	CodeNoAmount           = "no_amount"
	CodeInvalidAmount      = "invalid_amount"
	CodeNotFound           = "not_found"
	CodeInternal           = "internal"
)

type ErrorResponse struct {
	Code string `json:"code"`
}

type BudgetDTO struct {
	ID           int64   `json:"id"`
	Name         string  `json:"name"`
	BaseCurrency *string `json:"base_currency"`
}

// CurrencyInfoDTO.Units это число знаков после запятой, Coin: курс берётся отдельно по требованию.
type CurrencyInfoDTO struct {
	Code  string `json:"code"`
	Units int    `json:"units"`
	Coin  bool   `json:"coin"`
}

type MeResponse struct {
	ID             int64             `json:"id"`
	FirstName      string            `json:"first_name"`
	Timezone       string            `json:"timezone"`
	Budget         BudgetDTO         `json:"budget"`
	BaseCurrencies []string          `json:"base_currencies"`
	Currencies     []string          `json:"currencies"`
	CurrencyInfo   []CurrencyInfoDTO `json:"currency_info"`
}

type SetCurrencyRequest struct {
	Currency string `json:"currency"`
}

// BalanceDTO это остаток и расходы счёта в одной валюте. SpentMinor положительная: расходы за всё время.
// Base* это то же в базовой валюте бюджета: остаток по сегодняшнему курсу, расходы по курсу дня каждой записи.
// null, если базовая валюта не выбрана или курса нет.
type BalanceDTO struct {
	Currency         string `json:"currency"`
	BalanceMinor     int64  `json:"balance_minor"`
	SpentMinor       int64  `json:"spent_minor"`
	BaseBalanceMinor *int64 `json:"base_balance_minor"`
	BaseSpentMinor   *int64 `json:"base_spent_minor"`
}

// AccountDTO.Currency это основная валюта счёта, в Balances она первая.
type AccountDTO struct {
	ID            int64        `json:"id"`
	Name          string       `json:"name"`
	Kind          string       `json:"kind"`
	Currency      string       `json:"currency"`
	MultiCurrency bool         `json:"multi_currency"`
	IsDefault     bool         `json:"is_default"`
	Balances      []BalanceDTO `json:"balances"`
}

type AccountsResponse struct {
	Accounts []AccountDTO `json:"accounts"`
}

type CreateAccountRequest struct {
	Name                string `json:"name"`
	Kind                string `json:"kind"`
	Currency            string `json:"currency"`
	MultiCurrency       bool   `json:"multi_currency"`
	OpeningBalanceMinor int64  `json:"opening_balance_minor"`
}

type CategoryDTO struct {
	ID          int64  `json:"id"`
	GroupID     int64  `json:"group_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Hidden      bool   `json:"hidden"`
}

type CategoriesResponse struct {
	Categories []CategoryDTO `json:"categories"`
}

type CreateCategoryRequest struct {
	Name    string `json:"name"`
	GroupID int64  `json:"group_id"`
}

// CategoryGroupDTO приходит в порядке показа. Name уже на языке пользователя.
type CategoryGroupDTO struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type CategoryGroupsResponse struct {
	Groups []CategoryGroupDTO `json:"groups"`
}

type CreateCategoryGroupRequest struct {
	Name string `json:"name"`
}

type UpdateCategoryRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Hidden      *bool   `json:"hidden"`
}

// TransactionDTO.AmountMinor со знаком: расход и уходящая нога перевода отрицательные.
// BaseMinor это та же сумма в базовой валюте по курсу дня записи, null если курса нет.
type TransactionDTO struct {
	ID          int64  `json:"id"`
	AccountID   int64  `json:"account_id"`
	CategoryID  *int64 `json:"category_id"`
	Kind        string `json:"kind"`
	AmountMinor int64  `json:"amount_minor"`
	Currency    string `json:"currency"`
	BaseMinor   *int64 `json:"base_minor"`
	Date        string `json:"date"`
	Note        string `json:"note"`
	TransferID  *int64 `json:"transfer_id"`
}

type TransactionsResponse struct {
	Transactions []TransactionDTO `json:"transactions"`
}

// CreateTransactionRequest.AmountMinor всегда положительная, знак задаёт Kind.
type CreateTransactionRequest struct {
	Kind        string  `json:"kind"`
	AccountID   int64   `json:"account_id"`
	ToAccountID *int64  `json:"to_account_id"`
	CategoryID  *int64  `json:"category_id"`
	AmountMinor int64   `json:"amount_minor"`
	Currency    string  `json:"currency"`
	Date        *string `json:"date"`
	Note        string  `json:"note"`
}

type UpdateTransactionRequest struct {
	AccountID   *int64  `json:"account_id"`
	CategoryID  *int64  `json:"category_id"`
	AmountMinor *int64  `json:"amount_minor"`
	Currency    *string `json:"currency"`
	Date        *string `json:"date"`
	Note        *string `json:"note"`
}

type SnapshotDTO struct {
	AccountID   int64  `json:"account_id"`
	CategoryID  *int64 `json:"category_id"`
	AmountMinor int64  `json:"amount_minor"`
	Date        string `json:"date"`
	Note        string `json:"note"`
}

type TransactionEventDTO struct {
	ID     int64        `json:"id"`
	Action string       `json:"action"`
	At     string       `json:"at"`
	Before *SnapshotDTO `json:"before"`
	After  *SnapshotDTO `json:"after"`
}

type TransactionEventsResponse struct {
	Events []TransactionEventDTO `json:"events"`
}

// ParseRequest.Currency необязательна: валюта формы, по ней считается точность числа, если в тексте валюты нет.
type ParseRequest struct {
	Text     string `json:"text"`
	Currency string `json:"currency"`
}

// ParseResponse.Currency пустая, если в тексте валюты нет.
type ParseResponse struct {
	AmountMinor int64  `json:"amount_minor"`
	Currency    string `json:"currency"`
	Note        string `json:"note"`
}

// RateDTO.Rate: сколько единиц валюты In стоит одна целая единица Currency, десятичной строкой без округления.
type RateDTO struct {
	Currency string `json:"currency"`
	Rate     string `json:"rate"`
}

type RatesResponse struct {
	Day   string    `json:"day"`
	In    string    `json:"in"`
	Rates []RateDTO `json:"rates"`
}
