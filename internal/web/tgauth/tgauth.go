// Package tgauth проверяет подпись initData Telegram Mini App.
// Алгоритм: https://core.telegram.org/bots/webapps#validating-data-received-via-the-mini-app
package tgauth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"moneykeeper/internal/models"
)

const (
	hashField     = "hash"
	authDateField = "auth_date"
	userField     = "user"
	secretSalt    = "WebAppData"
)

type initDataUser struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
}

func Validate(initData, botToken string, maxAge time.Duration, now time.Time) (models.TelegramUser, error) {
	values, err := url.ParseQuery(initData)
	if err != nil {
		return models.TelegramUser{}, fmt.Errorf("parse init data: %w", models.ErrInitDataInvalid)
	}
	got := values.Get(hashField)
	if got == "" {
		return models.TelegramUser{}, fmt.Errorf("no hash: %w", models.ErrInitDataInvalid)
	}
	values.Del(hashField)

	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	lines := make([]string, 0, len(keys))
	for _, k := range keys {
		lines = append(lines, k+"="+values.Get(k))
	}

	secret := sign([]byte(secretSalt), []byte(botToken))
	want := hex.EncodeToString(sign(secret, []byte(strings.Join(lines, "\n"))))
	if !hmac.Equal([]byte(got), []byte(want)) {
		return models.TelegramUser{}, fmt.Errorf("hash mismatch: %w", models.ErrInitDataInvalid)
	}

	authDate, err := strconv.ParseInt(values.Get(authDateField), 10, 64)
	if err != nil {
		return models.TelegramUser{}, fmt.Errorf("auth_date: %w", models.ErrInitDataInvalid)
	}
	if now.Sub(time.Unix(authDate, 0)) > maxAge {
		return models.TelegramUser{}, fmt.Errorf("auth_date %d: %w", authDate, models.ErrInitDataExpired)
	}

	var u initDataUser
	if err := json.Unmarshal([]byte(values.Get(userField)), &u); err != nil || u.ID == 0 {
		return models.TelegramUser{}, fmt.Errorf("user: %w", models.ErrInitDataInvalid)
	}
	return models.TelegramUser{ID: u.ID, FirstName: u.FirstName}, nil
}

func sign(key, msg []byte) []byte {
	m := hmac.New(sha256.New, key)
	m.Write(msg)
	return m.Sum(nil)
}
