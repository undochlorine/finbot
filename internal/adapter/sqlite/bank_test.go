//go:build integration

package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"finbot/internal/domain"
)

func TestWALEnabled(t *testing.T) {
	db := openTemp(t)
	defer closeDB(t, db)
	var mode string
	if err := db.QueryRow(`PRAGMA journal_mode`).Scan(&mode); err != nil {
		t.Fatalf("pragma: %v", err)
	}
	if mode != "wal" {
		t.Fatalf("journal_mode=%q, want wal", mode)
	}
}

func TestBankRepositoryCRUD(t *testing.T) {
	db := openTemp(t)
	defer closeDB(t, db)
	ctx := context.Background()
	users := NewUserRepository(db)
	banks := NewBankRepository(db)
	now := time.Now().UTC().Truncate(time.Second)

	if _, err := users.Upsert(ctx, testUser(1, now)); err != nil {
		t.Fatalf("user: %v", err)
	}
	if _, err := users.Upsert(ctx, testUser(2, now)); err != nil {
		t.Fatalf("user2: %v", err)
	}

	created, err := banks.Create(ctx, 1, domain.Bank{
		Name:           "Live",
		Balance:        1250,
		IncludeInTotal: true,
		CreatedAt:      now,
		UpdatedAt:      now,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.ID == 0 || created.UserID != 1 || created.Name != "Live" || created.Balance != 1250 {
		t.Fatalf("created %+v", created)
	}

	byID, err := banks.GetByID(ctx, 1, created.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if byID.Name != "Live" {
		t.Fatalf("get %+v", byID)
	}

	byName, err := banks.GetByName(ctx, 1, "live")
	if err != nil {
		t.Fatalf("get by name: %v", err)
	}
	if byName.ID != created.ID {
		t.Fatalf("name lookup %+v", byName)
	}

	_, err = banks.Create(ctx, 1, domain.Bank{Name: "LIVE", CreatedAt: now, UpdatedAt: now})
	if !errors.Is(err, domain.ErrBankNameTaken) {
		t.Fatalf("duplicate: %v", err)
	}

	if _, err := banks.Create(ctx, 2, domain.Bank{
		Name:           "Live",
		Balance:        50,
		IncludeInTotal: true,
		CreatedAt:      now,
		UpdatedAt:      now,
	}); err != nil {
		t.Fatalf("other user same name: %v", err)
	}

	_, err = banks.GetByID(ctx, 2, created.ID)
	if !errors.Is(err, domain.ErrBankNotFound) {
		t.Fatalf("leaked bank: %v", err)
	}

	listed, err := banks.List(ctx, 1)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(listed) != 1 || listed[0].ID != created.ID {
		t.Fatalf("list %+v", listed)
	}

	created.Balance = 500
	created.IncludeInTotal = false
	created.UpdatedAt = now.Add(time.Minute)
	updated, err := banks.Update(ctx, 1, created)
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Balance != 500 || updated.IncludeInTotal {
		t.Fatalf("updated %+v", updated)
	}

	total, err := banks.TotalIncluded(ctx, 1)
	if err != nil {
		t.Fatalf("total: %v", err)
	}
	if total != 0 {
		t.Fatalf("excluded total %d", total)
	}
	otherTotal, err := banks.TotalIncluded(ctx, 2)
	if err != nil {
		t.Fatalf("other total: %v", err)
	}
	if otherTotal != 50 {
		t.Fatalf("other total %d", otherTotal)
	}

	if err := banks.Delete(ctx, 2, created.ID); !errors.Is(err, domain.ErrBankNotFound) {
		t.Fatalf("cross-delete: %v", err)
	}
	if _, err := banks.GetByID(ctx, 1, created.ID); err != nil {
		t.Fatalf("deleted by other user: %v", err)
	}

	if err := banks.Delete(ctx, 1, created.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := banks.GetByID(ctx, 1, created.ID); !errors.Is(err, domain.ErrBankNotFound) {
		t.Fatalf("after delete: %v", err)
	}

	empty, err := banks.TotalIncluded(ctx, 1)
	if err != nil {
		t.Fatalf("empty total: %v", err)
	}
	if empty != 0 {
		t.Fatalf("empty total %d", empty)
	}
}

func TestBankRepositoryUpdateNameTaken(t *testing.T) {
	db := openTemp(t)
	defer closeDB(t, db)
	ctx := context.Background()
	users := NewUserRepository(db)
	banks := NewBankRepository(db)
	now := time.Now().UTC().Truncate(time.Second)
	if _, err := users.Upsert(ctx, testUser(1, now)); err != nil {
		t.Fatalf("user: %v", err)
	}

	a, err := banks.Create(ctx, 1, domain.Bank{Name: "Live", CreatedAt: now, UpdatedAt: now})
	if err != nil {
		t.Fatalf("a: %v", err)
	}
	if _, err := banks.Create(ctx, 1, domain.Bank{Name: "Gifts", CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatalf("b: %v", err)
	}

	a.Name = "gifts"
	a.UpdatedAt = now.Add(time.Minute)
	_, err = banks.Update(ctx, 1, a)
	if !errors.Is(err, domain.ErrBankNameTaken) {
		t.Fatalf("got %v, want ErrBankNameTaken", err)
	}
}

func TestBankRepositoryEmptyList(t *testing.T) {
	db := openTemp(t)
	defer closeDB(t, db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	if _, err := NewUserRepository(db).Upsert(ctx, testUser(1, now)); err != nil {
		t.Fatalf("user: %v", err)
	}
	got, err := NewBankRepository(db).List(ctx, 1)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("got %v", got)
	}
}
