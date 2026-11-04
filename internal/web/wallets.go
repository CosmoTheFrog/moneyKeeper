package web

import (
	"net/http"

	"moneykeeper/internal/models"
	"moneykeeper/internal/web/apidto"
)

func walletDTO(w models.WalletAddress) apidto.WalletDTO {
	return apidto.WalletDTO{ID: w.ID, Chain: w.Chain, Address: w.Address}
}

func (h *handler) listWallets(w http.ResponseWriter, r *http.Request, u models.User) {
	id, ok := h.pathID(w, r)
	if !ok {
		return
	}
	list, err := h.svc.ListWallets(r.Context(), u, id)
	if err != nil {
		h.fail(w, err)
		return
	}
	out := apidto.WalletsResponse{Wallets: make([]apidto.WalletDTO, 0, len(list))}
	for _, x := range list {
		out.Wallets = append(out.Wallets, walletDTO(x))
	}
	h.writeJSON(w, http.StatusOK, out)
}

// addWallet привязывает публичный адрес к счёту-кошельку. Без consent адрес не сохраняется.
func (h *handler) addWallet(w http.ResponseWriter, r *http.Request, u models.User) {
	id, ok := h.pathID(w, r)
	if !ok {
		return
	}
	var req apidto.AddWalletRequest
	if !h.decode(w, r, &req) {
		return
	}
	x, err := h.svc.AddWallet(r.Context(), u, id, req.Chain, req.Address, req.Consent)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.writeJSON(w, http.StatusCreated, walletDTO(x))
}

func (h *handler) removeWallet(w http.ResponseWriter, r *http.Request, u models.User) {
	id, ok := h.pathID(w, r)
	if !ok {
		return
	}
	if err := h.svc.RemoveWallet(r.Context(), u, id); err != nil {
		h.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// previewWallet читает переводы адреса и возвращает строки для подтверждения, ничего не сохраняя.
func (h *handler) previewWallet(w http.ResponseWriter, r *http.Request, u models.User) {
	id, ok := h.pathID(w, r)
	if !ok {
		return
	}
	p, err := h.svc.PreviewWallet(r.Context(), u, id)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, previewResponse(p))
}
