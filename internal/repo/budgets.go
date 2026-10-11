package repo

import (
	"context"
	"fmt"

	"moneykeeper/internal/models"
	"moneykeeper/internal/pgclient"
	"moneykeeper/pkg/money"
)

type Budgets struct{ db *pgclient.Client }

func NewBudgets(db *pgclient.Client) *Budgets { return &Budgets{db: db} }

func (r *Budgets) Create(ctx context.Context, name string) (models.Budget, error) {
	b := models.Budget{Name: name}
	err := r.db.Q(ctx).QueryRow(ctx, `INSERT INTO budgets (name) VALUES ($1) RETURNING id`, name).Scan(&b.ID)
	if err != nil {
		return models.Budget{}, fmt.Errorf("create budget: %w", err)
	}
	return b, nil
}

func (r *Budgets) AddMember(ctx context.Context, budgetID, userID int64) error {
	_, err := r.db.Q(ctx).Exec(ctx, `INSERT INTO budget_members (budget_id, user_id) VALUES ($1, $2)`, budgetID, userID)
	if err != nil {
		return fmt.Errorf("add budget member: %w", err)
	}
	return nil
}

func (r *Budgets) Get(ctx context.Context, id int64) (models.Budget, error) {
	var (
		b    models.Budget
		code *string
	)
	err := r.db.Q(ctx).QueryRow(ctx, `SELECT id, name, base_currency FROM budgets WHERE id = $1`, id).
		Scan(&b.ID, &b.Name, &code)
	if err != nil {
		return models.Budget{}, fmt.Errorf("get budget: %w", mapErr(err))
	}
	if code != nil {
		c := money.Currency(*code)
		b.BaseCurrency = &c
	}
	return b, nil
}

// SetBaseCurrency задаёт базовую валюту один раз. Если она уже стоит, возвращает ErrBaseCurrencySet.
func (r *Budgets) SetBaseCurrency(ctx context.Context, id int64, currency money.Currency) error {
	tag, err := r.db.Q(ctx).Exec(ctx,
		`UPDATE budgets SET base_currency = $2 WHERE id = $1 AND base_currency IS NULL`, id, string(currency))
	if err != nil {
		return fmt.Errorf("set base currency: %w", err)
	}
	if tag.RowsAffected() == 0 {
		if _, err := r.Get(ctx, id); err != nil {
			return err
		}
		return fmt.Errorf("budget %d: %w", id, models.ErrBaseCurrencySet)
	}
	return nil
}
