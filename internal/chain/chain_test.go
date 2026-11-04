package chain

import (
	"context"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"moneykeeper/pkg/money"
)

func TestToMinor(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		decimals int
		cur      money.Currency
		want     int64
		ok       bool
	}{
		{"eth 0.5 rounds to 8 units", "500000000000000000", 18, coinETH, 50_000_000, true},
		{"eth rounds half up", "15000000000", 18, coinETH, 2, true},
		{"eth rounds down below half", "14999999999", 18, coinETH, 1, true},
		{"eth dust is dropped", "1000", 18, coinETH, 0, false},
		{"usdt 6 decimals to 2 units", "10000000", 6, money.USDT, 1000, true},
		{"usdc 2 units from 6 decimals", "503153870", 6, coinUSDC, 50315, true},
		{"trx from sun", "1000000", 6, coinTRX, 10_000, true},
		{"zero", "0", 6, coinTRX, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw, _ := new(big.Int).SetString(tt.raw, 10)
			got, ok := toMinor(raw, tt.decimals, tt.cur)
			if ok != tt.ok || got != tt.want {
				t.Errorf("toMinor = %d, %v; want %d, %v", got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestValid(t *testing.T) {
	tests := []struct {
		name    string
		chain   Chain
		address string
		want    bool
	}{
		{"eth", Ethereum, "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045", true},
		{"eth short", Ethereum, "0xd8dA6BF2", false},
		{"eth no prefix", Ethereum, "d8dA6BF26964aF9D7eEd9e03E53415D37aA96045", false},
		{"tron usdt contract", Tron, "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t", true},
		{"tron wrong checksum", Tron, "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6u", false},
		{"tron eth address", Tron, "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045", false},
		{"sol usdt mint", Solana, "Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB", true},
		{"sol too short", Solana, "Es9vMFrzaCER", false},
		{"unknown chain", Chain("btc"), "x", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Valid(tt.chain, tt.address); got != tt.want {
				t.Errorf("Valid(%s, %q) = %v, want %v", tt.chain, tt.address, got, tt.want)
			}
		})
	}
}

func testClient(url string) *Client {
	c := NewClient()
	c.ethURL, c.tronURL, c.sol = url, url, url
	return c
}

var longAgo = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

func TestEthereumTransfers(t *testing.T) {
	const me = "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/transactions"):
			fmt.Fprint(w, `{"items":[
				{"hash":"0xaa","timestamp":"2026-11-01T10:00:00.000000Z","value":"500000000000000000","status":"ok","from":{"hash":"0x2a82CB3176234da35B05D98aB8fA87Afdbc70A06"},"to":{"hash":"0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045"}},
				{"hash":"0xbb","timestamp":"2026-11-02T10:00:00.000000Z","value":"250000000000000000","status":"ok","from":{"hash":"0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045"},"to":{"hash":"0x2a82CB3176234da35B05D98aB8fA87Afdbc70A06"}},
				{"hash":"0xcc","timestamp":"2026-11-02T11:00:00.000000Z","value":"1000000000000000000","status":"error","from":{"hash":"0x2a82CB3176234da35B05D98aB8fA87Afdbc70A06"},"to":{"hash":"0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045"}},
				{"hash":"0xdd","timestamp":"2026-11-02T12:00:00.000000Z","value":"0","status":"ok","from":{"hash":"0x2a82CB3176234da35B05D98aB8fA87Afdbc70A06"},"to":{"hash":"0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045"}}
			],"next_page_params":null}`)
		case strings.HasSuffix(r.URL.Path, "/token-transfers"):
			fmt.Fprint(w, `{"items":[
				{"transaction_hash":"0xee","log_index":3,"timestamp":"2026-11-03T10:00:00.000000Z","from":{"hash":"0x2a82CB3176234da35B05D98aB8fA87Afdbc70A06"},"to":{"hash":"0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045"},"token":{"address_hash":"0xdAC17F958D2ee523a2206206994597C13D831ec7"},"total":{"value":"12500000","decimals":"6"}},
				{"transaction_hash":"0xff","log_index":1,"timestamp":"2026-11-03T11:00:00.000000Z","from":{"hash":"0x2a82CB3176234da35B05D98aB8fA87Afdbc70A06"},"to":{"hash":"0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045"},"token":{"address_hash":"0x64a22D8Eef40434CCCbbdA49008680edd94C8772"},"total":{"value":"99","decimals":"9"}}
			],"next_page_params":null}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	got, err := testClient(srv.URL).Transfers(context.Background(), Ethereum, me, longAgo)
	if err != nil {
		t.Fatal(err)
	}
	// Ошибочная операция, нулевая и токен-подделка отброшены; новые первыми.
	want := []struct {
		id     string
		amount int64
		cur    money.Currency
	}{
		{"eth:0xee:USDT:3", 1250, money.USDT},
		{"eth:0xbb:ETH", -25_000_000, coinETH},
		{"eth:0xaa:ETH", 50_000_000, coinETH},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d transfers %+v, want %d", len(got), got, len(want))
	}
	for i, w := range want {
		if got[i].ID != w.id || got[i].AmountMinor != w.amount || got[i].Currency != w.cur {
			t.Errorf("transfer %d = %+v, want %+v", i, got[i], w)
		}
	}
}

func TestTronTransfers(t *testing.T) {
	const me = "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"
	hexMe := fmt.Sprintf("%x", mustDecode(t, me)[:21])
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/trc20"):
			fmt.Fprintf(w, `{"data":[
				{"transaction_id":"t20","block_timestamp":1793804274000,"from":"TSUiKpHxhZhWN7CQjWoaQw3podEQcG8XBs","to":%q,"value":"10000000","type":"Transfer","token_info":{"address":"TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t","decimals":6}},
				{"transaction_id":"fake","block_timestamp":1793804274000,"from":"TSUiKpHxhZhWN7CQjWoaQw3podEQcG8XBs","to":%q,"value":"10000000","type":"Transfer","token_info":{"address":"TFake","decimals":6}}
			]}`, me, me)
		case strings.HasSuffix(r.URL.Path, "/transactions"):
			fmt.Fprintf(w, `{"data":[
				{"txID":"tx1","block_timestamp":1793804931000,"ret":[{"contractRet":"SUCCESS"}],"raw_data":{"contract":[{"type":"TransferContract","parameter":{"value":{"amount":2500000,"owner_address":%q,"to_address":"414916105a1b64ce5d7ce9c6a9c2a20fe2ff5f9122"}}}]}},
				{"txID":"tx2","block_timestamp":1793804931000,"ret":[{"contractRet":"REVERT"}],"raw_data":{"contract":[{"type":"TransferContract","parameter":{"value":{"amount":1,"owner_address":%q,"to_address":"414916105a1b64ce5d7ce9c6a9c2a20fe2ff5f9122"}}}]}},
				{"txID":"tx3","block_timestamp":1793804931000,"ret":[{"contractRet":"SUCCESS"}],"raw_data":{"contract":[{"type":"TriggerSmartContract","parameter":{"value":{}}}]}}
			]}`, hexMe, hexMe)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	got, err := testClient(srv.URL).Transfers(context.Background(), Tron, me, longAgo)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d transfers %+v, want 2", len(got), got)
	}
	byID := map[string]Transfer{}
	for _, tr := range got {
		byID[tr.ID] = tr
	}
	if tr := byID["tron:tx1:TRX"]; tr.AmountMinor != -25_000 || tr.Currency != coinTRX {
		t.Errorf("trx transfer = %+v, want -25000 TRX", tr)
	}
	if tr := byID["tron:t20:USDT"]; tr.AmountMinor != 1000 || tr.Currency != money.USDT {
		t.Errorf("usdt transfer = %+v, want 1000 USDT", tr)
	}
}

func TestSolanaTransfers(t *testing.T) {
	const me = "4raqd4tCEnm2yGeUocVqPgN2MPB1W6A3EZfC8LoMChvf"
	const mint = "Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB"
	now := time.Now().Unix()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if strings.HasPrefix(string(body), "[") {
			fmt.Fprintf(w, `[
				{"id":0,"result":{"blockTime":%d,"transaction":{"message":{"accountKeys":[%q,"x"]}},"meta":{"err":null,"fee":5000,"preBalances":[3000000000,1],"postBalances":[1999995000,1],"preTokenBalances":[],"postTokenBalances":[]}}},
				{"id":1,"result":{"blockTime":%d,"transaction":{"message":{"accountKeys":["z",%q]}},"meta":{"err":null,"fee":5000,"preBalances":[1,0],"postBalances":[1,0],"preTokenBalances":[{"owner":%q,"mint":%q,"uiTokenAmount":{"amount":"1000000","decimals":6}}],"postTokenBalances":[{"owner":%q,"mint":%q,"uiTokenAmount":{"amount":"3500000","decimals":6}}]}}},
				{"id":2,"result":null}
			]`, now, me, now-10, me, me, mint, me, mint)
			return
		}
		fmt.Fprintf(w, `{"id":1,"result":[
			{"signature":"sigA","blockTime":%d,"err":null},
			{"signature":"sigB","blockTime":%d,"err":null},
			{"signature":"sigC","blockTime":%d,"err":null},
			{"signature":"sigFail","blockTime":%d,"err":{"InstructionError":[0,"x"]}}
		]}`, now, now-10, now-20, now-30)
	}))
	defer srv.Close()

	got, err := testClient(srv.URL).Transfers(context.Background(), Solana, me, longAgo)
	if err != nil {
		t.Fatal(err)
	}
	// Первая операция: баланс упал на 1_000_005_000, из них 5000 комиссия, значит ушла ровно 1 SOL.
	byID := map[string]Transfer{}
	for _, tr := range got {
		byID[tr.ID] = tr
	}
	if len(got) != 2 {
		t.Fatalf("got %d transfers %+v, want 2", len(got), got)
	}
	if tr := byID["sol:sigA:SOL"]; tr.AmountMinor != -1_000_000 || tr.Currency != coinSOL {
		t.Errorf("sol transfer = %+v, want -1000000 SOL", tr)
	}
	if tr := byID["sol:sigB:USDT"]; tr.AmountMinor != 250 || tr.Currency != money.USDT {
		t.Errorf("usdt transfer = %+v, want +250 USDT", tr)
	}
}

func TestTransfersErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "limit", http.StatusTooManyRequests)
	}))
	defer srv.Close()
	c := testClient(srv.URL)

	if _, err := c.Transfers(context.Background(), Ethereum, "0x1", longAgo); !IsBadAddress(err) {
		t.Errorf("bad address error = %v, want IsBadAddress", err)
	}
	_, err := c.Transfers(context.Background(), Tron, "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t", longAgo)
	if err == nil || !strings.Contains(err.Error(), "unavailable") {
		t.Errorf("source error = %v, want unavailable", err)
	}
}

func TestUnreachableSourceHidesAddress(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	url := srv.URL
	srv.Close() // порт закрыт: запрос не дойдёт

	const me = "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"
	_, err := testClient(url).Transfers(context.Background(), Tron, me, longAgo)
	if err == nil || !strings.Contains(err.Error(), "unavailable") {
		t.Fatalf("err = %v, want unavailable", err)
	}
	if strings.Contains(err.Error(), me) {
		t.Errorf("error leaks the address: %v", err)
	}
}

func mustDecode(t *testing.T, s string) []byte {
	t.Helper()
	b, ok := decodeBase58(s)
	if !ok {
		t.Fatalf("decode %q", s)
	}
	return b
}
