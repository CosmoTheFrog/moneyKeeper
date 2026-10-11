package web

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"moneykeeper/internal/web/apidto"
	"moneykeeper/internal/web/tgauth"
)

const (
	authScheme  = "tma "
	initDataTTL = 24 * time.Hour
)

type handler struct {
	botToken string
	log      *slog.Logger
	now      func() time.Time
}

func NewHandler(botToken string, log *slog.Logger) http.Handler {
	h := &handler{botToken: botToken, log: log, now: time.Now}
	return h.routes()
}

func (h *handler) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", h.healthz)
	mux.HandleFunc("GET /api/me", h.me)
	return mux
}

func (h *handler) healthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (h *handler) me(w http.ResponseWriter, r *http.Request) {
	initData, ok := strings.CutPrefix(r.Header.Get("Authorization"), authScheme)
	if !ok {
		h.writeJSON(w, http.StatusUnauthorized, apidto.ErrorResponse{Code: apidto.CodeUnauthorized})
		return
	}
	user, err := tgauth.Validate(initData, h.botToken, initDataTTL, h.now())
	if err != nil {
		h.log.Info("init data rejected", "err", err)
		h.writeJSON(w, http.StatusUnauthorized, apidto.ErrorResponse{Code: apidto.CodeUnauthorized})
		return
	}
	h.writeJSON(w, http.StatusOK, apidto.MeResponse{ID: user.ID, FirstName: user.FirstName})
}

func (h *handler) writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		h.log.Error("write response", "err", err)
	}
}
