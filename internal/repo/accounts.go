package repo

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

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

// balanceSelect считает остаток из транзакций, в таблице счетов он не хранится.
const balanceSelect = `
	SELECT a.id, a.budget_id, a.name, a.kind, a.currency, a.is_default, COALESCE(SUM(t.amount_minor), 0)
	FROM accounts a
	LEFT JOIN transactions t ON t.account_id = a.id AND t.deleted_at IS NULL
	WHERE a.budget_id = $1`

func (r *Accounts) Get(ctx context.Context, budgetID, id int64) (models.Account, error) {
	a, err := scanAccount(r.db.Q(ctx).QueryRow(ctx, balanceSelect+` AND a.id = $2 GROUP BY a.id`, budgetID, id))
	if err != nil {
		return models.Account{}, fmt.Errorf("get account: %w", mapErr(err))
	}
	return a, nil
}

func (r *Accounts) List(ctx context.Context, budgetID int64) ([]models.Account, error) {
	rows, err := r.db.Q(ctx).Query(ctx, balanceSelect+` GROUP BY a.id ORDER BY a.id`, budgetID)
	if err != nil {
		return nil, fmt.Errorf("list accounts: %w", err)
	}
	defer rows.Close()

	var out []models.Account
	for rows.Next() {
		a, err := scanAccount(rows)
		if err != nil {
			return nil, fmt.Errorf("scan account: %w", err)
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list accounts: %w", err)
	}
	return out, nil
}

func scanAccount(row pgx.Row) (models.Account, error) {
	var (
		a        models.Account
		kind, cu string
	)
	if err := row.Scan(&a.ID, &a.BudgetID, &a.Name, &kind, &cu, &a.IsDefault, &a.BalanceMinor); err != nil {
		return models.Account{}, err
	}
	a.Kind, a.Currency = models.AccountKind(kind), money.Currency(cu)
	return a, nil
}
