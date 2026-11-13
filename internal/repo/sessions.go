package repo

import (
	"context"
	"fmt"
	"time"

	"moneykeeper/internal/models"
	"moneykeeper/internal/pgclient"
)

type Sessions struct{ db *pgclient.Client }

func NewSessions(db *pgclient.Client) *Sessions { return &Sessions{db: db} }

// Create сохраняет сессию по хэшу токена. Сам токен в базе не хранится.
func (r *Sessions) Create(ctx context.Context, userID int64, hash []byte, expires time.Time) error {
	_, err := r.db.Q(ctx).Exec(ctx,
		`INSERT INTO sessions (user_id, token_hash, expires_at) VALUES ($1, $2, $3)`, userID, hash, expires)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	return nil
}

// UserByHash возвращает владельца непросроченной сессии. Нет такой или она истекла: ErrNotFound.
func (r *Sessions) UserByHash(ctx context.Context, hash []byte, now time.Time) (models.User, error) {
	var u models.User
	err := r.db.Q(ctx).QueryRow(ctx,
		`SELECT u.id, COALESCE(u.telegram_id, 0), u.first_name, u.timezone, COALESCE(u.active_budget_id, 0), u.reminder_hour, u.tour_seen
		 FROM sessions s JOIN users u ON u.id = s.user_id
		 WHERE s.token_hash = $1 AND s.expires_at > $2`, hash, now,
	).Scan(&u.ID, &u.TelegramID, &u.FirstName, &u.Timezone, &u.ActiveBudgetID, &u.ReminderHour, &u.TourSeen)
	if err != nil {
		return models.User{}, fmt.Errorf("session user: %w", mapErr(err))
	}
	return u, nil
}

// Delete завершает сессию (выход). Несуществующая сессия не ошибка.
func (r *Sessions) Delete(ctx context.Context, hash []byte) error {
	if _, err := r.db.Q(ctx).Exec(ctx, `DELETE FROM sessions WHERE token_hash = $1`, hash); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}
