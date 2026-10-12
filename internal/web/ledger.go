package web

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"moneykeeper/internal/models"
	"moneykeeper/internal/web/apidto"
	"moneykeeper/pkg/apptime"
	"moneykeeper/pkg/quickinput"
)

func (h *handler) listCategories(w http.ResponseWriter, r *http.Request, u models.User) {
	cats, err := h.svc.ListCategories(r.Context(), u)
	if err != nil {
		h.fail(w, err)
		return
	}
	out := apidto.CategoriesResponse{Categories: make([]apidto.CategoryDTO, 0, len(cats))}
	for _, c := range cats {
		out.Categories = append(out.Categories, categoryDTO(c))
	}
	h.writeJSON(w, http.StatusOK, out)
}

func (h *handler) listCategoryGroups(w http.ResponseWriter, r *http.Request, u models.User) {
	groups, err := h.svc.ListCategoryGroups(r.Context(), u)
	if err != nil {
		h.fail(w, err)
		return
	}
	out := apidto.CategoryGroupsResponse{Groups: make([]apidto.CategoryGroupDTO, 0, len(groups))}
	for _, g := range groups {
		out.Groups = append(out.Groups, categoryGroupDTO(g))
	}
	h.writeJSON(w, http.StatusOK, out)
}

func (h *handler) createCategoryGroup(w http.ResponseWriter, r *http.Request, u models.User) {
	var req apidto.CreateCategoryGroupRequest
	if !h.decode(w, r, &req) {
		return
	}
	g, err := h.svc.CreateCategoryGroup(r.Context(), u, req.Name)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.writeJSON(w, http.StatusCreated, categoryGroupDTO(g))
}

func (h *handler) createCategory(w http.ResponseWriter, r *http.Request, u models.User) {
	var req apidto.CreateCategoryRequest
	if !h.decode(w, r, &req) {
		return
	}
	c, err := h.svc.CreateCategory(r.Context(), u, req.Name, req.GroupID)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.writeJSON(w, http.StatusCreated, categoryDTO(c))
}

func (h *handler) updateCategory(w http.ResponseWriter, r *http.Request, u models.User) {
	id, ok := h.pathID(w, r)
	if !ok {
		return
	}
	var req apidto.UpdateCategoryRequest
	if !h.decode(w, r, &req) {
		return
	}
	c, err := h.svc.UpdateCategory(r.Context(), u, id, models.CategoryPatch{Name: req.Name, Description: req.Description, Hidden: req.Hidden})
	if err != nil {
		h.fail(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, categoryDTO(c))
}

func (h *handler) listTransactions(w http.ResponseWriter, r *http.Request, u models.User) {
	limit, ok := h.queryInt(w, r, "limit")
	if !ok {
		return
	}
	offset, ok := h.queryInt(w, r, "offset")
	if !ok {
		return
	}
	txs, err := h.svc.ListTransactions(r.Context(), u, limit, offset)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, transactionsResponse(txs))
}

func (h *handler) createTransaction(w http.ResponseWriter, r *http.Request, u models.User) {
	var req apidto.CreateTransactionRequest
	if !h.decode(w, r, &req) {
		return
	}
	day, ok := h.optionalDay(w, req.Date)
	if !ok {
		return
	}
	txs, err := h.svc.CreateTransaction(r.Context(), u, models.NewTransaction{
		Kind: models.TransactionKind(req.Kind), AccountID: req.AccountID, ToAccountID: req.ToAccountID,
		CategoryID: req.CategoryID, AmountMinor: req.AmountMinor, Currency: req.Currency, OccurredOn: day, Note: req.Note,
	})
	if err != nil {
		h.fail(w, err)
		return
	}
	h.writeJSON(w, http.StatusCreated, transactionsResponse(txs))
}

func (h *handler) updateTransaction(w http.ResponseWriter, r *http.Request, u models.User) {
	id, ok := h.pathID(w, r)
	if !ok {
		return
	}
	var req apidto.UpdateTransactionRequest
	if !h.decode(w, r, &req) {
		return
	}
	day, ok := h.optionalDay(w, req.Date)
	if !ok {
		return
	}
	txs, err := h.svc.UpdateTransaction(r.Context(), u, id, models.TransactionPatch{
		AccountID: req.AccountID, CategoryID: req.CategoryID, AmountMinor: req.AmountMinor, Currency: req.Currency, OccurredOn: day, Note: req.Note,
	})
	if err != nil {
		h.fail(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, transactionsResponse(txs))
}

func (h *handler) deleteTransaction(w http.ResponseWriter, r *http.Request, u models.User) {
	id, ok := h.pathID(w, r)
	if !ok {
		return
	}
	if err := h.svc.DeleteTransaction(r.Context(), u, id); err != nil {
		h.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) transactionEvents(w http.ResponseWriter, r *http.Request, u models.User) {
	id, ok := h.pathID(w, r)
	if !ok {
		return
	}
	events, err := h.svc.TransactionEvents(r.Context(), u, id)
	if err != nil {
		h.fail(w, err)
		return
	}
	out := apidto.TransactionEventsResponse{Events: make([]apidto.TransactionEventDTO, 0, len(events))}
	for _, e := range events {
		out.Events = append(out.Events, apidto.TransactionEventDTO{
			ID: e.ID, Action: string(e.Action), At: e.At.UTC().Format(time.RFC3339),
			Before: snapshotDTO(e.Before), After: snapshotDTO(e.After),
		})
	}
	h.writeJSON(w, http.StatusOK, out)
}

// parseQuick разбирает текст быстрого ввода, ничего не сохраняя.
func (h *handler) parseQuick(w http.ResponseWriter, r *http.Request, _ models.User) {
	var req apidto.ParseRequest
	if !h.decode(w, r, &req) {
		return
	}
	res, err := quickinput.Parse(req.Text)
	switch {
	case errors.Is(err, quickinput.ErrNoAmount):
		h.writeError(w, http.StatusBadRequest, apidto.CodeNoAmount)
	case errors.Is(err, quickinput.ErrInvalidAmount):
		h.writeError(w, http.StatusBadRequest, apidto.CodeInvalidAmount)
	case err != nil:
		h.fail(w, err)
	default:
		h.writeJSON(w, http.StatusOK, apidto.ParseResponse{AmountMinor: res.AmountMinor, Currency: string(res.Currency), Note: res.Note})
	}
}

func (h *handler) pathID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		h.writeError(w, http.StatusBadRequest, apidto.CodeInvalidRequest)
		return 0, false
	}
	return id, true
}

// queryInt читает необязательный числовой параметр, отсутствующий даёт 0.
func (h *handler) queryInt(w http.ResponseWriter, r *http.Request, name string) (int, bool) {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return 0, true
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, apidto.CodeInvalidRequest)
		return 0, false
	}
	return v, true
}

func (h *handler) optionalDay(w http.ResponseWriter, s *string) (*apptime.Day, bool) {
	if s == nil {
		return nil, true
	}
	d, err := apptime.ParseDay(*s)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, apidto.CodeInvalidDate)
		return nil, false
	}
	return &d, true
}

func categoryDTO(c models.Category) apidto.CategoryDTO {
	return apidto.CategoryDTO{ID: c.ID, GroupID: c.GroupID, Name: c.Name, Description: c.Description, Hidden: c.Hidden}
}

func categoryGroupDTO(g models.CategoryGroup) apidto.CategoryGroupDTO {
	return apidto.CategoryGroupDTO{ID: g.ID, Name: g.Name}
}

func transactionsResponse(txs []models.Transaction) apidto.TransactionsResponse {
	out := apidto.TransactionsResponse{Transactions: make([]apidto.TransactionDTO, 0, len(txs))}
	for _, t := range txs {
		out.Transactions = append(out.Transactions, apidto.TransactionDTO{
			ID: t.ID, AccountID: t.AccountID, CategoryID: t.CategoryID, Kind: string(t.Kind),
			AmountMinor: t.AmountMinor, Currency: string(t.Currency), BaseMinor: t.BaseMinor, Date: t.OccurredOn.String(),
			Note: t.Note, TransferID: t.TransferID,
		})
	}
	return out
}

func snapshotDTO(s *models.TransactionSnapshot) *apidto.SnapshotDTO {
	if s == nil {
		return nil
	}
	return &apidto.SnapshotDTO{AccountID: s.AccountID, CategoryID: s.CategoryID, AmountMinor: s.AmountMinor, Date: s.OccurredOn.String(), Note: s.Note}
}
