// Package fx получает курсы валют к лари: фиат у Нацбанка Грузии, USDT через Binance.
// Курсы это цена одной целой единицы валюты в GEL.
package fx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"sync"
	"time"

	"moneykeeper/pkg/apptime"
	"moneykeeper/pkg/money"
)

const (
	nbgURL      = "https://nbg.gov.ge"
	binanceURL  = "https://api.binance.com"
	timeout     = 10 * time.Second
	maxParallel = 5
)

// fiat — валюты, которые берём у НБГ.
var fiat = []money.Currency{money.USD, money.EUR, money.RUB}

type Provider struct {
	nbg     string
	binance string
	client  *http.Client
}

func NewProvider() *Provider {
	return &Provider{nbg: nbgURL, binance: binanceURL, client: &http.Client{Timeout: timeout}}
}

// GelPerUnit возвращает курсы на день: фиат, USDT и монеты из coins. Если часть источников недоступна, в карте то,
// что удалось получить, а ошибка описывает недостающее: вызывающий сохраняет полученное и не падает.
func (p *Provider) GelPerUnit(ctx context.Context, day apptime.Day, coins []money.Currency) (map[money.Currency]*big.Rat, error) {
	out, err := p.fiatRates(ctx, day)
	if err != nil {
		return out, err
	}
	usdt, err := p.usdtRate(ctx, day, out[money.EUR])
	if err != nil {
		return out, err
	}
	out[money.USDT] = usdt

	// Монет бывает до двух десятков, поэтому запросы идут параллельно, но не больше maxParallel сразу.
	var (
		mu   sync.Mutex
		wg   sync.WaitGroup
		errs []error
		sem  = make(chan struct{}, maxParallel)
	)
	for _, c := range coins {
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			price, err := p.dailyClose(ctx, string(c)+"USDT", day)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errs = append(errs, fmt.Errorf("binance %s: %w", c, err))
				return
			}
			out[c] = new(big.Rat).Mul(price, usdt) // цена монеты в USDT на лари за USDT
		}()
	}
	wg.Wait()
	return out, errors.Join(errs...)
}

type nbgDay struct {
	Currencies []struct {
		Code     string      `json:"code"`
		Quantity int64       `json:"quantity"`
		Rate     json.Number `json:"rate"`
	} `json:"currencies"`
}

// fiatRates берёт официальные курсы НБГ. Для выходных и праздников НБГ отдаёт последний действующий курс.
func (p *Provider) fiatRates(ctx context.Context, day apptime.Day) (map[money.Currency]*big.Rat, error) {
	q := url.Values{"date": {day.String()}}
	var days []nbgDay
	if err := p.getJSON(ctx, p.nbg+"/gw/api/ct/monetarypolicy/currencies/en/json/?"+q.Encode(), &days); err != nil {
		return map[money.Currency]*big.Rat{}, fmt.Errorf("nbg: %w", err)
	}

	out := map[money.Currency]*big.Rat{}
	for _, d := range days {
		for _, c := range d.Currencies {
			cur := money.Currency(c.Code)
			if !isFiat(cur) || c.Quantity <= 0 {
				continue
			}
			rate, ok := new(big.Rat).SetString(c.Rate.String())
			if !ok || rate.Sign() <= 0 {
				return out, fmt.Errorf("nbg %s: bad rate %q", c.Code, c.Rate)
			}
			out[cur] = rate.Quo(rate, new(big.Rat).SetInt64(c.Quantity))
		}
	}
	for _, c := range fiat {
		if out[c] == nil {
			return out, fmt.Errorf("nbg: no rate for %s on %s", c, day)
		}
	}
	return out, nil
}

// usdtRate выводит цену USDT в лари из пары EURUSDT и евро НБГ, не принимая USDT равным доллару.
func (p *Provider) usdtRate(ctx context.Context, day apptime.Day, gelPerEUR *big.Rat) (*big.Rat, error) {
	if gelPerEUR == nil {
		return nil, errors.New("binance: no EUR rate to derive USDT from")
	}
	eurUSDT, err := p.dailyClose(ctx, "EURUSDT", day)
	if err != nil {
		return nil, fmt.Errorf("binance: %w", err)
	}
	return new(big.Rat).Quo(gelPerEUR, eurUSDT), nil // 1 USDT = (₾ за €) / (USDT за €)
}

// dailyClose возвращает цену закрытия дневной свечи пары. Для текущего дня это последняя цена. Берётся последняя свеча,
// начавшаяся не позже конца дня, поэтому при пропуске торгов в этот день используется ближайшая предыдущая.
func (p *Provider) dailyClose(ctx context.Context, symbol string, day apptime.Day) (*big.Rat, error) {
	q := url.Values{
		"symbol":   {symbol},
		"interval": {"1d"},
		"endTime":  {strconv.FormatInt(day.Date().Add(24*time.Hour).UnixMilli()-1, 10)},
		"limit":    {"1"},
	}
	var klines [][]json.RawMessage
	if err := p.getJSON(ctx, p.binance+"/api/v3/klines?"+q.Encode(), &klines); err != nil {
		return nil, err
	}
	if len(klines) == 0 || len(klines[0]) < 5 {
		return nil, fmt.Errorf("%s on %s: no candle", symbol, day)
	}
	var closeStr string
	if err := json.Unmarshal(klines[0][4], &closeStr); err != nil {
		return nil, fmt.Errorf("%s close: %w", symbol, err)
	}
	price, ok := new(big.Rat).SetString(closeStr)
	if !ok || price.Sign() <= 0 {
		return nil, fmt.Errorf("%s: bad price %q", symbol, closeStr)
	}
	return price, nil
}

func (p *Provider) getJSON(ctx context.Context, rawURL string, dst any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(dst)
}

func isFiat(c money.Currency) bool {
	return slices.Contains(fiat, c)
}
