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
			svc := New(mocks.NewMockBankRepository(t), users, clock, tt.trial)

			want := domain.User{
				TelegramID:     userA,
				Username:       tt.username,
				LastActivityAt: now,
				Plan:           domain.PlanTrial,
				TrialEndsAt:    now.Add(tt.wantEnd),
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
		svc := New(mocks.NewMockBankRepository(t), users, clock, 168*time.Hour)
		clock.EXPECT().Now().Return(now)
		users.EXPECT().TouchActivity(ctx, userA, now).Return(nil)

		err := svc.TouchActivity(ctx, userA)
		require.NoError(t, err)
	})

	t.Run("unknown user", func(t *testing.T) {
		users := mocks.NewMockUserRepository(t)
		clock := mocks.NewMockClock(t)
		svc := New(mocks.NewMockBankRepository(t), users, clock, 168*time.Hour)
		clock.EXPECT().Now().Return(now)
		users.EXPECT().TouchActivity(ctx, userB, now).Return(domain.ErrUserNotFound)

		err := svc.TouchActivity(ctx, userB)
		require.ErrorIs(t, err, domain.ErrUserNotFound)
	})
}
