//go:build integration

package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"finbot/internal/domain"
)

func TestUserRepositoryUpsertAndGet(t *testing.T) {
	db := openTemp(t)
	defer closeDB(t, db)
	users := NewUserRepository(db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	trial := now.Add(24 * time.Hour)
	pct := 50

	created, err := users.Upsert(ctx, domain.User{
		TelegramID:      7,
		Username:        "alice",
		LastActivityAt:  now,
		Plan:            domain.PlanTrial,
		TrialEndsAt:     trial,
		DiscountPercent: &pct,
		Locale:          domain.LocaleEN,
		CreatedAt:       now,
		UpdatedAt:       now,
	})
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if created.TelegramID != 7 || created.Username != "alice" || created.Plan != domain.PlanTrial {
		t.Fatalf("created %+v", created)
	}
	if created.Locale != domain.LocaleEN {
		t.Fatalf("locale %q", created.Locale)
	}
	if created.ReferredBy != nil {
		t.Fatalf("referred_by %+v", created.ReferredBy)
	}
	if created.DiscountPercent == nil || *created.DiscountPercent != pct {
		t.Fatalf("discount %+v", created.DiscountPercent)
	}

	later := now.Add(time.Hour)
	newPct := 100
	updated, err := users.Upsert(ctx, domain.User{
		TelegramID:      7,
		Username:        "alice2",
		LastActivityAt:  later,
		Plan:            domain.PlanFree,
		TrialEndsAt:     later.Add(48 * time.Hour),
		DiscountPercent: &newPct,
		Locale:          "fr",
		CreatedAt:       later,
		UpdatedAt:       later,
	})
	if err != nil {
		t.Fatalf("upsert again: %v", err)
	}
	if updated.Username != "alice2" {
		t.Fatalf("username %q", updated.Username)
	}
	if !updated.LastActivityAt.Equal(later) {
		t.Fatalf("activity %v", updated.LastActivityAt)
	}
	if !updated.TrialEndsAt.Equal(trial) {
		t.Fatalf("trial rewritten: %v", updated.TrialEndsAt)
	}
	if updated.Plan != domain.PlanTrial {
		t.Fatalf("plan rewritten: %q", updated.Plan)
	}
	if updated.DiscountPercent == nil || *updated.DiscountPercent != pct {
		t.Fatalf("discount rewritten: %+v", updated.DiscountPercent)
	}
	if updated.Locale != domain.LocaleEN {
		t.Fatalf("locale rewritten: %q", updated.Locale)
	}
	if updated.ReferredBy != nil {
		t.Fatalf("referred_by rewritten: %+v", updated.ReferredBy)
	}
}

func TestUserRepositoryGetNotFound(t *testing.T) {
	db := openTemp(t)
	defer closeDB(t, db)
	_, err := NewUserRepository(db).Get(context.Background(), 99)
	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("got %v, want ErrUserNotFound", err)
	}
}

func TestUserRepositoryTouchActivity(t *testing.T) {
	db := openTemp(t)
	defer closeDB(t, db)
	users := NewUserRepository(db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	if _, err := users.Upsert(ctx, testUser(1, now)); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	at := now.Add(2 * time.Hour)
	if err := users.TouchActivity(ctx, 1, at); err != nil {
		t.Fatalf("touch: %v", err)
	}
	got, err := users.Get(ctx, 1)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !got.LastActivityAt.Equal(at) {
		t.Fatalf("activity %v, want %v", got.LastActivityAt, at)
	}

	err = users.TouchActivity(ctx, 99, at)
	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("got %v, want ErrUserNotFound", err)
	}
}

func TestUserRepositorySetReferredByIfEmpty(t *testing.T) {
	db := openTemp(t)
	defer closeDB(t, db)
	users := NewUserRepository(db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	if _, err := users.Upsert(ctx, testUser(1, now)); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	if err := users.SetReferredByIfEmpty(ctx, 1, 1); err != nil {
		t.Fatalf("self: %v", err)
	}
	got, err := users.Get(ctx, 1)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.ReferredBy != nil {
		t.Fatalf("self wrote %+v", got.ReferredBy)
	}

	if err := users.SetReferredByIfEmpty(ctx, 1, 99); err != nil {
		t.Fatalf("first: %v", err)
	}
	got, err = users.Get(ctx, 1)
	if err != nil {
		t.Fatalf("get after first: %v", err)
	}
	if got.ReferredBy == nil || *got.ReferredBy != 99 {
		t.Fatalf("referred_by %+v", got.ReferredBy)
	}

	if err := users.SetReferredByIfEmpty(ctx, 1, 100); err != nil {
		t.Fatalf("second: %v", err)
	}
	got, err = users.Get(ctx, 1)
	if err != nil {
		t.Fatalf("get after second: %v", err)
	}
	if got.ReferredBy == nil || *got.ReferredBy != 99 {
		t.Fatalf("overwrote %+v", got.ReferredBy)
	}

	err = users.SetReferredByIfEmpty(ctx, 2, 99)
	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("missing user: %v", err)
	}
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
