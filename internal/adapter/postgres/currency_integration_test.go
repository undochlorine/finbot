//go:build integration

package postgres

import (
	"context"
	"testing"
	"time"

	"finbot/internal/domain"
)

func TestCurrencyModel(t *testing.T) {
	db := mustOpenReset(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	var n int
	err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM schema_migrations WHERE version = $1`, "002_currency_usd",
	).Scan(&n)
	if err != nil {
		t.Fatalf("count 002_currency_usd: %v", err)
	}
	if n != 1 {
		t.Fatalf("schema_migrations 002_currency_usd rows = %d, want 1", n)
	}

	_, err = db.ExecContext(ctx, `
		INSERT INTO users (telegram_id, last_activity_at, trial_ends_at, created_at, updated_at)
		VALUES ($1, $2, $2, $2, $2)`, 1, now)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
	var last string
	err = db.QueryRowContext(ctx, `SELECT last_currency FROM users WHERE telegram_id = $1`, 1).Scan(&last)
	if err != nil {
		t.Fatalf("last_currency: %v", err)
	}
	if last != domain.CurrencyUSD {
		t.Fatalf("omitted last_currency = %q, want USD", last)
	}

	_, err = db.ExecContext(ctx, `
		INSERT INTO banks (user_id, name, name_normalized, include_in_total, created_at, updated_at)
		VALUES ($1, 'Cash', 'cash', TRUE, $2, $2)`, 1, now)
	if err != nil {
		t.Fatalf("insert bank: %v", err)
	}
	var currency string
	err = db.QueryRowContext(ctx, `SELECT currency FROM banks WHERE user_id = $1`, 1).Scan(&currency)
	if err != nil {
		t.Fatalf("currency: %v", err)
	}
	if currency != domain.CurrencyUSD {
		t.Fatalf("omitted currency = %q, want USD", currency)
	}

	_, err = db.ExecContext(ctx, `
		INSERT INTO banks (user_id, name, name_normalized, include_in_total, currency, created_at, updated_at)
		VALUES ($1, 'Nope', 'nope', TRUE, NULL, $2, $2)`, 1, now)
	if err == nil {
		t.Fatal("NULL currency inserted")
	}
}

func TestUpsertLeavesLastCurrency(t *testing.T) {
	db := mustOpenReset(t)
	users := NewUserRepository(db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	created, err := users.Upsert(ctx, testUser(7, now))
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if created.LastCurrency != domain.CurrencyUSD {
		t.Fatalf("new last_currency %q, want USD", created.LastCurrency)
	}

	if _, err := db.ExecContext(ctx, `UPDATE users SET last_currency = 'UAH' WHERE telegram_id = $1`, 7); err != nil {
		t.Fatalf("set last_currency: %v", err)
	}

	later := now.Add(time.Hour)
	updated, err := users.Upsert(ctx, testUser(7, later))
	if err != nil {
		t.Fatalf("upsert again: %v", err)
	}
	if updated.LastCurrency != "UAH" {
		t.Fatalf("last_currency %q, want UAH", updated.LastCurrency)
	}
	if updated.Username != "u" {
		t.Fatalf("username %q", updated.Username)
	}
}
