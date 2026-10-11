package web

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"
)

const testToken = "123456:test-token"

var testNow = time.Unix(1_800_000_000, 0)

func TestMe(t *testing.T) {
	h := &handler{botToken: testToken, log: slog.New(slog.NewTextHandler(io.Discard, nil)), now: func() time.Time { return testNow }}
	srv := httptest.NewServer(h.routes())
	t.Cleanup(srv.Close)

	valid := signedInitData(testToken, testNow.Add(-time.Minute))
	tests := []struct {
		name       string
		authHeader string
		wantStatus int
		wantBody   string
	}{
		{"valid", "tma " + valid, http.StatusOK, `{"id":42,"first_name":"Igor"}` + "\n"},
		{"no header", "", http.StatusUnauthorized, `{"code":"unauthorized"}` + "\n"},
		{"other scheme", "Bearer " + valid, http.StatusUnauthorized, `{"code":"unauthorized"}` + "\n"},
		{"signed with another token", "tma " + signedInitData("999:other", testNow), http.StatusUnauthorized, `{"code":"unauthorized"}` + "\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, srv.URL+"/api/me", nil)
			if err != nil {
				t.Fatal(err)
			}
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			if resp.StatusCode != tt.wantStatus {
				t.Errorf("status = %d, want %d", resp.StatusCode, tt.wantStatus)
			}
			if string(body) != tt.wantBody {
				t.Errorf("body = %q, want %q", body, tt.wantBody)
			}
		})
	}
}

func hmacSHA256(key, msg []byte) []byte {
	m := hmac.New(sha256.New, key)
	m.Write(msg)
	return m.Sum(nil)
}

// signedInitData подписывает так же, как Telegram; алгоритм сверяется в тестах tgauth.
func signedInitData(token string, authDate time.Time) string {
	authDateStr := strconv.FormatInt(authDate.Unix(), 10)
	const user = `{"id":42,"first_name":"Igor"}`
	data := "auth_date=" + authDateStr + "\nuser=" + user
	hash := hex.EncodeToString(hmacSHA256(hmacSHA256([]byte("WebAppData"), []byte(token)), []byte(data)))

	q := url.Values{}
	q.Set("auth_date", authDateStr)
	q.Set("user", user)
	q.Set("hash", hash)
	return q.Encode()
}
