//go:build integration

package postgres

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"
)

const defaultTestURL = "postgres://finbot:finbot@127.0.0.1:5432/finbot_test?sslmode=disable"

func TestOpenIsIdempotent(t *testing.T) {
	ctx := context.Background()
	url := testURL()

	first, err := Open(ctx, url)
	if err != nil {
		t.Fatalf("first open: %v (start local Postgres with docker compose up -d)", err)
	}
	closeDB(t, first)

	second, err := Open(ctx, url)
	if err != nil {
		t.Fatalf("second open: %v (start local Postgres with docker compose up -d)", err)
	}
	defer closeDB(t, second)

	var n int
	err = second.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM schema_migrations WHERE version = $1`, "001_init",
	).Scan(&n)
	if err != nil {
		t.Fatalf("count 001_init: %v", err)
	}
	if n != 1 {
		t.Fatalf("schema_migrations 001_init rows = %d, want 1", n)
	}
}

func TestOpenCreatesExpectedSchema(t *testing.T) {
	db := mustOpen(t)
	ctx := context.Background()

	for _, table := range []string{"users", "banks", "operations", "schema_migrations"} {
		var exists bool
		err := db.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM information_schema.tables
				WHERE table_schema = 'public' AND table_name = $1
			)`, table).Scan(&exists)
		if err != nil {
			t.Fatalf("table %s: %v", table, err)
		}
		if !exists {
			t.Fatalf("missing table %s", table)
		}
	}

	assertColumnType(t, db, "users", "telegram_id", "bigint")
	assertColumnType(t, db, "banks", "id", "bigint")
	assertColumnType(t, db, "operations", "id", "bigint")
	assertColumnType(t, db, "users", "last_activity_at", "timestamp with time zone")
	assertColumnType(t, db, "banks", "include_in_total", "boolean")

	wantIndexes := []struct {
		table string
		name  string
	}{
		{table: "banks", name: "idx_banks_user_id"},
		{table: "operations", name: "idx_operations_user_created_at"},
		{table: "operations", name: "idx_operations_bank_id"},
		{table: "users", name: "idx_users_last_activity_at"},
	}
	for _, idx := range wantIndexes {
		var exists bool
		err := db.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM pg_indexes
				WHERE schemaname = 'public' AND tablename = $1 AND indexname = $2
			)`, idx.table, idx.name).Scan(&exists)
		if err != nil {
			t.Fatalf("index %s: %v", idx.name, err)
		}
		if !exists {
			t.Fatalf("missing index %s on %s", idx.name, idx.table)
		}
	}

	var uniqueExists bool
	err := db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM pg_indexes
			WHERE schemaname = 'public' AND tablename = 'banks'
			  AND indexdef ILIKE '%UNIQUE%'
			  AND indexdef ILIKE '%user_id%'
			  AND indexdef ILIKE '%name_normalized%'
		)`).Scan(&uniqueExists)
	if err != nil {
		t.Fatalf("unique index: %v", err)
	}
	if !uniqueExists {
		t.Fatal("missing unique (user_id, name_normalized) on banks")
	}
}

func TestOpenConfiguresPoolAndTimeouts(t *testing.T) {
	db := mustOpen(t)

	if got := db.Stats().MaxOpenConnections; got != MaxOpenConns {
		t.Fatalf("MaxOpenConnections = %d, want %d", got, MaxOpenConns)
	}

	assertSetting(t, db, "statement_timeout", "5s")
	assertSetting(t, db, "lock_timeout", "2s")
	assertSetting(t, db, "idle_in_transaction_session_timeout", "10s")
}

func TestOpenFailsWhenUnreachable(t *testing.T) {
	_, err := Open(context.Background(),
		"postgres://finbot:finbot@127.0.0.1:1/finbot_test?sslmode=disable&connect_timeout=1")
	if err == nil {
		t.Fatal("expected error when postgres is unreachable")
	}
}

func mustOpen(t *testing.T) *sql.DB {
	t.Helper()
	db, err := Open(context.Background(), testURL())
	if err != nil {
		t.Fatalf("open postgres: %v (start local Postgres with docker compose up -d)", err)
	}
	t.Cleanup(func() { closeDB(t, db) })
	return db
}

func closeDB(t *testing.T, db *sql.DB) {
	t.Helper()
	if err := db.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
}

func testURL() string {
	if u := strings.TrimSpace(os.Getenv("POSTGRES_TEST_URL")); u != "" {
		return u
	}
	return defaultTestURL
}

func assertColumnType(t *testing.T, db *sql.DB, table, column, want string) {
	t.Helper()
	var got string
	err := db.QueryRowContext(context.Background(), `
		SELECT data_type FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = $1 AND column_name = $2`,
		table, column).Scan(&got)
	if err != nil {
		t.Fatalf("%s.%s type: %v", table, column, err)
	}
	if got != want {
		t.Fatalf("%s.%s type = %q, want %q", table, column, got, want)
	}
}

func assertSetting(t *testing.T, db *sql.DB, name, want string) {
	t.Helper()
	var got string
	if err := db.QueryRowContext(context.Background(), "SHOW "+name).Scan(&got); err != nil {
		t.Fatalf("SHOW %s: %v", name, err)
	}
	if got != want {
		t.Fatalf("%s = %q, want %q", name, got, want)
	}
}
