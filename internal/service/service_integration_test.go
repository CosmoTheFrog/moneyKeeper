//go:build integration

package service

import (
	"context"
	"errors"
	"sync"
	"testing"

	"moneykeeper/internal/models"
	"moneykeeper/internal/pgclient"
	"moneykeeper/internal/repo"
	"moneykeeper/internal/testdb"
	"moneykeeper/pkg/money"
)

func newService(t *testing.T) (*Service, *pgclient.Client) {
	t.Helper()
	db := testdb.New(t)
	return New(db, repo.NewUsers(db), repo.NewBudgets(db), repo.NewAccounts(db),
		repo.NewCategories(db), repo.NewTransactions(db)), db
}

var tgIgor = models.TelegramUser{ID: 42, FirstName: "Igor"}

func TestResolve(t *testing.T) {
	ctx := context.Background()
	s, _ := newService(t)

	first, err := s.Resolve(ctx, tgIgor, "Asia/Tbilisi")
	if err != nil {
		t.Fatal(err)
	}
	if first.ID == 0 || first.ActiveBudgetID == 0 || first.Timezone != "Asia/Tbilisi" || first.FirstName != "Igor" {
		t.Fatalf("unexpected user: %+v", first)
	}

	// Повторный вход возвращает того же пользователя и не меняет пояс.
	again, err := s.Resolve(ctx, tgIgor, "Europe/Berlin")
	if err != nil {
		t.Fatal(err)
	}
	if again != first {
		t.Errorf("second resolve = %+v, want %+v", again, first)
	}

	b, err := s.Budget(ctx, first)
	if err != nil {
		t.Fatal(err)
	}
	if b.ID != first.ActiveBudgetID || b.BaseCurrency != nil {
		t.Errorf("budget = %+v", b)
	}
}

func TestResolveInvalidTimezone(t *testing.T) {
	s, _ := newService(t)
	if _, err := s.Resolve(context.Background(), tgIgor, "Mars/Olympus"); !errors.Is(err, models.ErrInvalidTimezone) {
		t.Fatalf("err = %v, want ErrInvalidTimezone", err)
	}
}

func TestResolveParallelFirstLogin(t *testing.T) {
	ctx := context.Background()
	s, db := newService(t)

	var wg sync.WaitGroup
	errs := make(chan error, 8)
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := s.Resolve(ctx, tgIgor, "UTC")
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Errorf("resolve: %v", err)
		}
	}

	var users, budgets int
	if err := db.Q(ctx).QueryRow(ctx, `SELECT count(*) FROM users`).Scan(&users); err != nil {
		t.Fatal(err)
	}
	if err := db.Q(ctx).QueryRow(ctx, `SELECT count(*) FROM budgets`).Scan(&budgets); err != nil {
		t.Fatal(err)
	}
	if users != 1 || budgets != 1 {
		t.Errorf("users = %d, budgets = %d, want 1 and 1", users, budgets)
	}
}

func TestSetBaseCurrency(t *testing.T) {
	ctx := context.Background()
	s, _ := newService(t)
	u, err := s.Resolve(ctx, tgIgor, "UTC")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		code    string
		wantErr error
	}{
		{"unknown", "XXX", money.ErrUnknownCurrency},
		{"not a base currency", "USDT", money.ErrUnknownCurrency},
		{"ok", "GEL", nil},
		{"second time", "EUR", models.ErrBaseCurrencySet},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := s.SetBaseCurrency(ctx, u, tt.code)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if err == nil && (b.BaseCurrency == nil || *b.BaseCurrency != money.GEL) {
				t.Errorf("base currency = %v, want GEL", b.BaseCurrency)
			}
		})
	}

	b, err := s.Budget(ctx, u)
	if err != nil {
		t.Fatal(err)
	}
	if b.BaseCurrency == nil || *b.BaseCurrency != money.GEL {
		t.Errorf("stored base currency = %v, want GEL (second call must not overwrite)", b.BaseCurrency)
	}
}

func TestCreateAccount(t *testing.T) {
	ctx := context.Background()
	s, _ := newService(t)
	u, err := s.Resolve(ctx, tgIgor, "UTC")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		acc      [3]string
		wantErr  error
		wantMain bool
	}{
		{"first account becomes default", [3]string{"TBC", "debit", "GEL"}, nil, true},
		{"second account is not default", [3]string{"Наличные", "cash", "EUR"}, nil, false},
		{"trims name", [3]string{"  Карта  ", "credit", "GEL"}, nil, false},
		{"empty name", [3]string{"  ", "cash", "GEL"}, models.ErrInvalidAccount, false},
		{"long name", [3]string{"ааааааааааааааааааааааааааааааааааааааааа", "cash", "GEL"}, models.ErrInvalidAccount, false},
		{"unknown kind", [3]string{"X", "savings", "GEL"}, models.ErrInvalidAccount, false},
		{"unknown currency", [3]string{"X", "cash", "XXX"}, money.ErrUnknownCurrency, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, err := s.CreateAccount(ctx, u, tt.acc[0], tt.acc[1], tt.acc[2], 0)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if err == nil && a.IsDefault != tt.wantMain {
				t.Errorf("is_default = %v, want %v", a.IsDefault, tt.wantMain)
			}
		})
	}

	list, err := s.ListAccounts(ctx, u)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 3 || list[2].Name != "Карта" {
		t.Errorf("list = %+v, want 3 accounts, last trimmed", list)
	}
}

func TestAccountsAreScopedToBudget(t *testing.T) {
	ctx := context.Background()
	s, _ := newService(t)
	igor, err := s.Resolve(ctx, tgIgor, "UTC")
	if err != nil {
		t.Fatal(err)
	}
	other, err := s.Resolve(ctx, models.TelegramUser{ID: 7, FirstName: "Anna"}, "UTC")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateAccount(ctx, igor, "TBC", "debit", "GEL", 0); err != nil {
		t.Fatal(err)
	}
	list, err := s.ListAccounts(ctx, other)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Errorf("other user sees %d accounts, want 0", len(list))
	}
}
