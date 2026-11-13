//go:build integration

package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"moneykeeper/internal/models"
	"moneykeeper/internal/repo"
)

func TestAppleUserAndSession(t *testing.T) {
	ctx := context.Background()
	s, db := newService(t)
	s.WithSessions(repo.NewSessions(db))
	now := time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return now }

	u, err := s.ResolveApple(ctx, "001.apple", "Анна", "Asia/Tbilisi")
	if err != nil || u.TelegramID != 0 || u.FirstName != "Анна" || u.ActiveBudgetID == 0 {
		t.Fatalf("new apple user = %+v, %v", u, err)
	}
	if cats, _ := s.ListCategories(ctx, u); len(cats) != 24 {
		t.Errorf("categories = %d, want the 24 seeded ones", len(cats))
	}
	// Повторный вход возвращает того же пользователя и ничего не создаёт.
	again, err := s.ResolveApple(ctx, "001.apple", "Другое имя", "UTC")
	if err != nil || again.ID != u.ID || again.FirstName != "Анна" {
		t.Errorf("repeat sign in = %+v, %v", again, err)
	}
	if _, err := s.ResolveApple(ctx, "002.apple", "X", "Not/AZone"); !errors.Is(err, models.ErrInvalidTimezone) {
		t.Errorf("bad timezone: err = %v, want ErrInvalidTimezone", err)
	}
	// Пользователь Telegram и пользователь Apple это разные люди.
	tg, err := s.Resolve(ctx, tgIgor, "Asia/Tbilisi")
	if err != nil || tg.ID == u.ID {
		t.Errorf("telegram user = %+v, %v", tg, err)
	}

	token, expires, err := s.StartSession(ctx, u)
	if err != nil || token == "" || !expires.Equal(now.Add(sessionTTL)) {
		t.Fatalf("start session = %q %v, %v", token, expires, err)
	}
	got, err := s.UserBySession(ctx, token)
	if err != nil || got.ID != u.ID {
		t.Fatalf("session user = %+v, %v", got, err)
	}
	if _, err := s.UserBySession(ctx, token+"x"); !errors.Is(err, models.ErrNotFound) {
		t.Errorf("wrong token: err = %v, want ErrNotFound", err)
	}

	// Просроченная сессия не принимается.
	s.now = func() time.Time { return now.Add(sessionTTL + time.Second) }
	if _, err := s.UserBySession(ctx, token); !errors.Is(err, models.ErrNotFound) {
		t.Errorf("expired: err = %v, want ErrNotFound", err)
	}
	s.now = func() time.Time { return now }

	if err := s.EndSession(ctx, token); err != nil {
		t.Fatal(err)
	}
	if _, err := s.UserBySession(ctx, token); !errors.Is(err, models.ErrNotFound) {
		t.Errorf("after sign out: err = %v, want ErrNotFound", err)
	}
}

func TestSessionsDisabled(t *testing.T) {
	s, _ := newService(t)
	if _, _, err := s.StartSession(context.Background(), models.User{ID: 1}); !errors.Is(err, models.ErrAuthUnavailable) {
		t.Errorf("err = %v, want ErrAuthUnavailable", err)
	}
}
