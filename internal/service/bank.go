package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"finbot/internal/domain"
)

func (s *Service) CreateBank(
	ctx context.Context,
	userID domain.UserID,
	name string,
	includeInTotal bool,
) (domain.Bank, error) {
	name = strings.TrimSpace(name)
	if _, err := domain.NormalizeBankName(name); err != nil {
		return domain.Bank{}, fmt.Errorf("bank name: %w", err)
	}

	_, err := s.banks.GetByName(ctx, userID, name)
	if err == nil {
		return domain.Bank{}, domain.ErrBankNameTaken
	}
	if !errors.Is(err, domain.ErrBankNotFound) {
		return domain.Bank{}, fmt.Errorf("check bank name: %w", err)
	}

	now := s.clock.Now()
	bank, err := s.banks.Create(ctx, userID, domain.Bank{
		UserID:         userID,
		Name:           name,
		Balance:        0,
		IncludeInTotal: includeInTotal,
		CreatedAt:      now,
		UpdatedAt:      now,
	})
	if err != nil {
		return domain.Bank{}, fmt.Errorf("create bank: %w", err)
	}
	return bank, nil
}

func (s *Service) Add(
	ctx context.Context,
	userID domain.UserID,
	bankID int64,
	amount domain.Money,
) (domain.Bank, error) {
	if amount < 0 {
		return domain.Bank{}, domain.ErrInvalidAmount
	}
	return s.adjust(ctx, userID, bankID, amount)
}

func (s *Service) Spend(
	ctx context.Context,
	userID domain.UserID,
	bankID int64,
	amount domain.Money,
) (domain.Bank, error) {
	if amount < 0 {
		return domain.Bank{}, domain.ErrInvalidAmount
	}
	return s.adjust(ctx, userID, bankID, -amount)
}

func (s *Service) Set(
	ctx context.Context,
	userID domain.UserID,
	bankID int64,
	amount domain.Money,
) (domain.Bank, error) {
	bank, err := s.banks.GetByID(ctx, userID, bankID)
	if err != nil {
		return domain.Bank{}, fmt.Errorf("get bank: %w", err)
	}
	bank.Balance = amount
	return s.save(ctx, userID, bank)
}

func (s *Service) Delete(ctx context.Context, userID domain.UserID, bankID int64) error {
	if err := s.banks.Delete(ctx, userID, bankID); err != nil {
		return fmt.Errorf("delete bank: %w", err)
	}
	return nil
}

func (s *Service) Get(ctx context.Context, userID domain.UserID, bankID int64) (domain.Bank, error) {
	bank, err := s.banks.GetByID(ctx, userID, bankID)
	if err != nil {
		return domain.Bank{}, fmt.Errorf("get bank: %w", err)
	}
	return bank, nil
}

func (s *Service) GetByName(ctx context.Context, userID domain.UserID, name string) (domain.Bank, error) {
	bank, err := s.banks.GetByName(ctx, userID, name)
	if err != nil {
		return domain.Bank{}, fmt.Errorf("get bank by name: %w", err)
	}
	return bank, nil
}

func (s *Service) List(ctx context.Context, userID domain.UserID) ([]domain.Bank, error) {
	banks, err := s.banks.List(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list banks: %w", err)
	}
	return banks, nil
}

func (s *Service) Total(ctx context.Context, userID domain.UserID) (domain.Money, error) {
	total, err := s.banks.TotalIncluded(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("total: %w", err)
	}
	return total, nil
}

func (s *Service) All(ctx context.Context, userID domain.UserID) ([]domain.Bank, domain.Money, error) {
	banks, err := s.List(ctx, userID)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.Total(ctx, userID)
	if err != nil {
		return nil, 0, err
	}
	return banks, total, nil
}

func (s *Service) Toggle(ctx context.Context, userID domain.UserID, bankID int64) (domain.Bank, error) {
	bank, err := s.banks.GetByID(ctx, userID, bankID)
	if err != nil {
		return domain.Bank{}, fmt.Errorf("get bank: %w", err)
	}
	bank.IncludeInTotal = !bank.IncludeInTotal
	return s.save(ctx, userID, bank)
}

func (s *Service) adjust(
	ctx context.Context,
	userID domain.UserID,
	bankID int64,
	delta domain.Money,
) (domain.Bank, error) {
	bank, err := s.banks.GetByID(ctx, userID, bankID)
	if err != nil {
		return domain.Bank{}, fmt.Errorf("get bank: %w", err)
	}
	sum, err := addMoney(bank.Balance, delta)
	if err != nil {
		return domain.Bank{}, err
	}
	bank.Balance = sum
	return s.save(ctx, userID, bank)
}

func (s *Service) save(ctx context.Context, userID domain.UserID, bank domain.Bank) (domain.Bank, error) {
	bank.UpdatedAt = s.clock.Now()
	updated, err := s.banks.Update(ctx, userID, bank)
	if err != nil {
		return domain.Bank{}, fmt.Errorf("update bank: %w", err)
	}
	return updated, nil
}

func addMoney(cur, delta domain.Money) (domain.Money, error) {
	a, b := int64(cur), int64(delta)
	sum := a + b
	if (b > 0 && sum < a) || (b < 0 && sum > a) {
		return 0, domain.ErrInvalidAmount
	}
	return domain.Money(sum), nil
}
