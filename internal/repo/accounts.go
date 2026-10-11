package repo

import (
	"context"
	"fmt"

	"moneykeeper/internal/models"
	"moneykeeper/internal/pgclient"
	"moneykeeper/pkg/money"
)

type Accounts struct{ db *pgclient.Client }

func NewAccounts(db *pgclient.Client) *Accounts { return &Accounts{db: db} }

// Create делает первый счёт бюджета основным.
func (r *Accounts) Create(ctx context.Context, budgetID int64, in models.NewAccount) (models.Account, error) {
	a := models.Account{BudgetID: budgetID, Name: in.Name, Kind: in.Kind, Currency: in.Currency}
	err := r.db.Q(ctx).QueryRow(ctx,
		`INSERT INTO accounts (budget_id, name, kind, currency, is_default)
		 VALUES ($1, $2, $3, $4, NOT EXISTS (SELECT 1 FROM accounts WHERE budget_id = $1))
		 RETURNING id, is_default`,
		budgetID, in.Name, string(in.Kind), string(in.Currency),
	).Scan(&a.ID, &a.IsDefault)
	if err != nil {
		return models.Account{}, fmt.Errorf("create account: %w", err)
	}
	return a, nil
}

func (r *Accounts) List(ctx context.Context, budgetID int64) ([]models.Account, error) {
	rows, err := r.db.Q(ctx).Query(ctx,
		`SELECT id, budget_id, name, kind, currency, is_default FROM accounts WHERE budget_id = $1 ORDER BY id`, budgetID)
	if err != nil {
		return nil, fmt.Errorf("list accounts: %w", err)
	}
	defer rows.Close()

	var out []models.Account
	for rows.Next() {
		var (
			a        models.Account
			kind, cu string
		)
		if err := rows.Scan(&a.ID, &a.BudgetID, &a.Name, &kind, &cu, &a.IsDefault); err != nil {
			return nil, fmt.Errorf("scan account: %w", err)
		}
		a.Kind, a.Currency = models.AccountKind(kind), money.Currency(cu)
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list accounts: %w", err)
	}
	return out, nil
}
