package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"moneykeeper/internal/models"
	"moneykeeper/pkg/apptime"
	"moneykeeper/pkg/money"
)

// CreatePiggy заводит копилку: счёт вне бюджета с целью и конвертом в группе «Цели накопления». Если в бюджете уже есть
// категория с таким названием, копилка использует её, иначе категория создаётся. Копилка пополняется переводом со счёта
// бюджета в категорию этого конверта: деньги сразу уходят из конверта и появляются в копилке.
func (s *Service) CreatePiggy(ctx context.Context, u models.User, name, currency string, targetMinor *int64, targetDate *apptime.Day) (models.Account, error) {
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > maxNameRunes {
		return models.Account{}, fmt.Errorf("name: %w", models.ErrInvalidAccount)
	}
	c, err := money.Parse(currency)
	if err != nil {
		return models.Account{}, err
	}
	if targetMinor != nil && (*targetMinor <= 0 || *targetMinor > maxAmountMinor) {
		return models.Account{}, fmt.Errorf("target: %w", models.ErrInvalidAccount)
	}

	var acc models.Account
	err = s.db.InTx(ctx, func(ctx context.Context) error {
		groups, err := s.categories.ListGroups(ctx, u.ActiveBudgetID)
		if err != nil {
			return err
		}
		var goal int64
		for _, g := range groups {
			if g.Key == models.GroupGoal {
				goal = g.ID
			}
		}
		if goal == 0 {
			return fmt.Errorf("goal group: %w", models.ErrNotFound)
		}
		cats, err := s.categories.List(ctx, u.ActiveBudgetID)
		if err != nil {
			return err
		}
		var category *int64
		for _, cat := range cats {
			if strings.EqualFold(cat.Name, name) {
				category = &cat.ID
				break
			}
		}
		if category == nil {
			cat, err := s.categories.Create(ctx, u.ActiveBudgetID, models.NewCategory{GroupID: goal, Name: name})
			if err != nil {
				return err
			}
			category = &cat.ID
		}
		acc, err = s.accounts.Create(ctx, u.ActiveBudgetID, models.NewAccount{
			Name: name, Kind: models.AccountPiggy, Currency: c, TargetMinor: targetMinor, TargetDate: targetDate, CategoryID: category,
		})
		return err
	})
	if err != nil {
		if errors.Is(err, models.ErrInvalidCategory) {
			return models.Account{}, fmt.Errorf("piggy name: %w", models.ErrInvalidAccount)
		}
		return models.Account{}, err
	}
	return acc, nil
}
