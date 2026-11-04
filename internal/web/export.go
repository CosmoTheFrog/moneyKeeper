package web

import (
	"net/http"

	"moneykeeper/internal/models"
	"moneykeeper/internal/web/apidto"
	"moneykeeper/pkg/apptime"
)

// exportTransactions отдаёт CSV с записями за период (параметры from и to, оба обязательные).
func (h *handler) exportTransactions(w http.ResponseWriter, r *http.Request, u models.User) {
	from, ferr := apptime.ParseDay(r.URL.Query().Get("from"))
	to, terr := apptime.ParseDay(r.URL.Query().Get("to"))
	if ferr != nil || terr != nil || to.Before(from) {
		h.writeError(w, http.StatusBadRequest, apidto.CodeInvalidDate)
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="moneykeeper-`+from.String()+`_`+to.String()+`.csv"`)
	// BOM нужен Excel, иначе он читает кириллицу в чужой кодировке.
	if _, err := w.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
		return
	}
	if err := h.svc.ExportTransactions(r.Context(), u, from, to, w); err != nil {
		h.log.Error("export transactions", "err", err)
	}
}
