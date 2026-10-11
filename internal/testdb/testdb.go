// Package testdb поднимает изолированную схему в тестовой базе Postgres для интеграционных тестов.
// Адрес базы берётся из TEST_DATABASE_URL, без него тест пропускается.
package testdb

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"moneykeeper/internal/pgclient"
)

var counter atomic.Int64

func New(t *testing.T) *pgclient.Client {
	t.Helper()
	baseURL := os.Getenv("TEST_DATABASE_URL")
	if baseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	schema := fmt.Sprintf("t_%d_%d", time.Now().UnixNano(), counter.Add(1))

	admin, err := pgxpool.New(ctx, baseURL)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() {
		_, _ = admin.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE")
		admin.Close()
	})
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatalf("create schema: %v", err)
	}

	u, err := url.Parse(baseURL)
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}
	q := u.Query()
	q.Set("search_path", schema)
	u.RawQuery = q.Encode()

	db, err := pgclient.New(ctx, u.String())
	if err != nil {
		t.Fatalf("pgclient: %v", err)
	}
	t.Cleanup(db.Close)

	for _, sql := range upMigrations(t) {
		if _, err := db.Q(ctx).Exec(ctx, sql); err != nil {
			t.Fatalf("apply migration: %v", err)
		}
	}
	return db
}

func upMigrations(t *testing.T) []string {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	dir := filepath.Join(filepath.Dir(file), "..", "..", "migrations")
	files, err := filepath.Glob(filepath.Join(dir, "*.up.sql"))
	if err != nil || len(files) == 0 {
		t.Fatalf("migrations not found in %s: %v", dir, err)
	}
	out := make([]string, 0, len(files))
	for _, f := range files { // Glob возвращает по алфавиту, номера с ведущими нулями
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		out = append(out, strings.TrimSpace(string(b)))
	}
	return out
}
