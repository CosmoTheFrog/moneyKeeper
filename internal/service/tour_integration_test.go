//go:build integration

package service

import (
	"context"
	"testing"
)

func TestTourSeen(t *testing.T) {
	ctx := context.Background()
	l := newLedger(t)
	read := func() bool {
		t.Helper()
		u, err := l.s.users.GetByTelegramID(ctx, tgIgor.ID)
		if err != nil {
			t.Fatal(err)
		}
		return u.TourSeen
	}

	if read() {
		t.Fatal("tour must be unseen for a new user")
	}
	for _, seen := range []bool{true, false} {
		if err := l.s.SetTourSeen(ctx, l.u, seen); err != nil {
			t.Fatal(err)
		}
		if got := read(); got != seen {
			t.Errorf("after SetTourSeen(%v): %v", seen, got)
		}
	}
}
