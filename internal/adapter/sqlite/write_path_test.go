//go:build integration

package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strconv"
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
	if len(versions) != 3 ||
		versions[0] != "001_init" ||
		versions[1] != "002_write_path_foundations" ||
		versions[2] != "003_rename_operation" {
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

func TestRenameCheckMigrationPreservesRows(t *testing.T) {
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
	for _, version := range []string{"001_init", "002_write_path_foundations"} {
		body, err := migrationsFS.ReadFile("migrations/" + version + ".sql")
		if err != nil {
			t.Fatalf("read %s: %v", version, err)
		}
		if err := applyMigration(ctx, db, version, string(body)); err != nil {
			t.Fatalf("apply %s: %v", version, err)
		}
	}
	insertUser(t, db, 1, now)
	insertBank(t, db, 1, "Live", "live", now)
	var bankID int64
	if err := db.QueryRowContext(ctx, `SELECT id FROM banks WHERE name = 'Live'`).Scan(&bankID); err != nil {
		t.Fatalf("bank id: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO operations (user_id, bank_id, type, amount_cents, balance_after_cents, meta, created_at)
		VALUES (1, ?, 'add', 2500, 2500, 'Live', ?)`, bankID, now); err != nil {
		t.Fatalf("seed op: %v", err)
	}
	closeDB(t, db)

	opened, err := Open(ctx, path)
	if err != nil {
		t.Fatalf("upgrade: %v", err)
	}
	defer closeDB(t, opened)

	var (
		typ   string
		meta  string
		after int64
	)
	if err := opened.QueryRowContext(ctx, `
		SELECT type, meta, balance_after_cents FROM operations WHERE type = 'add'`).Scan(&typ, &meta, &after); err != nil {
		t.Fatalf("preserved: %v", err)
	}
	if typ != "add" || meta != "Live" || after != 2500 {
		t.Fatalf("preserved type=%s meta=%s after=%d", typ, meta, after)
	}

	if _, err := opened.ExecContext(ctx, `
		INSERT INTO operations (user_id, bank_id, type, amount_cents, balance_after_cents, meta, created_at)
		VALUES (1, ?, 'rename', 0, 2500, 'Live -> live', ?)`, bankID, now); err != nil {
		t.Fatalf("insert rename: %v", err)
	}
}

func TestRenameWritesOperation(t *testing.T) {
	db := openTemp(t)
	defer closeDB(t, db)
	ctx := context.Background()
	svc := testService(db, "USD")
	if _, err := svc.UpsertUser(ctx, 1, "alice"); err != nil {
		t.Fatalf("user: %v", err)
	}
	holiday, err := svc.CreateBank(ctx, 1, "Holiday", true)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := svc.Add(ctx, 1, holiday.ID, 2500); err != nil {
		t.Fatalf("add: %v", err)
	}
	gifts, err := svc.CreateBank(ctx, 1, "Gifts", false)
	if err != nil {
		t.Fatalf("create gifts: %v", err)
	}

	renamed, err := svc.Rename(ctx, 1, holiday.ID, "holiday")
	if err != nil {
		t.Fatalf("recase: %v", err)
	}
	if renamed.Name != "holiday" || renamed.Balance != 2500 {
		t.Fatalf("recase %+v", renamed)
	}

	var (
		typ    string
		amount int64
		after  sql.NullInt64
		bankID sql.NullInt64
		meta   sql.NullString
	)
	if err := db.QueryRowContext(ctx, `
		SELECT type, amount_cents, balance_after_cents, bank_id, meta
		FROM operations WHERE type = 'rename'`).Scan(&typ, &amount, &after, &bankID, &meta); err != nil {
		t.Fatalf("rename row: %v", err)
	}
	if typ != "rename" || amount != 0 || !after.Valid || after.Int64 != 2500 {
		t.Fatalf("rename op type=%s amount=%d after=%v", typ, amount, after)
	}
	if !bankID.Valid || bankID.Int64 != holiday.ID {
		t.Fatalf("rename bank_id %+v", bankID)
	}
	if !meta.Valid || meta.String != "Holiday -> holiday" {
		t.Fatalf("rename meta %+v", meta)
	}

	if _, err := svc.Rename(ctx, 1, holiday.ID, "gifts"); !errors.Is(err, domain.ErrBankNameTaken) {
		t.Fatalf("taken: %v", err)
	}
	got, err := NewBankRepository(db).GetByID(ctx, 1, holiday.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Name != "holiday" {
		t.Fatalf("overwrote name %q", got.Name)
	}
	var n int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM operations WHERE type = 'rename'`).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 1 {
		t.Fatalf("rename ops %d, want 1", n)
	}

	got, err = svc.Rename(ctx, 1, gifts.ID, "Trips")
	if err != nil {
		t.Fatalf("rename gifts: %v", err)
	}
	if got.Name != "Trips" {
		t.Fatalf("renamed %+v", got)
	}
}

func TestTransferWritesOneOperation(t *testing.T) {
	db := openTemp(t)
	defer closeDB(t, db)
	ctx := context.Background()
	svc := testService(db, "USD")
	if _, err := svc.UpsertUser(ctx, 1, "alice"); err != nil {
		t.Fatalf("user: %v", err)
	}
	holiday, err := svc.CreateBank(ctx, 1, "Holiday", true)
	if err != nil {
		t.Fatalf("create holiday: %v", err)
	}
	if _, err := svc.Add(ctx, 1, holiday.ID, 5000); err != nil {
		t.Fatalf("add: %v", err)
	}
	gifts, err := svc.CreateBank(ctx, 1, "Gifts", false)
	if err != nil {
		t.Fatalf("create gifts: %v", err)
	}

	from, to, err := svc.Transfer(ctx, 1, holiday.ID, gifts.ID, 2500)
	if err != nil {
		t.Fatalf("transfer: %v", err)
	}
	if from.Balance != 2500 || to.Balance != 2500 {
		t.Fatalf("balances from=%d to=%d", from.Balance, to.Balance)
	}

	var (
		typ    string
		amount int64
		after  sql.NullInt64
		bankID sql.NullInt64
		meta   sql.NullString
	)
	if err := db.QueryRowContext(ctx, `
		SELECT type, amount_cents, balance_after_cents, bank_id, meta
		FROM operations WHERE type = 'transfer'`).Scan(&typ, &amount, &after, &bankID, &meta); err != nil {
		t.Fatalf("transfer row: %v", err)
	}
	if typ != "transfer" || amount != 2500 || !after.Valid || after.Int64 != 2500 {
		t.Fatalf("transfer op type=%s amount=%d after=%v", typ, amount, after)
	}
	if !bankID.Valid || bankID.Int64 != holiday.ID {
		t.Fatalf("transfer bank_id %+v", bankID)
	}
	if !meta.Valid || meta.String != strconv.FormatInt(gifts.ID, 10) {
		t.Fatalf("transfer meta %+v", meta)
	}
	var n int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM operations WHERE type = 'transfer'`).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 1 {
		t.Fatalf("transfer ops %d, want 1", n)
	}

	if _, _, err := svc.Transfer(ctx, 1, holiday.ID, holiday.ID, 100); !errors.Is(err, domain.ErrSameBank) {
		t.Fatalf("same bank: %v", err)
	}
	if _, _, err := svc.Transfer(ctx, 1, holiday.ID, gifts.ID, 0); !errors.Is(err, domain.ErrInvalidAmount) {
		t.Fatalf("zero: %v", err)
	}

	gotFrom, err := NewBankRepository(db).GetByID(ctx, 1, holiday.ID)
	if err != nil {
		t.Fatalf("get from: %v", err)
	}
	gotTo, err := NewBankRepository(db).GetByID(ctx, 1, gifts.ID)
	if err != nil {
		t.Fatalf("get to: %v", err)
	}
	if gotFrom.Balance != 2500 || gotTo.Balance != 2500 {
		t.Fatalf("rejected transfer changed balances from=%d to=%d", gotFrom.Balance, gotTo.Balance)
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

	if _, err := svc.Rename(ctx, 1, bank.ID, "holiday"); err == nil {
		t.Fatal("rename: want error")
	}
	got, err = NewBankRepository(db).GetByID(ctx, 1, bank.ID)
	if err != nil {
		t.Fatalf("get after rename: %v", err)
	}
	if got.Name != "Live" {
		t.Fatalf("rename left name %q", got.Name)
	}

	other, err := svc.CreateBank(ctx, 1, "Gifts", false)
	if err != nil {
		t.Fatalf("create gifts: %v", err)
	}
	if _, _, err := svc.Transfer(ctx, 1, bank.ID, other.ID, 100); err == nil {
		t.Fatal("transfer: want error")
	}
	got, err = NewBankRepository(db).GetByID(ctx, 1, bank.ID)
	if err != nil {
		t.Fatalf("get after transfer: %v", err)
	}
	if got.Balance != 0 {
		t.Fatalf("transfer left balance %d", got.Balance)
	}
	got, err = NewBankRepository(db).GetByID(ctx, 1, other.ID)
	if err != nil {
		t.Fatalf("get to after transfer: %v", err)
	}
	if got.Balance != 0 {
		t.Fatalf("transfer credited to %d", got.Balance)
	}
}
