// Package notify раз в несколько минут находит, о чём пора напомнить, и отправляет сообщения в Telegram: платежи по
// расписанию, ежедневное «внесли траты?» и предупреждения о лимите категории.
package notify

import (
	"context"
	"log/slog"
	"time"

	"moneykeeper/internal/i18n"
	"moneykeeper/internal/models"
)

const interval = 5 * time.Minute

type Service interface {
	DueReminders(ctx context.Context) ([]models.Reminder, error)
	MarkReminded(ctx context.Context, r models.Reminder) error
	DueDailyReminders(ctx context.Context) ([]models.DailyReminder, error)
	MarkDailyReminded(ctx context.Context, r models.DailyReminder) error
	DueLimitAlerts(ctx context.Context) ([]models.LimitAlert, error)
	MarkLimitAlerted(ctx context.Context, a models.LimitAlert) error
}

type Sender interface {
	Send(ctx context.Context, chatID int64, text string) error
}

type Notifier struct {
	svc    Service
	sender Sender
	log    *slog.Logger
}

func New(svc Service, sender Sender, log *slog.Logger) *Notifier {
	return &Notifier{svc: svc, sender: sender, log: log}
}

// Run проверяет напоминания сразу и затем каждые пять минут, пока не закрыт ctx.
func (n *Notifier) Run(ctx context.Context) {
	n.Tick(ctx)
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			n.Tick(ctx)
		}
	}
}

// Tick отправляет все созревшие сообщения. Запись помечается отправленной, когда сообщение ушло хотя бы одному
// участнику; суммы и названия в журнал не пишем.
func (n *Notifier) Tick(ctx context.Context) {
	n.payments(ctx)
	n.daily(ctx)
	n.limits(ctx)
}

func (n *Notifier) payments(ctx context.Context) {
	reminders, err := n.svc.DueReminders(ctx)
	if err != nil {
		n.log.Error("find reminders", "err", err)
		return
	}
	type key struct {
		schedule int64
		due      string
		lead     int
	}
	delivered := map[key]models.Reminder{}
	for _, r := range reminders {
		if err := n.sender.Send(ctx, r.TelegramID, i18n.ReminderText(r)); err != nil {
			n.log.Warn("send reminder", "schedule", r.Schedule.ID, "err", err)
			continue
		}
		delivered[key{r.Schedule.ID, r.DueOn.String(), r.Lead}] = r
	}
	for _, r := range delivered {
		if err := n.svc.MarkReminded(ctx, r); err != nil {
			n.log.Error("mark reminded", "schedule", r.Schedule.ID, "err", err)
		}
	}
}

func (n *Notifier) daily(ctx context.Context) {
	reminders, err := n.svc.DueDailyReminders(ctx)
	if err != nil {
		n.log.Error("find daily reminders", "err", err)
		return
	}
	for _, r := range reminders {
		if err := n.sender.Send(ctx, r.TelegramID, i18n.DailyReminderText()); err != nil {
			n.log.Warn("send daily reminder", "user", r.UserID, "err", err)
			continue
		}
		if err := n.svc.MarkDailyReminded(ctx, r); err != nil {
			n.log.Error("mark daily reminded", "user", r.UserID, "err", err)
		}
	}
}

func (n *Notifier) limits(ctx context.Context) {
	alerts, err := n.svc.DueLimitAlerts(ctx)
	if err != nil {
		n.log.Error("find limit alerts", "err", err)
		return
	}
	type key struct {
		category  int64
		month     string
		threshold int
	}
	delivered := map[key]models.LimitAlert{}
	for _, a := range alerts {
		if err := n.sender.Send(ctx, a.TelegramID, i18n.LimitAlertText(a)); err != nil {
			n.log.Warn("send limit alert", "category", a.CategoryID, "err", err)
			continue
		}
		delivered[key{a.CategoryID, a.Month.String(), a.Threshold}] = a
	}
	for _, a := range delivered {
		if err := n.svc.MarkLimitAlerted(ctx, a); err != nil {
			n.log.Error("mark limit alerted", "category", a.CategoryID, "err", err)
		}
	}
}
