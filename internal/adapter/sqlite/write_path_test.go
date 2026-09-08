//go:build integration

package sqlite

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"finbot/internal/adapter/clock"
	"finbot/internal/domain"
	"finbot/internal/service"
)

func TestWritePathMigrationIdempotent(t *testing.T) {
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

	var versions []string
	rows, err := second.QueryContext(ctx, `SELECT version FROM schema_migrations ORDER BY version`)
	if err != nil {
		t.Fatalf("versions: %v", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			t.Errorf("close: %v", err)
		}
	}()
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			t.Fatalf("scan: %v", err)
		}
		versions = append(versions, v)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}
	if len(versions) != 2 || versions[0] != "001_init" || versions[1] != "002_write_path_foundations" {
		t.Fatalf("versions %v", versions)
	}
}

func testService(db *sql.DB, currency string) *service.Service {
	return service.New(
		NewBankRepository(db),
		NewUserRepository(db),
		NewOperationRepository(db),
		NewTransactor(db),
		clock.New(),
		time.Hour,
		currency,
	)
}

func TestExistingBankGetsUSDAfterMigrate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "finbot.db")
	ctx := context.Background()
	now := time.Now().UTC().Format(time.RFC3339)

	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := pingAndPragma(ctx, db); err != nil {
		t.Fatalf("pragma: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
		CREATE TABLE schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TEXT NOT NULL
		)`); err != nil {
		t.Fatalf("schema_migrations: %v", err)
	}
	body, err := migrationsFS.ReadFile("migrations/001_init.sql")
	if err != nil {
		t.Fatalf("read 001: %v", err)
	}
	if err := applyMigration(ctx, db, "001_init", string(body)); err != nil {
		t.Fatalf("apply 001: %v", err)
	}
	insertUser(t, db, 1, now)
	insertBank(t, db, 1, "Live", "live", now)
	closeDB(t, db)

	opened, err := Open(ctx, path)
	if err != nil {
		t.Fatalf("upgrade: %v", err)
	}
	defer closeDB(t, opened)

	var currency string
	if err := opened.QueryRowContext(ctx, `SELECT currency FROM banks WHERE name = 'Live'`).Scan(&currency); err != nil {
		t.Fatalf("currency: %v", err)
	}
	if currency != "USD" {
		t.Fatalf("currency %q, want USD", currency)
	}
}

func TestCreateBankUsesDefaultCurrency(t *testing.T) {
	db := openTemp(t)
	defer closeDB(t, db)
	ctx := context.Background()
	svc := testService(db, "EUR")
	if _, err := svc.UpsertUser(ctx, 1, "alice"); err != nil {
		t.Fatalf("user: %v", err)
	}
	bank, err := svc.CreateBank(ctx, 1, "Holiday", true)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if bank.Currency != "EUR" {
		t.Fatalf("currency %q, want EUR", bank.Currency)
	}
	got, err := NewBankRepository(db).GetByID(ctx, 1, bank.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Currency != "EUR" {
		t.Fatalf("stored %q, want EUR", got.Currency)
	}
}

func TestAddAndDeleteWriteOperations(t *testing.T) {
	db := openTemp(t)
	defer closeDB(t, db)
	ctx := context.Background()
	svc := testService(db, "USD")
	if _, err := svc.UpsertUser(ctx, 1, "alice"); err != nil {
		t.Fatalf("user: %v", err)
	}
	bank, err := svc.CreateBank(ctx, 1, "Live", true)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := svc.Add(ctx, 1, bank.ID, 2500); err != nil {
		t.Fatalf("add: %v", err)
	}

	var (
		typ     string
		amount  int64
		after   sql.NullInt64
		bankID  sql.NullInt64
		meta    sql.NullString
		opCount int
	)
	if err := db.QueryRowContext(ctx, `
		SELECT type, amount_cents, balance_after_cents, bank_id, meta, COUNT(*) OVER ()
		FROM operations WHERE type = 'add'`).Scan(&typ, &amount, &after, &bankID, &meta, &opCount); err != nil {
		t.Fatalf("add row: %v", err)
	}
	if opCount != 1 || typ != "add" || amount != 2500 || !after.Valid || after.Int64 != 2500 {
		t.Fatalf("add op type=%s amount=%d after=%v count=%d", typ, amount, after, opCount)
	}
	if !bankID.Valid || bankID.Int64 != bank.ID {
		t.Fatalf("add bank_id %+v", bankID)
	}
	if !meta.Valid || meta.String != "Live" {
		t.Fatalf("add meta %+v", meta)
	}

	if err := svc.Delete(ctx, 1, bank.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}

	if err := db.QueryRowContext(ctx, `
		SELECT type, amount_cents, balance_after_cents, bank_id, meta
		FROM operations WHERE type = 'delete'`).Scan(&typ, &amount, &after, &bankID, &meta); err != nil {
		t.Fatalf("delete row: %v", err)
	}
	if typ != "delete" || amount != 2500 || after.Valid {
		t.Fatalf("delete op type=%s amount=%d after=%v", typ, amount, after)
	}
	if bankID.Valid {
		t.Fatalf("delete bank_id should be null after cascade, got %v", bankID.Int64)
	}
	if !meta.Valid || meta.String != "Live" {
		t.Fatalf("delete meta %+v", meta)
	}

	if err := db.QueryRowContext(ctx, `
		SELECT bank_id, meta FROM operations WHERE type = 'add'`).Scan(&bankID, &meta); err != nil {
		t.Fatalf("add after delete: %v", err)
	}
	if bankID.Valid {
		t.Fatalf("add bank_id should be null after delete, got %v", bankID.Int64)
	}
	if !meta.Valid || meta.String != "Live" {
		t.Fatalf("add meta lost after delete: %+v", meta)
	}
}

func TestExistingUserGetsLocaleENAfterMigrate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "finbot.db")
	ctx := context.Background()
	now := time.Now().UTC().Format(time.RFC3339)

	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := pingAndPragma(ctx, db); err != nil {
		t.Fatalf("pragma: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
		CREATE TABLE schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TEXT NOT NULL
		)`); err != nil {
		t.Fatalf("schema_migrations: %v", err)
	}
	body, err := migrationsFS.ReadFile("migrations/001_init.sql")
	if err != nil {
		t.Fatalf("read 001: %v", err)
	}
	if err := applyMigration(ctx, db, "001_init", string(body)); err != nil {
		t.Fatalf("apply 001: %v", err)
	}
	insertUser(t, db, 1, now)
	closeDB(t, db)

	opened, err := Open(ctx, path)
	if err != nil {
		t.Fatalf("upgrade: %v", err)
	}
	defer closeDB(t, opened)

	var locale string
	if err := opened.QueryRowContext(ctx, `SELECT locale FROM users WHERE telegram_id = 1`).Scan(&locale); err != nil {
		t.Fatalf("locale: %v", err)
	}
	if locale != "en" {
		t.Fatalf("locale %q, want en", locale)
	}
}

func TestCreateEmptyCurrencyDefaultsUSD(t *testing.T) {
	db := openTemp(t)
	defer closeDB(t, db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	if _, err := NewUserRepository(db).Upsert(ctx, testUser(1, now)); err != nil {
		t.Fatalf("user: %v", err)
	}
	got, err := NewBankRepository(db).Create(ctx, 1, domain.Bank{
		Name:      "Live",
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if got.Currency != "USD" {
		t.Fatalf("currency %q, want USD", got.Currency)
	}
}

func TestMutatorsRollBackWhenAppendFails(t *testing.T) {
	db := openTemp(t)
	defer closeDB(t, db)
	ctx := context.Background()
	svc := testService(db, "USD")
	if _, err := svc.UpsertUser(ctx, 1, "alice"); err != nil {
		t.Fatalf("user: %v", err)
	}
	bank, err := svc.CreateBank(ctx, 1, "Live", true)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
		CREATE TRIGGER ops_fail BEFORE INSERT ON operations
		BEGIN
			SELECT RAISE(ABORT, 'boom');
		END`); err != nil {
		t.Fatalf("trigger: %v", err)
	}

	if _, err := svc.Add(ctx, 1, bank.ID, 2500); err == nil {
		t.Fatal("add: want error")
	}
	got, err := NewBankRepository(db).GetByID(ctx, 1, bank.ID)
	if err != nil {
		t.Fatalf("get after add: %v", err)
	}
	if got.Balance != 0 {
		t.Fatalf("add left balance %d", got.Balance)
	}

	if err := svc.Delete(ctx, 1, bank.ID); err == nil {
		t.Fatal("delete: want error")
	}
	if _, err := NewBankRepository(db).GetByID(ctx, 1, bank.ID); err != nil {
		t.Fatalf("delete removed bank: %v", err)
	}
}
