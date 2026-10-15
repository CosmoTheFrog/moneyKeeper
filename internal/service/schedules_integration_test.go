//go:build integration

package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"moneykeeper/internal/models"
	"moneykeeper/pkg/apptime"
)

func (l ledger) rent(t *testing.T) models.Schedule {
	t.Helper()
	cat := l.category(t, "Аренда и ипотека")
	s, err := l.s.CreateSchedule(context.Background(), l.u, models.NewSchedule{
		Name: "Аренда", CategoryID: cat, AccountID: &l.main.ID, Currency: "GEL", AmountKind: models.AmountExact, AmountMinor: 50000,
		EveryMonths: 1, Days: []int{15}, Shift: models.ShiftForward, RemindBefore: 1, StartMonth: monthPtr(t, "2026-09"),
	})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func monthPtr(t *testing.T, s string) *apptime.Month {
	t.Helper()
	m := month(t, s)
	return &m
}

func (l ledger) occurrences(t *testing.T) map[string]models.Occurrence {
	t.Helper()
	list, err := l.s.Occurrences(context.Background(), l.u, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]models.Occurrence{}
	for _, o := range list {
		out[o.DueOn.String()+"/"+o.Schedule.Name] = o
	}
	return out
}

func (l ledger) spendOn(t *testing.T, on string, amount int64, category int64) models.Transaction {
	t.Helper()
	rows, err := l.s.CreateTransaction(context.Background(), l.u, models.NewTransaction{
		Kind: models.KindExpense, AccountID: l.main.ID, CategoryID: &category, AmountMinor: amount, OccurredOn: day(t, on),
	})
	if err != nil {
		t.Fatal(err)
	}
	return rows[0]
}

func TestScheduleOccurrencesAndStatuses(t *testing.T) {
	ctx := context.Background()
	l := newLedger(t)
	rent := l.rent(t)
	cat := rent.CategoryID

	got := l.occurrences(t)
	// Сегодня 12 октября; 15 сентября просрочено, 15 октября впереди, 15 ноября воскресенье и переносится на понедельник; декабрьский платёж за пределами окна в 60 дней.
	want := map[string]models.OccurrenceStatus{
		"2026-09-15/Аренда": models.StatusOverdue,
		"2026-10-15/Аренда": models.StatusUpcoming,
		"2026-11-16/Аренда": models.StatusUpcoming,
	}
	if len(got) != len(want) {
		t.Fatalf("occurrences = %v", got)
	}
	for k, st := range want {
		if got[k].Status != st {
			t.Errorf("%s = %s, want %s", k, got[k].Status, st)
		}
	}

	// Трата на день раньше подходящей суммы закрывает сентябрьский платёж; трата другой суммы нет.
	paid := l.spendOn(t, "2026-09-14", 50000, cat)
	l.spendOn(t, "2026-10-15", 49999, cat)
	got = l.occurrences(t)
	if o := got["2026-09-15/Аренда"]; o.Status != models.StatusPaid || o.TransactionID == nil || *o.TransactionID != paid.ID {
		t.Errorf("september = %+v", o)
	}
	if got["2026-10-15/Аренда"].Status != models.StatusUpcoming {
		t.Errorf("october with a wrong amount = %s, want upcoming", got["2026-10-15/Аренда"].Status)
	}

	// Пропуск снимает просрочку; пропустить можно только дату платежа.
	if err := l.s.SkipOccurrence(ctx, l.u, rent.ID, d(t, "2026-10-15")); err != nil {
		t.Fatal(err)
	}
	if err := l.s.SkipOccurrence(ctx, l.u, rent.ID, d(t, "2026-10-15")); err != nil {
		t.Errorf("skipping twice: %v", err)
	}
	if err := l.s.SkipOccurrence(ctx, l.u, rent.ID, d(t, "2026-10-16")); !errors.Is(err, models.ErrInvalidSchedule) {
		t.Errorf("skipping a day without a payment: err = %v, want ErrInvalidSchedule", err)
	}
	if err := l.s.SkipOccurrence(ctx, l.u, 999999, d(t, "2026-10-15")); !errors.Is(err, models.ErrNotFound) {
		t.Errorf("skipping an unknown schedule: err = %v, want ErrNotFound", err)
	}
	if got := l.occurrences(t)["2026-10-15/Аренда"].Status; got != models.StatusSkipped {
		t.Errorf("october after skip = %s, want skipped", got)
	}

	// Окно задаётся параметрами.
	from, to := day(t, "2026-11-01"), day(t, "2026-11-30")
	list, err := l.s.Occurrences(ctx, l.u, from, to)
	if err != nil || len(list) != 1 || list[0].DueOn.String() != "2026-11-16" {
		t.Errorf("november window = %+v, %v", list, err)
	}
}

func TestScheduleCrudAndValidation(t *testing.T) {
	ctx := context.Background()
	l := newLedger(t)
	rent := l.rent(t)
	cat := l.category(t, "Аренда и ипотека")
	maxAmount := int64(80000)
	good := models.NewSchedule{
		Name: "Интернет", CategoryID: cat, Currency: "GEL", AmountKind: models.AmountRange, AmountMinor: 3000, AmountMaxMinor: &maxAmount,
		EveryMonths: 1, Days: []int{5, 20, 5}, Shift: models.ShiftNone,
	}

	created, err := l.s.CreateSchedule(ctx, l.u, good)
	if err != nil {
		t.Fatal(err)
	}
	if len(created.Days) != 2 || created.Days[0] != 5 || created.Days[1] != 20 || created.StartMonth.String() != "2026-10" {
		t.Errorf("created = %+v (days sorted without duplicates, start month is the current one)", created)
	}

	// Правка заменяет поля и сохраняет начальный месяц.
	good.Name, good.AmountKind, good.AmountMaxMinor = "Интернет и связь", models.AmountApprox, nil
	updated, err := l.s.UpdateSchedule(ctx, l.u, created.ID, good)
	if err != nil || updated.Name != "Интернет и связь" || updated.AmountKind != models.AmountApprox || updated.StartMonth != created.StartMonth {
		t.Fatalf("updated = %+v, %v", updated, err)
	}

	list, _ := l.s.ListSchedules(ctx, l.u)
	if len(list) != 2 {
		t.Errorf("%d schedules, want 2", len(list))
	}
	if err := l.s.DeleteSchedule(ctx, l.u, created.ID); err != nil {
		t.Fatal(err)
	}
	if err := l.s.DeleteSchedule(ctx, l.u, created.ID); !errors.Is(err, models.ErrNotFound) {
		t.Errorf("second delete: err = %v, want ErrNotFound", err)
	}

	base := models.NewSchedule{Name: "X", CategoryID: cat, Currency: "GEL", AmountKind: models.AmountExact, AmountMinor: 100, EveryMonths: 1, Days: []int{1}, Shift: models.ShiftNone}
	with := func(f func(*models.NewSchedule)) models.NewSchedule { s := base; f(&s); return s }
	lo := int64(50)
	hi := int64(maxAmountMinor + 1)
	tooMany := []int{1, 2, 3, 4, 5, 6}
	bad := map[string]models.NewSchedule{
		"empty name":              with(func(s *models.NewSchedule) { s.Name = "  " }),
		"zero amount":             with(func(s *models.NewSchedule) { s.AmountMinor = 0 }),
		"amount too large":        with(func(s *models.NewSchedule) { s.AmountMinor = maxAmountMinor + 1 }),
		"unknown kind":            with(func(s *models.NewSchedule) { s.AmountKind = "weird" }),
		"exact with upper bound":  with(func(s *models.NewSchedule) { s.AmountMaxMinor = &maxAmount }),
		"range without bound":     with(func(s *models.NewSchedule) { s.AmountKind = models.AmountRange }),
		"range upper below lower": with(func(s *models.NewSchedule) { s.AmountKind, s.AmountMaxMinor = models.AmountRange, &lo }),
		"range upper too large":   with(func(s *models.NewSchedule) { s.AmountKind, s.AmountMaxMinor = models.AmountRange, &hi }),
		"period zero":             with(func(s *models.NewSchedule) { s.EveryMonths = 0 }),
		"period too long":         with(func(s *models.NewSchedule) { s.EveryMonths = 25 }),
		"no days":                 with(func(s *models.NewSchedule) { s.Days = nil }),
		"day 32":                  with(func(s *models.NewSchedule) { s.Days = []int{32} }),
		"negative day":            with(func(s *models.NewSchedule) { s.Days = []int{-1} }),
		"too many days":           with(func(s *models.NewSchedule) { s.Days = tooMany }),
		"unknown shift":           with(func(s *models.NewSchedule) { s.Shift = "sideways" }),
		"reminder too early":      with(func(s *models.NewSchedule) { s.RemindBefore = 8 }),
		"negative reminder":       with(func(s *models.NewSchedule) { s.RemindBefore = -1 }),
		"unknown category":        with(func(s *models.NewSchedule) { s.CategoryID = 999999 }),
		"unknown account":         with(func(s *models.NewSchedule) { a := int64(999999); s.AccountID = &a }),
	}
	for name, in := range bad {
		if _, err := l.s.CreateSchedule(ctx, l.u, in); !errors.Is(err, models.ErrInvalidSchedule) {
			t.Errorf("%s: err = %v, want ErrInvalidSchedule", name, err)
		}
	}
	if _, err := l.s.CreateSchedule(ctx, l.u, with(func(s *models.NewSchedule) { s.Currency = "XXX" })); err == nil {
		t.Error("unknown currency must fail")
	}
	// Чужое расписание недоступно.
	other, err := l.s.Resolve(ctx, models.TelegramUser{ID: 7, FirstName: "Anna"}, "Asia/Tbilisi")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := l.s.UpdateSchedule(ctx, other, rent.ID, base); !errors.Is(err, models.ErrNotFound) {
		t.Errorf("foreign update: err = %v, want ErrNotFound", err)
	}
	if err := l.s.SkipOccurrence(ctx, other, rent.ID, d(t, "2026-10-15")); !errors.Is(err, models.ErrNotFound) {
		t.Errorf("foreign skip: err = %v, want ErrNotFound", err)
	}
}

func TestScheduleReminders(t *testing.T) {
	ctx := context.Background()
	l := newLedger(t)
	rent := l.rent(t)

	at := func(s string) {
		t.Helper()
		now, err := time.Parse(time.RFC3339, s)
		if err != nil {
			t.Fatal(err)
		}
		l.s.now = func() time.Time { return now }
	}
	due := func() []models.Reminder {
		t.Helper()
		r, err := l.s.DueReminders(ctx)
		if err != nil {
			t.Fatal(err)
		}
		return r
	}

	// 1:30 ночи по Тбилиси 12 октября: рано, и до платежа далеко.
	if got := due(); len(got) != 0 {
		t.Errorf("night reminders = %+v", got)
	}
	// 14 октября в 08:59 по Тбилиси (04:59 UTC) ещё рано, в 09:00 (05:00 UTC) приходит напоминание за день до платежа.
	at("2026-10-14T04:59:00Z")
	if got := due(); len(got) != 0 {
		t.Errorf("before nine = %+v", got)
	}
	at("2026-10-14T05:00:00Z")
	got := due()
	if len(got) != 1 || got[0].Lead != 1 || got[0].DueOn.String() != "2026-10-15" || got[0].TelegramID != 42 || got[0].Schedule.ID != rent.ID {
		t.Fatalf("a day before = %+v", got)
	}
	if err := l.s.MarkReminded(ctx, got[0]); err != nil {
		t.Fatal(err)
	}
	if got := due(); len(got) != 0 {
		t.Errorf("after sending = %+v, the reminder must not repeat", got)
	}

	// В день платежа напоминание приходит ещё раз.
	at("2026-10-15T06:00:00Z")
	got = due()
	if len(got) != 1 || got[0].Lead != 0 {
		t.Fatalf("on the day = %+v", got)
	}

	// Оплаченный платёж не напоминает.
	l.spendOn(t, "2026-10-15", 50000, rent.CategoryID)
	if got := due(); len(got) != 0 {
		t.Errorf("paid payment reminds: %+v", got)
	}
}

func TestSkippedPaymentDoesNotRemind(t *testing.T) {
	ctx := context.Background()
	l := newLedger(t)
	rent := l.rent(t)
	if err := l.s.SkipOccurrence(ctx, l.u, rent.ID, d(t, "2026-10-15")); err != nil {
		t.Fatal(err)
	}
	now, _ := time.Parse(time.RFC3339, "2026-10-15T06:00:00Z")
	l.s.now = func() time.Time { return now }
	if got, err := l.s.DueReminders(ctx); err != nil || len(got) != 0 {
		t.Errorf("reminders = %+v, %v", got, err)
	}
}
