package notify

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"

	"moneykeeper/internal/models"
	"moneykeeper/pkg/apptime"
)

type fakeService struct {
	daily       []models.DailyReminder
	dailyMarked []models.DailyReminder
	alerts      []models.LimitAlert
	alertMarked []models.LimitAlert
	due         []models.Reminder
	err         error
	marked      []models.Reminder
}

func (f *fakeService) DueReminders(context.Context) ([]models.Reminder, error) { return f.due, f.err }
func (f *fakeService) MarkReminded(_ context.Context, r models.Reminder) error {
	f.marked = append(f.marked, r)
	return nil
}

type fakeSender struct {
	failFor map[int64]bool
	sent    []string
	to      []int64
}

func (f *fakeSender) Send(_ context.Context, chatID int64, text string) error {
	if f.failFor[chatID] {
		return errors.New("down")
	}
	f.to, f.sent = append(f.to, chatID), append(f.sent, text)
	return nil
}

func reminder(t *testing.T, telegramID, scheduleID int64, due string, lead int) models.Reminder {
	t.Helper()
	d, err := apptime.ParseDay(due)
	if err != nil {
		t.Fatal(err)
	}
	return models.Reminder{
		TelegramID: telegramID, DueOn: d, Lead: lead,
		Schedule: models.Schedule{ID: scheduleID, Name: "Аренда", Currency: "GEL", AmountKind: models.AmountExact, AmountMinor: 150000},
	}
}

func run(svc *fakeService, sender *fakeSender) {
	New(svc, sender, slog.New(slog.NewTextHandler(io.Discard, nil))).Tick(context.Background())
}

func TestTickSendsAndMarks(t *testing.T) {
	svc := &fakeService{due: []models.Reminder{reminder(t, 1, 10, "2026-10-12", 0), reminder(t, 1, 11, "2026-10-13", 1)}}
	sender := &fakeSender{}
	run(svc, sender)

	if len(sender.sent) != 2 || len(svc.marked) != 2 {
		t.Fatalf("sent %d marked %d, want 2 and 2", len(sender.sent), len(svc.marked))
	}
	if !strings.Contains(sender.sent[0], "Сегодня платёж «Аренда»") || !strings.Contains(sender.sent[1], "Завтра платёж") {
		t.Errorf("texts = %q", sender.sent)
	}
}

func TestTickDoesNotMarkWhatWasNotSent(t *testing.T) {
	svc := &fakeService{due: []models.Reminder{reminder(t, 1, 10, "2026-10-12", 0), reminder(t, 2, 11, "2026-10-12", 0)}}
	sender := &fakeSender{failFor: map[int64]bool{1: true}}
	run(svc, sender)

	if len(sender.sent) != 1 || len(svc.marked) != 1 || svc.marked[0].Schedule.ID != 11 {
		t.Errorf("sent %v marked %+v: the failed one must stay unmarked to be retried", sender.sent, svc.marked)
	}
}

func TestTickMarksOnceWhenSeveralMembersGotIt(t *testing.T) {
	svc := &fakeService{due: []models.Reminder{reminder(t, 1, 10, "2026-10-12", 0), reminder(t, 2, 10, "2026-10-12", 0)}}
	sender := &fakeSender{failFor: map[int64]bool{2: true}}
	run(svc, sender)

	// Один участник получил, другому не дошло: платёж помечен, повторять нечего.
	if len(sender.sent) != 1 || len(svc.marked) != 1 {
		t.Errorf("sent %d marked %d", len(sender.sent), len(svc.marked))
	}
}

func TestTickSurvivesAFailingSource(t *testing.T) {
	svc := &fakeService{err: errors.New("db down")}
	sender := &fakeSender{}
	run(svc, sender)
	if len(sender.sent) != 0 || len(svc.marked) != 0 {
		t.Error("nothing must be sent when reminders cannot be read")
	}
}

func (f *fakeService) DueDailyReminders(context.Context) ([]models.DailyReminder, error) {
	return f.daily, nil
}
func (f *fakeService) MarkDailyReminded(_ context.Context, r models.DailyReminder) error {
	f.dailyMarked = append(f.dailyMarked, r)
	return nil
}
func (f *fakeService) DueLimitAlerts(context.Context) ([]models.LimitAlert, error) {
	return f.alerts, nil
}
func (f *fakeService) MarkLimitAlerted(_ context.Context, a models.LimitAlert) error {
	f.alertMarked = append(f.alertMarked, a)
	return nil
}

func TestTickDailyReminder(t *testing.T) {
	svc := &fakeService{daily: []models.DailyReminder{{UserID: 1, TelegramID: 10}, {UserID: 2, TelegramID: 20}}}
	sender := &fakeSender{failFor: map[int64]bool{20: true}}
	run(svc, sender)

	if len(sender.sent) != 1 || !strings.Contains(sender.sent[0], "внесли траты") {
		t.Errorf("sent %q", sender.sent)
	}
	if len(svc.dailyMarked) != 1 || svc.dailyMarked[0].UserID != 1 {
		t.Errorf("marked %+v: the failed one must stay unmarked", svc.dailyMarked)
	}
}

func limitAlert(telegramID int64, threshold int) models.LimitAlert {
	return models.LimitAlert{
		TelegramID: telegramID, CategoryID: 5, CategoryName: "Продукты", Threshold: threshold,
		SpentMinor: 8500, BudgetedMinor: 10000, Currency: "GEL",
	}
}

func TestTickLimitAlertsMarkOncePerCategory(t *testing.T) {
	svc := &fakeService{alerts: []models.LimitAlert{limitAlert(10, 80), limitAlert(20, 80)}}
	sender := &fakeSender{failFor: map[int64]bool{20: true}}
	run(svc, sender)

	if len(sender.sent) != 1 || len(svc.alertMarked) != 1 {
		t.Fatalf("sent %d marked %d, want 1 and 1", len(sender.sent), len(svc.alertMarked))
	}
	if !strings.Contains(sender.sent[0], "«Продукты»") || !strings.Contains(sender.sent[0], "80%") {
		t.Errorf("text = %q", sender.sent[0])
	}
}

func TestTickLimitAlertFullText(t *testing.T) {
	svc := &fakeService{alerts: []models.LimitAlert{limitAlert(10, 100)}}
	sender := &fakeSender{}
	run(svc, sender)
	if len(sender.sent) != 1 || !strings.Contains(sender.sent[0], "исчерпан") {
		t.Errorf("sent %q", sender.sent)
	}
}
