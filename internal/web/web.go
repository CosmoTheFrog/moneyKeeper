package web

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"moneykeeper/internal/models"
	"moneykeeper/internal/web/apidto"
	"moneykeeper/internal/web/tgauth"
	"moneykeeper/pkg/apptime"
	"moneykeeper/pkg/money"
)

const (
	authScheme     = "tma "
	timezoneHeader = "X-Timezone"
	defaultTZ      = "UTC"
	initDataTTL    = 24 * time.Hour
	maxBodyBytes   = 4096
)

type Service interface {
	Resolve(ctx context.Context, tg models.TelegramUser, timezone string) (models.User, error)
	Budget(ctx context.Context, u models.User) (models.Budget, error)
	SetBaseCurrency(ctx context.Context, u models.User, code string) (models.Budget, error)
	ListAccounts(ctx context.Context, u models.User, in string) ([]models.Account, error)
	CreateAccount(ctx context.Context, u models.User, name, kind, currency string, multiCurrency bool, openingMinor int64) (models.Account, error)
	ArchiveAccount(ctx context.Context, u models.User, id int64) error
	AddAsset(ctx context.Context, code string, stock bool) (money.Info, error)
	CreatePiggy(ctx context.Context, u models.User, name, currency string, targetMinor *int64, targetDate *apptime.Day) (models.Account, error)

	ListCategories(ctx context.Context, u models.User) ([]models.Category, error)
	CreateCategory(ctx context.Context, u models.User, name string, groupID int64) (models.Category, error)
	ListCategoryGroups(ctx context.Context, u models.User) ([]models.CategoryGroup, error)
	CreateCategoryGroup(ctx context.Context, u models.User, name string) (models.CategoryGroup, error)
	UpdateCategory(ctx context.Context, u models.User, id int64, p models.CategoryPatch) (models.Category, error)

	ListTransactions(ctx context.Context, u models.User, limit, offset int, in string) ([]models.Transaction, error)
	Rates(ctx context.Context, u models.User, in string) (models.RateTable, error)
	Portfolio(ctx context.Context, u models.User, in string) (models.Portfolio, error)
	BudgetMonth(ctx context.Context, u models.User, month apptime.Month) (models.BudgetMonth, error)

	ListSchedules(ctx context.Context, u models.User) ([]models.Schedule, error)
	CreateSchedule(ctx context.Context, u models.User, in models.NewSchedule) (models.Schedule, error)
	UpdateSchedule(ctx context.Context, u models.User, id int64, in models.NewSchedule) (models.Schedule, error)
	DeleteSchedule(ctx context.Context, u models.User, id int64) error
	Occurrences(ctx context.Context, u models.User, from, to *apptime.Day) ([]models.Occurrence, error)
	SkipOccurrence(ctx context.Context, u models.User, id int64, due apptime.Day) error

	SetBudgeted(ctx context.Context, u models.User, categoryID int64, month apptime.Month, minor int64) error
	CreateTransaction(ctx context.Context, u models.User, in models.NewTransaction) ([]models.Transaction, error)
	CreateExchange(ctx context.Context, u models.User, in models.NewExchange) ([]models.Transaction, error)
	UpdateTransaction(ctx context.Context, u models.User, id int64, p models.TransactionPatch) ([]models.Transaction, error)
	DeleteTransaction(ctx context.Context, u models.User, id int64) error
	TransactionEvents(ctx context.Context, u models.User, id int64) ([]models.TransactionEvent, error)
}

type handler struct {
	svc      Service
	botToken string
	log      *slog.Logger
	now      func() time.Time
}

func NewHandler(svc Service, botToken string, log *slog.Logger) http.Handler {
	h := &handler{svc: svc, botToken: botToken, log: log, now: time.Now}
	return h.routes()
}

func (h *handler) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", h.healthz)
	mux.HandleFunc("GET /api/me", h.auth(h.me))
	mux.HandleFunc("PUT /api/budget/currency", h.auth(h.setBaseCurrency))
	mux.HandleFunc("POST /api/assets", h.auth(h.addAsset))
	mux.HandleFunc("GET /api/rates", h.auth(h.rates))
	mux.HandleFunc("GET /api/portfolio", h.auth(h.portfolio))
	mux.HandleFunc("GET /api/schedules", h.auth(h.listSchedules))
	mux.HandleFunc("POST /api/schedules", h.auth(h.createSchedule))
	mux.HandleFunc("GET /api/schedules/occurrences", h.auth(h.occurrences))
	mux.HandleFunc("PUT /api/schedules/{id}", h.auth(h.updateSchedule))
	mux.HandleFunc("DELETE /api/schedules/{id}", h.auth(h.deleteSchedule))
	mux.HandleFunc("POST /api/schedules/{id}/skip", h.auth(h.skipOccurrence))
	mux.HandleFunc("GET /api/budget/months/{month}", h.auth(h.budgetMonth))
	mux.HandleFunc("PUT /api/budget/months/{month}/categories/{id}", h.auth(h.setBudgeted))
	mux.HandleFunc("GET /api/accounts", h.auth(h.listAccounts))
	mux.HandleFunc("POST /api/accounts", h.auth(h.createAccount))
	mux.HandleFunc("POST /api/accounts/{id}/archive", h.auth(h.archiveAccount))
	mux.HandleFunc("GET /api/category-groups", h.auth(h.listCategoryGroups))
	mux.HandleFunc("POST /api/category-groups", h.auth(h.createCategoryGroup))
	mux.HandleFunc("GET /api/categories", h.auth(h.listCategories))
	mux.HandleFunc("POST /api/categories", h.auth(h.createCategory))
	mux.HandleFunc("PATCH /api/categories/{id}", h.auth(h.updateCategory))
	mux.HandleFunc("GET /api/transactions", h.auth(h.listTransactions))
	mux.HandleFunc("POST /api/transactions", h.auth(h.createTransaction))
	mux.HandleFunc("POST /api/exchanges", h.auth(h.createExchange))
	mux.HandleFunc("POST /api/transactions/parse", h.auth(h.parseQuick))
	mux.HandleFunc("PATCH /api/transactions/{id}", h.auth(h.updateTransaction))
	mux.HandleFunc("DELETE /api/transactions/{id}", h.auth(h.deleteTransaction))
	mux.HandleFunc("GET /api/transactions/{id}/events", h.auth(h.transactionEvents))
	return mux
}

func (h *handler) healthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

type userHandler func(w http.ResponseWriter, r *http.Request, u models.User)

// auth проверяет подпись initData и находит (или создаёт при первом входе) пользователя.
func (h *handler) auth(next userHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		initData, ok := strings.CutPrefix(r.Header.Get("Authorization"), authScheme)
		if !ok {
			h.writeError(w, http.StatusUnauthorized, apidto.CodeUnauthorized)
			return
		}
		tg, err := tgauth.Validate(initData, h.botToken, initDataTTL, h.now())
		if err != nil {
			h.log.Info("init data rejected", "err", err)
			h.writeError(w, http.StatusUnauthorized, apidto.CodeUnauthorized)
			return
		}
		tz := r.Header.Get(timezoneHeader)
		if tz == "" {
			tz = defaultTZ
		}
		u, err := h.svc.Resolve(r.Context(), tg, tz)
		if err != nil {
			h.fail(w, err)
			return
		}
		next(w, r, u)
	}
}

func (h *handler) me(w http.ResponseWriter, r *http.Request, u models.User) {
	b, err := h.svc.Budget(r.Context(), u)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, apidto.MeResponse{
		ID:             u.ID,
		FirstName:      u.FirstName,
		Timezone:       u.Timezone,
		Budget:         budgetDTO(b),
		BaseCurrencies: currencyCodes(money.Base),
		Currencies:     currencyCodes(money.All()),
		CurrencyInfo:   currencyInfo(),
	})
}

// addAsset добавляет монету или акцию по коду, цена проверяется у источника. Ответ это описание актива.
func (h *handler) addAsset(w http.ResponseWriter, r *http.Request, _ models.User) {
	var req apidto.AddAssetRequest
	if !h.decode(w, r, &req) {
		return
	}
	i, err := h.svc.AddAsset(r.Context(), req.Code, req.Stock)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.writeJSON(w, http.StatusCreated, apidto.CurrencyInfoDTO{Code: string(i.Code), Units: i.Units, Coin: i.Coin, Stock: i.Stock})
}

func currencyInfo() []apidto.CurrencyInfoDTO {
	infos := money.Infos()
	out := make([]apidto.CurrencyInfoDTO, 0, len(infos))
	for _, i := range infos {
		out = append(out, apidto.CurrencyInfoDTO{Code: string(i.Code), Units: i.Units, Coin: i.Coin, Stock: i.Stock})
	}
	return out
}

func currencyCodes(cs []money.Currency) []string {
	out := make([]string, 0, len(cs))
	for _, c := range cs {
		out = append(out, string(c))
	}
	return out
}

func (h *handler) setBaseCurrency(w http.ResponseWriter, r *http.Request, u models.User) {
	var req apidto.SetCurrencyRequest
	if !h.decode(w, r, &req) {
		return
	}
	b, err := h.svc.SetBaseCurrency(r.Context(), u, req.Currency)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, budgetDTO(b))
}

func (h *handler) listAccounts(w http.ResponseWriter, r *http.Request, u models.User) {
	accounts, err := h.svc.ListAccounts(r.Context(), u, r.URL.Query().Get("in"))
	if err != nil {
		h.fail(w, err)
		return
	}
	out := apidto.AccountsResponse{Accounts: make([]apidto.AccountDTO, 0, len(accounts))}
	for _, a := range accounts {
		out.Accounts = append(out.Accounts, accountDTO(a))
	}
	h.writeJSON(w, http.StatusOK, out)
}

func (h *handler) createAccount(w http.ResponseWriter, r *http.Request, u models.User) {
	var req apidto.CreateAccountRequest
	if !h.decode(w, r, &req) {
		return
	}
	var (
		a   models.Account
		err error
	)
	if req.Kind == string(models.AccountPiggy) {
		day, ok := h.optionalDay(w, req.TargetDate)
		if !ok {
			return
		}
		a, err = h.svc.CreatePiggy(r.Context(), u, req.Name, req.Currency, req.TargetMinor, day)
	} else {
		a, err = h.svc.CreateAccount(r.Context(), u, req.Name, req.Kind, req.Currency, req.MultiCurrency, req.OpeningBalanceMinor)
	}
	if err != nil {
		h.fail(w, err)
		return
	}
	h.writeJSON(w, http.StatusCreated, accountDTO(a))
}

// archiveAccount закрывает пустой неосновной счёт: он пропадает из списков, записи остаются в истории.
func (h *handler) archiveAccount(w http.ResponseWriter, r *http.Request, u models.User) {
	id, ok := h.pathID(w, r)
	if !ok {
		return
	}
	if err := h.svc.ArchiveAccount(r.Context(), u, id); err != nil {
		h.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func budgetDTO(b models.Budget) apidto.BudgetDTO {
	dto := apidto.BudgetDTO{ID: b.ID, Name: b.Name}
	if b.BaseCurrency != nil {
		c := string(*b.BaseCurrency)
		dto.BaseCurrency = &c
	}
	return dto
}

func accountDTO(a models.Account) apidto.AccountDTO {
	balances := make([]apidto.BalanceDTO, 0, len(a.Balances))
	for _, b := range a.Balances {
		balances = append(balances, apidto.BalanceDTO{
			Currency: string(b.Currency), BalanceMinor: b.BalanceMinor, SpentMinor: b.SpentMinor,
			BaseBalanceMinor: b.BaseBalanceMinor, BaseSpentMinor: b.BaseSpentMinor,
		})
	}
	dto := apidto.AccountDTO{
		ID: a.ID, Name: a.Name, Kind: string(a.Kind), Currency: string(a.Currency),
		MultiCurrency: a.MultiCurrency, IsDefault: a.IsDefault, OffBudget: a.OffBudget, Archived: a.Archived,
		TargetMinor: a.TargetMinor, CategoryID: a.CategoryID, Balances: balances,
	}
	if a.TargetDate != nil {
		d := a.TargetDate.String()
		dto.TargetDate = &d
	}
	return dto
}

func (h *handler) decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		h.writeError(w, http.StatusBadRequest, apidto.CodeInvalidRequest)
		return false
	}
	return true
}

// fail сопоставляет ошибку сервиса с HTTP-статусом и устойчивым кодом.
func (h *handler) fail(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, models.ErrAssetNotFound):
		h.writeError(w, http.StatusNotFound, apidto.CodeAssetNotFound)
	case errors.Is(err, money.ErrBadCode):
		h.writeError(w, http.StatusBadRequest, apidto.CodeInvalidCurrency)
	case errors.Is(err, money.ErrUnknownCurrency):
		h.writeError(w, http.StatusBadRequest, apidto.CodeInvalidCurrency)
	case errors.Is(err, models.ErrInvalidAccount):
		h.writeError(w, http.StatusBadRequest, apidto.CodeInvalidAccount)
	case errors.Is(err, models.ErrInvalidTimezone):
		h.writeError(w, http.StatusBadRequest, apidto.CodeInvalidTimezone)
	case errors.Is(err, models.ErrInvalidCategory):
		h.writeError(w, http.StatusBadRequest, apidto.CodeInvalidCategory)
	case errors.Is(err, models.ErrInvalidSchedule):
		h.writeError(w, http.StatusBadRequest, apidto.CodeInvalidSchedule)
	case errors.Is(err, models.ErrInvalidBudget):
		h.writeError(w, http.StatusBadRequest, apidto.CodeInvalidBudget)
	case errors.Is(err, models.ErrInvalidTransaction):
		h.writeError(w, http.StatusBadRequest, apidto.CodeInvalidTransaction)
	case errors.Is(err, models.ErrCurrencyMismatch):
		h.writeError(w, http.StatusBadRequest, apidto.CodeCurrencyMismatch)
	case errors.Is(err, models.ErrBaseCurrencySet):
		h.writeError(w, http.StatusConflict, apidto.CodeBaseCurrencySet)
	case errors.Is(err, models.ErrNotFound):
		h.writeError(w, http.StatusNotFound, apidto.CodeNotFound)
	default:
		h.log.Error("request failed", "err", err)
		h.writeError(w, http.StatusInternalServerError, apidto.CodeInternal)
	}
}

func (h *handler) writeError(w http.ResponseWriter, status int, code string) {
	h.writeJSON(w, status, apidto.ErrorResponse{Code: code})
}

func (h *handler) writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		h.log.Error("write response", "err", err)
	}
}
