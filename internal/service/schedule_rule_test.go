package service

import (
	"slices"
	"testing"

	"moneykeeper/internal/models"
	"moneykeeper/pkg/apptime"
)

func d(t *testing.T, s string) apptime.Day {
	t.Helper()
	v, err := apptime.ParseDay(s)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func strs(days []apptime.Day) []string {
	out := make([]string, len(days))
	for i, v := range days {
		out[i] = v.String()
	}
	return out
}

func rule(t *testing.T, every int, start string, shift models.WeekendShift, days ...int) models.Schedule {
	t.Helper()
	return models.Schedule{EveryMonths: every, StartMonth: month(t, start), Days: days, Shift: shift}
}

func TestOccurrenceDays(t *testing.T) {
	tests := []struct {
		name     string
		s        models.Schedule
		from, to string
		want     []string
	}{
		{"monthly on the 15th", rule(t, 1, "2026-08", models.ShiftNone, 15), "2026-09-01", "2026-11-30", []string{"2026-09-15", "2026-10-15", "2026-11-15"}},
		{"two dates a month", rule(t, 1, "2026-10", models.ShiftNone, 1, 15), "2026-10-01", "2026-11-01", []string{"2026-10-01", "2026-10-15", "2026-11-01"}},
		{"every third month from the start", rule(t, 3, "2026-01", models.ShiftNone, 10), "2026-01-01", "2026-12-31", []string{"2026-01-10", "2026-04-10", "2026-07-10", "2026-10-10"}},
		{"yearly", rule(t, 12, "2025-05", models.ShiftNone, 20), "2026-01-01", "2027-12-31", []string{"2026-05-20", "2027-05-20"}},
		{"last day of the month", rule(t, 1, "2026-01", models.ShiftNone, 0), "2026-02-01", "2026-04-30", []string{"2026-02-28", "2026-03-31", "2026-04-30"}},
		{"a day beyond the month takes its last day", rule(t, 1, "2026-01", models.ShiftNone, 31), "2026-02-01", "2026-04-30", []string{"2026-02-28", "2026-03-31", "2026-04-30"}},
		{"leap february", rule(t, 1, "2024-01", models.ShiftNone, 30), "2024-02-01", "2024-02-29", []string{"2024-02-29"}},
		// 3 октября 2026 года суббота, 4 воскресенье.
		{"saturday stays", rule(t, 1, "2026-10", models.ShiftNone, 3), "2026-10-01", "2026-10-31", []string{"2026-10-03"}},
		{"saturday back", rule(t, 1, "2026-10", models.ShiftBack, 3), "2026-10-01", "2026-10-31", []string{"2026-10-02"}},
		{"saturday forward", rule(t, 1, "2026-10", models.ShiftForward, 3), "2026-10-01", "2026-10-31", []string{"2026-10-05"}},
		{"sunday back", rule(t, 1, "2026-10", models.ShiftBack, 4), "2026-10-01", "2026-10-31", []string{"2026-10-02"}},
		{"sunday forward", rule(t, 1, "2026-10", models.ShiftForward, 4), "2026-10-01", "2026-10-31", []string{"2026-10-05"}},
		// 1 ноября 2026 года воскресенье: перенос назад уходит в октябрь, вперёд остаётся в ноябре.
		{"shift crosses into the previous month", rule(t, 1, "2026-10", models.ShiftBack, 1), "2026-10-25", "2026-11-05", []string{"2026-10-30"}},
		{"shift crosses into the next month", rule(t, 1, "2026-10", models.ShiftForward, 31), "2026-10-25", "2026-11-05", []string{"2026-11-02"}},
		{"weekday is not shifted", rule(t, 1, "2026-10", models.ShiftBack, 12), "2026-10-01", "2026-10-31", []string{"2026-10-12"}},
		{"nothing before the start month", rule(t, 1, "2026-10", models.ShiftNone, 5), "2026-01-01", "2026-09-30", nil},
		{"window inside a month", rule(t, 1, "2026-01", models.ShiftNone, 5, 25), "2026-10-10", "2026-10-20", nil},
		{"duplicated days after a shift collapse", rule(t, 1, "2026-10", models.ShiftBack, 3, 4), "2026-10-01", "2026-10-31", []string{"2026-10-02"}},
		{"from after to", rule(t, 1, "2026-01", models.ShiftNone, 5), "2026-10-20", "2026-10-01", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := strs(occurrenceDays(tt.s, d(t, tt.from), d(t, tt.to)))
			if !slices.Equal(got, tt.want) && !(len(got) == 0 && len(tt.want) == 0) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAmountFits(t *testing.T) {
	max := int64(12000)
	tests := []struct {
		name   string
		s      models.Schedule
		amount int64
		want   bool
	}{
		{"exact equal", models.Schedule{AmountKind: models.AmountExact, AmountMinor: 10000}, 10000, true},
		{"exact differs by a cent", models.Schedule{AmountKind: models.AmountExact, AmountMinor: 10000}, 10001, false},
		{"approx within 7,5%", models.Schedule{AmountKind: models.AmountApprox, AmountMinor: 10000}, 10750, true},
		{"approx lower edge", models.Schedule{AmountKind: models.AmountApprox, AmountMinor: 10000}, 9250, true},
		{"approx just above", models.Schedule{AmountKind: models.AmountApprox, AmountMinor: 10000}, 10751, false},
		{"approx just below", models.Schedule{AmountKind: models.AmountApprox, AmountMinor: 10000}, 9249, false},
		{"range low edge", models.Schedule{AmountKind: models.AmountRange, AmountMinor: 8000, AmountMaxMinor: &max}, 8000, true},
		{"range high edge", models.Schedule{AmountKind: models.AmountRange, AmountMinor: 8000, AmountMaxMinor: &max}, 12000, true},
		{"range above", models.Schedule{AmountKind: models.AmountRange, AmountMinor: 8000, AmountMaxMinor: &max}, 12001, false},
		{"range below", models.Schedule{AmountKind: models.AmountRange, AmountMinor: 8000, AmountMaxMinor: &max}, 7999, false},
		{"range without a bound never fits", models.Schedule{AmountKind: models.AmountRange, AmountMinor: 8000}, 8000, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := amountFits(tt.s, tt.amount); got != tt.want {
				t.Errorf("amountFits = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAssignSpends(t *testing.T) {
	acc := int64(7)
	s := models.Schedule{CategoryID: 1, Currency: "GEL", AmountKind: models.AmountExact, AmountMinor: 50000, AccountID: &acc}
	dues := []apptime.Day{d(t, "2026-09-10"), d(t, "2026-10-10")}
	mk := func(id int64, day string, amount int64) spend {
		return spend{ID: id, Day: d(t, day), Amount: amount, Currency: "GEL", CategoryID: 1, AccountID: 7}
	}

	tests := []struct {
		name   string
		spends []spend
		want   map[string]int64
	}{
		{"exact day", []spend{mk(1, "2026-10-10", 50000)}, map[string]int64{"2026-10-10": 1}},
		{"two days off still matches", []spend{mk(1, "2026-10-12", 50000)}, map[string]int64{"2026-10-10": 1}},
		{"three days off does not", []spend{mk(1, "2026-10-13", 50000)}, map[string]int64{}},
		{"wrong amount", []spend{mk(1, "2026-10-10", 49999)}, map[string]int64{}},
		{"nearest day wins", []spend{mk(1, "2026-10-12", 50000), mk(2, "2026-10-09", 50000)}, map[string]int64{"2026-10-10": 2}},
		{"each spend closes one payment", []spend{mk(1, "2026-09-10", 50000), mk(2, "2026-10-10", 50000)}, map[string]int64{"2026-09-10": 1, "2026-10-10": 2}},
		{"one spend between two payments closes only one", []spend{mk(1, "2026-09-11", 50000)}, map[string]int64{"2026-09-10": 1}},
		{"other category", []spend{{ID: 1, Day: d(t, "2026-10-10"), Amount: 50000, Currency: "GEL", CategoryID: 2, AccountID: 7}}, map[string]int64{}},
		{"other currency", []spend{{ID: 1, Day: d(t, "2026-10-10"), Amount: 50000, Currency: "EUR", CategoryID: 1, AccountID: 7}}, map[string]int64{}},
		{"other account", []spend{{ID: 1, Day: d(t, "2026-10-10"), Amount: 50000, Currency: "GEL", CategoryID: 1, AccountID: 8}}, map[string]int64{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := map[string]int64{}
			for due, id := range assignSpends(s, dues, tt.spends) {
				got[due.String()] = id
			}
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("got %v, want %v", got, tt.want)
				}
			}
		})
	}

	t.Run("a schedule without an account matches any account", func(t *testing.T) {
		any := s
		any.AccountID = nil
		got := assignSpends(any, dues, []spend{{ID: 1, Day: d(t, "2026-10-10"), Amount: 50000, Currency: "GEL", CategoryID: 1, AccountID: 99}})
		if got[d(t, "2026-10-10")] != 1 {
			t.Errorf("got %v", got)
		}
	})
}
