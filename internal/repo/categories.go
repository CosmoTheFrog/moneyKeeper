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

const categoryColumns = `id, budget_id, group_key, name, description, hidden`

func (r *Categories) SeedDefaults(ctx context.Context, budgetID int64, defaults []models.NewCategory) error {
	groups := make([]string, len(defaults))
	names := make([]string, len(defaults))
	descriptions := make([]string, len(defaults))
	for i, c := range defaults {
		groups[i], names[i], descriptions[i] = string(c.Group), c.Name, c.Description
	}
	_, err := r.db.Q(ctx).Exec(ctx,
		`INSERT INTO categories (budget_id, group_key, name, description)
		 SELECT $1, g, n, d FROM unnest($2::text[], $3::text[], $4::text[]) AS u(g, n, d)`,
		budgetID, groups, names, descriptions)
	if err != nil {
		return fmt.Errorf("seed categories: %w", err)
	}
	return nil
}

func (r *Categories) Create(ctx context.Context, budgetID int64, in models.NewCategory) (models.Category, error) {
	c, err := scanCategory(r.db.Q(ctx).QueryRow(ctx,
		`INSERT INTO categories (budget_id, group_key, name, description) VALUES ($1, $2, $3, $4) RETURNING `+categoryColumns,
		budgetID, string(in.Group), in.Name, in.Description))
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
		`UPDATE categories SET name = COALESCE($3, name), description = COALESCE($4, description), hidden = COALESCE($5, hidden)
		 WHERE budget_id = $1 AND id = $2 RETURNING `+categoryColumns,
		budgetID, id, p.Name, p.Description, p.Hidden))
	if err != nil {
		return models.Category{}, fmt.Errorf("update category: %w", mapCategoryErr(err))
	}
	return c, nil
}

func scanCategory(row pgx.Row) (models.Category, error) {
	var (
		c     models.Category
		group string
	)
	if err := row.Scan(&c.ID, &c.BudgetID, &group, &c.Name, &c.Description, &c.Hidden); err != nil {
		return models.Category{}, err
	}
	c.Group = models.CategoryGroup(group)
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
