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
	CodeInvalidBudget      = "invalid_budget"
	CodeInvalidSchedule    = "invalid_schedule"
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
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	Kind          string `json:"kind"`
	Currency      string `json:"currency"`
	MultiCurrency bool   `json:"multi_currency"`
	IsDefault     bool   `json:"is_default"`
	// OffBudget: кошелёк или копилка, их остатки не входят в бюджет. У копилки есть цель и конверт category_id.
	OffBudget   bool         `json:"off_budget"`
	Archived    bool         `json:"archived"`
	TargetMinor *int64       `json:"target_minor"`
	TargetDate  *string      `json:"target_date"`
	CategoryID  *int64       `json:"category_id"`
	Balances    []BalanceDTO `json:"balances"`
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
	// Цель копилки (kind piggy): сумма и дата, оба поля необязательны.
	TargetMinor *int64  `json:"target_minor"`
	TargetDate  *string `json:"target_date"`
}

type CategoryDTO struct {
	ID          int64  `json:"id"`
	GroupID     int64  `json:"group_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Hidden      bool   `json:"hidden"`
	Rollover    bool   `json:"rollover"`
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
	Rollover    *bool   `json:"rollover"`
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
	// FeeMinor: комиссия сети при переводе между кошельками вне бюджета, приходит на amount_minor минус комиссия.
	FeeMinor int64 `json:"fee_minor"`
}

// CreateExchangeRequest: отдаётся out_minor в out_currency со счёта from, приходит in_minor в in_currency на счёт to.
// category_id нужна, если деньги уходят из бюджета на счёт вне бюджета. fee_minor в валюте отдаваемой суммы,
// fee_category_id для комиссии со счёта внутри бюджета.
type CreateExchangeRequest struct {
	FromAccountID int64   `json:"from_account_id"`
	ToAccountID   int64   `json:"to_account_id"`
	OutMinor      int64   `json:"out_minor"`
	OutCurrency   string  `json:"out_currency"`
	InMinor       int64   `json:"in_minor"`
	InCurrency    string  `json:"in_currency"`
	CategoryID    *int64  `json:"category_id"`
	FeeMinor      int64   `json:"fee_minor"`
	FeeCategoryID *int64  `json:"fee_category_id"`
	Date          *string `json:"date"`
	Note          string  `json:"note"`
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

// BudgetRowDTO: суммы категории за месяц в базовой валюте. SpentMinor положительная.
type BudgetRowDTO struct {
	CategoryID    int64 `json:"category_id"`
	BudgetedMinor int64 `json:"budgeted_minor"`
	SpentMinor    int64 `json:"spent_minor"`
	BalanceMinor  int64 `json:"balance_minor"`
}

// BudgetMonthResponse: ToBudgetMinor накопительное («К распределению»), MissingRates число сумм без курса, не вошедших в итоги.
type BudgetMonthResponse struct {
	Month         string         `json:"month"`
	Currency      string         `json:"currency"`
	IncomeMinor   int64          `json:"income_minor"`
	BudgetedMinor int64          `json:"budgeted_minor"`
	ToBudgetMinor int64          `json:"to_budget_minor"`
	MissingRates  int            `json:"missing_rates"`
	Categories    []BudgetRowDTO `json:"categories"`
}

type SetBudgetedRequest struct {
	BudgetedMinor int64 `json:"budgeted_minor"`
}

// PortfolioItemDTO: позиция по монете в валюте портфеля. ProfitBP прибыль в сотых долях процента, null без цены покупки.
// AvgPrice средняя цена покупки за целую монету десятичной строкой.
type PortfolioItemDTO struct {
	Currency      string `json:"currency"`
	QuantityMinor int64  `json:"quantity_minor"`
	ValueMinor    int64  `json:"value_minor"`
	CostMinor     int64  `json:"cost_minor"`
	ProfitMinor   int64  `json:"profit_minor"`
	ProfitBP      *int64 `json:"profit_bp"`
	RealizedMinor int64  `json:"realized_minor"`
	AvgPrice      string `json:"avg_price"`
}

type PortfolioResponse struct {
	Currency     string             `json:"currency"`
	MissingRates int                `json:"missing_rates"`
	Items        []PortfolioItemDTO `json:"items"`
}

// ScheduleDTO это регулярный платёж. Повторяется раз в every_months месяцев с start_month в числа days (0 последний день
// месяца). amount_kind: exact, approx (допуск 7,5%) или range (до amount_max_minor). weekend_shift: none, back, forward.
type ScheduleDTO struct {
	ID             int64  `json:"id"`
	Name           string `json:"name"`
	CategoryID     int64  `json:"category_id"`
	AccountID      *int64 `json:"account_id"`
	Currency       string `json:"currency"`
	AmountKind     string `json:"amount_kind"`
	AmountMinor    int64  `json:"amount_minor"`
	AmountMaxMinor *int64 `json:"amount_max_minor"`
	EveryMonths    int    `json:"every_months"`
	Days           []int  `json:"days"`
	StartMonth     string `json:"start_month"`
	WeekendShift   string `json:"weekend_shift"`
	RemindBefore   int    `json:"remind_before"`
}

type SchedulesResponse struct {
	Schedules []ScheduleDTO `json:"schedules"`
}

// ScheduleRequest создаёт или заменяет расписание. start_month необязателен (пусто: текущий месяц, при правке прежний).
type ScheduleRequest struct {
	Name           string  `json:"name"`
	CategoryID     int64   `json:"category_id"`
	AccountID      *int64  `json:"account_id"`
	Currency       string  `json:"currency"`
	AmountKind     string  `json:"amount_kind"`
	AmountMinor    int64   `json:"amount_minor"`
	AmountMaxMinor *int64  `json:"amount_max_minor"`
	EveryMonths    int     `json:"every_months"`
	Days           []int   `json:"days"`
	StartMonth     *string `json:"start_month"`
	WeekendShift   string  `json:"weekend_shift"`
	RemindBefore   int     `json:"remind_before"`
}

// OccurrenceDTO это платёж на дату со статусом: upcoming, overdue, paid или skipped. transaction_id у оплаченного.
type OccurrenceDTO struct {
	Schedule      ScheduleDTO `json:"schedule"`
	DueOn         string      `json:"due_on"`
	Status        string      `json:"status"`
	TransactionID *int64      `json:"transaction_id"`
}

type OccurrencesResponse struct {
	Occurrences []OccurrenceDTO `json:"occurrences"`
}

type SkipRequest struct {
	DueOn string `json:"due_on"`
}
