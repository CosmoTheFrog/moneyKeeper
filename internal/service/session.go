package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"moneykeeper/internal/models"
	"moneykeeper/internal/repo"
)

// sessionTTL это срок жизни сессии нативного приложения. Токен выдаётся при входе через Apple.
const sessionTTL = 90 * 24 * time.Hour

// WithSessions включает сессии нативного приложения. Без них вход через Apple отвечает ErrAuthUnavailable.
func (s *Service) WithSessions(r *repo.Sessions) *Service {
	s.sessions = r
	return s
}

// ResolveApple находит пользователя по идентификатору Apple (поле sub токена). При первом входе создаёт пользователя,
// личный бюджет и стартовые категории одной транзакцией, как Resolve. Имя и часовой пояс сохраняются только при создании.
func (s *Service) ResolveApple(ctx context.Context, sub, firstName, timezone string) (models.User, error) {
	u, err := s.users.GetByAppleSub(ctx, sub)
	if err == nil {
		return u, nil
	}
	if !errors.Is(err, models.ErrNotFound) {
		return models.User{}, err
	}
	if _, err := time.LoadLocation(timezone); err != nil {
		return models.User{}, fmt.Errorf("%q: %w", timezone, models.ErrInvalidTimezone)
	}

	err = s.db.InTx(ctx, func(ctx context.Context) error {
		created, err := s.users.CreateAppleIfAbsent(ctx, sub, firstName, timezone)
		if errors.Is(err, models.ErrNotFound) {
			return nil // параллельный запрос успел создать пользователя
		}
		if err != nil {
			return err
		}
		return s.provision(ctx, created.ID)
	})
	if err != nil {
		return models.User{}, err
	}
	return s.users.GetByAppleSub(ctx, sub)
}

// StartSession выдаёт приложению новый токен сессии. В базе остаётся только хэш токена.
func (s *Service) StartSession(ctx context.Context, u models.User) (string, time.Time, error) {
	if s.sessions == nil {
		return "", time.Time{}, models.ErrAuthUnavailable
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", time.Time{}, fmt.Errorf("session token: %w", err)
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	expires := s.now().Add(sessionTTL)
	if err := s.sessions.Create(ctx, u.ID, tokenHash(token), expires); err != nil {
		return "", time.Time{}, err
	}
	return token, expires, nil
}

// UserBySession находит владельца действующей сессии. Нет такой или она истекла: ErrNotFound.
func (s *Service) UserBySession(ctx context.Context, token string) (models.User, error) {
	if s.sessions == nil {
		return models.User{}, models.ErrNotFound
	}
	return s.sessions.UserByHash(ctx, tokenHash(token), s.now())
}

// EndSession завершает сессию: токен больше не принимается.
func (s *Service) EndSession(ctx context.Context, token string) error {
	if s.sessions == nil {
		return nil
	}
	return s.sessions.Delete(ctx, tokenHash(token))
}

func tokenHash(token string) []byte {
	h := sha256.Sum256([]byte(token))
	return h[:]
}
