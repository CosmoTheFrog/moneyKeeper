//go:build integration

package service

import (
	"context"
	"errors"
	"testing"

	"moneykeeper/internal/models"
	"moneykeeper/internal/repo"
	"moneykeeper/pkg/money"
)

func TestAddAsset(t *testing.T) {
	ctx := context.Background()
	s, db := newService(t)

	tests := []struct {
		name    string
		code    string
		stock   bool
		want    money.Info
		wantErr error
	}{
		{"new coin gets precision from price", " pepe ", false, money.Info{Code: "PEPE", Units: 0, Coin: true}, nil},
		{"new stock", "COIN", true, money.Info{Code: "COIN", Units: 4, Coin: true, Stock: true}, nil},
		{"known code comes back as is", "btc", false, money.Info{Code: "BTC", Units: 8, Coin: true}, nil},
		{"unknown at the source", "NOSUCH", false, money.Info{}, models.ErrAssetNotFound},
		{"not a ticker", "a b;", false, money.Info{}, money.ErrBadCode},
		{"fiat code is not an asset", "USD", false, money.Info{Code: "USD", Units: 2}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := s.AddAsset(ctx, tt.code, tt.stock)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if err == nil && got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}

	// Сохранённое доступно после перезапуска: оно лежит в таблице, а не только в памяти.
	saved, err := repo.NewAssets(db).List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(saved) != 2 || saved[0].Code != "PEPE" || !saved[1].Stock {
		t.Errorf("saved = %+v", saved)
	}
	if err := s.LoadAssets(ctx); err != nil {
		t.Errorf("load twice must be harmless: %v", err)
	}
}