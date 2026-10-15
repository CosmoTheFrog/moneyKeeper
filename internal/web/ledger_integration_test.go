//go:build integration

package web

import (
	"encoding/json"
	"fmt"
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

type client struct {
	t    *testing.T
	base string
	auth string
}

func (c client) do(method, path, body string) (int, []byte) {
	c.t.Helper()
	req, err := http.NewRequest(method, c.base+path, strings.NewReader(body))
	if err != nil {
		c.t.Fatal(err)
	}
	req.Header.Set("Authorization", c.auth)
	req.Header.Set("X-Timezone", "Asia/Tbilisi")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, b
}

// ok выполняет запрос, проверяет статус и разбирает JSON в dst (если он нужен).
func (c client) ok(method, path, body string, wantStatus int, dst any) {
	c.t.Helper()
	status, b := c.do(method, path, body)
	if status != wantStatus {
		c.t.Fatalf("%s %s: status %d, want %d (body %s)", method, path, status, wantStatus, b)
	}
	if dst != nil {
		if err := json.Unmarshal(b, dst); err != nil {
			c.t.Fatalf("%s %s: decode %s: %v", method, path, b, err)
		}
	}
}

func (c client) fails(method, path, body string, wantStatus int, wantCode string) {
	c.t.Helper()
	var e apidto.ErrorResponse
	c.ok(method, path, body, wantStatus, &e)
	if e.Code != wantCode {
		c.t.Errorf("%s %s: code %q, want %q", method, path, e.Code, wantCode)
	}
}

func TestLedgerFlow(t *testing.T) {
	db := testdb.New(t)
	svc := service.New(db, repo.NewUsers(db), repo.NewBudgets(db), repo.NewAccounts(db),
		repo.NewCategories(db), repo.NewTransactions(db), repo.NewPlan(db), repo.NewRates(db), &testdb.Rates{}, slog.New(slog.DiscardHandler))
	srv := httptest.NewServer(newTestHandler(svc).routes())
	t.Cleanup(srv.Close)

	fresh := testNow.Add(-time.Minute)
	igor := client{t, srv.URL, "tma " + signedInitData(testToken, 42, "Igor", fresh)}
	anna := client{t, srv.URL, "tma " + signedInitData(testToken, 7, "Anna", fresh)}

	// Счета: основной с остатком и наличные без остатка.
	var acc apidto.AccountDTO
	igor.ok("POST", "/api/accounts", `{"name":"TBC","kind":"debit","currency":"GEL","opening_balance_minor":100000}`, 201, &acc)
	if acc.ID != 1 || len(acc.Balances) != 1 || acc.Balances[0].BalanceMinor != 100000 {
		t.Fatalf("account = %+v", acc)
	}
	igor.ok("POST", "/api/accounts", `{"name":"Наличные","kind":"cash","currency":"GEL"}`, 201, &acc)
	igor.fails("POST", "/api/accounts", `{"name":"X","kind":"cash","currency":"GEL","opening_balance_minor":-1}`, 400, apidto.CodeInvalidAccount)

	var groups apidto.CategoryGroupsResponse
	igor.ok("GET", "/api/category-groups", "", 200, &groups)
	if len(groups.Groups) != 5 || groups.Groups[0].Name != "Критичные" || groups.Groups[2].Name != "Ежедневные" {
		t.Fatalf("groups = %+v", groups.Groups)
	}

	var cats apidto.CategoriesResponse
	igor.ok("GET", "/api/categories", "", 200, &cats)
	if len(cats.Categories) != 15 || cats.Categories[4].Name != "Продукты" || cats.Categories[4].GroupID != groups.Groups[2].ID {
		t.Fatalf("categories = %+v", cats.Categories)
	}
	food := cats.Categories[4].ID

	t.Run("categories", func(t *testing.T) {
		daily := groups.Groups[2].ID
		var c apidto.CategoryDTO
		igor.ok("POST", "/api/categories", fmt.Sprintf(`{"name":"Питомцы","group_id":%d}`, daily), 201, &c)
		igor.fails("POST", "/api/categories", fmt.Sprintf(`{"name":"Питомцы","group_id":%d}`, daily), 400, apidto.CodeInvalidCategory)
		igor.fails("POST", "/api/categories", `{"name":"Х","group_id":999999}`, 400, apidto.CodeInvalidCategory)
		igor.ok("PATCH", fmt.Sprintf("/api/categories/%d", c.ID), `{"hidden":true,"description":"Корм"}`, 200, &c)
		if !c.Hidden || c.Description != "Корм" || c.Name != "Питомцы" {
			t.Errorf("updated = %+v", c)
		}
		igor.fails("PATCH", "/api/categories/abc", `{}`, 400, apidto.CodeInvalidRequest)
		anna.fails("PATCH", fmt.Sprintf("/api/categories/%d", c.ID), `{"hidden":false}`, 404, apidto.CodeNotFound)
	})

	t.Run("category groups", func(t *testing.T) {
		var g apidto.CategoryGroupDTO
		igor.ok("POST", "/api/category-groups", `{"name":"Путешествия"}`, 201, &g)
		if g.ID == 0 || g.Name != "Путешествия" {
			t.Fatalf("created = %+v", g)
		}
		igor.fails("POST", "/api/category-groups", `{"name":"Путешествия"}`, 400, apidto.CodeInvalidCategory)
		igor.fails("POST", "/api/category-groups", `{"name":""}`, 400, apidto.CodeInvalidCategory)

		var list apidto.CategoryGroupsResponse
		igor.ok("GET", "/api/category-groups", "", 200, &list)
		if len(list.Groups) != 6 || list.Groups[5].ID != g.ID {
			t.Errorf("groups = %+v", list.Groups)
		}
		// Чужая группа недоступна для категорий.
		anna.fails("POST", "/api/categories", fmt.Sprintf(`{"name":"Х","group_id":%d}`, g.ID), 400, apidto.CodeInvalidCategory)
	})

	t.Run("rates and display currency", func(t *testing.T) {
		var r apidto.RatesResponse
		igor.ok("GET", "/api/rates?in=USD", "", 200, &r)
		if r.In != "USD" || r.Day == "" || len(r.Rates) == 0 {
			t.Fatalf("rates = %+v", r)
		}
		for _, q := range r.Rates {
			if q.Currency == "USD" || q.Rate == "" {
				t.Errorf("bad quote %+v", q)
			}
		}
		igor.fails("GET", "/api/rates?in=XXX", "", 400, apidto.CodeInvalidCurrency)
		igor.fails("GET", "/api/accounts?in=XXX", "", 400, apidto.CodeInvalidCurrency)
		igor.fails("GET", "/api/transactions?in=XXX", "", 400, apidto.CodeInvalidCurrency)
		igor.ok("GET", "/api/accounts?in=USDT", "", 200, nil)
	})

	t.Run("budget", func(t *testing.T) {
		igor.do("PUT", "/api/budget/currency", `{"currency":"GEL"}`) // могла быть выбрана раньше в этом тесте

		// Месяц берём у последней записи: начальный остаток счёта создан «сегодня».
		var list apidto.TransactionsResponse
		igor.ok("GET", "/api/transactions?limit=1", "", 200, &list)
		m := list.Transactions[0].Date[:7]

		var b apidto.BudgetMonthResponse
		igor.ok("GET", "/api/budget/months/"+m, "", 200, &b)
		if b.Month != m || b.Currency != "GEL" || b.IncomeMinor < 100000 || len(b.Categories) < 15 || b.BudgetedMinor != 0 {
			t.Fatalf("empty month = %+v", b)
		}
		before := b.ToBudgetMinor

		igor.ok("PUT", fmt.Sprintf("/api/budget/months/%s/categories/%d", m, food), `{"budgeted_minor":30000}`, 200, &b)
		var row apidto.BudgetRowDTO
		for _, r := range b.Categories {
			if r.CategoryID == food {
				row = r
			}
		}
		if row.BudgetedMinor != 30000 || row.BalanceMinor < 30000-row.SpentMinor || b.BudgetedMinor != 30000 || b.ToBudgetMinor != before-30000 {
			t.Errorf("after setting: row %+v, month %+v (was %d)", row, b, before)
		}

		igor.fails("PUT", fmt.Sprintf("/api/budget/months/%s/categories/%d", m, food), `{"budgeted_minor":-1}`, 400, apidto.CodeInvalidBudget)
		igor.fails("GET", "/api/budget/months/2026-13", "", 400, apidto.CodeInvalidDate)
		igor.fails("PUT", fmt.Sprintf("/api/budget/months/%s/categories/%d", m, food), `{"budgeted_minor":1,"x":1}`, 400, apidto.CodeInvalidRequest)
		anna.fails("PUT", fmt.Sprintf("/api/budget/months/%s/categories/%d", m, food), `{"budgeted_minor":100}`, 404, apidto.CodeNotFound)

		var c apidto.CategoryDTO
		igor.ok("PATCH", fmt.Sprintf("/api/categories/%d", food), `{"rollover":true}`, 200, &c)
		if !c.Rollover {
			t.Errorf("rollover not set: %+v", c)
		}
	})

	t.Run("exchange and portfolio", func(t *testing.T) {
		dana := client{t, srv.URL, "tma " + signedInitData(testToken, 9, "Dana", fresh)} // отдельный пользователь, чтобы не менять записи других проверок
		dana.do("PUT", "/api/budget/currency", `{"currency":"GEL"}`)
		var w apidto.AccountDTO
		dana.ok("POST", "/api/accounts", `{"name":"Binance","kind":"wallet","currency":"USDT","multi_currency":true,"opening_balance_minor":100000}`, 201, &w)

		body := fmt.Sprintf(`{"from_account_id":%d,"to_account_id":%d,"out_minor":8000,"out_currency":"USDT","in_minor":100000,"in_currency":"BTC","fee_minor":50,"note":"покупка"}`, w.ID, w.ID)
		var rows apidto.TransactionsResponse
		dana.ok("POST", "/api/exchanges", body, 201, &rows)
		if len(rows.Transactions) != 3 || rows.Transactions[0].Kind != "exchange" || rows.Transactions[0].AmountMinor != -8000 ||
			rows.Transactions[1].Currency != "BTC" || rows.Transactions[2].Kind != "fee" || rows.Transactions[0].TransferID == nil {
			t.Fatalf("exchange = %+v", rows.Transactions)
		}

		dana.fails("POST", "/api/exchanges", fmt.Sprintf(`{"from_account_id":%d,"to_account_id":%d,"out_minor":100,"out_currency":"USDT","in_minor":100,"in_currency":"USDT"}`, w.ID, w.ID), 400, apidto.CodeInvalidTransaction)
		dana.fails("POST", "/api/exchanges", `{"from_account_id":1,"to_account_id":1,"x":1}`, 400, apidto.CodeInvalidRequest)
		anna.fails("POST", "/api/exchanges", body, 404, apidto.CodeNotFound)

		var p apidto.PortfolioResponse
		dana.ok("GET", "/api/portfolio", "", 200, &p)
		if p.Currency != "GEL" || len(p.Items) != 1 || p.Items[0].Currency != "BTC" || p.Items[0].QuantityMinor != 100000 || p.Items[0].ValueMinor != 25000 || p.Items[0].ProfitBP == nil {
			t.Errorf("portfolio = %+v", p)
		}
		dana.fails("GET", "/api/portfolio?in=XXX", "", 400, apidto.CodeInvalidCurrency)
	})

	t.Run("quick input", func(t *testing.T) {
		var p apidto.ParseResponse
		igor.ok("POST", "/api/transactions/parse", `{"text":"кофе 5 EUR"}`, 200, &p)
		if p.AmountMinor != 500 || p.Currency != "EUR" || p.Note != "кофе" {
			t.Errorf("parse = %+v", p)
		}
		igor.ok("POST", "/api/transactions/parse", `{"text":"такси 12,5"}`, 200, &p)
		if p.AmountMinor != 1250 || p.Currency != "" {
			t.Errorf("parse without currency = %+v", p)
		}
		igor.fails("POST", "/api/transactions/parse", `{"text":"кофе"}`, 400, apidto.CodeNoAmount)
		igor.fails("POST", "/api/transactions/parse", `{"text":"кофе 5.555"}`, 400, apidto.CodeInvalidAmount)
	})

	var expense, income apidto.TransactionDTO
	var list apidto.TransactionsResponse

	igor.ok("POST", "/api/transactions",
		fmt.Sprintf(`{"kind":"expense","account_id":1,"category_id":%d,"amount_minor":5000,"date":"2026-10-01","note":"молоко"}`, food), 201, &list)
	expense = list.Transactions[0]
	if expense.AmountMinor != -5000 || expense.Date != "2026-10-01" || expense.Kind != "expense" || expense.Currency != "GEL" {
		t.Fatalf("expense = %+v", expense)
	}
	igor.ok("POST", "/api/transactions", `{"kind":"income","account_id":1,"amount_minor":20000}`, 201, &list)
	income = list.Transactions[0]
	if income.AmountMinor != 20000 || income.Date == "" || income.CategoryID != nil {
		t.Fatalf("income = %+v", income)
	}

	t.Run("create errors", func(t *testing.T) {
		igor.fails("POST", "/api/transactions", `{"kind":"income","account_id":1,"amount_minor":100,"date":"11.10.2026"}`, 400, apidto.CodeInvalidDate)
		igor.fails("POST", "/api/transactions", `{"kind":"expense","account_id":1,"amount_minor":100}`, 400, apidto.CodeInvalidTransaction)
		igor.fails("POST", "/api/transactions", `{"kind":"income","account_id":99,"amount_minor":100}`, 404, apidto.CodeNotFound)
		igor.fails("POST", "/api/transactions", `{"kind":"income","account_id":1,"amount_minor":100,"extra":1}`, 400, apidto.CodeInvalidRequest)
		anna.fails("POST", "/api/transactions", `{"kind":"income","account_id":1,"amount_minor":100}`, 404, apidto.CodeNotFound)
	})

	igor.ok("POST", "/api/transactions", `{"kind":"transfer","account_id":1,"to_account_id":2,"amount_minor":3000}`, 201, &list)
	if len(list.Transactions) != 2 || list.Transactions[0].AmountMinor != -3000 || list.Transactions[1].AmountMinor != 3000 ||
		list.Transactions[0].TransferID == nil || *list.Transactions[0].TransferID != *list.Transactions[1].TransferID {
		t.Fatalf("transfer = %+v", list.Transactions)
	}

	t.Run("list", func(t *testing.T) {
		igor.ok("GET", "/api/transactions", "", 200, &list)
		if len(list.Transactions) != 5 { // остаток, расход, доход и две ноги перевода
			t.Errorf("got %d records, want 5", len(list.Transactions))
		}
		igor.ok("GET", "/api/transactions?limit=2&offset=1", "", 200, &list)
		if len(list.Transactions) != 2 {
			t.Errorf("page has %d records, want 2", len(list.Transactions))
		}
		igor.fails("GET", "/api/transactions?limit=abc", "", 400, apidto.CodeInvalidRequest)
		anna.ok("GET", "/api/transactions", "", 200, &list)
		if len(list.Transactions) != 0 {
			t.Errorf("other user sees %d records", len(list.Transactions))
		}
	})

	t.Run("update and history", func(t *testing.T) {
		path := fmt.Sprintf("/api/transactions/%d", expense.ID)
		igor.ok("PATCH", path, `{"amount_minor":7000,"note":"молоко и хлеб"}`, 200, &list)
		if list.Transactions[0].AmountMinor != -7000 || list.Transactions[0].Note != "молоко и хлеб" {
			t.Errorf("updated = %+v", list.Transactions[0])
		}
		igor.fails("PATCH", path, `{"amount_minor":0}`, 400, apidto.CodeInvalidTransaction)
		igor.fails("PATCH", path, `{"date":"вчера"}`, 400, apidto.CodeInvalidDate)
		anna.fails("PATCH", path, `{"note":"чужая"}`, 404, apidto.CodeNotFound)

		var ev apidto.TransactionEventsResponse
		igor.ok("GET", path+"/events", "", 200, &ev)
		if len(ev.Events) != 2 || ev.Events[0].Action != "create" || ev.Events[1].Action != "update" ||
			ev.Events[1].Before.AmountMinor != -5000 || ev.Events[1].After.AmountMinor != -7000 || ev.Events[1].After.Date != "2026-10-01" {
			t.Errorf("events = %+v", ev.Events)
		}
		anna.fails("GET", path+"/events", "", 404, apidto.CodeNotFound)
	})

	t.Run("delete", func(t *testing.T) {
		path := fmt.Sprintf("/api/transactions/%d", expense.ID)
		anna.fails("DELETE", path, "", 404, apidto.CodeNotFound)
		igor.ok("DELETE", path, "", 204, nil)
		igor.fails("DELETE", path, "", 404, apidto.CodeNotFound)
		igor.fails("PATCH", path, `{"note":"поздно"}`, 404, apidto.CodeNotFound)
	})

	t.Run("multi-currency account", func(t *testing.T) {
		var wallet apidto.AccountDTO
		igor.ok("POST", "/api/accounts", `{"name":"Binance","kind":"cash","currency":"USDT","multi_currency":true}`, 201, &wallet)
		if !wallet.MultiCurrency || wallet.Currency != "USDT" {
			t.Fatalf("wallet = %+v", wallet)
		}
		path := fmt.Sprintf(`{"kind":"income","account_id":%d,"amount_minor":25000,"currency":"EUR"}`, wallet.ID)
		igor.ok("POST", "/api/transactions", path, 201, &list)
		if list.Transactions[0].Currency != "EUR" || list.Transactions[0].AccountID != wallet.ID {
			t.Errorf("record = %+v", list.Transactions[0])
		}
		igor.fails("POST", "/api/transactions", `{"kind":"income","account_id":1,"amount_minor":100,"currency":"EUR"}`, 400, apidto.CodeCurrencyMismatch)
		igor.fails("POST", "/api/transactions", fmt.Sprintf(`{"kind":"income","account_id":%d,"amount_minor":100,"currency":"XXX"}`, wallet.ID), 400, apidto.CodeInvalidCurrency)

		var accs apidto.AccountsResponse
		igor.ok("GET", "/api/accounts", "", 200, &accs)
		got := accs.Accounts[len(accs.Accounts)-1]
		if got.ID != wallet.ID || len(got.Balances) != 2 || got.Balances[0].Currency != "USDT" || got.Balances[1].Currency != "EUR" || got.Balances[1].BalanceMinor != 25000 {
			t.Errorf("balances = %+v", got.Balances)
		}
	})

	// 100000 остаток + 20000 доход - 3000 перевод; расход удалён. На наличных 3000.
	var accounts apidto.AccountsResponse
	igor.ok("GET", "/api/accounts", "", 200, &accounts)
	if accounts.Accounts[0].Balances[0].BalanceMinor != 117000 || accounts.Accounts[1].Balances[0].BalanceMinor != 3000 {
		t.Errorf("balances = %d and %d, want 117000 and 3000", accounts.Accounts[0].Balances[0].BalanceMinor, accounts.Accounts[1].Balances[0].BalanceMinor)
	}
}
