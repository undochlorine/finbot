//go:build integration

package postgres

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"finbot/internal/adapter/clock"
	"finbot/internal/domain"
	"finbot/internal/service"
)

func resetDB(t *testing.T, db *sql.DB) {
	t.Helper()
	if _, err := db.ExecContext(context.Background(), `TRUNCATE users RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("truncate: %v", err)
	}
}

func testService(db *sql.DB, currency string) *service.Service {
	return service.New(
		NewBankRepository(db),
		NewUserRepository(db),
		NewOperationRepository(db),
		NewTransactor(db),
		clock.New(),
		service.Config{
			Trial:   service.TrialConfig{Duration: time.Hour.String()},
			Default: service.DefaultConfig{Currency: currency},
		},
	)
}

func testUser(id domain.UserID, now time.Time) domain.User {
	return domain.User{
		TelegramID:     id,
		Username:       "u",
		LastActivityAt: now,
		Plan:           domain.PlanTrial,
		TrialEndsAt:    now.Add(24 * time.Hour),
		Locale:         domain.LocaleEN,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

func upsertTestUser(t *testing.T, users *UserRepository, id domain.UserID, now time.Time) {
	t.Helper()
	if _, err := users.Upsert(context.Background(), testUser(id, now)); err != nil {
		t.Fatalf("user %d: %v", id, err)
	}
}

func createBank(
	t *testing.T,
	banks *BankRepository,
	user domain.UserID,
	name string,
	bal domain.Money,
	include bool,
	now time.Time,
) domain.Bank {
	t.Helper()
	got, err := banks.Create(context.Background(), user, domain.Bank{
		Name:           name,
		Balance:        bal,
		IncludeInTotal: include,
		CreatedAt:      now,
		UpdatedAt:      now,
	})
	if err != nil {
		t.Fatalf("create %q: %v", name, err)
	}
	return got
}

func mustOpenReset(t *testing.T) *sql.DB {
	t.Helper()
	db := mustOpen(t)
	resetDB(t, db)
	return db
}
