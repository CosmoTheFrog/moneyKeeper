package chain

import (
	"context"
	"encoding/hex"
	"math/big"
	"net/url"
	"strconv"
	"time"

	"moneykeeper/pkg/money"
)

// tronTokens: контракты TRC-20, чьи переводы берутся.
var tronTokens = map[string]money.Currency{
	"TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t": money.USDT,
	"TEkxiTehnzSmSe2XqrBj4w32RUN966rdz8": coinUSDC,
}

type tronTx struct {
	TxID      string `json:"txID"`
	Timestamp int64  `json:"block_timestamp"`
	Ret       []struct {
		ContractRet string `json:"contractRet"`
	} `json:"ret"`
	RawData struct {
		Contract []struct {
			Type      string `json:"type"`
			Parameter struct {
				Value struct {
					Amount int64  `json:"amount"`
					Owner  string `json:"owner_address"`
					To     string `json:"to_address"`
				} `json:"value"`
			} `json:"parameter"`
		} `json:"contract"`
	} `json:"raw_data"`
}

type tronTRC20 struct {
	TxID      string `json:"transaction_id"`
	Timestamp int64  `json:"block_timestamp"`
	From      string `json:"from"`
	To        string `json:"to"`
	Value     string `json:"value"`
	Type      string `json:"type"`
	Token     struct {
		Address  string `json:"address"`
		Decimals int    `json:"decimals"`
	} `json:"token_info"`
}

func (c *Client) tron(ctx context.Context, address string, since time.Time) ([]Transfer, error) {
	q := url.Values{"limit": {"200"}, "only_confirmed": {"true"}, "min_timestamp": {strconv.FormatInt(since.UnixMilli(), 10)}}
	base := c.tronURL + "/v1/accounts/" + address

	var native struct {
		Data []tronTx `json:"data"`
	}
	if err := c.getJSON(ctx, base+"/transactions?"+q.Encode(), &native); err != nil {
		return nil, err
	}
	var out []Transfer
	for _, t := range native.Data {
		if len(t.Ret) == 0 || t.Ret[0].ContractRet != "SUCCESS" || len(t.RawData.Contract) == 0 || t.RawData.Contract[0].Type != "TransferContract" {
			continue
		}
		v := t.RawData.Contract[0].Parameter.Value
		minor, ok := toMinor(big.NewInt(v.Amount), 6, coinTRX)
		if !ok {
			continue
		}
		if tr, ok := signed("tron:"+t.TxID+":TRX", time.UnixMilli(t.Timestamp), coinTRX, minor, address, tronAddr(v.Owner), tronAddr(v.To)); ok {
			out = append(out, tr)
		}
	}

	var tokens struct {
		Data []tronTRC20 `json:"data"`
	}
	if err := c.getJSON(ctx, base+"/transactions/trc20?"+q.Encode(), &tokens); err != nil {
		return nil, err
	}
	for _, t := range tokens.Data {
		cur, ok := tronTokens[t.Token.Address]
		raw, ok1 := parseBig(t.Value)
		if !ok || !ok1 || t.Type != "Transfer" {
			continue
		}
		minor, ok := toMinor(raw, t.Token.Decimals, cur)
		if !ok {
			continue
		}
		if tr, ok := signed("tron:"+t.TxID+":"+string(cur), time.UnixMilli(t.Timestamp), cur, minor, address, t.From, t.To); ok {
			out = append(out, tr)
		}
	}
	return sortNewest(out), nil
}

// tronAddr переводит hex-адрес из TransferContract в base58. Пустая строка, если он не разобрался.
func tronAddr(h string) string {
	b, err := hex.DecodeString(h)
	if err != nil || len(b) != 21 {
		return ""
	}
	return tronHexToBase58(b)
}
