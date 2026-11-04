package chain

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
	"time"

	"moneykeeper/pkg/money"
)

// ethPages: сколько страниц по 50 записей читается на каждый запрос.
const ethPages = 3

// ethTokens: контракты, чьи переводы берутся (в нижнем регистре). Остальные токены, в том числе подделки, пропускаются.
var ethTokens = map[string]money.Currency{
	"0xdac17f958d2ee523a2206206994597c13d831ec7": money.USDT,
	"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48": coinUSDC,
}

type bsAddr struct {
	Hash string `json:"hash"`
}

type bsTx struct {
	Hash      string    `json:"hash"`
	Timestamp time.Time `json:"timestamp"`
	Value     string    `json:"value"`
	Status    string    `json:"status"`
	From      bsAddr    `json:"from"`
	To        bsAddr    `json:"to"`
}

type bsTokenTransfer struct {
	TxHash    string    `json:"transaction_hash"`
	LogIndex  int       `json:"log_index"`
	Timestamp time.Time `json:"timestamp"`
	From      bsAddr    `json:"from"`
	To        bsAddr    `json:"to"`
	Token     struct {
		Address string `json:"address_hash"`
	} `json:"token"`
	Total struct {
		Value    string `json:"value"`
		Decimals string `json:"decimals"`
	} `json:"total"`
}

type bsPage[T any] struct {
	Items []T                        `json:"items"`
	Next  map[string]json.RawMessage `json:"next_page_params"`
}

// bsFetch читает страницы списка Blockscout, пока записи не станут старше since или не кончатся страницы.
func bsFetch[T any](ctx context.Context, c *Client, path string, since time.Time, at func(T) time.Time) ([]T, error) {
	var (
		out    []T
		cursor = url.Values{}
	)
	for range ethPages {
		var page bsPage[T]
		if err := c.getJSON(ctx, c.ethURL+path+"?"+cursor.Encode(), &page); err != nil {
			return nil, err
		}
		old := false
		for _, it := range page.Items {
			if at(it).Before(since) {
				old = true
				continue
			}
			out = append(out, it)
		}
		if old || len(page.Next) == 0 {
			break
		}
		cursor = url.Values{}
		for k, v := range page.Next {
			cursor.Set(k, strings.Trim(string(v), `"`))
		}
	}
	return out, nil
}

func (c *Client) ethereum(ctx context.Context, address string, since time.Time) ([]Transfer, error) {
	me := strings.ToLower(address)
	base := "/api/v2/addresses/" + address

	txs, err := bsFetch(ctx, c, base+"/transactions", since, func(t bsTx) time.Time { return t.Timestamp })
	if err != nil {
		return nil, err
	}
	var out []Transfer
	for _, t := range txs {
		if t.Status != "ok" {
			continue
		}
		raw, ok := parseBig(t.Value)
		if !ok {
			continue
		}
		minor, ok := toMinor(raw, 18, coinETH)
		if !ok {
			continue
		}
		if tr, ok := signed("eth:"+t.Hash+":ETH", t.Timestamp, coinETH, minor, me, strings.ToLower(t.From.Hash), strings.ToLower(t.To.Hash)); ok {
			out = append(out, tr)
		}
	}

	tokens, err := bsFetch(ctx, c, base+"/token-transfers", since, func(t bsTokenTransfer) time.Time { return t.Timestamp })
	if err != nil {
		return nil, err
	}
	for _, t := range tokens {
		cur, ok := ethTokens[strings.ToLower(t.Token.Address)]
		if !ok {
			continue
		}
		raw, ok1 := parseBig(t.Total.Value)
		decimals, err := strconv.Atoi(t.Total.Decimals)
		if !ok1 || err != nil {
			continue
		}
		minor, ok := toMinor(raw, decimals, cur)
		if !ok {
			continue
		}
		id := "eth:" + t.TxHash + ":" + string(cur) + ":" + strconv.Itoa(t.LogIndex)
		if tr, ok := signed(id, t.Timestamp, cur, minor, me, strings.ToLower(t.From.Hash), strings.ToLower(t.To.Hash)); ok {
			out = append(out, tr)
		}
	}
	return sortNewest(out), nil
}
