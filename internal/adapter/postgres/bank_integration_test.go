//go:build integration

package postgres

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"finbot/internal/domain"
)

func TestUsersCannotSeeEachOthersBanks(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)

	tests := []struct {
		name string
		run  func(*testing.T, context.Context, *BankRepository, domain.Bank, domain.Bank)
	}{
		{
			name: "list is per user",
			run: func(t *testing.T, ctx context.Context, banks *BankRepository, a, b domain.Bank) {
				got, err := banks.List(ctx, b.UserID)
				if err != nil {
					t.Fatalf("list: %v", err)
				}
				if len(got) != 1 || got[0].ID != b.ID {
					t.Fatalf("list %+v", got)
				}
			},
		},
		{
			name: "get by id does not leak",
			run: func(t *testing.T, ctx context.Context, banks *BankRepository, a, b domain.Bank) {
				_, err := banks.GetByID(ctx, b.UserID, a.ID)
				if !errors.Is(err, domain.ErrBankNotFound) {
					t.Fatalf("got %v, want ErrBankNotFound", err)
				}
			},
		},
		{
			name: "get by name is scoped to user",
			run: func(t *testing.T, ctx context.Context, banks *BankRepository, a, b domain.Bank) {
				_, err := banks.GetByName(ctx, b.UserID, a.Name)
				if !errors.Is(err, domain.ErrBankNotFound) {
					t.Fatalf("got %v, want ErrBankNotFound", err)
				}
				got, err := banks.GetByName(ctx, a.UserID, a.Name)
				if err != nil {
					t.Fatalf("own name: %v", err)
				}
				if got.ID != a.ID {
					t.Fatalf("got %+v", got)
				}
			},
		},
		{
			name: "update other user's bank is not found",
			run: func(t *testing.T, ctx context.Context, banks *BankRepository, a, b domain.Bank) {
				patch := a
				patch.Balance = 1
				patch.UpdatedAt = now.Add(time.Minute)
				_, err := banks.Update(ctx, b.UserID, patch)
				if !errors.Is(err, domain.ErrBankNotFound) {
					t.Fatalf("got %v, want ErrBankNotFound", err)
				}
				got, err := banks.GetByID(ctx, a.UserID, a.ID)
				if err != nil {
					t.Fatalf("owner get: %v", err)
				}
				if got.Balance != a.Balance {
					t.Fatalf("balance changed to %d", got.Balance)
				}
			},
		},
		{
			name: "delete other user's bank is not found",
			run: func(t *testing.T, ctx context.Context, banks *BankRepository, a, b domain.Bank) {
				err := banks.Delete(ctx, b.UserID, a.ID)
				if !errors.Is(err, domain.ErrBankNotFound) {
					t.Fatalf("got %v, want ErrBankNotFound", err)
				}
				if _, err := banks.GetByID(ctx, a.UserID, a.ID); err != nil {
					t.Fatalf("owner get: %v", err)
				}
			},
		},
		{
			name: "total is per user",
			run: func(t *testing.T, ctx context.Context, banks *BankRepository, a, b domain.Bank) {
				totalA, err := banks.TotalIncluded(ctx, a.UserID)
				if err != nil {
					t.Fatalf("total a: %v", err)
				}
				totalB, err := banks.TotalIncluded(ctx, b.UserID)
				if err != nil {
					t.Fatalf("total b: %v", err)
				}
				if totalA != a.Balance {
					t.Fatalf("total a %d, want %d", totalA, a.Balance)
				}
				if totalB != b.Balance {
					t.Fatalf("total b %d, want %d", totalB, b.Balance)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := mustOpenReset(t)
			ctx := context.Background()
			users := NewUserRepository(db)
			banks := NewBankRepository(db)
			upsertTestUser(t, users, 1, now)
			upsertTestUser(t, users, 2, now)
			a := createBank(t, banks, 1, "Holiday", 10000, true, now)
			b := createBank(t, banks, 2, "Gifts", 2500, true, now)
			tt.run(t, ctx, banks, a, b)
		})
	}
}

func TestBankNameUniquePerUser(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)

	tests := []struct {
		name       string
		firstUser  domain.UserID
		secondUser domain.UserID
		first      string
		second     string
		wantErr    error
	}{
		{name: "same spelling", firstUser: 1, secondUser: 1, first: "Live", second: "Live", wantErr: domain.ErrBankNameTaken},
		{name: "different case", firstUser: 1, secondUser: 1, first: "Live", second: "LIVE", wantErr: domain.ErrBankNameTaken},
		{name: "unicode case", firstUser: 1, secondUser: 1, first: "Подарки", second: "подарки", wantErr: domain.ErrBankNameTaken},
		{name: "other user same name", firstUser: 1, secondUser: 2, first: "Live", second: "Live"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := mustOpenReset(t)
			ctx := context.Background()
			users := NewUserRepository(db)
			banks := NewBankRepository(db)
			upsertTestUser(t, users, tt.firstUser, now)
			if tt.secondUser != tt.firstUser {
				upsertTestUser(t, users, tt.secondUser, now)
			}
			createBank(t, banks, tt.firstUser, tt.first, 0, true, now)

			_, err := banks.Create(ctx, tt.secondUser, domain.Bank{
				Name:      tt.second,
				CreatedAt: now,
				UpdatedAt: now,
			})
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("got %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("create: %v", err)
			}
		})
	}
}

func TestReopenKeepsData(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	db, err := Open(ctx, testURL())
	if err != nil {
		t.Fatalf("first open: %v (start local Postgres with docker compose up -d)", err)
	}
	resetDB(t, db)
	users := NewUserRepository(db)
	banks := NewBankRepository(db)
	upsertTestUser(t, users, 1, now)
	upsertTestUser(t, users, 2, now)
	holiday := createBank(t, banks, 1, "Holiday", 10000, true, now)
	createBank(t, banks, 1, "Gifts", 2500, false, now)
	createBank(t, banks, 2, "Holiday", 50, true, now)
	closeDB(t, db)

	reopened, err := Open(ctx, testURL())
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer closeDB(t, reopened)
	banks = NewBankRepository(reopened)

	listed, err := banks.List(ctx, 1)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(listed) != 2 {
		t.Fatalf("list %+v", listed)
	}
	got, err := banks.GetByID(ctx, 1, holiday.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Name != "Holiday" || got.Balance != 10000 || !got.IncludeInTotal {
		t.Fatalf("persisted %+v", got)
	}

	total, err := banks.TotalIncluded(ctx, 1)
	if err != nil {
		t.Fatalf("total: %v", err)
	}
	if total != 10000 {
		t.Fatalf("total %d, want 10000", total)
	}

	other, err := banks.List(ctx, 2)
	if err != nil {
		t.Fatalf("other list: %v", err)
	}
	if len(other) != 1 || other[0].Name != "Holiday" || other[0].Balance != 50 {
		t.Fatalf("other %+v", other)
	}
	if _, err := banks.GetByID(ctx, 2, holiday.ID); !errors.Is(err, domain.ErrBankNotFound) {
		t.Fatalf("leaked after reopen: %v", err)
	}

	_, err = banks.Create(ctx, 1, domain.Bank{Name: "holiday", CreatedAt: now, UpdatedAt: now})
	if !errors.Is(err, domain.ErrBankNameTaken) {
		t.Fatalf("unique after reopen: %v", err)
	}
}

func TestBankRepositoryCRUD(t *testing.T) {
	db := mustOpenReset(t)
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
	if created.Currency != "USD" {
		t.Fatalf("currency %q, want USD", created.Currency)
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

	created.Name = "live"
	created.UpdatedAt = now.Add(2 * time.Minute)
	recased, err := banks.Update(ctx, 1, created)
	if err != nil {
		t.Fatalf("recase: %v", err)
	}
	if recased.Name != "live" {
		t.Fatalf("recase %+v", recased)
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
	db := mustOpenReset(t)
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
	db := mustOpenReset(t)
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

func TestConcurrentCreateSameName(t *testing.T) {
	db := mustOpenReset(t)
	now := time.Now().UTC().Truncate(time.Second)
	users := NewUserRepository(db)
	banks := NewBankRepository(db)
	upsertTestUser(t, users, 1, now)

	const n = 8
	errc := make(chan error, n)
	var wg sync.WaitGroup
	for range n {
		wg.Go(func() {
			_, err := banks.Create(context.Background(), 1, domain.Bank{
				Name:      "Live",
				CreatedAt: now,
				UpdatedAt: now,
			})
			errc <- err
		})
	}
	wg.Wait()
	close(errc)

	var ok, taken int
	for err := range errc {
		switch {
		case err == nil:
			ok++
		case errors.Is(err, domain.ErrBankNameTaken):
			taken++
		default:
			t.Fatalf("create: %v", err)
		}
	}
	if ok != 1 || taken != n-1 {
		t.Fatalf("ok=%d taken=%d, want 1 unique and %d taken", ok, taken, n-1)
	}
}
