package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"finbot/internal/domain"
	"finbot/internal/service/mocks"
)

func TestUpsertUser(t *testing.T) {
	ctx := context.Background()
	now := fixedNow()

	tests := []struct {
		name     string
		trial    time.Duration
		username string
		wantEnd  time.Duration
	}{
		{
			name:     "first insert freezes trial from duration",
			trial:    168 * time.Hour,
			username: "alice",
			wantEnd:  168 * time.Hour,
		},
		{
			name:     "zero duration means trial already ended",
			trial:    0,
			username: "alice",
			wantEnd:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			users := mocks.NewMockUserRepository(t)
			clock := mocks.NewMockClock(t)
			svc := New(mocks.NewMockBankRepository(t), users, mocks.NewMockOperationRepository(t), passthroughTx(t), clock, testCfg(tt.trial, "USD"))

			want := domain.User{
				TelegramID:     userA,
				Username:       tt.username,
				LastActivityAt: now,
				Plan:           domain.PlanTrial,
				TrialEndsAt:    now.Add(tt.wantEnd),
				Locale:         "",
				CreatedAt:      now,
				UpdatedAt:      now,
			}
			clock.EXPECT().Now().Return(now)
			users.EXPECT().Upsert(ctx, want).Return(want, nil)

			got, err := svc.UpsertUser(ctx, userA, tt.username)
			require.NoError(t, err)
			require.Equal(t, want, got)
			require.Nil(t, got.DiscountPercent)
		})
	}
}

func TestTouchActivity(t *testing.T) {
	ctx := context.Background()
	now := fixedNow()

	t.Run("updates last activity", func(t *testing.T) {
		users := mocks.NewMockUserRepository(t)
		clock := mocks.NewMockClock(t)
		svc := New(mocks.NewMockBankRepository(t), users, mocks.NewMockOperationRepository(t), passthroughTx(t), clock, testCfg(168*time.Hour, "USD"))
		clock.EXPECT().Now().Return(now)
		users.EXPECT().TouchActivity(ctx, userA, now).Return(nil)

		err := svc.TouchActivity(ctx, userA)
		require.NoError(t, err)
	})

	t.Run("unknown user", func(t *testing.T) {
		users := mocks.NewMockUserRepository(t)
		clock := mocks.NewMockClock(t)
		svc := New(mocks.NewMockBankRepository(t), users, mocks.NewMockOperationRepository(t), passthroughTx(t), clock, testCfg(168*time.Hour, "USD"))
		clock.EXPECT().Now().Return(now)
		users.EXPECT().TouchActivity(ctx, userB, now).Return(domain.ErrUserNotFound)

		err := svc.TouchActivity(ctx, userB)
		require.ErrorIs(t, err, domain.ErrUserNotFound)
	})
}

func TestSetReferredByIfEmpty(t *testing.T) {
	ctx := context.Background()

	t.Run("stores referrer", func(t *testing.T) {
		users := mocks.NewMockUserRepository(t)
		svc := New(mocks.NewMockBankRepository(t), users, mocks.NewMockOperationRepository(t), passthroughTx(t), mocks.NewMockClock(t), testCfg(168*time.Hour, "USD"))
		users.EXPECT().SetReferredByIfEmpty(ctx, userA, userB).Return(nil)

		require.NoError(t, svc.SetReferredByIfEmpty(ctx, userA, userB))
	})

	t.Run("self does not call repository", func(t *testing.T) {
		users := mocks.NewMockUserRepository(t)
		svc := New(mocks.NewMockBankRepository(t), users, mocks.NewMockOperationRepository(t), passthroughTx(t), mocks.NewMockClock(t), testCfg(168*time.Hour, "USD"))

		require.NoError(t, svc.SetReferredByIfEmpty(ctx, userA, userA))
		require.True(t, users.AssertNotCalled(t, "SetReferredByIfEmpty"))
	})
}

func TestSetLocale(t *testing.T) {
	ctx := context.Background()
	now := fixedNow()

	t.Run("persists a known locale", func(t *testing.T) {
		users := mocks.NewMockUserRepository(t)
		clock := mocks.NewMockClock(t)
		svc := New(mocks.NewMockBankRepository(t), users, mocks.NewMockOperationRepository(t), passthroughTx(t), clock, testCfg(168*time.Hour, "USD"))
		clock.EXPECT().Now().Return(now)
		users.EXPECT().SetLocale(ctx, userA, domain.LocaleRU, now).Return(nil)

		require.NoError(t, svc.SetLocale(ctx, userA, domain.LocaleRU))
	})

	t.Run("rejects unknown locale", func(t *testing.T) {
		users := mocks.NewMockUserRepository(t)
		svc := New(mocks.NewMockBankRepository(t), users, mocks.NewMockOperationRepository(t), passthroughTx(t), mocks.NewMockClock(t), testCfg(168*time.Hour, "USD"))

		err := svc.SetLocale(ctx, userA, "fr")
		require.ErrorIs(t, err, domain.ErrUnknownLocale)
		require.True(t, users.AssertNotCalled(t, "SetLocale"))
	})

	t.Run("unknown user", func(t *testing.T) {
		users := mocks.NewMockUserRepository(t)
		clock := mocks.NewMockClock(t)
		svc := New(mocks.NewMockBankRepository(t), users, mocks.NewMockOperationRepository(t), passthroughTx(t), clock, testCfg(168*time.Hour, "USD"))
		clock.EXPECT().Now().Return(now)
		users.EXPECT().SetLocale(ctx, userB, domain.LocaleUK, now).Return(domain.ErrUserNotFound)

		err := svc.SetLocale(ctx, userB, domain.LocaleUK)
		require.ErrorIs(t, err, domain.ErrUserNotFound)
	})
}
