package chain

import (
	"bytes"
	"context"
	"encoding/json"
	"math/big"
	"time"

	"moneykeeper/pkg/money"
)

// solSignatures: сколько последних операций адреса читается за раз; каждая это отдельный getTransaction в одном батче.
const solSignatures = 50

// solMints: токены SPL, чьи переводы берутся.
var solMints = map[string]money.Currency{
	"Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB": money.USDT,
	"EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v": coinUSDC,
}

type solRPC[T any] struct {
	ID     int `json:"id"`
	Result *T  `json:"result"`
}

type solSignature struct {
	Signature string          `json:"signature"`
	BlockTime int64           `json:"blockTime"`
	Err       json.RawMessage `json:"err"`
}

type solTokenBalance struct {
	Owner string `json:"owner"`
	Mint  string `json:"mint"`
	UI    struct {
		Amount   string `json:"amount"`
		Decimals int    `json:"decimals"`
	} `json:"uiTokenAmount"`
}

type solTx struct {
	BlockTime   int64 `json:"blockTime"`
	Transaction struct {
		Message struct {
			Keys []string `json:"accountKeys"`
		} `json:"message"`
	} `json:"transaction"`
	Meta struct {
		Err      json.RawMessage   `json:"err"`
		Fee      int64             `json:"fee"`
		Pre      []int64           `json:"preBalances"`
		Post     []int64           `json:"postBalances"`
		PreToken []solTokenBalance `json:"preTokenBalances"`
		PstToken []solTokenBalance `json:"postTokenBalances"`
	} `json:"meta"`
}

func isNull(raw json.RawMessage) bool { return len(raw) == 0 || string(raw) == "null" }

func (c *Client) solanaCall(ctx context.Context, body any, out any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	return c.postJSON(ctx, c.sol, bytes.NewReader(b), out)
}

func (c *Client) solana(ctx context.Context, address string, since time.Time) ([]Transfer, error) {
	var sigs solRPC[[]solSignature]
	err := c.solanaCall(ctx, map[string]any{
		"jsonrpc": "2.0", "id": 1, "method": "getSignaturesForAddress",
		"params": []any{address, map[string]any{"limit": solSignatures}},
	}, &sigs)
	if err != nil {
		return nil, err
	}
	if sigs.Result == nil {
		return nil, ErrUnavailable
	}

	var batch []map[string]any
	for i, s := range *sigs.Result {
		if !isNull(s.Err) || s.BlockTime < since.Unix() {
			continue
		}
		batch = append(batch, map[string]any{
			"jsonrpc": "2.0", "id": i, "method": "getTransaction",
			"params": []any{s.Signature, map[string]any{"encoding": "json", "maxSupportedTransactionVersion": 0}},
		})
	}
	if len(batch) == 0 {
		return nil, nil
	}
	var txs []solRPC[solTx]
	if err := c.solanaCall(ctx, batch, &txs); err != nil {
		return nil, err
	}

	var out []Transfer
	for _, r := range txs {
		if r.Result == nil || r.ID < 0 || r.ID >= len(*sigs.Result) || !isNull(r.Result.Meta.Err) {
			continue // операции без результата источник иногда ещё не отдаёт, они подтянутся при следующем запросе
		}
		out = append(out, solTransfers((*sigs.Result)[r.ID].Signature, address, *r.Result)...)
	}
	return sortNewest(out), nil
}

// solTransfers считает, как операция изменила остатки адреса: SOL по балансам аккаунта, токены по их балансам.
// Комиссию платит первый аккаунт, она возвращается в сумму, чтобы переводом считалось только отправленное.
func solTransfers(sig, address string, t solTx) []Transfer {
	at := time.Unix(t.BlockTime, 0)
	var out []Transfer

	for i, key := range t.Transaction.Message.Keys {
		if key != address || i >= len(t.Meta.Pre) || i >= len(t.Meta.Post) {
			continue
		}
		delta := t.Meta.Post[i] - t.Meta.Pre[i]
		if i == 0 {
			delta += t.Meta.Fee
		}
		if minor, ok := toMinor(big.NewInt(abs64(delta)), 9, coinSOL); ok {
			if delta < 0 {
				minor = -minor
			}
			out = append(out, Transfer{ID: "sol:" + sig + ":SOL", At: at, Currency: coinSOL, AmountMinor: minor})
		}
	}

	for mint, cur := range solMints {
		delta, decimals := new(big.Int), 0
		for _, b := range t.Meta.PstToken {
			if b.Owner == address && b.Mint == mint {
				if v, ok := parseBig(b.UI.Amount); ok {
					delta.Add(delta, v)
					decimals = b.UI.Decimals
				}
			}
		}
		for _, b := range t.Meta.PreToken {
			if b.Owner == address && b.Mint == mint {
				if v, ok := parseBig(b.UI.Amount); ok {
					delta.Sub(delta, v)
					decimals = b.UI.Decimals
				}
			}
		}
		if minor, ok := toMinor(new(big.Int).Abs(delta), decimals, cur); ok {
			if delta.Sign() < 0 {
				minor = -minor
			}
			out = append(out, Transfer{ID: "sol:" + sig + ":" + string(cur), At: at, Currency: cur, AmountMinor: minor})
		}
	}
	return out
}

func abs64(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}
