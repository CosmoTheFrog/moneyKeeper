package repo

import (
	"context"
	"fmt"

	"moneykeeper/internal/models"
	"moneykeeper/internal/pgclient"
)

type Users struct{ db *pgclient.Client }

func NewUsers(db *pgclient.Client) *Users { return &Users{db: db} }

const userColumns = `id, telegram_id, first_name, timezone, COALESCE(active_budget_id, 0)`

func (r *Users) GetByTelegramID(ctx context.Context, telegramID int64) (models.User, error) {
	var u models.User
	err := r.db.Q(ctx).QueryRow(ctx,
		`SELECT `+userColumns+` FROM users WHERE telegram_id = $1`, telegramID,
	).Scan(&u.ID, &u.TelegramID, &u.FirstName, &u.Timezone, &u.ActiveBudgetID)
	if err != nil {
		return models.User{}, fmt.Errorf("get user by telegram id: %w", mapErr(err))
	}
	return u, nil
}

// CreateIfAbsent возвращает ErrNotFound, если пользователь с таким telegram_id уже есть
// (его создал параллельный запрос).
func (r *Users) CreateIfAbsent(ctx context.Context, telegramID int64, firstName, timezone string) (models.User, error) {
	u := models.User{TelegramID: telegramID, FirstName: firstName, Timezone: timezone}
	err := r.db.Q(ctx).QueryRow(ctx,
		`INSERT INTO users (telegram_id, first_name, timezone) VALUES ($1, $2, $3)
		 ON CONFLICT (telegram_id) DO NOTHING RETURNING id`,
		telegramID, firstName, timezone,
	).Scan(&u.ID)
	if err != nil {
		return models.User{}, fmt.Errorf("create user: %w", mapErr(err))
	}
	return u, nil
}

func (r *Users) SetActiveBudget(ctx context.Context, userID, budgetID int64) error {
	tag, err := r.db.Q(ctx).Exec(ctx, `UPDATE users SET active_budget_id = $2 WHERE id = $1`, userID, budgetID)
	if err != nil {
		return fmt.Errorf("set active budget: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("set active budget: %w", models.ErrNotFound)
	}
	return nil
}

// Members возвращает участников бюджета: им уходят напоминания.
func (r *Users) Members(ctx context.Context, budgetID int64) ([]models.User, error) {
	rows, err := r.db.Q(ctx).Query(ctx,
		`SELECT u.id, u.telegram_id, u.first_name, u.timezone, COALESCE(u.active_budget_id, 0)
		 FROM users u JOIN budget_members m ON m.user_id = u.id WHERE m.budget_id = $1 ORDER BY u.id`, budgetID)
	if err != nil {
		return nil, fmt.Errorf("budget members: %w", err)
	}
	defer rows.Close()

	var out []models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.TelegramID, &u.FirstName, &u.Timezone, &u.ActiveBudgetID); err != nil {
			return nil, fmt.Errorf("scan member: %w", err)
		}
		out = append(out, u)
	}
	return out, rows.Err()
}
