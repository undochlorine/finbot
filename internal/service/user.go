package service

import (
	"context"
	"fmt"

	"finbot/internal/domain"
)

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
