//go:build integration

package web

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"moneykeeper/internal/repo"
	"moneykeeper/internal/service"
	"moneykeeper/internal/testdb"
	"moneykeeper/internal/web/apidto"
)

func TestFlow(t *testing.T) {
	db := testdb.New(t)
	svc := service.New(db, repo.NewUsers(db), repo.NewBudgets(db), repo.NewAccounts(db),
		repo.NewCategories(db), repo.NewTransactions(db), repo.NewPlan(db), repo.NewSchedules(db), repo.NewRates(db), &testdb.Rates{}, slog.New(slog.DiscardHandler))
	srv := httptest.NewServer(newTestHandler(svc).routes())
	t.Cleanup(srv.Close)

	auth := "tma " + signedInitData(testToken, 42, "Igor", testNow.Add(-time.Minute))

	do := func(method, path, body string) (int, string) {
		t.Helper()
		req, err := http.NewRequest(method, srv.URL+path, strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Authorization", auth)
		req.Header.Set("X-Timezone", "Asia/Tbilisi")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, string(b)
	}

	steps := []struct {
		name       string
		method     string
		path       string
		body       string
		wantStatus int
		wantBody   string
	}{
		{"first login creates user and budget", "GET", "/api/me", "", 200,
			`{"id":1,"first_name":"Igor","timezone":"Asia/Tbilisi","budget":{"id":1,"name":"","base_currency":null},"base_currencies":["GEL","EUR","USD","RUB"],"currencies":["GEL","EUR","USD","RUB","USDT","BTC",…`},
		{"invalid base currency", "PUT", "/api/budget/currency", `{"currency":"USDT"}`, 400, `{"code":"invalid_currency"}`},
		{"set base currency", "PUT", "/api/budget/currency", `{"currency":"GEL"}`, 200, `{"id":1,"name":"","base_currency":"GEL"}`},
		{"base currency is set once", "PUT", "/api/budget/currency", `{"currency":"EUR"}`, 409, `{"code":"base_currency_set"}`},
		{"unknown field is rejected", "PUT", "/api/budget/currency", `{"currency":"GEL","x":1}`, 400, `{"code":"invalid_request"}`},
		{"no accounts yet", "GET", "/api/accounts", "", 200, `{"accounts":[]}`},
		{"create first account", "POST", "/api/accounts", `{"name":"TBC","kind":"debit","currency":"GEL"}`, 201,
			`{"id":1,"name":"TBC","kind":"debit","currency":"GEL","multi_currency":false,"is_default":true,"off_budget":false,"archived":false,"target_minor":null,"target_date":null,"category_id":null,"balances":[{"currency":"GEL","balance_minor":0,"spent_minor":0,"base_balance_minor":0,"base_spent_minor":0}]}`},
		{"invalid account kind", "POST", "/api/accounts", `{"name":"X","kind":"bank","currency":"GEL"}`, 400, `{"code":"invalid_account"}`},
		{"invalid account currency", "POST", "/api/accounts", `{"name":"X","kind":"cash","currency":"XXX"}`, 400, `{"code":"invalid_currency"}`},
		{"list accounts", "GET", "/api/accounts", "", 200,
			`{"accounts":[{"id":1,"name":"TBC","kind":"debit","currency":"GEL","multi_currency":false,"is_default":true,"off_budget":false,"archived":false,"target_minor":null,"target_date":null,"category_id":null,"balances":[{"currency":"GEL","balance_minor":0,"spent_minor":0,"base_balance_minor":0,"base_spent_minor":0}]}]}`},
	}
	t.Run("currency precision is part of /api/me", func(t *testing.T) {
		_, body := do("GET", "/api/me", "")
		var me apidto.MeResponse
		if err := json.Unmarshal([]byte(body), &me); err != nil {
			t.Fatal(err)
		}
		units := map[string]int{}
		coin := map[string]bool{}
		for _, c := range me.CurrencyInfo {
			units[c.Code], coin[c.Code] = c.Units, c.Coin
		}
		if units["GEL"] != 2 || units["BTC"] != 8 || units["SHIB"] != 0 || coin["GEL"] || coin["USDT"] || !coin["BTC"] || len(me.CurrencyInfo) != len(me.Currencies) {
			t.Errorf("currency info = %+v", me.CurrencyInfo)
		}
	})

	for _, s := range steps {
		t.Run(s.name, func(t *testing.T) {
			status, body := do(s.method, s.path, s.body)
			if status != s.wantStatus {
				t.Errorf("status = %d, want %d (body %s)", status, s.wantStatus, body)
			}
			got := strings.TrimSpace(body)
			// Окончание «…» значит, что ответ должен лишь начинаться с wantBody: длинные списки проверяются отдельно.
			if head, ok := strings.CutSuffix(s.wantBody, "…"); ok && strings.HasPrefix(got, head) {
				return
			}
			if got != s.wantBody {
				t.Errorf("body = %s\nwant   %s", strings.TrimSpace(body), s.wantBody)
			}
		})
	}
}
