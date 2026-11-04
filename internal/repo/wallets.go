package repo

import (
	"context"
	"fmt"

	"moneykeeper/internal/models"
	"moneykeeper/internal/pgclient"
)

type Wallets struct{ db *pgclient.Client }

func NewWallets(db *pgclient.Client) *Wallets { return &Wallets{db: db} }

const walletColumns = `w.id, w.account_id, w.chain, w.address, w.consent_at`

// Add привязывает адрес к счёту. Повторная привязка того же адреса возвращает уже сохранённую запись.
func (r *Wallets) Add(ctx context.Context, accountID int64, chain, address string) (models.WalletAddress, error) {
	var w models.WalletAddress
	err := r.db.Q(ctx).QueryRow(ctx,
		`INSERT INTO wallet_addresses (account_id, chain, address) VALUES ($1, $2, $3)
		 ON CONFLICT (account_id, chain, address) DO UPDATE SET address = EXCLUDED.address
		 RETURNING id, account_id, chain, address, consent_at`, accountID, chain, address,
	).Scan(&w.ID, &w.AccountID, &w.Chain, &w.Address, &w.ConsentAt)
	if err != nil {
		return models.WalletAddress{}, fmt.Errorf("add wallet address: %w", err)
	}
	return w, nil
}

// Get возвращает адрес, если его счёт принадлежит бюджету.
func (r *Wallets) Get(ctx context.Context, budgetID, id int64) (models.WalletAddress, error) {
	var w models.WalletAddress
	err := r.db.Q(ctx).QueryRow(ctx,
		`SELECT `+walletColumns+` FROM wallet_addresses w JOIN accounts a ON a.id = w.account_id
		 WHERE w.id = $1 AND a.budget_id = $2`, id, budgetID,
	).Scan(&w.ID, &w.AccountID, &w.Chain, &w.Address, &w.ConsentAt)
	if err != nil {
		return models.WalletAddress{}, fmt.Errorf("get wallet address: %w", mapErr(err))
	}
	return w, nil
}

// List возвращает адреса счёта в порядке привязки.
func (r *Wallets) List(ctx context.Context, budgetID, accountID int64) ([]models.WalletAddress, error) {
	rows, err := r.db.Q(ctx).Query(ctx,
		`SELECT `+walletColumns+` FROM wallet_addresses w JOIN accounts a ON a.id = w.account_id
		 WHERE a.budget_id = $1 AND w.account_id = $2 ORDER BY w.id`, budgetID, accountID)
	if err != nil {
		return nil, fmt.Errorf("list wallet addresses: %w", err)
	}
	defer rows.Close()

	var out []models.WalletAddress
	for rows.Next() {
		var w models.WalletAddress
		if err := rows.Scan(&w.ID, &w.AccountID, &w.Chain, &w.Address, &w.ConsentAt); err != nil {
			return nil, fmt.Errorf("scan wallet address: %w", err)
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

// Delete отвязывает адрес. Уже внесённые записи остаются.
func (r *Wallets) Delete(ctx context.Context, budgetID, id int64) error {
	tag, err := r.db.Q(ctx).Exec(ctx,
		`DELETE FROM wallet_addresses w USING accounts a WHERE a.id = w.account_id AND w.id = $1 AND a.budget_id = $2`, id, budgetID)
	if err != nil {
		return fmt.Errorf("delete wallet address: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("wallet address %d: %w", id, models.ErrNotFound)
	}
	return nil
}

// Imported возвращает, какие из переводов сети уже внесены в записи счёта.
func (r *Wallets) Imported(ctx context.Context, accountID int64, ids []string) (map[string]bool, error) {
	rows, err := r.db.Q(ctx).Query(ctx,
		`SELECT external_id FROM chain_imported WHERE account_id = $1 AND external_id = ANY($2)`, accountID, ids)
	if err != nil {
		return nil, fmt.Errorf("imported transfers: %w", err)
	}
	defer rows.Close()

	out := map[string]bool{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan imported transfer: %w", err)
		}
		out[id] = true
	}
	return out, rows.Err()
}

// MarkImported записывает перевод как внесённый. false: он уже был внесён раньше.
func (r *Wallets) MarkImported(ctx context.Context, accountID int64, externalID string) (bool, error) {
	tag, err := r.db.Q(ctx).Exec(ctx,
		`INSERT INTO chain_imported (account_id, external_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, accountID, externalID)
	if err != nil {
		return false, fmt.Errorf("mark imported: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}
