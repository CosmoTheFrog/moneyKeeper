package web

import (
	"io"
	"net/http"
	"strconv"

	"moneykeeper/internal/models"
	"moneykeeper/internal/web/apidto"
	"moneykeeper/pkg/apptime"
	"moneykeeper/pkg/money"
)

// previewImport разбирает выписку моделью и возвращает строки для подтверждения, ничего не сохраняя.
func (h *handler) previewImport(w http.ResponseWriter, r *http.Request, u models.User) {
	var req apidto.PreviewImportRequest
	if !h.decodeLimit(w, r, &req, maxImportBodyBytes) {
		return
	}
	p, err := h.svc.PreviewImport(r.Context(), u, req.AccountID, req.Text, req.Consent)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, previewResponse(p))
}

// previewImportFile читает xlsx-отчёт прежнего бота или PDF-выписку (multipart: account_id, file, consent) и возвращает
// строки для подтверждения. PDF не от TBC уходит на разбор модели и требует consent=true.
func (h *handler) previewImportFile(w http.ResponseWriter, r *http.Request, u models.User) {
	r.Body = http.MaxBytesReader(w, r.Body, maxImportFileBytes)
	if err := r.ParseMultipartForm(maxImportFileBytes); err != nil {
		h.writeError(w, http.StatusBadRequest, apidto.CodeInvalidRequest)
		return
	}
	accountID, err := strconv.ParseInt(r.FormValue("account_id"), 10, 64)
	f, _, ferr := r.FormFile("file")
	if err != nil || ferr != nil {
		h.writeError(w, http.StatusBadRequest, apidto.CodeInvalidRequest)
		return
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, apidto.CodeInvalidRequest)
		return
	}
	p, err := h.svc.PreviewFile(r.Context(), u, accountID, data, r.FormValue("consent") == "true")
	if err != nil {
		h.fail(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, previewResponse(p))
}

func previewResponse(p models.ImportPreview) apidto.PreviewImportResponse {
	out := apidto.PreviewImportResponse{Rows: make([]apidto.ImportRowDTO, 0, len(p.Rows)), Skipped: p.Skipped}
	for _, row := range p.Rows {
		out.Rows = append(out.Rows, apidto.ImportRowDTO{
			Date: row.Date.String(), AmountMinor: row.AmountMinor, Currency: string(row.Currency),
			Description: row.Description, CategoryID: row.CategoryID, SourceCategory: row.SourceCategory,
			Suggested: row.Suggested, Duplicate: row.Duplicate, ExternalID: row.ExternalID,
		})
	}
	return out
}

// commitImport сохраняет строки, которые пользователь подтвердил в превью.
func (h *handler) commitImport(w http.ResponseWriter, r *http.Request, u models.User) {
	var req apidto.CommitImportRequest
	if !h.decodeLimit(w, r, &req, maxImportBodyBytes) {
		return
	}
	rows := make([]models.ImportRow, 0, len(req.Rows))
	for _, d := range req.Rows {
		day, err := apptime.ParseDay(d.Date)
		if err != nil {
			h.writeError(w, http.StatusBadRequest, apidto.CodeInvalidDate)
			return
		}
		cur, err := money.Parse(d.Currency)
		if err != nil {
			h.fail(w, err)
			return
		}
		rows = append(rows, models.ImportRow{
			Date: day, AmountMinor: d.AmountMinor, Currency: cur, Description: d.Description, CategoryID: d.CategoryID,
			ExternalID: d.ExternalID,
		})
	}
	ids, err := h.svc.CommitImport(r.Context(), u, req.AccountID, rows)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.writeJSON(w, http.StatusCreated, apidto.CommitImportResponse{Saved: len(ids), IDs: ids})
}

// undoImport отменяет только что выполненный импорт: удаляет созданные им записи.
func (h *handler) undoImport(w http.ResponseWriter, r *http.Request, u models.User) {
	var req apidto.UndoImportRequest
	if !h.decodeLimit(w, r, &req, maxImportBodyBytes) {
		return
	}
	if err := h.svc.UndoImport(r.Context(), u, req.IDs); err != nil {
		h.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
