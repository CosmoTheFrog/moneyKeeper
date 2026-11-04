package service

import (
	"context"
	"time"

	"moneykeeper/internal/models"
	"moneykeeper/internal/repo"
	"moneykeeper/pkg/apptime"
)

// alertWindow: предупреждение о лимите ищется только в бюджетах, где расход вносили за последние сутки. Отправленные
// пороги хранятся, поэтому повторный проход ничего не пошлёт второй раз.
const alertWindow = 24 * time.Hour

// SetReminderHour включает ежедневное напоминание внести траты на час (0-23 по времени пользователя) или выключает (nil).
func (s *Service) SetReminderHour(ctx context.Context, u models.User, hour *int) error {
	return s.users.SetReminderHour(ctx, u.ID, hour)
}

// DueDailyReminders находит тех, кому пора напомнить внести траты: наступил выбранный час, напоминание за сегодня ещё
// не уходило и расхода за сегодня пользователь не вносил.
func (s *Service) DueDailyReminders(ctx context.Context) ([]models.DailyReminder, error) {
	users, err := s.users.WithReminder(ctx)
	if err != nil {
		return nil, err
	}
	var out []models.DailyReminder
	for _, u := range users {
		if u.ReminderHour == nil || u.ActiveBudgetID == 0 {
			continue
		}
		loc, err := time.LoadLocation(u.Timezone)
		if err != nil {
			loc = time.UTC
		}
		if s.now().In(loc).Hour() < *u.ReminderHour {
			continue
		}
		today := apptime.Today(loc, s.now())
		sent, err := s.users.ReminderSent(ctx, u.ID, today)
		if err != nil {
			return nil, err
		}
		if sent {
			continue
		}
		entered, err := s.transactions.HasExpenseBy(ctx, u.ActiveBudgetID, u.ID, today)
		if err != nil {
			return nil, err
		}
		if !entered {
			out = append(out, models.DailyReminder{UserID: u.ID, TelegramID: u.TelegramID, Day: today})
		}
	}
	return out, nil
}

func (s *Service) MarkDailyReminded(ctx context.Context, r models.DailyReminder) error {
	return s.users.MarkReminderSent(ctx, r.UserID, r.Day)
}

// limitLevel возвращает достигнутый порог лимита: 100, если потрачено всё заложенное, 80, если не меньше 80%, иначе 0.
func limitLevel(spent, budgeted int64) int {
	switch {
	case budgeted <= 0:
		return 0
	case spent >= budgeted:
		return models.LimitFull
	case spent*100 >= budgeted*models.LimitWarn:
		return models.LimitWarn
	}
	return 0
}

// DueLimitAlerts находит категории, где расход текущего месяца дошёл до 80% или 100% заложенного и об этом ещё не
// предупреждали. Предупреждение получает каждый участник бюджета.
func (s *Service) DueLimitAlerts(ctx context.Context) ([]models.LimitAlert, error) {
	budgets, err := s.transactions.BudgetsWithExpensesSince(ctx, s.now().Add(-alertWindow))
	if err != nil {
		return nil, err
	}
	var out []models.LimitAlert
	for _, budgetID := range budgets {
		// Лимиты есть только в режиме конвертов; при выключении заложенные суммы остаются, но молчат.
		budget, err := s.budgets.Get(ctx, budgetID)
		if err != nil {
			return nil, err
		}
		if !budget.EnvelopesEnabled {
			continue
		}
		members, err := s.users.Members(ctx, budgetID)
		if err != nil {
			return nil, err
		}
		if len(members) == 0 {
			continue
		}
		// Бюджет считается глазами первого участника: пояс задаёт, какой месяц текущий.
		viewer := members[0]
		viewer.ActiveBudgetID = budgetID
		month := apptime.MonthOf(s.today(viewer))
		bm, err := s.BudgetMonth(ctx, viewer, month)
		if err != nil {
			return nil, err
		}
		cats, err := s.categories.List(ctx, budgetID)
		if err != nil {
			return nil, err
		}
		names := make(map[int64]string, len(cats))
		hidden := map[int64]bool{}
		for _, c := range cats {
			names[c.ID], hidden[c.ID] = c.Name, c.Hidden
		}
		sent, err := s.plan.LimitAlerted(ctx, budgetID, month)
		if err != nil {
			return nil, err
		}
		for _, row := range bm.Rows {
			level := limitLevel(row.SpentMinor, row.BudgetedMinor)
			if level == 0 || hidden[row.CategoryID] || sent[repo.AlertKey{CategoryID: row.CategoryID, Threshold: level}] {
				continue
			}
			for _, m := range members {
				out = append(out, models.LimitAlert{
					TelegramID: m.TelegramID, CategoryID: row.CategoryID, CategoryName: names[row.CategoryID], Month: month,
					Threshold: level, SpentMinor: row.SpentMinor, BudgetedMinor: row.BudgetedMinor, Currency: bm.Currency,
				})
			}
		}
	}
	return out, nil
}

func (s *Service) MarkLimitAlerted(ctx context.Context, a models.LimitAlert) error {
	return s.plan.MarkLimitAlerted(ctx, a.CategoryID, a.Month, a.Threshold)
}

// SetTourSeen отмечает, что обучение показано, или возвращает его на следующий вход (false).
func (s *Service) SetTourSeen(ctx context.Context, u models.User, seen bool) error {
	return s.users.SetTourSeen(ctx, u.ID, seen)
}
