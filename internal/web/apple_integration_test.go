//go:build integration

package web

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"moneykeeper/internal/repo"
	"moneykeeper/internal/service"
	"moneykeeper/internal/testdb"
	"moneykeeper/internal/web/apidto"
)

// fakeApple признаёт один токен и один nonce, остальное отклоняет.
type fakeApple struct{ token, nonce, sub string }

func (f fakeApple) Verify(_ context.Context, token, nonce string) (string, error) {
	if token != f.token || nonce != f.nonce {
		return "", errors.New("rejected")
	}
	return f.sub, nil
}

func TestAppleSignInFlow(t *testing.T) {
	db := testdb.New(t)
	svc := service.New(db, repo.NewUsers(db), repo.NewBudgets(db), repo.NewAccounts(db),
		repo.NewCategories(db), repo.NewTransactions(db), repo.NewPlan(db), repo.NewSchedules(db), repo.NewRates(db), repo.NewAssets(db), &testdb.Rates{}, slog.New(slog.DiscardHandler))
	svc.WithSessions(repo.NewSessions(db))
	h := newTestHandler(svc)
	h.apple = fakeApple{token: "good-token", nonce: "n1", sub: "001.apple"}
	srv := httptest.NewServer(h.routes())
	t.Cleanup(srv.Close)

	do := func(method, path, auth, body string) (int, string) {
		t.Helper()
		req, err := http.NewRequest(method, srv.URL+path, strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		if auth != "" {
			req.Header.Set("Authorization", auth)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, string(b)
	}

	if code, _ := do("POST", "/api/auth/apple", "", `{"identity_token":"bad","nonce":"n1","timezone":"Asia/Tbilisi"}`); code != 401 {
		t.Errorf("rejected token: status %d, want 401", code)
	}
	if code, _ := do("POST", "/api/auth/apple", "", `{"identity_token":"good-token","nonce":"other"}`); code != 401 {
		t.Errorf("wrong nonce: status %d, want 401", code)
	}
	if code, body := do("POST", "/api/auth/apple", "", `{"identity_token":"good-token","nonce":"n1","timezone":"Nowhere/Land"}`); code != 400 || !strings.Contains(body, apidto.CodeInvalidTimezone) {
		t.Errorf("bad timezone: %d %s", code, body)
	}

	code, body := do("POST", "/api/auth/apple", "", `{"identity_token":"good-token","nonce":"n1","first_name":"Анна","timezone":"Asia/Tbilisi"}`)
	var session apidto.SessionResponse
	if code != 200 || json.Unmarshal([]byte(body), &session) != nil || session.Token == "" || session.ExpiresAt == "" {
		t.Fatalf("sign in: %d %s", code, body)
	}
	bearer := "Bearer " + session.Token

	if code, body := do("GET", "/api/me", bearer, ""); code != 200 || !strings.Contains(body, `"first_name":"Анна"`) || !strings.Contains(body, `"timezone":"Asia/Tbilisi"`) {
		t.Errorf("me with session: %d %s", code, body)
	}
	if code, _ := do("GET", "/api/categories", bearer, ""); code != 200 {
		t.Errorf("categories with session: status %d", code)
	}
	if code, _ := do("GET", "/api/me", "Bearer wrong", ""); code != 401 {
		t.Errorf("wrong token: status %d, want 401", code)
	}

	if code, _ := do("DELETE", "/api/auth/session", "tma whatever", ""); code != 401 {
		t.Errorf("sign out without a session token: status %d, want 401", code)
	}
	if code, _ := do("DELETE", "/api/auth/session", bearer, ""); code != 204 {
		t.Errorf("sign out: status %d, want 204", code)
	}
	if code, _ := do("GET", "/api/me", bearer, ""); code != 401 {
		t.Errorf("after sign out: status %d, want 401", code)
	}
}

func TestAppleSignInUnavailable(t *testing.T) {
	srv := httptest.NewServer(newTestHandler(nil).routes())
	t.Cleanup(srv.Close)
	resp, err := http.Post(srv.URL+"/api/auth/apple", "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 503 {
		t.Errorf("status %d, want 503 without an Apple verifier", resp.StatusCode)
	}
}
