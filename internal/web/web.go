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
	ListAccounts(ctx context.Context, u models.User) ([]models.Account, error)
	CreateAccount(ctx context.Context, u models.User, name, kind, currency string) (models.Account, error)
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
	mux.HandleFunc("GET /api/accounts", h.auth(h.listAccounts))
	mux.HandleFunc("POST /api/accounts", h.auth(h.createAccount))
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
		Currencies:     currencyCodes(money.All),
	})
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
	accounts, err := h.svc.ListAccounts(r.Context(), u)
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
	a, err := h.svc.CreateAccount(r.Context(), u, req.Name, req.Kind, req.Currency)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.writeJSON(w, http.StatusCreated, accountDTO(a))
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
	return apidto.AccountDTO{ID: a.ID, Name: a.Name, Kind: string(a.Kind), Currency: string(a.Currency), IsDefault: a.IsDefault}
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
	case errors.Is(err, money.ErrUnknownCurrency):
		h.writeError(w, http.StatusBadRequest, apidto.CodeInvalidCurrency)
	case errors.Is(err, models.ErrInvalidAccount):
		h.writeError(w, http.StatusBadRequest, apidto.CodeInvalidAccount)
	case errors.Is(err, models.ErrInvalidTimezone):
		h.writeError(w, http.StatusBadRequest, apidto.CodeInvalidTimezone)
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
