//go:build integration

package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"moneykeeper/internal/chain"
	"moneykeeper/internal/models"
	"moneykeeper/internal/repo"
)

type fakeChain struct {
	transfers []chain.Transfer
	err       error
}

func (f fakeChain) Transfers(context.Context, chain.Chain, string, time.Time) ([]chain.Transfer, error) {
	return f.transfers, f.err
}

const tronAddress = "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"

func TestWalletImport(t *testing.T) {
	ctx := context.Background()
	s, db := newService(t)
	s.now = func() time.Time { return ledgerNow }
	u, err := s.Resolve(ctx, tgIgor, "Asia/Tbilisi")
	if err != nil {
		t.Fatal(err)
	}
	wallet, err := s.CreateAccount(ctx, u, "Ledger", "wallet", "USDT", false, 0)
	if err != nil {
		t.Fatal(err)
	}
	card, err := s.CreateAccount(ctx, u, "TBC", "debit", "GEL", false, 0)
	if err != nil {
		t.Fatal(err)
	}
	fake := fakeChain{transfers: []chain.Transfer{
		{ID: "tron:in1:USDT", At: ledgerNow, Currency: "USDT", AmountMinor: 5000, Counterparty: "TSUiKpHxhZhWN7CQjWoaQw3podEQcG8XBs"},
		{ID: "tron:out1:USDT", At: ledgerNow, Currency: "USDT", AmountMinor: -1000, Counterparty: "TBP2rNp6aZXJ7RUMqh21FpZGWXvyYUEfDL"},
		{ID: "tron:in2:TRX", At: ledgerNow, Currency: "TRX", AmountMinor: 20000},
		{ID: "tron:in3:DOGE", At: ledgerNow, Currency: "DOGE", AmountMinor: 100},
	}}

	t.Run("not configured", func(t *testing.T) {
		if _, err := s.AddWallet(ctx, u, wallet.ID, "tron", tronAddress, true); !errors.Is(err, models.ErrImportUnavailable) {
			t.Errorf("err = %v, want ErrImportUnavailable", err)
		}
	})

	s.WithWallets(repo.NewWallets(db), fake)

	t.Run("link checks", func(t *testing.T) {
		tests := []struct {
			name    string
			account int64
			chain   string
			address string
			consent bool
			want    error
		}{
			{"no consent", wallet.ID, "tron", tronAddress, false, models.ErrImportConsent},
			{"bad address", wallet.ID, "tron", "nope", true, models.ErrInvalidImport},
			{"unknown chain", wallet.ID, "btc", tronAddress, true, models.ErrInvalidImport},
			{"not a wallet account", card.ID, "tron", tronAddress, true, models.ErrInvalidImport},
			{"foreign account", 999999, "tron", tronAddress, true, models.ErrNotFound},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if _, err := s.AddWallet(ctx, u, tt.account, tt.chain, tt.address, tt.consent); !errors.Is(err, tt.want) {
					t.Errorf("err = %v, want %v", err, tt.want)
				}
			})
		}
	})

	w, err := s.AddWallet(ctx, u, wallet.ID, "tron", tronAddress, true)
	if err != nil {
		t.Fatal(err)
	}
	if again, err := s.AddWallet(ctx, u, wallet.ID, "tron", tronAddress, true); err != nil || again.ID != w.ID {
		t.Fatalf("second link = %+v, %v; want the same wallet %d", again, err, w.ID)
	}

	p, err := s.PreviewWallet(ctx, u, w.ID)
	if err != nil {
		t.Fatal(err)
	}
	// Счёт держит только USDT: TRX и DOGE считаются пропущенными.
	if len(p.Rows) != 2 || p.Skipped != 2 {
		t.Fatalf("preview = %d rows, %d skipped; want 2 and 2", len(p.Rows), p.Skipped)
	}
	if p.Rows[0].ExternalID != "tron:in1:USDT" || p.Rows[0].Duplicate {
		t.Errorf("first row = %+v", p.Rows[0])
	}

	// Расход без категории не сохраняется, и ничего не остаётся помеченным внесённым.
	if _, err := s.CommitImport(ctx, u, wallet.ID, p.Rows[:2]); !errors.Is(err, models.ErrInvalidTransaction) {
		t.Fatalf("expense without category: err = %v, want ErrInvalidTransaction", err)
	}
	if again, _ := s.PreviewWallet(ctx, u, w.ID); again.Rows[0].Duplicate {
		t.Fatal("failed commit must not mark transfers as imported")
	}

	n, err := s.CommitImport(ctx, u, wallet.ID, p.Rows[:1])
	if err != nil || n != 1 {
		t.Fatalf("commit = %d, %v; want 1", n, err)
	}
	n, err = s.CommitImport(ctx, u, wallet.ID, p.Rows[:1])
	if err != nil || n != 0 {
		t.Fatalf("repeat commit = %d, %v; want 0 saved", n, err)
	}
	again, _ := s.PreviewWallet(ctx, u, w.ID)
	if !again.Rows[0].Duplicate || again.Rows[1].Duplicate {
		t.Errorf("duplicates after commit = %v %v; want true false", again.Rows[0].Duplicate, again.Rows[1].Duplicate)
	}

	t.Run("source down", func(t *testing.T) {
		s.chains = fakeChain{err: chain.ErrUnavailable}
		if _, err := s.PreviewWallet(ctx, u, w.ID); !errors.Is(err, models.ErrImportUnavailable) {
			t.Errorf("err = %v, want ErrImportUnavailable", err)
		}
	})

	t.Run("unlink", func(t *testing.T) {
		if err := s.RemoveWallet(ctx, u, w.ID); err != nil {
			t.Fatal(err)
		}
		if err := s.RemoveWallet(ctx, u, w.ID); !errors.Is(err, models.ErrNotFound) {
			t.Errorf("second unlink err = %v, want ErrNotFound", err)
		}
	})
}
