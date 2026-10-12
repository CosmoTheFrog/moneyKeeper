package repo

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"moneykeeper/internal/pgclient"
	"moneykeeper/pkg/apptime"
	"moneykeeper/pkg/money"
)

// storedRateScale совпадает с NUMERIC(24,10) в миграции.
const storedRateScale = 10

type Rates struct{ db *pgclient.Client }

func NewRates(db *pgclient.Client) *Rates { return &Rates{db: db} }

// Store сохраняет курсы дня. Уже записанный курс не меняется.
func (r *Rates) Store(ctx context.Context, day apptime.Day, gelPerUnit map[money.Currency]*big.Rat) error {
	for cur, rate := range gelPerUnit {
		_, err := r.db.Q(ctx).Exec(ctx,
			`INSERT INTO rates (day, currency, gel_per_unit) VALUES ($1, $2, $3::numeric)
			 ON CONFLICT (day, currency) DO NOTHING`,
			day.Date(), string(cur), rate.FloatString(storedRateScale))
		if err != nil {
			return fmt.Errorf("store rate %s: %w", cur, err)
		}
	}
	return nil
}

// ForDays возвращает курсы на заданные дни: день, затем валюта.
func (r *Rates) ForDays(ctx context.Context, days []apptime.Day) (map[apptime.Day]map[money.Currency]*big.Rat, error) {
	dates := make([]time.Time, 0, len(days))
	for _, d := range days {
		dates = append(dates, d.Date())
	}
	rows, err := r.db.Q(ctx).Query(ctx,
		`SELECT day, currency, gel_per_unit::text FROM rates WHERE day = ANY($1)`, dates)
	if err != nil {
		return nil, fmt.Errorf("rates for days: %w", err)
	}
	defer rows.Close()

	out := map[apptime.Day]map[money.Currency]*big.Rat{}
	for rows.Next() {
		var (
			d    time.Time
			cu   string
			text string
		)
		if err := rows.Scan(&d, &cu, &text); err != nil {
			return nil, fmt.Errorf("scan rate: %w", err)
		}
		rate, ok := new(big.Rat).SetString(text)
		if !ok {
			return nil, fmt.Errorf("rate %s %q: bad number", cu, text)
		}
		day := apptime.FromDate(d)
		if out[day] == nil {
			out[day] = map[money.Currency]*big.Rat{}
		}
		out[day][money.Currency(cu)] = rate
	}
	return out, rows.Err()
}
