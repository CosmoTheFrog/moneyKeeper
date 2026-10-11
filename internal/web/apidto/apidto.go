// Package apidto описывает формат JSON API для Mini App. Модели наружу не сериализуются.
package apidto

const (
	CodeUnauthorized    = "unauthorized"
	CodeInvalidRequest  = "invalid_request"
	CodeInvalidCurrency = "invalid_currency"
	CodeBaseCurrencySet = "base_currency_set"
	CodeInvalidAccount  = "invalid_account"
	CodeInvalidTimezone = "invalid_timezone"
	CodeNotFound        = "not_found"
	CodeInternal        = "internal"
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
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	Currency  string `json:"currency"`
	IsDefault bool   `json:"is_default"`
}

type AccountsResponse struct {
	Accounts []AccountDTO `json:"accounts"`
}

type CreateAccountRequest struct {
	Name     string `json:"name"`
	Kind     string `json:"kind"`
	Currency string `json:"currency"`
}
