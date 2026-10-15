// Package notify раз в несколько минут находит платежи, о которых пора напомнить, и отправляет напоминания в Telegram.
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

// Tick отправляет все созревшие напоминания. Платёж помечается отправленным, когда сообщение ушло хотя бы одному
// участнику; сумму и название платежа в журнал не пишем.
func (n *Notifier) Tick(ctx context.Context) {
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
