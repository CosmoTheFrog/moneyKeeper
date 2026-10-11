// Package apptime хранит календарный день отдельным типом: дата без времени и без пояса.
// «Сегодня» всегда считается в поясе пользователя.
package apptime

import (
	"errors"
	"fmt"
	"time"
)

const dayLayout = "2006-01-02"

var ErrInvalidDay = errors.New("invalid day")

type Day struct {
	year  int
	month time.Month
	day   int
}

func NewDay(year int, month time.Month, day int) (Day, error) {
	t := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	if t.Year() != year || t.Month() != month || t.Day() != day {
		return Day{}, fmt.Errorf("%04d-%02d-%02d: %w", year, month, day, ErrInvalidDay)
	}
	return Day{year, month, day}, nil
}

func ParseDay(s string) (Day, error) {
	t, err := time.Parse(dayLayout, s)
	if err != nil {
		return Day{}, fmt.Errorf("%q: %w", s, ErrInvalidDay)
	}
	return Day{t.Year(), t.Month(), t.Day()}, nil
}

// Today возвращает календарный день момента now в поясе loc.
func Today(loc *time.Location, now time.Time) Day {
	t := now.In(loc)
	return Day{t.Year(), t.Month(), t.Day()}
}

// FromDate переводит значение колонки DATE (UTC-полночь) в Day.
func FromDate(t time.Time) Day {
	return Day{t.Year(), t.Month(), t.Day()}
}

// Date отдаёт UTC-полночь этого дня для записи в колонку DATE.
func (d Day) Date() time.Time {
	return time.Date(d.year, d.month, d.day, 0, 0, 0, 0, time.UTC)
}

func (d Day) String() string { return d.Date().Format(dayLayout) }

func (d Day) MarshalText() ([]byte, error) { return []byte(d.String()), nil }

func (d *Day) UnmarshalText(b []byte) error {
	parsed, err := ParseDay(string(b))
	if err != nil {
		return err
	}
	*d = parsed
	return nil
}

func (d Day) IsZero() bool { return d == Day{} }

func (d Day) Before(o Day) bool { return d.Date().Before(o.Date()) }
