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

func newTestHandler(svc Service) *handler {
	return &handler{svc: svc, botToken: testToken, log: slog.New(slog.NewTextHandler(io.Discard, nil)), now: func() time.Time { return testNow }}
}

// До сервиса запрос не доходит: проверка подписи отсекает его раньше.
func TestAuthRejected(t *testing.T) {
	srv := httptest.NewServer(newTestHandler(nil).routes())
	t.Cleanup(srv.Close)

	valid := signedInitData(testToken, 42, "Igor", testNow.Add(-time.Minute))
	tests := []struct {
		name       string
		authHeader string
	}{
		{"no header", ""},
		{"other scheme", "Bearer " + valid},
		{"signed with another token", "tma " + signedInitData("999:other", 42, "Igor", testNow)},
		{"expired", "tma " + signedInitData(testToken, 42, "Igor", testNow.Add(-25*time.Hour))},
	}
	for _, path := range []string{"/api/me", "/api/accounts"} {
		for _, tt := range tests {
			t.Run(path+" "+tt.name, func(t *testing.T) {
				req, err := http.NewRequest(http.MethodGet, srv.URL+path, nil)
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
				if resp.StatusCode != http.StatusUnauthorized {
					t.Errorf("status = %d, want 401", resp.StatusCode)
				}
				if want := `{"code":"unauthorized"}` + "\n"; string(body) != want {
					t.Errorf("body = %q, want %q", body, want)
				}
			})
		}
	}
}

func hmacSHA256(key, msg []byte) []byte {
	m := hmac.New(sha256.New, key)
	m.Write(msg)
	return m.Sum(nil)
}

// signedInitData подписывает так же, как Telegram; алгоритм сверяется в тестах tgauth.
func signedInitData(token string, userID int64, firstName string, authDate time.Time) string {
	authDateStr := strconv.FormatInt(authDate.Unix(), 10)
	user := `{"id":` + strconv.FormatInt(userID, 10) + `,"first_name":"` + firstName + `"}`
	data := "auth_date=" + authDateStr + "\nuser=" + user
	hash := hex.EncodeToString(hmacSHA256(hmacSHA256([]byte("WebAppData"), []byte(token)), []byte(data)))

	q := url.Values{}
	q.Set("auth_date", authDateStr)
	q.Set("user", user)
	q.Set("hash", hash)
	return q.Encode()
}
