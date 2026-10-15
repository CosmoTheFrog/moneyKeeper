package web

import (
	"net/http"

	"moneykeeper/internal/models"
	"moneykeeper/internal/web/apidto"
	"moneykeeper/pkg/apptime"
)

func (h *handler) month(w http.ResponseWriter, r *http.Request) (apptime.Month, bool) {
	m, err := apptime.ParseMonth(r.PathValue("month"))
	if err != nil {
		h.writeError(w, http.StatusBadRequest, apidto.CodeInvalidDate)
		return apptime.Month{}, false
	}
	return m, true
}

// budgetMonth отдаёт бюджет месяца: «К распределению» и строку на каждую категорию, всё в базовой валюте.
func (h *handler) budgetMonth(w http.ResponseWriter, r *http.Request, u models.User) {
	m, ok := h.month(w, r)
	if !ok {
		return
	}
	h.writeBudget(w, r, u, m, http.StatusOK)
}

// setBudgeted задаёт сумму, заложенную на категорию в месяце, и возвращает пересчитанный бюджет месяца.
func (h *handler) setBudgeted(w http.ResponseWriter, r *http.Request, u models.User) {
	m, ok := h.month(w, r)
	if !ok {
		return
	}
	id, ok := h.pathID(w, r)
	if !ok {
		return
	}
	var req apidto.SetBudgetedRequest
	if !h.decode(w, r, &req) {
		return
	}
	if err := h.svc.SetBudgeted(r.Context(), u, id, m, req.BudgetedMinor); err != nil {
		h.fail(w, err)
		return
	}
	h.writeBudget(w, r, u, m, http.StatusOK)
}

func (h *handler) writeBudget(w http.ResponseWriter, r *http.Request, u models.User, m apptime.Month, status int) {
	b, err := h.svc.BudgetMonth(r.Context(), u, m)
	if err != nil {
		h.fail(w, err)
		return
	}
	out := apidto.BudgetMonthResponse{
		Month: b.Month.String(), Currency: string(b.Currency), IncomeMinor: b.IncomeMinor, BudgetedMinor: b.BudgetedMinor,
		ToBudgetMinor: b.ToBudgetMinor, MissingRates: b.MissingRates, Categories: make([]apidto.BudgetRowDTO, 0, len(b.Rows)),
	}
	for _, row := range b.Rows {
		out.Categories = append(out.Categories, apidto.BudgetRowDTO{
			CategoryID: row.CategoryID, BudgetedMinor: row.BudgetedMinor, SpentMinor: row.SpentMinor, BalanceMinor: row.BalanceMinor,
		})
	}
	h.writeJSON(w, status, out)
}
