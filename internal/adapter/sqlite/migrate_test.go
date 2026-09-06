package sqlite

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"
)

func TestOpenCreatesSchema(t *testing.T) {
	db := openTemp(t)
	defer closeDB(t, db)

	wantUsers := []string{
		"telegram_id", "username", "last_activity_at", "plan",
		"trial_ends_at", "discount_percent", "created_at", "updated_at",
	}
	wantBanks := []string{
		"id", "user_id", "name", "name_normalized", "balance_cents",
		"include_in_total", "created_at", "updated_at",
	}
	assertColumns(t, db, "users", wantUsers)
	assertColumns(t, db, "banks", wantBanks)
}

func TestOpenIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "finbot.db")
	ctx := context.Background()

	first, err := Open(ctx, path)
	if err != nil {
		t.Fatalf("first open: %v", err)
	}
	closeDB(t, first)

	second, err := Open(ctx, path)
	if err != nil {
		t.Fatalf("second open: %v", err)
	}
	defer closeDB(t, second)

	var n int
	if err := second.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations`).Scan(&n); err != nil {
		t.Fatalf("count migrations: %v", err)
	}
	if n != 1 {
		t.Fatalf("schema_migrations rows = %d, want 1", n)
	}
}

func TestUniqueBankNamePerUser(t *testing.T) {
	db := openTemp(t)
	defer closeDB(t, db)
	ctx := context.Background()
	now := time.Now().UTC().Format(time.RFC3339)

	insertUser(t, db, 1, now)
	insertUser(t, db, 2, now)
	insertBank(t, db, 1, "Live", "live", now)

	tests := []struct {
		name    string
		userID  int64
		bank    string
		norm    string
		wantErr bool
	}{
		{name: "same user same name", userID: 1, bank: "live", norm: "live", wantErr: true},
		{name: "same user different case", userID: 1, bank: "LIVE", norm: "live", wantErr: true},
		{name: "same user other name", userID: 1, bank: "Gifts", norm: "gifts", wantErr: false},
		{name: "other user same name", userID: 2, bank: "Live", norm: "live", wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := insertBankErr(ctx, db, tt.userID, tt.bank, tt.norm, now)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected unique constraint error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func openTemp(t *testing.T) *sql.DB {
	t.Helper()
	db, err := Open(context.Background(), filepath.Join(t.TempDir(), "finbot.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	return db
}

func closeDB(t *testing.T, db *sql.DB) {
	t.Helper()
	if err := db.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
}

func assertColumns(t *testing.T, db *sql.DB, table string, want []string) {
	t.Helper()
	rows, err := db.Query(`SELECT name FROM pragma_table_info(?) ORDER BY cid`, table)
	if err != nil {
		t.Fatalf("pragma %s: %v", table, err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			t.Errorf("close rows: %v", err)
		}
	}()

	var got []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan: %v", err)
		}
		got = append(got, name)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("%s columns = %v, want %v", table, got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%s columns = %v, want %v", table, got, want)
		}
	}
}

func insertUser(t *testing.T, db *sql.DB, id int64, now string) {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO users (telegram_id, last_activity_at, trial_ends_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)`, id, now, now, now, now)
	if err != nil {
		t.Fatalf("insert user %d: %v", id, err)
	}
}

func insertBank(t *testing.T, db *sql.DB, userID int64, name, norm, now string) {
	t.Helper()
	if err := insertBankErr(context.Background(), db, userID, name, norm, now); err != nil {
		t.Fatalf("insert bank %q: %v", name, err)
	}
}

func insertBankErr(ctx context.Context, db *sql.DB, userID int64, name, norm, now string) error {
	_, err := db.ExecContext(ctx, `
		INSERT INTO banks (user_id, name, name_normalized, include_in_total, created_at, updated_at)
		VALUES (?, ?, ?, 1, ?, ?)`, userID, name, norm, now, now)
	return err
}
