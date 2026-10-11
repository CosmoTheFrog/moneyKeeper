//go:build integration

package web

import (
	"encoding/json"
	"fmt"
	"io"
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
		repo.NewCategories(db), repo.NewTransactions(db))
	srv := httptest.NewServer(newTestHandler(svc).routes())
	t.Cleanup(srv.Close)

	fresh := testNow.Add(-time.Minute)
	igor := client{t, srv.URL, "tma " + signedInitData(testToken, 42, "Igor", fresh)}
	anna := client{t, srv.URL, "tma " + signedInitData(testToken, 7, "Anna", fresh)}

	// Счета: основной с остатком и наличные без остатка.
	var acc apidto.AccountDTO
	igor.ok("POST", "/api/accounts", `{"name":"TBC","kind":"debit","currency":"GEL","opening_balance_minor":100000}`, 201, &acc)
	if acc.ID != 1 || acc.BalanceMinor != 100000 {
		t.Fatalf("account = %+v", acc)
	}
	igor.ok("POST", "/api/accounts", `{"name":"Наличные","kind":"cash","currency":"GEL"}`, 201, &acc)
	igor.fails("POST", "/api/accounts", `{"name":"X","kind":"cash","currency":"GEL","opening_balance_minor":-1}`, 400, apidto.CodeInvalidAccount)

	var cats apidto.CategoriesResponse
	igor.ok("GET", "/api/categories", "", 200, &cats)
	if len(cats.Categories) != 15 || cats.Categories[4].Name != "Продукты" || cats.Categories[4].Group != "daily" {
		t.Fatalf("categories = %+v", cats.Categories)
	}
	food := cats.Categories[4].ID

	t.Run("categories", func(t *testing.T) {
		var c apidto.CategoryDTO
		igor.ok("POST", "/api/categories", `{"name":"Питомцы","group":"daily"}`, 201, &c)
		igor.fails("POST", "/api/categories", `{"name":"Питомцы","group":"daily"}`, 400, apidto.CodeInvalidCategory)
		igor.ok("PATCH", fmt.Sprintf("/api/categories/%d", c.ID), `{"hidden":true,"description":"Корм"}`, 200, &c)
		if !c.Hidden || c.Description != "Корм" || c.Name != "Питомцы" {
			t.Errorf("updated = %+v", c)
		}
		igor.fails("PATCH", "/api/categories/abc", `{}`, 400, apidto.CodeInvalidRequest)
		anna.fails("PATCH", fmt.Sprintf("/api/categories/%d", c.ID), `{"hidden":false}`, 404, apidto.CodeNotFound)
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

	// 100000 остаток + 20000 доход - 3000 перевод; расход удалён. На наличных 3000.
	var accounts apidto.AccountsResponse
	igor.ok("GET", "/api/accounts", "", 200, &accounts)
	if accounts.Accounts[0].BalanceMinor != 117000 || accounts.Accounts[1].BalanceMinor != 3000 {
		t.Errorf("balances = %d and %d, want 117000 and 3000", accounts.Accounts[0].BalanceMinor, accounts.Accounts[1].BalanceMinor)
	}
}
