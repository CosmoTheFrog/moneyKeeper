package tgauth

import (
	"encoding/hex"
	"errors"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"moneykeeper/internal/models"
)

const testToken = "123456:test-token"

var testNow = time.Unix(1_800_000_000, 0)

// signed собирает initData так, как это делает Telegram.
func signed(token string, fields map[string]string) string {
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	lines := make([]string, 0, len(keys))
	for _, k := range keys {
		lines = append(lines, k+"="+fields[k])
	}
	hash := hex.EncodeToString(sign(sign([]byte(secretSalt), []byte(token)), []byte(strings.Join(lines, "\n"))))

	q := url.Values{}
	for k, v := range fields {
		q.Set(k, v)
	}
	q.Set(hashField, hash)
	return q.Encode()
}

func fields(authDate time.Time, user string) map[string]string {
	return map[string]string{
		"auth_date": strconv.FormatInt(authDate.Unix(), 10),
		"query_id":  "AAH",
		"user":      user,
	}
}

func TestValidate(t *testing.T) {
	const user = `{"id":42,"first_name":"Igor"}`
	tamper := func(s string) string { return strings.Replace(s, "Igor", "Evil", 1) }

	tests := []struct {
		name     string
		initData string
		want     models.TelegramUser
		wantErr  error
	}{
		{
			name:     "valid",
			initData: signed(testToken, fields(testNow.Add(-time.Minute), user)),
			want:     models.TelegramUser{ID: 42, FirstName: "Igor"},
		},
		{
			name:     "signed with another token",
			initData: signed("999:other", fields(testNow, user)),
			wantErr:  models.ErrInitDataInvalid,
		},
		{
			name:     "payload changed after signing",
			initData: tamper(signed(testToken, fields(testNow, user))),
			wantErr:  models.ErrInitDataInvalid,
		},
		{
			name:     "no hash",
			initData: "auth_date=1&user=%7B%7D",
			wantErr:  models.ErrInitDataInvalid,
		},
		{
			name:     "empty",
			initData: "",
			wantErr:  models.ErrInitDataInvalid,
		},
		{
			name:     "expired",
			initData: signed(testToken, fields(testNow.Add(-25*time.Hour), user)),
			wantErr:  models.ErrInitDataExpired,
		},
		{
			name:     "auth_date not a number",
			initData: signed(testToken, map[string]string{"auth_date": "soon", "user": user}),
			wantErr:  models.ErrInitDataInvalid,
		},
		{
			name:     "user is not json",
			initData: signed(testToken, fields(testNow, "Igor")),
			wantErr:  models.ErrInitDataInvalid,
		},
		{
			name:     "user without id",
			initData: signed(testToken, fields(testNow, `{"first_name":"Igor"}`)),
			wantErr:  models.ErrInitDataInvalid,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Validate(tt.initData, testToken, 24*time.Hour, testNow)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("user = %+v, want %+v", got, tt.want)
			}
		})
	}
}
