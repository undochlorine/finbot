//go:build integration

package postgres

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"finbot/internal/domain"
)

func TestGetLocksOnlyInTx(t *testing.T) {
	db := mustOpenReset(t)
	now := time.Now().UTC().Truncate(time.Second)
	users := NewUserRepository(db)
	banks := NewBankRepository(db)
	upsertTestUser(t, users, 1, now)
	bank := createBank(t, banks, 1, "Live", 0, true, now)

	tests := []struct {
		name    string
		hold    func(context.Context) error
		wantErr bool
	}{
		{
			name: "get by id in tx",
			hold: func(ctx context.Context) error {
				_, err := banks.GetByID(ctx, 1, bank.ID)
				return err
			},
			wantErr: true,
		},
		{
			name: "get by name in tx",
			hold: func(ctx context.Context) error {
				_, err := banks.GetByName(ctx, 1, "Live")
				return err
			},
			wantErr: true,
		},
		{
			name: "list in tx",
			hold: func(ctx context.Context) error {
				_, err := banks.List(ctx, 1)
				return err
			},
		},
		{
			name: "total in tx",
			hold: func(ctx context.Context) error {
				_, err := banks.TotalIncluded(ctx, 1)
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			release, errc := holdTx(t, db, tt.hold)
			err := lockNowait(db, 1, bank.ID)
			release()
			if txErr := <-errc; txErr != nil {
				t.Fatalf("hold tx: %v", txErr)
			}
			if tt.wantErr {
				if !isLockNotAvailable(err) {
					t.Fatalf("got %v, want lock_not_available", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("display read locked row: %v", err)
			}
		})
	}
}

func TestInTxRetriesDeadlock(t *testing.T) {
	db := mustOpenReset(t)
	now := time.Now().UTC().Truncate(time.Second)
	users := NewUserRepository(db)
	banks := NewBankRepository(db)
	tx := NewTransactor(db)
	upsertTestUser(t, users, 1, now)
	a := createBank(t, banks, 1, "A", 0, true, now)
	b := createBank(t, banks, 1, "B", 0, true, now)

	start := make(chan struct{})
	var wg sync.WaitGroup
	errs := make([]error, 2)
	run := func(i int, first, second int64) {
		defer wg.Done()
		<-start
		errs[i] = tx.InTx(context.Background(), func(ctx context.Context) error {
			if _, err := banks.GetByID(ctx, 1, first); err != nil {
				return err
			}
			time.Sleep(80 * time.Millisecond)
			_, err := banks.GetByID(ctx, 1, second)
			return err
		})
	}
	wg.Add(2)
	go run(0, a.ID, b.ID)
	go run(1, b.ID, a.ID)
	close(start)
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Fatalf("tx %d: %v", i, err)
		}
	}
}

func holdTx(t *testing.T, db *sql.DB, fn func(context.Context) error) (func(), <-chan error) {
	t.Helper()
	locked := make(chan struct{})
	release := make(chan struct{})
	errc := make(chan error, 1)
	go func() {
		errc <- NewTransactor(db).InTx(context.Background(), func(ctx context.Context) error {
			if err := fn(ctx); err != nil {
				return err
			}
			close(locked)
			<-release
			return nil
		})
	}()
	select {
	case <-locked:
	case err := <-errc:
		t.Fatalf("tx: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for tx")
	}
	return func() { close(release) }, errc
}

func lockNowait(db *sql.DB, userID domain.UserID, bankID int64) error {
	ctx := context.Background()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer rollbackTx(tx)
	_, err = tx.ExecContext(ctx, `
		SELECT id FROM banks WHERE id = $1 AND user_id = $2 FOR UPDATE NOWAIT`, bankID, userID)
	return err
}

func isLockNotAvailable(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "55P03"
}
