package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"moneykeeper/internal/models"
)

const maxDescriptionRunes = 200

func (s *Service) ListCategories(ctx context.Context, u models.User) ([]models.Category, error) {
	return s.categories.List(ctx, u.ActiveBudgetID)
}

func (s *Service) ListCategoryGroups(ctx context.Context, u models.User) ([]models.CategoryGroup, error) {
	return s.categories.ListGroups(ctx, u.ActiveBudgetID)
}

func (s *Service) CreateCategoryGroup(ctx context.Context, u models.User, name string) (models.CategoryGroup, error) {
	name, err := validName(name)
	if err != nil {
		return models.CategoryGroup{}, err
	}
	return s.categories.CreateGroup(ctx, u.ActiveBudgetID, name)
}

func (s *Service) CreateCategory(ctx context.Context, u models.User, name string, groupID int64) (models.Category, error) {
	name, err := validName(name)
	if err != nil {
		return models.Category{}, err
	}
	// Группа должна быть из этого бюджета: внешний ключ этого не проверяет.
	if _, err := s.categories.GetGroup(ctx, u.ActiveBudgetID, groupID); err != nil {
		if errors.Is(err, models.ErrNotFound) {
			return models.Category{}, fmt.Errorf("group %d: %w", groupID, models.ErrInvalidCategory)
		}
		return models.Category{}, err
	}
	return s.categories.Create(ctx, u.ActiveBudgetID, models.NewCategory{GroupID: groupID, Name: name})
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
