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
	categoryColumns = `id, budget_id, group_id, name, description, hidden`
	groupColumns    = `id, budget_id, COALESCE(key, ''), name, position`
)

// SeedDefaults кладёт в новый бюджет стартовые группы и категории. Порядок групп задаёт порядок в списке.
func (r *Categories) SeedDefaults(ctx context.Context, budgetID int64, groups []models.DefaultGroup, defaults []models.DefaultCategory) error {
	keys := make([]string, len(groups))
	groupNames := make([]string, len(groups))
	positions := make([]int32, len(groups))
	for i, g := range groups {
		keys[i], groupNames[i], positions[i] = string(g.Key), g.Name, int32(i+1)
	}
	_, err := r.db.Q(ctx).Exec(ctx,
		`INSERT INTO category_groups (budget_id, key, name, position)
		 SELECT $1, k, n, p FROM unnest($2::text[], $3::text[], $4::int[]) AS u(k, n, p)`,
		budgetID, keys, groupNames, positions)
	if err != nil {
		return fmt.Errorf("seed category groups: %w", err)
	}

	groupKeys := make([]string, len(defaults))
	names := make([]string, len(defaults))
	descriptions := make([]string, len(defaults))
	for i, c := range defaults {
		groupKeys[i], names[i], descriptions[i] = string(c.Group), c.Name, c.Description
	}
	_, err = r.db.Q(ctx).Exec(ctx,
		`INSERT INTO categories (budget_id, group_id, name, description)
		 SELECT $1, g.id, u.n, u.d
		 FROM unnest($2::text[], $3::text[], $4::text[]) WITH ORDINALITY AS u(k, n, d, ord)
		 JOIN category_groups g ON g.budget_id = $1 AND g.key = u.k
		 ORDER BY u.ord`,
		budgetID, groupKeys, names, descriptions)
	if err != nil {
		return fmt.Errorf("seed categories: %w", err)
	}
	return nil
}

// CreateGroup ставит новую группу в конец списка. Повтор названия в бюджете это ErrInvalidCategory.
func (r *Categories) CreateGroup(ctx context.Context, budgetID int64, name string) (models.CategoryGroup, error) {
	g, err := scanGroup(r.db.Q(ctx).QueryRow(ctx,
		`INSERT INTO category_groups (budget_id, name, position)
		 VALUES ($1, $2, COALESCE((SELECT MAX(position) FROM category_groups WHERE budget_id = $1), 0) + 1)
		 RETURNING `+groupColumns,
		budgetID, name))
	if err != nil {
		return models.CategoryGroup{}, fmt.Errorf("create category group: %w", mapCategoryErr(err))
	}
	return g, nil
}

func (r *Categories) GetGroup(ctx context.Context, budgetID, id int64) (models.CategoryGroup, error) {
	g, err := scanGroup(r.db.Q(ctx).QueryRow(ctx,
		`SELECT `+groupColumns+` FROM category_groups WHERE budget_id = $1 AND id = $2`, budgetID, id))
	if err != nil {
		return models.CategoryGroup{}, fmt.Errorf("get category group: %w", mapErr(err))
	}
	return g, nil
}

func (r *Categories) ListGroups(ctx context.Context, budgetID int64) ([]models.CategoryGroup, error) {
	rows, err := r.db.Q(ctx).Query(ctx,
		`SELECT `+groupColumns+` FROM category_groups WHERE budget_id = $1 ORDER BY position, id`, budgetID)
	if err != nil {
		return nil, fmt.Errorf("list category groups: %w", err)
	}
	defer rows.Close()

	var out []models.CategoryGroup
	for rows.Next() {
		g, err := scanGroup(rows)
		if err != nil {
			return nil, fmt.Errorf("scan category group: %w", err)
		}
		out = append(out, g)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list category groups: %w", err)
	}
	return out, nil
}

func (r *Categories) Create(ctx context.Context, budgetID int64, in models.NewCategory) (models.Category, error) {
	c, err := scanCategory(r.db.Q(ctx).QueryRow(ctx,
		`INSERT INTO categories (budget_id, group_id, name, description) VALUES ($1, $2, $3, $4) RETURNING `+categoryColumns,
		budgetID, in.GroupID, in.Name, in.Description))
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
	var c models.Category
	if err := row.Scan(&c.ID, &c.BudgetID, &c.GroupID, &c.Name, &c.Description, &c.Hidden); err != nil {
		return models.Category{}, err
	}
	return c, nil
}

func scanGroup(row pgx.Row) (models.CategoryGroup, error) {
	var (
		g   models.CategoryGroup
		key string
	)
	if err := row.Scan(&g.ID, &g.BudgetID, &key, &g.Name, &g.Position); err != nil {
		return models.CategoryGroup{}, err
	}
	g.Key = models.GroupKey(key)
	return g, nil
}

// mapCategoryErr превращает повтор имени в ErrInvalidCategory, остальное как mapErr.
func mapCategoryErr(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
		return models.ErrInvalidCategory
	}
	return mapErr(err)
}
