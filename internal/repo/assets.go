package repo

import (
	"context"
	"fmt"

	"moneykeeper/internal/pgclient"
	"moneykeeper/pkg/money"
)

type Assets struct{ db *pgclient.Client }

func NewAssets(db *pgclient.Client) *Assets { return &Assets{db: db} }

// List возвращает добавленные пользователями активы в порядке добавления.
func (a *Assets) List(ctx context.Context) ([]money.Info, error) {
	rows, err := a.db.Q(ctx).Query(ctx, `SELECT code, stock, units FROM assets ORDER BY created_at, code`)
	if err != nil {
		return nil, fmt.Errorf("list assets: %w", err)
	}
	defer rows.Close()

	var out []money.Info
	for rows.Next() {
		var (
			code  string
			stock bool
			units int
		)
		if err := rows.Scan(&code, &stock, &units); err != nil {
			return nil, fmt.Errorf("scan asset: %w", err)
		}
		out = append(out, money.Info{Code: money.Currency(code), Units: units, Coin: true, Stock: stock})
	}
	return out, rows.Err()
}

// Create сохраняет актив. Уже существующий код не меняется: точность нельзя менять после первых записей.
func (a *Assets) Create(ctx context.Context, i money.Info) error {
	_, err := a.db.Q(ctx).Exec(ctx,
		`INSERT INTO assets (code, stock, units) VALUES ($1, $2, $3) ON CONFLICT (code) DO NOTHING`,
		string(i.Code), i.Stock, i.Units)
	if err != nil {
		return fmt.Errorf("create asset %s: %w", i.Code, err)
	}
	return nil
}
