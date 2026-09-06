package service

import (
	"context"
	"fmt"
	"time"

	"finbot/internal/domain"
)

type UserRepository interface {
	// Upsert inserts on first seen. Existing trial_ends_at, plan, and discount_percent are left unchanged.
	Upsert(ctx context.Context, user domain.User) (domain.User, error)
	TouchActivity(ctx context.Context, userID domain.UserID, at time.Time) error
}

func (s *Service) UpsertUser(ctx context.Context, userID domain.UserID, username string) (domain.User, error) {
	now := s.clock.Now()
	user, err := s.users.Upsert(ctx, domain.User{
		TelegramID:     userID,
		Username:       username,
		LastActivityAt: now,
		Plan:           domain.PlanTrial,
		TrialEndsAt:    now.Add(s.trialDuration),
		CreatedAt:      now,
		UpdatedAt:      now,
	})
	if err != nil {
		return domain.User{}, fmt.Errorf("upsert user: %w", err)
	}
	return user, nil
}

func (s *Service) TouchActivity(ctx context.Context, userID domain.UserID) error {
	if err := s.users.TouchActivity(ctx, userID, s.clock.Now()); err != nil {
		return fmt.Errorf("touch activity: %w", err)
	}
	return nil
}
