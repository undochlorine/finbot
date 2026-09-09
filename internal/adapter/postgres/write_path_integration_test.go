//go:build integration

package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"testing"
	"time"

	"finbot/internal/domain"
)

func TestCreateBankUsesDefaultCurrency(t *testing.T) {
	db := mustOpenReset(t)
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

func TestCreateEmptyCurrencyDefaultsUSD(t *testing.T) {
	db := mustOpenReset(t)
	ctx := context.Background()
	now := timeNowTruncated()
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

func TestAddSpendSetAndDeleteWriteOperations(t *testing.T) {
	db := mustOpenReset(t)
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
	assertOp(t, db, "add", 2500, 2500, true, bank.ID, true, "Live")

	if _, err := svc.Spend(ctx, 1, bank.ID, 500); err != nil {
		t.Fatalf("spend: %v", err)
	}
	assertOp(t, db, "spend", 500, 2000, true, bank.ID, true, "Live")

	if _, err := svc.Set(ctx, 1, bank.ID, 1000); err != nil {
		t.Fatalf("set: %v", err)
	}
	assertOp(t, db, "set", 1000, 1000, true, bank.ID, true, "Live")

	if err := svc.Delete(ctx, 1, bank.ID); err != nil {
		t.Fatalf("delete: %v", err)
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
		FROM operations WHERE type = 'delete'`).Scan(&typ, &amount, &after, &bankID, &meta); err != nil {
		t.Fatalf("delete row: %v", err)
	}
	if typ != "delete" || amount != 1000 || after.Valid {
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

func TestRenameWritesOperation(t *testing.T) {
	db := mustOpenReset(t)
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

	assertOp(t, db, "rename", 0, 2500, true, holiday.ID, true, "Holiday -> holiday")

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
	db := mustOpenReset(t)
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

	assertOp(t, db, "transfer", 2500, 2500, true, holiday.ID, true, strconv.FormatInt(gifts.ID, 10))
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
	db := mustOpenReset(t)
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
		CREATE OR REPLACE FUNCTION operations_fail() RETURNS trigger AS $$
		BEGIN
			RAISE EXCEPTION 'boom';
		END;
		$$ LANGUAGE plpgsql`); err != nil {
		t.Fatalf("function: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
		CREATE TRIGGER operations_fail BEFORE INSERT ON operations
		FOR EACH ROW EXECUTE FUNCTION operations_fail()`); err != nil {
		t.Fatalf("trigger: %v", err)
	}
	t.Cleanup(func() {
		if _, err := db.ExecContext(ctx, `DROP TRIGGER IF EXISTS operations_fail ON operations`); err != nil {
			t.Errorf("drop trigger: %v", err)
		}
		if _, err := db.ExecContext(ctx, `DROP FUNCTION IF EXISTS operations_fail()`); err != nil {
			t.Errorf("drop function: %v", err)
		}
	})

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

	if _, err := svc.Spend(ctx, 1, bank.ID, 100); err == nil {
		t.Fatal("spend: want error")
	}
	got, err = NewBankRepository(db).GetByID(ctx, 1, bank.ID)
	if err != nil {
		t.Fatalf("get after spend: %v", err)
	}
	if got.Balance != 0 {
		t.Fatalf("spend left balance %d", got.Balance)
	}

	if _, err := svc.Set(ctx, 1, bank.ID, 100); err == nil {
		t.Fatal("set: want error")
	}
	got, err = NewBankRepository(db).GetByID(ctx, 1, bank.ID)
	if err != nil {
		t.Fatalf("get after set: %v", err)
	}
	if got.Balance != 0 {
		t.Fatalf("set left balance %d", got.Balance)
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

func assertOp(
	t *testing.T,
	db *sql.DB,
	typ string,
	amount, afterCents int64,
	afterValid bool,
	bankID int64,
	bankValid bool,
	meta string,
) {
	t.Helper()
	var (
		gotType   string
		gotAmount int64
		gotAfter  sql.NullInt64
		gotBank   sql.NullInt64
		gotMeta   sql.NullString
	)
	if err := db.QueryRowContext(context.Background(), `
		SELECT type, amount_cents, balance_after_cents, bank_id, meta
		FROM operations WHERE type = $1
		ORDER BY id DESC LIMIT 1`, typ).Scan(&gotType, &gotAmount, &gotAfter, &gotBank, &gotMeta); err != nil {
		t.Fatalf("%s row: %v", typ, err)
	}
	if gotType != typ || gotAmount != amount || gotAfter.Valid != afterValid {
		t.Fatalf("%s op type=%s amount=%d after=%v", typ, gotType, gotAmount, gotAfter)
	}
	if afterValid && gotAfter.Int64 != afterCents {
		t.Fatalf("%s after %d, want %d", typ, gotAfter.Int64, afterCents)
	}
	if gotBank.Valid != bankValid || (bankValid && gotBank.Int64 != bankID) {
		t.Fatalf("%s bank_id %+v, want valid=%v id=%d", typ, gotBank, bankValid, bankID)
	}
	if !gotMeta.Valid || gotMeta.String != meta {
		t.Fatalf("%s meta %+v, want %q", typ, gotMeta, meta)
	}
}

func timeNowTruncated() time.Time {
	return time.Now().UTC().Truncate(time.Second)
}
