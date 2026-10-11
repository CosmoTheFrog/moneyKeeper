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
