package repo

import (
	"context"
	"fmt"

	"moneykeeper/internal/models"
	"moneykeeper/internal/pgclient"
	"moneykeeper/pkg/apptime"
)

type Users struct{ db *pgclient.Client }

func NewUsers(db *pgclient.Client) *Users { return &Users{db: db} }

const userColumns = `id, telegram_id, first_name, timezone, COALESCE(active_budget_id, 0), reminder_hour, tour_seen`

func (r *Users) GetByTelegramID(ctx context.Context, telegramID int64) (models.User, error) {
	var u models.User
	err := r.db.Q(ctx).QueryRow(ctx,
		`SELECT `+userColumns+` FROM users WHERE telegram_id = $1`, telegramID,
	).Scan(&u.ID, &u.TelegramID, &u.FirstName, &u.Timezone, &u.ActiveBudgetID, &u.ReminderHour, &u.TourSeen)
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
		`SELECT u.id, u.telegram_id, u.first_name, u.timezone, COALESCE(u.active_budget_id, 0), u.reminder_hour, u.tour_seen
		 FROM users u JOIN budget_members m ON m.user_id = u.id WHERE m.budget_id = $1 ORDER BY u.id`, budgetID)
	if err != nil {
		return nil, fmt.Errorf("budget members: %w", err)
	}
	defer rows.Close()

	var out []models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.TelegramID, &u.FirstName, &u.Timezone, &u.ActiveBudgetID, &u.ReminderHour, &u.TourSeen); err != nil {
			return nil, fmt.Errorf("scan member: %w", err)
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

// SetReminderHour включает ежедневное напоминание на час или выключает его (nil).
func (r *Users) SetReminderHour(ctx context.Context, userID int64, hour *int) error {
	tag, err := r.db.Q(ctx).Exec(ctx, `UPDATE users SET reminder_hour = $2 WHERE id = $1`, userID, hour)
	if err != nil {
		return fmt.Errorf("set reminder hour: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("set reminder hour: %w", models.ErrNotFound)
	}
	return nil
}

// WithReminder возвращает пользователей с включённым ежедневным напоминанием.
func (r *Users) WithReminder(ctx context.Context) ([]models.User, error) {
	rows, err := r.db.Q(ctx).Query(ctx, `SELECT `+userColumns+` FROM users WHERE reminder_hour IS NOT NULL ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("users with reminder: %w", err)
	}
	defer rows.Close()

	var out []models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.TelegramID, &u.FirstName, &u.Timezone, &u.ActiveBudgetID, &u.ReminderHour, &u.TourSeen); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

// ReminderSent сообщает, ушло ли пользователю напоминание за этот день.
func (r *Users) ReminderSent(ctx context.Context, userID int64, day apptime.Day) (bool, error) {
	var sent bool
	err := r.db.Q(ctx).QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM daily_reminders WHERE user_id = $1 AND day = $2)`, userID, day.Date()).Scan(&sent)
	if err != nil {
		return false, fmt.Errorf("reminder sent: %w", err)
	}
	return sent, nil
}

func (r *Users) MarkReminderSent(ctx context.Context, userID int64, day apptime.Day) error {
	_, err := r.db.Q(ctx).Exec(ctx,
		`INSERT INTO daily_reminders (user_id, day) VALUES ($1, $2) ON CONFLICT DO NOTHING`, userID, day.Date())
	if err != nil {
		return fmt.Errorf("mark reminder sent: %w", err)
	}
	return nil
}

// SetTourSeen отмечает, что обучение показано (true) или должно показаться снова (false).
func (r *Users) SetTourSeen(ctx context.Context, userID int64, seen bool) error {
	tag, err := r.db.Q(ctx).Exec(ctx, `UPDATE users SET tour_seen = $2 WHERE id = $1`, userID, seen)
	if err != nil {
		return fmt.Errorf("set tour seen: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("set tour seen: %w", models.ErrNotFound)
	}
	return nil
}
