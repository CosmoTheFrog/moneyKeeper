package apptime

import (
	"errors"
	"fmt"
	"time"
)

const monthLayout = "2006-01"

var ErrInvalidMonth = errors.New("invalid month")

// Month это календарный месяц без дня, например 2026-10. Бюджет заложен по месяцам.
type Month struct {
	year  int
	month time.Month
}

func ParseMonth(s string) (Month, error) {
	t, err := time.Parse(monthLayout, s)
	if err != nil {
		return Month{}, fmt.Errorf("%q: %w", s, ErrInvalidMonth)
	}
	return Month{t.Year(), t.Month()}, nil
}

// MonthOf возвращает месяц, в который попадает день.
func MonthOf(d Day) Month { return Month{d.year, d.month} }

// First это первый день месяца.
func (m Month) First() Day { return Day{m.year, m.month, 1} }

// Last это последний день месяца.
func (m Month) Last() Day { return FromDate(m.Next().First().Date().AddDate(0, 0, -1)) }

func (m Month) Next() Month {
	t := time.Date(m.year, m.month+1, 1, 0, 0, 0, 0, time.UTC)
	return Month{t.Year(), t.Month()}
}

func (m Month) String() string { return m.First().Date().Format(monthLayout) }

func (m Month) IsZero() bool { return m == Month{} }

func (m Month) Before(o Month) bool {
	return m.year < o.year || (m.year == o.year && m.month < o.month)
}
