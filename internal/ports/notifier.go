package ports

import (
	"context"

	"finbot/internal/domain"
)

type Notifier interface {
	Notify(ctx context.Context, userID domain.UserID, message string) error
}
