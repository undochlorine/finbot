package ports

import (
	"context"

	"finbot/internal/domain"
)

type BankRepository interface {
	Create(ctx context.Context, userID domain.UserID, bank domain.Bank) (domain.Bank, error)
	GetByID(ctx context.Context, userID domain.UserID, bankID int64) (domain.Bank, error)
	GetByName(ctx context.Context, userID domain.UserID, name string) (domain.Bank, error)
	List(ctx context.Context, userID domain.UserID) ([]domain.Bank, error)
	Update(ctx context.Context, userID domain.UserID, bank domain.Bank) (domain.Bank, error)
	Delete(ctx context.Context, userID domain.UserID, bankID int64) error
	TotalIncluded(ctx context.Context, userID domain.UserID) (domain.Money, error)
}
