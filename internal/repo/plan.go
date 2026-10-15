package repo

import (
	"context"
	"fmt"
	"time"

	"moneykeeper/internal/models"
	"moneykeeper/internal/pgclient"
	"moneykeeper/pkg/apptime"
	"moneykeeper/pkg/money"
)

// Plan хранит заложенное по категориям и месяцам и отдаёт суммы записей для расчёта бюджета.
type Plan struct{ db *pgclient.Client }

func NewPlan(db *pgclient.Client) *Plan { return &Plan{db: db} }

// SetBudgeted записывает заложенную на категорию сумму месяца. Чужая категория это ErrNotFound.
func (r *Plan) SetBudgeted(ctx context.Context, budgetID, categoryID int64, month apptime.Month, minor int64) error {
	tag, err := r.db.Q(ctx).Exec(ctx,
		`INSERT INTO category_months (category_id, month, budgeted_minor)
		 SELECT id, $3, $4 FROM categories WHERE id = $2 AND budget_id = $1
		 ON CONFLICT (category_id, month) DO UPDATE SET budgeted_minor = EXCLUDED.budgeted_minor`,
		budgetID, categoryID, month.First().Date(), minor)
	if err != nil {
		return fmt.Errorf("set budgeted: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("category %d: %w", categoryID, models.ErrNotFound)
	}
	return nil
}

// Budgeted возвращает всё заложенное по бюджету до месяца upTo включительно.
func (r *Plan) Budgeted(ctx context.Context, budgetID int64, upTo apptime.Month) ([]models.MonthAmount, error) {
	rows, err := r.db.Q(ctx).Query(ctx,
		`SELECT cm.category_id, cm.month, cm.budgeted_minor
		 FROM category_months cm JOIN categories c ON c.id = cm.category_id
		 WHERE c.budget_id = $1 AND cm.month <= $2`, budgetID, upTo.First().Date())
	if err != nil {
		return nil, fmt.Errorf("budgeted: %w", err)
	}
	defer rows.Close()

	var out []models.MonthAmount
	for rows.Next() {
		var (
			a models.MonthAmount
			m time.Time
		)
		if err := rows.Scan(&a.CategoryID, &m, &a.Minor); err != nil {
			return nil, fmt.Errorf("scan budgeted: %w", err)
		}
		a.Month = apptime.MonthOf(apptime.FromDate(m))
		out = append(out, a)
	}
	return out, rows.Err()
}

// Flows возвращает расходы по категориям и доходы бюджета до конца месяца upTo, сгруппированно по дню и валюте.
// В бюджет входят: расходы любых счетов; доходы и начальные остатки счетов внутри бюджета; ноги переводов и обменов
// между счётом внутри бюджета и счётом вне бюджета (с категорией это расход конверта, без неё приход в бюджет).
// Переводы внутри бюджета и между счетами вне бюджета в него не входят.
func (r *Plan) Flows(ctx context.Context, budgetID int64, upTo apptime.Month) ([]models.Flow, error) {
	rows, err := r.db.Q(ctx).Query(ctx,
		`SELECT t.category_id, t.currency, t.occurred_on, SUM(t.amount_minor)
		 FROM transactions t JOIN accounts a ON a.id = t.account_id
		 WHERE t.budget_id = $1 AND t.deleted_at IS NULL AND t.occurred_on <= $2 AND (
		       t.kind = 'expense'
		    OR (t.kind IN ('income', 'opening') AND NOT a.off_budget)
		    OR (t.kind IN ('transfer', 'exchange') AND NOT a.off_budget AND EXISTS (
		          SELECT 1 FROM transactions o JOIN accounts oa ON oa.id = o.account_id
		          WHERE o.transfer_id = t.transfer_id AND o.id <> t.id AND o.deleted_at IS NULL AND oa.off_budget))
		 )
		 GROUP BY t.category_id, t.currency, t.occurred_on`, budgetID, upTo.Last().Date())
	if err != nil {
		return nil, fmt.Errorf("flows: %w", err)
	}
	defer rows.Close()
	var out []models.Flow
	for rows.Next() {
		var (
			f  models.Flow
			cu string
			d  time.Time
		)
		if err := rows.Scan(&f.CategoryID, &cu, &d, &f.AmountMinor); err != nil {
			return nil, fmt.Errorf("scan flow: %w", err)
		}
		f.Currency, f.Day = money.Currency(cu), apptime.FromDate(d)
		out = append(out, f)
	}
	return out, rows.Err()
}
