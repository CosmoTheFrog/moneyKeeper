package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"moneykeeper/internal/chain"
	"moneykeeper/internal/models"
	"moneykeeper/internal/repo"
	"moneykeeper/pkg/apptime"
)

// chainHistory: переводы старше этого срока при чтении сети не берутся.
const chainHistory = 90 * 24 * time.Hour

// maxExternalIDLen ограничивает идентификатор перевода из сети (сеть, хэш, актив, номер события).
const maxExternalIDLen = 200

// ChainSource читает переводы публичного адреса. Реализует chain.Client.
type ChainSource interface {
	Transfers(ctx context.Context, ch chain.Chain, address string, since time.Time) ([]chain.Transfer, error)
}

// WithWallets включает привязку криптокошельков. Без него методы кошельков возвращают ErrImportUnavailable.
func (s *Service) WithWallets(w *repo.Wallets, src ChainSource) *Service {
	s.wallets, s.chains = w, src
	return s
}

// AddWallet привязывает публичный адрес к счёту-кошельку. consent это явное согласие отправить адрес во внешний
// источник; без него адрес не сохраняется.
func (s *Service) AddWallet(ctx context.Context, u models.User, accountID int64, chainName, address string, consent bool) (models.WalletAddress, error) {
	if !consent {
		return models.WalletAddress{}, models.ErrImportConsent
	}
	if s.wallets == nil {
		return models.WalletAddress{}, models.ErrImportUnavailable
	}
	address = strings.TrimSpace(address)
	if !chain.Valid(chain.Chain(chainName), address) {
		return models.WalletAddress{}, fmt.Errorf("address for %s: %w", chainName, models.ErrInvalidImport)
	}
	acc, err := s.accounts.Get(ctx, u.ActiveBudgetID, accountID)
	if err != nil {
		return models.WalletAddress{}, err
	}
	if acc.Kind != models.AccountWallet {
		return models.WalletAddress{}, fmt.Errorf("account %d is not a wallet: %w", acc.ID, models.ErrInvalidImport)
	}
	return s.wallets.Add(ctx, acc.ID, chainName, address)
}

func (s *Service) ListWallets(ctx context.Context, u models.User, accountID int64) ([]models.WalletAddress, error) {
	if s.wallets == nil {
		return nil, models.ErrImportUnavailable
	}
	return s.wallets.List(ctx, u.ActiveBudgetID, accountID)
}

func (s *Service) RemoveWallet(ctx context.Context, u models.User, id int64) error {
	if s.wallets == nil {
		return models.ErrImportUnavailable
	}
	return s.wallets.Delete(ctx, u.ActiveBudgetID, id)
}

// PreviewWallet читает переводы адреса за последние chainHistory и возвращает их строками импорта, ничего не
// сохраняя. Уже внесённые помечены Duplicate, активы, которых нет на счёте, считаются в Skipped. Тип операции
// выбирает пользователь: пришедшее станет доходом, ушедшее расходом с категорией, любую строку можно не вносить.
func (s *Service) PreviewWallet(ctx context.Context, u models.User, walletID int64) (models.ImportPreview, error) {
	if s.wallets == nil || s.chains == nil {
		return models.ImportPreview{}, models.ErrImportUnavailable
	}
	w, err := s.wallets.Get(ctx, u.ActiveBudgetID, walletID)
	if err != nil {
		return models.ImportPreview{}, err
	}
	acc, err := s.accounts.Get(ctx, u.ActiveBudgetID, w.AccountID)
	if err != nil {
		return models.ImportPreview{}, err
	}
	transfers, err := s.chains.Transfers(ctx, chain.Chain(w.Chain), w.Address, s.now().Add(-chainHistory))
	switch {
	case chain.IsBadAddress(err):
		return models.ImportPreview{}, fmt.Errorf("%w: %w", models.ErrInvalidImport, err)
	case errors.Is(err, chain.ErrUnavailable):
		return models.ImportPreview{}, fmt.Errorf("%w: %w", models.ErrImportUnavailable, err)
	case err != nil:
		return models.ImportPreview{}, err
	}

	ids := make([]string, 0, len(transfers))
	for _, t := range transfers {
		ids = append(ids, t.ID)
	}
	done, err := s.wallets.Imported(ctx, acc.ID, ids)
	if err != nil {
		return models.ImportPreview{}, err
	}

	loc, err := time.LoadLocation(u.Timezone)
	if err != nil {
		loc = time.UTC
	}
	var out models.ImportPreview
	for _, t := range transfers {
		if !acc.Allows(t.Currency) {
			out.Skipped++
			continue
		}
		out.Rows = append(out.Rows, models.ImportRow{
			Date: apptime.Today(loc, t.At), AmountMinor: t.AmountMinor, Currency: t.Currency,
			Description: transferNote(t), Duplicate: done[t.ID], ExternalID: t.ID,
		})
	}
	return out, nil
}

// transferNote подписывает перевод стрелкой и сокращённым адресом второй стороны, если она известна.
func transferNote(t chain.Transfer) string {
	if t.Counterparty == "" {
		return ""
	}
	short := t.Counterparty
	if len(short) > 12 {
		short = short[:6] + "…" + short[len(short)-4:]
	}
	if t.AmountMinor < 0 {
		return "→ " + short
	}
	return "← " + short
}
