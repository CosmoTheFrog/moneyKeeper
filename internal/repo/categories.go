package repo

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"moneykeeper/internal/models"
	"moneykeeper/internal/pgclient"
)

const pgUniqueViolation = "23505"

type Categories struct{ db *pgclient.Client }

func NewCategories(db *pgclient.Client) *Categories { return &Categories{db: db} }

const (
	categoryColumns = `id, budget_id, name, description, hidden, rollover`
)

// SeedDefaults кладёт в новый бюджет стартовые категории. Порядок набора задаёт порядок в списке.
func (r *Categories) SeedDefaults(ctx context.Context, budgetID int64, defaults []models.DefaultCategory) error {
	names := make([]string, len(defaults))
	descriptions := make([]string, len(defaults))
	rollovers := make([]bool, len(defaults))
	for i, c := range defaults {
		names[i], descriptions[i], rollovers[i] = c.Name, c.Description, c.Rollover
	}
	_, err := r.db.Q(ctx).Exec(ctx,
		`INSERT INTO categories (budget_id, name, description, rollover)
		 SELECT $1, n, d, k FROM unnest($2::text[], $3::text[], $4::bool[]) WITH ORDINALITY AS u(n, d, k, ord)
		 ORDER BY ord`,
		budgetID, names, descriptions, rollovers)
	if err != nil {
		return fmt.Errorf("seed categories: %w", err)
	}
	return nil
}

func (r *Categories) Create(ctx context.Context, budgetID int64, in models.NewCategory) (models.Category, error) {
	c, err := scanCategory(r.db.Q(ctx).QueryRow(ctx,
		`INSERT INTO categories (budget_id, name, description, rollover)
		 VALUES ($1, $2, $3, $4) RETURNING `+categoryColumns,
		budgetID, in.Name, in.Description, in.Rollover))
	if err != nil {
		return models.Category{}, fmt.Errorf("create category: %w", mapCategoryErr(err))
	}
	return c, nil
}

func (r *Categories) Get(ctx context.Context, budgetID, id int64) (models.Category, error) {
	c, err := scanCategory(r.db.Q(ctx).QueryRow(ctx,
		`SELECT `+categoryColumns+` FROM categories WHERE budget_id = $1 AND id = $2`, budgetID, id))
	if err != nil {
		return models.Category{}, fmt.Errorf("get category: %w", mapErr(err))
	}
	return c, nil
}

func (r *Categories) List(ctx context.Context, budgetID int64) ([]models.Category, error) {
	rows, err := r.db.Q(ctx).Query(ctx,
		`SELECT `+categoryColumns+` FROM categories WHERE budget_id = $1 ORDER BY id`, budgetID)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	defer rows.Close()

	var out []models.Category
	for rows.Next() {
		c, err := scanCategory(rows)
		if err != nil {
			return nil, fmt.Errorf("scan category: %w", err)
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	return out, nil
}

func (r *Categories) Update(ctx context.Context, budgetID, id int64, p models.CategoryPatch) (models.Category, error) {
	c, err := scanCategory(r.db.Q(ctx).QueryRow(ctx,
		`UPDATE categories SET name = COALESCE($3, name), description = COALESCE($4, description), hidden = COALESCE($5, hidden),
		   rollover = COALESCE($6, rollover)
		 WHERE budget_id = $1 AND id = $2 RETURNING `+categoryColumns,
		budgetID, id, p.Name, p.Description, p.Hidden, p.Rollover))
	if err != nil {
		return models.Category{}, fmt.Errorf("update category: %w", mapCategoryErr(err))
	}
	return c, nil
}

func scanCategory(row pgx.Row) (models.Category, error) {
	var c models.Category
	if err := row.Scan(&c.ID, &c.BudgetID, &c.Name, &c.Description, &c.Hidden, &c.Rollover); err != nil {
		return models.Category{}, err
	}
	return c, nil
}

// mapCategoryErr превращает повтор имени в ErrInvalidCategory, остальное как mapErr.
func mapCategoryErr(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
		return models.ErrInvalidCategory
	}
	return mapErr(err)
}
