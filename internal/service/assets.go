package service

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"moneykeeper/internal/models"
	"moneykeeper/pkg/money"
)

// assetUnits подбирает точность по цене одной штуки в долларах: чем дешевле штука, тем меньше знаков нужно,
// чтобы предел суммы в 10^12 минорных единиц не превращался в мизер (у SHIB знаков нет, у BTC восемь).
// Акции всегда с четырьмя знаками для дробных долей.
func assetUnits(stock bool, usd *big.Rat) int {
	if stock {
		return 4
	}
	steps := []struct {
		from  string
		units int
	}{{"1000", 8}, {"10", 6}, {"1/100", 4}, {"1/10000", 2}}
	for _, st := range steps {
		min, _ := new(big.Rat).SetString(st.from)
		if usd.Cmp(min) >= 0 {
			return st.units
		}
	}
	return 0
}

// LoadAssets регистрирует сохранённые активы при старте сервера.
func (s *Service) LoadAssets(ctx context.Context) error {
	infos, err := s.assets.List(ctx)
	if err != nil {
		return err
	}
	for _, i := range infos {
		if err := money.Register(i); err != nil && !errors.Is(err, money.ErrAssetExists) {
			return fmt.Errorf("register asset: %w", err)
		}
	}
	return nil
}

// AddAsset добавляет монету или акцию по коду: цена проверяется у источника (Binance для монет, Yahoo Finance для акций),
// точность подбирается по цене. Уже известный код возвращается как есть.
func (s *Service) AddAsset(ctx context.Context, code string, stock bool) (money.Info, error) {
	c := money.Currency(strings.ToUpper(strings.TrimSpace(code)))
	if known, ok := money.Lookup(c); ok {
		return known, nil
	}
	if _, err := money.Parse(string(c)); !errors.Is(err, money.ErrUnknownCurrency) {
		return money.Info{}, err
	}
	// Код проверяется до обращения к источнику: в запрос уходит только то, что похоже на тикер.
	probe := money.Info{Code: c, Coin: true, Stock: stock}
	if err := money.CheckAsset(probe); err != nil {
		return money.Info{}, err
	}
	price, err := s.fx.AssetPrice(ctx, c, stock)
	if err != nil {
		s.log.Info("asset not found", "asset", string(c), "err", err)
		return money.Info{}, fmt.Errorf("%s: %w", c, models.ErrAssetNotFound)
	}
	probe.Units = assetUnits(stock, price)
	if err := s.assets.Create(ctx, probe); err != nil {
		return money.Info{}, err
	}
	if err := money.Register(probe); err != nil && !errors.Is(err, money.ErrAssetExists) {
		return money.Info{}, err
	}
	return probe, nil
}
