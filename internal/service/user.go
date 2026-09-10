package service

import (
	"context"
	"fmt"
	"time"

	"finbot/internal/domain"
)

type UserRepository interface {
	// Upsert inserts on first seen. Existing trial_ends_at, plan, discount_percent,
	// locale, and referred_by are left unchanged. New users are inserted with an empty
	// locale until they pick one.
	Upsert(ctx context.Context, user domain.User) (domain.User, error)
	TouchActivity(ctx context.Context, userID domain.UserID, at time.Time) error
	SetReferredByIfEmpty(ctx context.Context, userID, referredBy domain.UserID) error
	SetLocale(ctx context.Context, userID domain.UserID, locale string, at time.Time) error
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

func (s *Service) SetReferredByIfEmpty(ctx context.Context, userID, referredBy domain.UserID) error {
	if referredBy == userID {
		return nil
	}
	if err := s.users.SetReferredByIfEmpty(ctx, userID, referredBy); err != nil {
		return fmt.Errorf("set referred by: %w", err)
	}
	return nil
}

func (s *Service) SetLocale(ctx context.Context, userID domain.UserID, locale string) error {
	if !domain.KnownLocale(locale) {
		return fmt.Errorf("set locale: %w", domain.ErrUnknownLocale)
	}
	if err := s.users.SetLocale(ctx, userID, locale, s.clock.Now()); err != nil {
		return fmt.Errorf("set locale: %w", err)
	}
	return nil
}
