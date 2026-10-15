package repo

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"moneykeeper/internal/models"
	"moneykeeper/internal/pgclient"
	"moneykeeper/pkg/apptime"
	"moneykeeper/pkg/money"
)

type Schedules struct{ db *pgclient.Client }

func NewSchedules(db *pgclient.Client) *Schedules { return &Schedules{db: db} }

const scheduleColumns = `id, budget_id, name, category_id, account_id, currency, amount_kind, amount_minor, amount_max_minor,
	every_months, days, start_month, weekend_shift, remind_before`

func (r *Schedules) Create(ctx context.Context, s models.Schedule) (models.Schedule, error) {
	out, err := scanSchedule(r.db.Q(ctx).QueryRow(ctx,
		`INSERT INTO schedules (budget_id, name, category_id, account_id, currency, amount_kind, amount_minor, amount_max_minor,
		                        every_months, days, start_month, weekend_shift, remind_before)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13) RETURNING `+scheduleColumns,
		s.BudgetID, s.Name, s.CategoryID, s.AccountID, string(s.Currency), string(s.AmountKind), s.AmountMinor, s.AmountMaxMinor,
		s.EveryMonths, smallints(s.Days), s.StartMonth.First().Date(), string(s.Shift), s.RemindBefore))
	if err != nil {
		return models.Schedule{}, fmt.Errorf("create schedule: %w", mapErr(err))
	}
	return out, nil
}

// Update заменяет все поля расписания. Чужое или несуществующее это ErrNotFound.
func (r *Schedules) Update(ctx context.Context, s models.Schedule) (models.Schedule, error) {
	out, err := scanSchedule(r.db.Q(ctx).QueryRow(ctx,
		`UPDATE schedules SET name = $3, category_id = $4, account_id = $5, currency = $6, amount_kind = $7, amount_minor = $8,
		        amount_max_minor = $9, every_months = $10, days = $11, start_month = $12, weekend_shift = $13, remind_before = $14
		 WHERE budget_id = $1 AND id = $2 RETURNING `+scheduleColumns,
		s.BudgetID, s.ID, s.Name, s.CategoryID, s.AccountID, string(s.Currency), string(s.AmountKind), s.AmountMinor, s.AmountMaxMinor,
		s.EveryMonths, smallints(s.Days), s.StartMonth.First().Date(), string(s.Shift), s.RemindBefore))
	if err != nil {
		return models.Schedule{}, fmt.Errorf("update schedule: %w", mapErr(err))
	}
	return out, nil
}

func (r *Schedules) Get(ctx context.Context, budgetID, id int64) (models.Schedule, error) {
	s, err := scanSchedule(r.db.Q(ctx).QueryRow(ctx,
		`SELECT `+scheduleColumns+` FROM schedules WHERE budget_id = $1 AND id = $2`, budgetID, id))
	if err != nil {
		return models.Schedule{}, fmt.Errorf("get schedule: %w", mapErr(err))
	}
	return s, nil
}

func (r *Schedules) List(ctx context.Context, budgetID int64) ([]models.Schedule, error) {
	rows, err := r.db.Q(ctx).Query(ctx, `SELECT `+scheduleColumns+` FROM schedules WHERE budget_id = $1 ORDER BY id`, budgetID)
	if err != nil {
		return nil, fmt.Errorf("list schedules: %w", err)
	}
	defer rows.Close()

	var out []models.Schedule
	for rows.Next() {
		s, err := scanSchedule(rows)
		if err != nil {
			return nil, fmt.Errorf("scan schedule: %w", err)
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *Schedules) Delete(ctx context.Context, budgetID, id int64) error {
	tag, err := r.db.Q(ctx).Exec(ctx, `DELETE FROM schedules WHERE budget_id = $1 AND id = $2`, budgetID, id)
	if err != nil {
		return fmt.Errorf("delete schedule: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("schedule %d: %w", id, models.ErrNotFound)
	}
	return nil
}

// Budgets возвращает бюджеты, у которых есть расписания: по ним планировщик ищет напоминания.
func (r *Schedules) Budgets(ctx context.Context) ([]int64, error) {
	rows, err := r.db.Q(ctx).Query(ctx, `SELECT DISTINCT budget_id FROM schedules ORDER BY budget_id`)
	if err != nil {
		return nil, fmt.Errorf("schedule budgets: %w", err)
	}
	defer rows.Close()
	var out []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan budget: %w", err)
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// Mark это пара «расписание и дата платежа» для пропусков и напоминаний.
type Mark struct {
	ScheduleID int64
	DueOn      apptime.Day
	Lead       int
}

// AddSkip отмечает платёж пропущенным. Чужое расписание это ErrNotFound, повтор безвреден.
func (r *Schedules) AddSkip(ctx context.Context, budgetID, id int64, due apptime.Day) error {
	tag, err := r.db.Q(ctx).Exec(ctx,
		`INSERT INTO schedule_skips (schedule_id, due_on)
		 SELECT id, $3 FROM schedules WHERE budget_id = $1 AND id = $2
		 ON CONFLICT DO NOTHING`, budgetID, id, due.Date())
	if err != nil {
		return fmt.Errorf("skip schedule: %w", err)
	}
	if tag.RowsAffected() == 0 {
		// Либо расписания нет, либо дата уже пропущена.
		if _, err := r.Get(ctx, budgetID, id); err != nil {
			return err
		}
	}
	return nil
}

// Skips возвращает пропущенные платежи бюджета за период.
func (r *Schedules) Skips(ctx context.Context, budgetID int64, from, to apptime.Day) ([]Mark, error) {
	rows, err := r.db.Q(ctx).Query(ctx,
		`SELECT k.schedule_id, k.due_on FROM schedule_skips k JOIN schedules s ON s.id = k.schedule_id
		 WHERE s.budget_id = $1 AND k.due_on BETWEEN $2 AND $3`, budgetID, from.Date(), to.Date())
	if err != nil {
		return nil, fmt.Errorf("skips: %w", err)
	}
	return scanMarks(rows, false)
}

// Reminded возвращает отправленные напоминания бюджета за период.
func (r *Schedules) Reminded(ctx context.Context, budgetID int64, from, to apptime.Day) ([]Mark, error) {
	rows, err := r.db.Q(ctx).Query(ctx,
		`SELECT m.schedule_id, m.due_on, m.lead FROM schedule_reminders m JOIN schedules s ON s.id = m.schedule_id
		 WHERE s.budget_id = $1 AND m.due_on BETWEEN $2 AND $3`, budgetID, from.Date(), to.Date())
	if err != nil {
		return nil, fmt.Errorf("reminded: %w", err)
	}
	return scanMarks(rows, true)
}

func (r *Schedules) MarkReminded(ctx context.Context, m Mark) error {
	_, err := r.db.Q(ctx).Exec(ctx,
		`INSERT INTO schedule_reminders (schedule_id, due_on, lead) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`,
		m.ScheduleID, m.DueOn.Date(), m.Lead)
	if err != nil {
		return fmt.Errorf("mark reminded: %w", err)
	}
	return nil
}

func scanMarks(rows pgx.Rows, withLead bool) ([]Mark, error) {
	defer rows.Close()
	var out []Mark
	for rows.Next() {
		var (
			m Mark
			d time.Time
		)
		var err error
		if withLead {
			err = rows.Scan(&m.ScheduleID, &d, &m.Lead)
		} else {
			err = rows.Scan(&m.ScheduleID, &d)
		}
		if err != nil {
			return nil, fmt.Errorf("scan mark: %w", err)
		}
		m.DueOn = apptime.FromDate(d)
		out = append(out, m)
	}
	return out, rows.Err()
}

func scanSchedule(row pgx.Row) (models.Schedule, error) {
	var (
		s               models.Schedule
		cu, kind, shift string
		days            []int16
		start           time.Time
	)
	if err := row.Scan(&s.ID, &s.BudgetID, &s.Name, &s.CategoryID, &s.AccountID, &cu, &kind, &s.AmountMinor, &s.AmountMaxMinor,
		&s.EveryMonths, &days, &start, &shift, &s.RemindBefore); err != nil {
		return models.Schedule{}, err
	}
	s.Currency, s.AmountKind, s.Shift = money.Currency(cu), models.AmountKind(kind), models.WeekendShift(shift)
	s.StartMonth = apptime.MonthOf(apptime.FromDate(start))
	for _, d := range days {
		s.Days = append(s.Days, int(d))
	}
	return s, nil
}

func smallints(in []int) []int16 {
	out := make([]int16, len(in))
	for i, v := range in {
		out[i] = int16(v)
	}
	return out
}
