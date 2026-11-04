//go:build integration

package service

import (
	"context"
	"testing"

	"moneykeeper/internal/models"
)

func TestDailyReminder(t *testing.T) {
	ctx := context.Background()
	l := newLedger(t)
	due := func() []models.DailyReminder {
		t.Helper()
		got, err := l.s.DueDailyReminders(ctx)
		if err != nil {
			t.Fatal(err)
		}
		return got
	}

	if got := due(); len(got) != 0 {
		t.Fatalf("reminder is off by default, got %+v", got)
	}
	// Сейчас у пользователя 01:30 (Тбилиси): в два часа ещё рано, в час уже пора.
	two, one := 2, 1
	if err := l.s.SetReminderHour(ctx, l.u, &two); err != nil {
		t.Fatal(err)
	}
	if got := due(); len(got) != 0 {
		t.Errorf("before the chosen hour: %+v", got)
	}
	if err := l.s.SetReminderHour(ctx, l.u, &one); err != nil {
		t.Fatal(err)
	}
	got := due()
	if len(got) != 1 || got[0].TelegramID != tgIgor.ID {
		t.Fatalf("at the chosen hour: %+v", got)
	}

	if err := l.s.MarkDailyReminded(ctx, got[0]); err != nil {
		t.Fatal(err)
	}
	if got := due(); len(got) != 0 {
		t.Errorf("already sent today: %+v", got)
	}
}

func TestDailyReminderSkipsWhenExpenseEntered(t *testing.T) {
	ctx := context.Background()
	l := newLedger(t)
	one := 1
	if err := l.s.SetReminderHour(ctx, l.u, &one); err != nil {
		t.Fatal(err)
	}
	l.expense(t, l.main.ID, 500, "Продукты")
	if got, err := l.s.DueDailyReminders(ctx); err != nil || len(got) != 0 {
		t.Errorf("expense already entered today: %+v, %v", got, err)
	}
}

func TestLimitAlerts(t *testing.T) {
	ctx := context.Background()
	l := newLedger(t)
	l.setBase(t, "GEL")
	food := l.category(t, "Продукты")
	if err := l.s.SetBudgeted(ctx, l.u, food, month(t, "2026-10"), 10000); err != nil {
		t.Fatal(err)
	}
	alerts := func() []models.LimitAlert {
		t.Helper()
		got, err := l.s.DueLimitAlerts(ctx)
		if err != nil {
			t.Fatal(err)
		}
		return got
	}

	l.expense(t, l.main.ID, 7000, "Продукты")
	enable := func(on bool) {
		t.Helper()
		if b, err := l.s.SetEnvelopes(ctx, l.u, on); err != nil || b.EnvelopesEnabled != on {
			t.Fatalf("set envelopes %v: %+v, %v", on, b, err)
		}
	}
	// Режим конвертов выключен по умолчанию: заложенная сумма молчит, даже когда расход её почти выбрал.
	l.expense(t, l.main.ID, 1500, "Продукты")
	if got := alerts(); len(got) != 0 {
		t.Fatalf("envelopes are off: %+v", got)
	}

	enable(true)
	got := alerts()
	if len(got) != 1 || got[0].Threshold != models.LimitWarn || got[0].CategoryName != "Продукты" || got[0].SpentMinor != 8500 {
		t.Fatalf("85%% of the limit: %+v", got)
	}
	if err := l.s.MarkLimitAlerted(ctx, got[0]); err != nil {
		t.Fatal(err)
	}
	if got := alerts(); len(got) != 0 {
		t.Errorf("80%% already sent: %+v", got)
	}

	// Расход перескочил через 100%: приходит предупреждение о лимите, и о 80% уже не напоминаем.
	l.expense(t, l.main.ID, 3000, "Продукты")
	got = alerts()
	if len(got) != 1 || got[0].Threshold != models.LimitFull {
		t.Fatalf("over the limit: %+v", got)
	}
	if err := l.s.MarkLimitAlerted(ctx, got[0]); err != nil {
		t.Fatal(err)
	}
	if got := alerts(); len(got) != 0 {
		t.Errorf("100%% already sent: %+v", got)
	}

	// Выключение прячет конверты, но заложенная сумма остаётся.
	enable(false)
	if rows := l.budget(t, "2026-10").Rows; len(rows) == 0 {
		t.Fatal("budget rows disappeared")
	}
	if b := l.budget(t, "2026-10"); b.BudgetedMinor != 10000 {
		t.Errorf("budgeted after disabling = %d, want 10000", b.BudgetedMinor)
	}
}
