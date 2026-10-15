package service

import (
	"slices"
	"sort"
	"time"

	"moneykeeper/internal/models"
	"moneykeeper/pkg/apptime"
	"moneykeeper/pkg/money"
)

const (
	// matchWindowDays: трата считается платежом расписания, если она не дальше двух дней от даты платежа.
	matchWindowDays = 2
	// approxPermille: допуск примерной суммы, 7,5% в тысячных долях.
	approxPermille = 75
)

// occurrenceDays возвращает даты платежей расписания в промежутке from..to включительно по порядку. Повторение идёт
// от месяца StartMonth раз в EveryMonths месяцев; число больше длины месяца и нулевое число дают последний день месяца;
// платёж на выходной переносится по правилу расписания. Перенос может вынести платёж в соседний месяц.
func occurrenceDays(s models.Schedule, from, to apptime.Day) []apptime.Day {
	var out []apptime.Day
	// Перенос сдвигает дату не больше чем на два дня, поэтому месяцы смотрим с запасом.
	limit := to.Date().AddDate(0, 0, 3)
	for m := s.StartMonth; !m.First().Date().After(limit); m = addMonths(m, s.EveryMonths) {
		last := m.Last().Date().Day()
		for _, d := range s.Days {
			day := d
			if day <= 0 || day > last {
				day = last
			}
			date := time.Date(m.First().Date().Year(), m.First().Date().Month(), day, 0, 0, 0, 0, time.UTC)
			due := apptime.FromDate(shiftWeekend(date, s.Shift))
			if !due.Before(from) && !to.Before(due) && !slices.Contains(out, due) {
				out = append(out, due)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Before(out[j]) })
	return out
}

func addMonths(m apptime.Month, n int) apptime.Month {
	for range n {
		m = m.Next()
	}
	return m
}

func shiftWeekend(d time.Time, shift models.WeekendShift) time.Time {
	switch d.Weekday() {
	case time.Saturday:
		switch shift {
		case models.ShiftBack:
			return d.AddDate(0, 0, -1)
		case models.ShiftForward:
			return d.AddDate(0, 0, 2)
		}
	case time.Sunday:
		switch shift {
		case models.ShiftBack:
			return d.AddDate(0, 0, -2)
		case models.ShiftForward:
			return d.AddDate(0, 0, 1)
		}
	}
	return d
}

// spend это расход, который может оказаться платежом расписания. Amount положительная.
type spend struct {
	ID         int64
	Day        apptime.Day
	Amount     int64
	Currency   money.Currency
	CategoryID int64
	AccountID  int64
}

func dayDistance(a, b apptime.Day) int {
	d := int(a.Date().Sub(b.Date()).Hours() / 24)
	if d < 0 {
		return -d
	}
	return d
}

// amountFits проверяет сумму траты по виду суммы расписания.
func amountFits(s models.Schedule, amount int64) bool {
	switch s.AmountKind {
	case models.AmountApprox:
		diff := amount - s.AmountMinor
		if diff < 0 {
			diff = -diff
		}
		return diff*1000 <= s.AmountMinor*approxPermille
	case models.AmountRange:
		return s.AmountMaxMinor != nil && amount >= s.AmountMinor && amount <= *s.AmountMaxMinor
	default:
		return amount == s.AmountMinor
	}
}

// spendMatches: та же категория и валюта, подходящая сумма, тот же счёт (если он задан) и дата в окне вокруг платежа.
func spendMatches(s models.Schedule, due apptime.Day, t spend) bool {
	if t.CategoryID != s.CategoryID || t.Currency != s.Currency || !amountFits(s, t.Amount) {
		return false
	}
	if s.AccountID != nil && *s.AccountID != t.AccountID {
		return false
	}
	return dayDistance(due, t.Day) <= matchWindowDays
}

// assignSpends сопоставляет платежи с тратами по порядку дат: каждому платежу ближайшая по дате подходящая трата,
// одна трата закрывает только один платёж. Возвращает номер траты по дате платежа.
func assignSpends(s models.Schedule, dues []apptime.Day, spends []spend) map[apptime.Day]int64 {
	used := map[int64]bool{}
	out := map[apptime.Day]int64{}
	for _, due := range dues {
		best := -1
		for i, t := range spends {
			if used[t.ID] || !spendMatches(s, due, t) {
				continue
			}
			if best < 0 || dayDistance(due, t.Day) < dayDistance(due, spends[best].Day) {
				best = i
			}
		}
		if best >= 0 {
			used[spends[best].ID] = true
			out[due] = spends[best].ID
		}
	}
	return out
}
