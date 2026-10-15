package service

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"moneykeeper/internal/models"
	"moneykeeper/internal/repo"
	"moneykeeper/pkg/apptime"
	"moneykeeper/pkg/money"
)

const (
	// Окно ближайших платежей по умолчанию: месяц назад, чтобы видеть просроченные, и два месяца вперёд.
	occurrencesBack   = 30
	occurrencesAhead  = 60
	maxScheduleDays   = 5
	maxEveryMonths    = 24
	maxRemindBefore   = 7
	reminderLocalHour = 9
	reminderLookAhead = maxRemindBefore + 1
)

// validateSchedule приводит ввод к расписанию: проверяет поля, подставляет начальный месяц.
func (s *Service) validateSchedule(ctx context.Context, u models.User, in models.NewSchedule) (models.Schedule, error) {
	bad := func(what string) (models.Schedule, error) {
		return models.Schedule{}, fmt.Errorf("%s: %w", what, models.ErrInvalidSchedule)
	}
	name := strings.TrimSpace(in.Name)
	if name == "" || utf8.RuneCountInString(name) > maxNameRunes {
		return bad("name")
	}
	cur, err := money.Parse(in.Currency)
	if err != nil {
		return models.Schedule{}, err
	}
	if in.AmountMinor <= 0 || in.AmountMinor > maxAmountMinor {
		return bad("amount")
	}
	switch in.AmountKind {
	case models.AmountExact, models.AmountApprox:
		if in.AmountMaxMinor != nil {
			return bad("upper bound of a non-range amount")
		}
	case models.AmountRange:
		if in.AmountMaxMinor == nil || *in.AmountMaxMinor < in.AmountMinor || *in.AmountMaxMinor > maxAmountMinor {
			return bad("range")
		}
	default:
		return bad("amount kind")
	}
	if in.EveryMonths < 1 || in.EveryMonths > maxEveryMonths {
		return bad("period")
	}
	days := slices.Clone(in.Days)
	slices.Sort(days)
	days = slices.Compact(days)
	if len(days) == 0 || len(days) > maxScheduleDays || days[0] < 0 || days[len(days)-1] > 31 {
		return bad("days")
	}
	switch in.Shift {
	case models.ShiftNone, models.ShiftBack, models.ShiftForward:
	default:
		return bad("weekend shift")
	}
	if in.RemindBefore < 0 || in.RemindBefore > maxRemindBefore {
		return bad("reminder")
	}

	cat, err := s.categories.Get(ctx, u.ActiveBudgetID, in.CategoryID)
	if err != nil {
		return bad("category")
	}
	if in.AccountID != nil {
		if _, err := s.accounts.Get(ctx, u.ActiveBudgetID, *in.AccountID); err != nil {
			return bad("account")
		}
	}
	start := apptime.MonthOf(s.today(u))
	if in.StartMonth != nil {
		start = *in.StartMonth
	}
	return models.Schedule{
		BudgetID: u.ActiveBudgetID, Name: name, CategoryID: cat.ID, AccountID: in.AccountID, Currency: cur,
		AmountKind: in.AmountKind, AmountMinor: in.AmountMinor, AmountMaxMinor: in.AmountMaxMinor,
		EveryMonths: in.EveryMonths, Days: days, StartMonth: start, Shift: in.Shift, RemindBefore: in.RemindBefore,
	}, nil
}

func (s *Service) CreateSchedule(ctx context.Context, u models.User, in models.NewSchedule) (models.Schedule, error) {
	sc, err := s.validateSchedule(ctx, u, in)
	if err != nil {
		return models.Schedule{}, err
	}
	return s.schedules.Create(ctx, sc)
}

// UpdateSchedule заменяет расписание целиком. Начальный месяц остаётся прежним, если новый не задан.
func (s *Service) UpdateSchedule(ctx context.Context, u models.User, id int64, in models.NewSchedule) (models.Schedule, error) {
	old, err := s.schedules.Get(ctx, u.ActiveBudgetID, id)
	if err != nil {
		return models.Schedule{}, err
	}
	if in.StartMonth == nil {
		in.StartMonth = &old.StartMonth
	}
	sc, err := s.validateSchedule(ctx, u, in)
	if err != nil {
		return models.Schedule{}, err
	}
	sc.ID = id
	return s.schedules.Update(ctx, sc)
}

func (s *Service) DeleteSchedule(ctx context.Context, u models.User, id int64) error {
	return s.schedules.Delete(ctx, u.ActiveBudgetID, id)
}

func (s *Service) ListSchedules(ctx context.Context, u models.User) ([]models.Schedule, error) {
	return s.schedules.List(ctx, u.ActiveBudgetID)
}

// SkipOccurrence пропускает платёж на дату: он больше не просрочен и не напоминает. Дата должна быть платежом расписания.
func (s *Service) SkipOccurrence(ctx context.Context, u models.User, id int64, due apptime.Day) error {
	sc, err := s.schedules.Get(ctx, u.ActiveBudgetID, id)
	if err != nil {
		return err
	}
	if len(occurrenceDays(sc, due, due)) == 0 {
		return fmt.Errorf("%s is not a payment day: %w", due, models.ErrInvalidSchedule)
	}
	return s.schedules.AddSkip(ctx, u.ActiveBudgetID, id, due)
}

// Occurrences возвращает платежи всех расписаний за период (по умолчанию от месяца назад до двух месяцев вперёд) со
// статусом: оплачен (нашлась подходящая трата), пропущен, просрочен или впереди.
func (s *Service) Occurrences(ctx context.Context, u models.User, from, to *apptime.Day) ([]models.Occurrence, error) {
	today := s.today(u)
	start, end := addDays(today, -occurrencesBack), addDays(today, occurrencesAhead)
	if from != nil {
		start = *from
	}
	if to != nil {
		end = *to
	}
	return s.occurrencesIn(ctx, u.ActiveBudgetID, today, start, end)
}

func (s *Service) occurrencesIn(ctx context.Context, budgetID int64, today, from, to apptime.Day) ([]models.Occurrence, error) {
	schedules, err := s.schedules.List(ctx, budgetID)
	if err != nil || len(schedules) == 0 {
		return nil, err
	}
	skipRows, err := s.schedules.Skips(ctx, budgetID, from, to)
	if err != nil {
		return nil, err
	}
	skipped := map[repo.Mark]bool{}
	for _, m := range skipRows {
		skipped[repo.Mark{ScheduleID: m.ScheduleID, DueOn: m.DueOn}] = true
	}
	txs, err := s.transactions.Expenses(ctx, budgetID, addDays(from, -matchWindowDays), addDays(to, matchWindowDays))
	if err != nil {
		return nil, err
	}
	spends := make([]spend, 0, len(txs))
	for _, t := range txs {
		if t.CategoryID != nil {
			spends = append(spends, spend{ID: t.ID, Day: t.OccurredOn, Amount: -t.AmountMinor, Currency: t.Currency, CategoryID: *t.CategoryID, AccountID: t.AccountID})
		}
	}

	var out []models.Occurrence
	for _, sc := range schedules {
		dues := occurrenceDays(sc, from, to)
		paid := assignSpends(sc, dues, spends)
		for _, due := range dues {
			o := models.Occurrence{Schedule: sc, DueOn: due, Status: models.StatusUpcoming}
			switch id, isPaid := paid[due]; {
			case isPaid:
				o.Status, o.TransactionID = models.StatusPaid, &id
			case skipped[repo.Mark{ScheduleID: sc.ID, DueOn: due}]:
				o.Status = models.StatusSkipped
			case due.Before(today):
				o.Status = models.StatusOverdue
			}
			out = append(out, o)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].DueOn != out[j].DueOn {
			return out[i].DueOn.Before(out[j].DueOn)
		}
		return out[i].Schedule.Name < out[j].Schedule.Name
	})
	return out, nil
}

// DueReminders находит напоминания, которые пора отправить: за RemindBefore дней до платежа и в день платежа, если
// платёж не оплачен и не пропущен. Напоминание уходит с девяти утра по времени пользователя и только один раз.
func (s *Service) DueReminders(ctx context.Context) ([]models.Reminder, error) {
	budgets, err := s.schedules.Budgets(ctx)
	if err != nil {
		return nil, err
	}
	var out []models.Reminder
	for _, budgetID := range budgets {
		members, err := s.users.Members(ctx, budgetID)
		if err != nil {
			return nil, err
		}
		for _, m := range members {
			rs, err := s.remindersFor(ctx, budgetID, m)
			if err != nil {
				return nil, err
			}
			out = append(out, rs...)
		}
	}
	return out, nil
}

func (s *Service) remindersFor(ctx context.Context, budgetID int64, m models.User) ([]models.Reminder, error) {
	loc, err := time.LoadLocation(m.Timezone)
	if err != nil {
		loc = time.UTC
	}
	now := s.now().In(loc)
	if now.Hour() < reminderLocalHour {
		return nil, nil
	}
	today := apptime.Today(loc, s.now())
	from, to := today, addDays(today, reminderLookAhead)

	occs, err := s.occurrencesIn(ctx, budgetID, today, from, to)
	if err != nil {
		return nil, err
	}
	sentRows, err := s.schedules.Reminded(ctx, budgetID, from, to)
	if err != nil {
		return nil, err
	}
	sent := map[repo.Mark]bool{}
	for _, r := range sentRows {
		sent[r] = true
	}

	var out []models.Reminder
	for _, o := range occs {
		if o.Status != models.StatusUpcoming && o.Status != models.StatusOverdue {
			continue
		}
		lead := dayDistanceSigned(o.DueOn, today)
		// Заранее: ровно за RemindBefore дней, и ещё раз в день платежа.
		if lead != o.Schedule.RemindBefore && lead != 0 {
			continue
		}
		mark := repo.Mark{ScheduleID: o.Schedule.ID, DueOn: o.DueOn, Lead: lead}
		if sent[mark] {
			continue
		}
		out = append(out, models.Reminder{TelegramID: m.TelegramID, Schedule: o.Schedule, DueOn: o.DueOn, Lead: lead})
	}
	return out, nil
}

// MarkReminded запоминает, что напоминание отправлено, чтобы оно не уходило повторно.
func (s *Service) MarkReminded(ctx context.Context, r models.Reminder) error {
	return s.schedules.MarkReminded(ctx, repo.Mark{ScheduleID: r.Schedule.ID, DueOn: r.DueOn, Lead: r.Lead})
}

func addDays(d apptime.Day, n int) apptime.Day {
	return apptime.FromDate(d.Date().AddDate(0, 0, n))
}

// dayDistanceSigned считает, через сколько дней due после today (в прошлом отрицательное).
func dayDistanceSigned(due, today apptime.Day) int {
	return int(due.Date().Sub(today.Date()).Hours() / 24)
}
