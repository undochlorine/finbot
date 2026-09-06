package ports

import (
	"context"
	"time"

	"finbot/internal/domain"
)

type UserRepository interface {
	Get(ctx context.Context, userID domain.UserID) (domain.User, error)
	// Upsert inserts on first seen. Existing trial_ends_at, plan, and discount_percent are left unchanged.
	Upsert(ctx context.Context, user domain.User) (domain.User, error)
	TouchActivity(ctx context.Context, userID domain.UserID, at time.Time) error
}
