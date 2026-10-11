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

type MeResponse struct {
	ID             int64     `json:"id"`
	FirstName      string    `json:"first_name"`
	Timezone       string    `json:"timezone"`
	Budget         BudgetDTO `json:"budget"`
	BaseCurrencies []string  `json:"base_currencies"`
	Currencies     []string  `json:"currencies"`
}

type SetCurrencyRequest struct {
	Currency string `json:"currency"`
}

type AccountDTO struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	Kind         string `json:"kind"`
	Currency     string `json:"currency"`
	IsDefault    bool   `json:"is_default"`
	BalanceMinor int64  `json:"balance_minor"`
}

type AccountsResponse struct {
	Accounts []AccountDTO `json:"accounts"`
}

type CreateAccountRequest struct {
	Name                string `json:"name"`
	Kind                string `json:"kind"`
	Currency            string `json:"currency"`
	OpeningBalanceMinor int64  `json:"opening_balance_minor"`
}

type CategoryDTO struct {
	ID          int64  `json:"id"`
	Group       string `json:"group"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Hidden      bool   `json:"hidden"`
}

type CategoriesResponse struct {
	Categories []CategoryDTO `json:"categories"`
}

type CreateCategoryRequest struct {
	Name  string `json:"name"`
	Group string `json:"group"`
}

type UpdateCategoryRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Hidden      *bool   `json:"hidden"`
}

// TransactionDTO.AmountMinor со знаком: расход и уходящая нога перевода отрицательные.
type TransactionDTO struct {
	ID          int64  `json:"id"`
	AccountID   int64  `json:"account_id"`
	CategoryID  *int64 `json:"category_id"`
	Kind        string `json:"kind"`
	AmountMinor int64  `json:"amount_minor"`
	Currency    string `json:"currency"`
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
	Date        *string `json:"date"`
	Note        string  `json:"note"`
}

type UpdateTransactionRequest struct {
	AccountID   *int64  `json:"account_id"`
	CategoryID  *int64  `json:"category_id"`
	AmountMinor *int64  `json:"amount_minor"`
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

type ParseRequest struct {
	Text string `json:"text"`
}

// ParseResponse.Currency пустая, если в тексте валюты нет.
type ParseResponse struct {
	AmountMinor int64  `json:"amount_minor"`
	Currency    string `json:"currency"`
	Note        string `json:"note"`
}
