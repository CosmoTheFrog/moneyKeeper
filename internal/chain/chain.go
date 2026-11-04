// Package chain читает переводы публичного адреса кошелька: Ethereum (Blockscout), Tron (TronGrid) и Solana (публичный RPC).
// Ключей и seed-фраз тут нет, источник получает только адрес. Из активов берутся родная монета сети и USDT/USDC
// по адресам контрактов из списка: токены-подделки с чужими контрактами отбрасываются.
package chain

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"sort"
	"time"

	"moneykeeper/pkg/money"
)

// Монеты, которые читаются с сетей. Они же есть в реестре валют, откуда берётся точность.
const (
	coinETH  money.Currency = "ETH"
	coinSOL  money.Currency = "SOL"
	coinTRX  money.Currency = "TRX"
	coinUSDC money.Currency = "USDC"
)

type Chain string

const (
	Ethereum Chain = "eth"
	Tron     Chain = "tron"
	Solana   Chain = "sol"
)

const (
	timeout = 15 * time.Second
	// maxTransfers ограничивает один запрос: старше этой границы история подтягивается следующими нажатиями.
	maxTransfers = 200
	// maxBody защищает от слишком большого ответа источника.
	maxBody = 4 << 20
)

// ErrUnavailable: источник не ответил или ответил ошибкой.
var ErrUnavailable = errors.New("chain source unavailable")

// Transfer это один перевод актива. AmountMinor со знаком: положительный пришёл на адрес, отрицательный ушёл с него.
type Transfer struct {
	// ID стабилен между запросами и уникален по сети и активу: по нему повторный импорт пропускает уже внесённое.
	ID           string
	At           time.Time
	Currency     money.Currency
	AmountMinor  int64
	Counterparty string
}

// Client ходит к источникам. Адреса источников можно подменить в тестах.
type Client struct {
	http                 *http.Client
	ethURL, tronURL, sol string
}

func NewClient() *Client {
	return &Client{
		http:    &http.Client{Timeout: timeout},
		ethURL:  "https://eth.blockscout.com",
		tronURL: "https://api.trongrid.io",
		sol:     "https://api.mainnet-beta.solana.com",
	}
}

// Transfers возвращает переводы адреса не старше since, новые первыми, не больше maxTransfers.
func (c *Client) Transfers(ctx context.Context, ch Chain, address string, since time.Time) ([]Transfer, error) {
	if !Valid(ch, address) {
		return nil, fmt.Errorf("address %q for %s: %w", address, ch, errBadAddress)
	}
	var (
		out []Transfer
		err error
	)
	switch ch {
	case Ethereum:
		out, err = c.ethereum(ctx, address, since)
	case Tron:
		out, err = c.tron(ctx, address, since)
	case Solana:
		out, err = c.solana(ctx, address, since)
	}
	if err != nil {
		return nil, err
	}
	if len(out) > maxTransfers {
		out = out[:maxTransfers]
	}
	return out, nil
}

var errBadAddress = errors.New("bad address")

// IsBadAddress: адрес не подходит сети.
func IsBadAddress(err error) bool { return errors.Is(err, errBadAddress) }

func (c *Client) getJSON(ctx context.Context, url string, v any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	return c.do(req, v)
}

func (c *Client) postJSON(ctx context.Context, url string, body io.Reader, v any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.do(req, v)
}

func (c *Client) do(req *http.Request, v any) error {
	resp, err := c.http.Do(req)
	if err != nil {
		// В тексте ошибки http-клиента лежит весь адрес запроса, а в нём публичный адрес кошелька: в лог он не идёт.
		var ue *url.Error
		if errors.As(err, &ue) {
			err = ue.Err
		}
		return fmt.Errorf("%w: %w", ErrUnavailable, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: status %d", ErrUnavailable, resp.StatusCode)
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxBody)).Decode(v); err != nil {
		return fmt.Errorf("%w: decode: %w", ErrUnavailable, err)
	}
	return nil
}

// toMinor переводит целое число наименьших единиц сети (decimals знаков) в минорные единицы валюты с её точностью.
// Лишние знаки округляются. false: ноль после округления или не помещается в int64.
func toMinor(raw *big.Int, decimals int, cur money.Currency) (int64, bool) {
	units := money.MinorUnits(cur)
	v := new(big.Int).Set(raw)
	if units >= decimals {
		v.Mul(v, pow10(units-decimals))
	} else {
		d := pow10(decimals - units)
		v.Mul(v, big.NewInt(2))
		v.Add(v, d)
		v.Quo(v, new(big.Int).Mul(d, big.NewInt(2)))
	}
	if !v.IsInt64() || v.Sign() <= 0 {
		return 0, false
	}
	return v.Int64(), true
}

// signed задаёт знак перевода для адреса me: пришёл (+) или ушёл (−). Перевод самому себе и чужие пропускаются.
func signed(id string, at time.Time, cur money.Currency, minor int64, me, from, to string) (Transfer, bool) {
	switch {
	case from == to:
		return Transfer{}, false
	case to == me:
		return Transfer{ID: id, At: at, Currency: cur, AmountMinor: minor, Counterparty: from}, true
	case from == me:
		return Transfer{ID: id, At: at, Currency: cur, AmountMinor: -minor, Counterparty: to}, true
	}
	return Transfer{}, false
}

func sortNewest(ts []Transfer) []Transfer {
	sort.SliceStable(ts, func(i, j int) bool { return ts[i].At.After(ts[j].At) })
	return ts
}

func pow10(n int) *big.Int { return new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(n)), nil) }

func parseBig(s string) (*big.Int, bool) { return new(big.Int).SetString(s, 10) }
