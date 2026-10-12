//go:build integration

package service

import (
	"context"
	"errors"
	"testing"

	"moneykeeper/internal/models"
)

// group находит id стартовой группы по ключу.
func (l ledger) group(t *testing.T, key models.GroupKey) int64 {
	t.Helper()
	groups, err := l.s.ListCategoryGroups(context.Background(), l.u)
	if err != nil {
		t.Fatal(err)
	}
	for _, g := range groups {
		if g.Key == key {
			return g.ID
		}
	}
	t.Fatalf("group %q not found", key)
	return 0
}

func TestCategoryGroupManagement(t *testing.T) {
	ctx := context.Background()
	l := newLedger(t)

	created, err := l.s.CreateCategoryGroup(ctx, l.u, "  Путешествия ")
	if err != nil || created.Name != "Путешествия" || created.Key != "" || created.Position != 6 {
		t.Fatalf("create = %+v, %v", created, err)
	}
	// Новая группа идёт в конец списка.
	groups, err := l.s.ListCategoryGroups(ctx, l.u)
	if err != nil || len(groups) != 6 || groups[5].ID != created.ID {
		t.Fatalf("groups = %+v, %v", groups, err)
	}

	tests := []struct {
		name  string
		input string
	}{
		{"duplicate of the new group", "Путешествия"},
		{"duplicate of a default group", "Долги"},
		{"empty", "   "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := l.s.CreateCategoryGroup(ctx, l.u, tt.input); !errors.Is(err, models.ErrInvalidCategory) {
				t.Errorf("err = %v, want ErrInvalidCategory", err)
			}
		})
	}

	// Категория в своей группе создаётся и привязана к ней.
	cat, err := l.s.CreateCategory(ctx, l.u, "Авиабилеты", created.ID)
	if err != nil || cat.GroupID != created.ID {
		t.Fatalf("category in the new group = %+v, %v", cat, err)
	}
}

func TestCategoryCannotJoinForeignGroup(t *testing.T) {
	ctx := context.Background()
	l := newLedger(t)

	other, err := l.s.Resolve(ctx, models.TelegramUser{ID: 7, FirstName: "Anna"}, "Asia/Tbilisi")
	if err != nil {
		t.Fatal(err)
	}
	foreign, err := l.s.CreateCategoryGroup(ctx, other, "Чужая")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := l.s.CreateCategory(ctx, l.u, "Х", foreign.ID); !errors.Is(err, models.ErrInvalidCategory) {
		t.Errorf("foreign group: err = %v, want ErrInvalidCategory", err)
	}
}

func TestSameGroupNameInDifferentBudgets(t *testing.T) {
	ctx := context.Background()
	l := newLedger(t)

	other, err := l.s.Resolve(ctx, models.TelegramUser{ID: 7, FirstName: "Anna"}, "Asia/Tbilisi")
	if err != nil {
		t.Fatal(err)
	}
	for _, u := range []models.User{l.u, other} {
		if _, err := l.s.CreateCategoryGroup(ctx, u, "Путешествия"); err != nil {
			t.Errorf("user %d: %v", u.ID, err)
		}
	}
}
