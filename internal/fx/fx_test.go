package fx

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"moneykeeper/pkg/apptime"
	"moneykeeper/pkg/money"
)

// Ответ НБГ: цена за quantity единиц, у рубля и многих валют quantity не 1.
const nbgBody = `[{"date":"2026-10-10T00:00:00.000Z","currencies":[
  {"code":"AMD","quantity":1000,"rate":7.1720,"name":"Armenian Dram"},
  {"code":"EUR","quantity":1,"rate":3.1594,"name":"Euro"},
  {"code":"RUB","quantity":100,"rate":2.99,"name":"Russian Ruble"},
  {"code":"USD","quantity":1,"rate":2.7084,"name":"US Dollar"}]}]`

// Ответ Binance klines: [время, open, high, low, close, ...], цена закрытия пятым элементом строкой.
const klineBody = `[[1760227200000,"1.14270000","1.15920000","1.14220000","1.15730000","34278522.90000000",1760313599999,"39441090.99","116574","1","1","0"]]`

func newProvider(t *testing.T, nbg, binance http.HandlerFunc) *Provider {
	t.Helper()
	n := httptest.NewServer(nbg)
	b := httptest.NewServer(binance)
	t.Cleanup(n.Close)
	t.Cleanup(b.Close)
	return &Provider{nbg: n.URL, binance: b.URL, client: n.Client()}
}

func ok(body string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, body) }
}

func fail(status int) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(status) }
}

func TestGelPerUnit(t *testing.T) {
	var gotNBG, gotBinance string
	p := newProvider(t,
		func(w http.ResponseWriter, r *http.Request) { gotNBG = r.URL.RawQuery; fmt.Fprint(w, nbgBody) },
		func(w http.ResponseWriter, r *http.Request) { gotBinance = r.URL.RawQuery; fmt.Fprint(w, klineBody) },
	)
	day, _ := apptime.ParseDay("2026-10-12")

	got, err := p.GelPerUnit(context.Background(), day)
	if err != nil {
		t.Fatal(err)
	}
	want := map[money.Currency]string{
		money.USD:  "6771/2500",  // 2,7084
		money.EUR:  "15797/5000", // 3,1594
		money.RUB:  "299/10000",  // 2,99 за 100 = 0,0299
		money.USDT: "",           // выводится ниже
	}
	for cur, w := range want {
		if cur == money.USDT {
			continue
		}
		if got[cur] == nil || got[cur].String() != w {
			t.Errorf("%s = %v, want %s", cur, got[cur], w)
		}
	}
	// USDT = (₾ за €) / (USDT за €) = 3,1594 / 1,1573 ≈ 2,7300
	usdt, _ := got[money.USDT].Float64()
	if usdt < 2.729 || usdt > 2.731 {
		t.Errorf("USDT = %v, want about 2.730", usdt)
	}
	if !strings.Contains(gotNBG, "date=2026-10-12") {
		t.Errorf("NBG query %q has no date", gotNBG)
	}
	if !strings.Contains(gotBinance, "symbol=EURUSDT") || !strings.Contains(gotBinance, "startTime=1791763200000") || !strings.Contains(gotBinance, "interval=1d") {
		t.Errorf("Binance query %q", gotBinance)
	}
}

func TestGelPerUnitKeepsWhatItGotWhenBinanceFails(t *testing.T) {
	p := newProvider(t, ok(nbgBody), fail(http.StatusServiceUnavailable))
	day, _ := apptime.ParseDay("2026-10-12")

	got, err := p.GelPerUnit(context.Background(), day)
	if err == nil || !strings.Contains(err.Error(), "binance") {
		t.Fatalf("err = %v, want a binance error", err)
	}
	if got[money.EUR] == nil || got[money.USD] == nil || got[money.RUB] == nil {
		t.Errorf("fiat rates lost: %v", got)
	}
	if got[money.USDT] != nil {
		t.Errorf("USDT = %v, want none", got[money.USDT])
	}
}

func TestGelPerUnitErrors(t *testing.T) {
	day, _ := apptime.ParseDay("2026-10-12")
	tests := []struct {
		name    string
		nbg     http.HandlerFunc
		binance http.HandlerFunc
		wantIn  string
	}{
		{"nbg down", fail(http.StatusBadGateway), ok(klineBody), "nbg"},
		{"nbg without a needed currency", ok(`[{"currencies":[{"code":"EUR","quantity":1,"rate":3.1}]}]`), ok(klineBody), "no rate for USD"},
		{"nbg bad rate", ok(`[{"currencies":[{"code":"USD","quantity":1,"rate":0}]}]`), ok(klineBody), "bad rate"},
		{"binance empty", ok(nbgBody), ok(`[]`), "no candle"},
		{"binance bad price", ok(nbgBody), ok(`[[1,"1","1","1","oops"]]`), "bad price"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newProvider(t, tt.nbg, tt.binance)
			_, err := p.GelPerUnit(context.Background(), day)
			if err == nil || !strings.Contains(err.Error(), tt.wantIn) {
				t.Errorf("err = %v, want it to mention %q", err, tt.wantIn)
			}
		})
	}
}
