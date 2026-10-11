package service

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"moneykeeper/internal/models"
)

const maxDescriptionRunes = 200

func (s *Service) ListCategories(ctx context.Context, u models.User) ([]models.Category, error) {
	return s.categories.List(ctx, u.ActiveBudgetID)
}

func (s *Service) CreateCategory(ctx context.Context, u models.User, name, group string) (models.Category, error) {
	name, err := validName(name)
	if err != nil {
		return models.Category{}, err
	}
	g := models.CategoryGroup(group)
	switch g {
	case models.GroupCritical, models.GroupDebt, models.GroupDaily, models.GroupRainy, models.GroupGoal:
	default:
		return models.Category{}, fmt.Errorf("group %q: %w", group, models.ErrInvalidCategory)
	}
	return s.categories.Create(ctx, u.ActiveBudgetID, models.NewCategory{Group: g, Name: name})
}

func (s *Service) UpdateCategory(ctx context.Context, u models.User, id int64, p models.CategoryPatch) (models.Category, error) {
	if p.Name != nil {
		name, err := validName(*p.Name)
		if err != nil {
			return models.Category{}, err
		}
		p.Name = &name
	}
	if p.Description != nil {
		d := strings.TrimSpace(*p.Description)
		if utf8.RuneCountInString(d) > maxDescriptionRunes {
			return models.Category{}, fmt.Errorf("description: %w", models.ErrInvalidCategory)
		}
		p.Description = &d
	}
	return s.categories.Update(ctx, u.ActiveBudgetID, id, p)
}

func validName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > maxNameRunes {
		return "", fmt.Errorf("name: %w", models.ErrInvalidCategory)
	}
	return name, nil
}
