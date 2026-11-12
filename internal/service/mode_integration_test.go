//go:build integration

package service

import (
	"context"
	"errors"
	"testing"

	"moneykeeper/internal/models"
)

func TestInterfaceMode(t *testing.T) {
	ctx := context.Background()
	s, _ := newService(t)
	u, err := s.Resolve(ctx, tgIgor, "Asia/Tbilisi")
	if err != nil {
		t.Fatal(err)
	}
	if b, _ := s.Budget(ctx, u); b.InterfaceMode != "" {
		t.Fatalf("new budget mode = %q, want not chosen", b.InterfaceMode)
	}
	// Без базовой валюты режим не выбрать: простому режиму нечем завести счёт.
	if _, err := s.SetInterfaceMode(ctx, u, "simple"); !errors.Is(err, models.ErrInvalidAccount) {
		t.Errorf("no base currency: err = %v, want ErrInvalidAccount", err)
	}
	if _, err := s.SetBaseCurrency(ctx, u, "GEL"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SetInterfaceMode(ctx, u, "weird"); !errors.Is(err, models.ErrInvalidAccount) {
		t.Errorf("unknown mode: err = %v, want ErrInvalidAccount", err)
	}

	b, err := s.SetInterfaceMode(ctx, u, "simple")
	if err != nil || b.InterfaceMode != models.ModeSimple {
		t.Fatalf("simple = %+v, %v", b, err)
	}
	accs, _ := s.ListAccounts(ctx, u, "")
	if len(accs) != 1 || !accs[0].IsDefault || accs[0].Currency != "GEL" || accs[0].Name != "Основной" {
		t.Fatalf("accounts = %+v", accs)
	}

	// Повторный выбор и возврат в полный режим счёт не дублируют и не удаляют.
	for _, m := range []string{"simple", "full", "simple"} {
		if _, err := s.SetInterfaceMode(ctx, u, m); err != nil {
			t.Fatal(err)
		}
	}
	if accs, _ := s.ListAccounts(ctx, u, ""); len(accs) != 1 {
		t.Errorf("accounts after switching = %d, want 1", len(accs))
	}
}
