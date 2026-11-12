package repo

import (
	"context"
	"fmt"

	"moneykeeper/internal/pgclient"
)

type ImportRules struct{ db *pgclient.Client }

func NewImportRules(db *pgclient.Client) *ImportRules { return &ImportRules{db: db} }

// All возвращает запомненные соответствия бюджета: ключ и категория.
func (r *ImportRules) All(ctx context.Context, budgetID int64) (map[string]int64, error) {
	rows, err := r.db.Q(ctx).Query(ctx, `SELECT key, category_id FROM import_rules WHERE budget_id = $1`, budgetID)
	if err != nil {
		return nil, fmt.Errorf("list import rules: %w", err)
	}
	defer rows.Close()

	out := map[string]int64{}
	for rows.Next() {
		var (
			key string
			id  int64
		)
		if err := rows.Scan(&key, &id); err != nil {
			return nil, fmt.Errorf("scan import rule: %w", err)
		}
		out[key] = id
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list import rules: %w", err)
	}
	return out, nil
}

// Save запоминает соответствия, новое выбранное заменяет прежнее.
func (r *ImportRules) Save(ctx context.Context, budgetID int64, rules map[string]int64) error {
	for key, categoryID := range rules {
		_, err := r.db.Q(ctx).Exec(ctx,
			`INSERT INTO import_rules (budget_id, key, category_id) VALUES ($1, $2, $3)
			 ON CONFLICT (budget_id, key) DO UPDATE SET category_id = EXCLUDED.category_id, updated_at = now()`,
			budgetID, key, categoryID)
		if err != nil {
			return fmt.Errorf("save import rule: %w", err)
		}
	}
	return nil
}
