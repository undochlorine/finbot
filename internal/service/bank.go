package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

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
		Currency:       s.defaultCurrency,
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
	return s.changeBalance(ctx, userID, domain.OperationAdd, amount, func(ctx context.Context) (domain.Bank, error) {
		return s.adjust(ctx, userID, bankID, amount)
	})
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
	return s.changeBalance(ctx, userID, domain.OperationSpend, amount, func(ctx context.Context) (domain.Bank, error) {
		return s.adjust(ctx, userID, bankID, -amount)
	})
}

func (s *Service) Set(
	ctx context.Context,
	userID domain.UserID,
	bankID int64,
	amount domain.Money,
) (domain.Bank, error) {
	return s.changeBalance(ctx, userID, domain.OperationSet, amount, func(ctx context.Context) (domain.Bank, error) {
		bank, err := s.banks.GetByID(ctx, userID, bankID)
		if err != nil {
			return domain.Bank{}, fmt.Errorf("get bank: %w", err)
		}
		bank.Balance = amount
		return s.save(ctx, userID, bank)
	})
}

func (s *Service) Delete(ctx context.Context, userID domain.UserID, bankID int64) error {
	if err := s.tx.InTx(ctx, func(ctx context.Context) error {
		bank, err := s.banks.GetByID(ctx, userID, bankID)
		if err != nil {
			return fmt.Errorf("get bank: %w", err)
		}
		if err := s.appendOp(ctx, domain.Operation{
			UserID:    userID,
			BankID:    ptr(bank.ID),
			Type:      domain.OperationDelete,
			Amount:    bank.Balance,
			Meta:      bank.Name,
			CreatedAt: s.clock.Now(),
		}); err != nil {
			return err
		}
		if err := s.banks.Delete(ctx, userID, bankID); err != nil {
			return fmt.Errorf("delete bank: %w", err)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("delete: %w", err)
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

func (s *Service) Rename(ctx context.Context, userID domain.UserID, bankID int64, newName string) (domain.Bank, error) {
	newName = strings.TrimSpace(newName)
	if _, err := domain.NormalizeBankName(newName); err != nil {
		return domain.Bank{}, fmt.Errorf("bank name: %w", err)
	}

	var updated domain.Bank
	err := s.tx.InTx(ctx, func(ctx context.Context) error {
		bank, err := s.banks.GetByID(ctx, userID, bankID)
		if err != nil {
			return fmt.Errorf("get bank: %w", err)
		}
		existing, err := s.banks.GetByName(ctx, userID, newName)
		if err == nil && existing.ID != bank.ID {
			return domain.ErrBankNameTaken
		}
		if err != nil && !errors.Is(err, domain.ErrBankNotFound) {
			return fmt.Errorf("check bank name: %w", err)
		}
		oldName := bank.Name
		bank.Name = newName
		updated, err = s.save(ctx, userID, bank)
		if err != nil {
			return err
		}
		return s.appendOp(ctx, domain.Operation{
			UserID:       userID,
			BankID:       ptr(updated.ID),
			Type:         domain.OperationRename,
			Amount:       0,
			BalanceAfter: ptr(updated.Balance),
			Meta:         oldName + " -> " + newName,
			CreatedAt:    updated.UpdatedAt,
		})
	})
	if err != nil {
		return domain.Bank{}, fmt.Errorf("rename: %w", err)
	}
	return updated, nil
}

func (s *Service) Transfer(
	ctx context.Context,
	userID domain.UserID,
	fromID, toID int64,
	amount domain.Money,
) (domain.Bank, domain.Bank, error) {
	if amount <= 0 {
		return domain.Bank{}, domain.Bank{}, domain.ErrInvalidAmount
	}
	if fromID == toID {
		return domain.Bank{}, domain.Bank{}, domain.ErrSameBank
	}

	var from, to domain.Bank
	err := s.tx.InTx(ctx, func(ctx context.Context) error {
		var err error
		from, err = s.banks.GetByID(ctx, userID, fromID)
		if err != nil {
			return fmt.Errorf("get from bank: %w", err)
		}
		to, err = s.banks.GetByID(ctx, userID, toID)
		if err != nil {
			return fmt.Errorf("get to bank: %w", err)
		}
		if from.Currency != to.Currency {
			return domain.ErrCurrencyMismatch
		}
		fromBal, err := addMoney(from.Balance, -amount)
		if err != nil {
			return err
		}
		toBal, err := addMoney(to.Balance, amount)
		if err != nil {
			return err
		}
		from.Balance = fromBal
		from, err = s.save(ctx, userID, from)
		if err != nil {
			return err
		}
		to.Balance = toBal
		to, err = s.save(ctx, userID, to)
		if err != nil {
			return err
		}
		return s.appendOp(ctx, domain.Operation{
			UserID:       userID,
			BankID:       ptr(from.ID),
			Type:         domain.OperationTransfer,
			Amount:       amount,
			BalanceAfter: ptr(from.Balance),
			Meta:         strconv.FormatInt(to.ID, 10),
			CreatedAt:    from.UpdatedAt,
		})
	})
	if err != nil {
		return domain.Bank{}, domain.Bank{}, fmt.Errorf("transfer: %w", err)
	}
	return from, to, nil
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

func (s *Service) changeBalance(
	ctx context.Context,
	userID domain.UserID,
	typ domain.OperationType,
	amount domain.Money,
	mutate func(ctx context.Context) (domain.Bank, error),
) (domain.Bank, error) {
	var updated domain.Bank
	err := s.tx.InTx(ctx, func(ctx context.Context) error {
		var err error
		updated, err = mutate(ctx)
		if err != nil {
			return err
		}
		return s.appendOp(ctx, domain.Operation{
			UserID:       userID,
			BankID:       ptr(updated.ID),
			Type:         typ,
			Amount:       amount,
			BalanceAfter: ptr(updated.Balance),
			Meta:         updated.Name,
			CreatedAt:    updated.UpdatedAt,
		})
	})
	if err != nil {
		return domain.Bank{}, fmt.Errorf("change balance: %w", err)
	}
	return updated, nil
}

func (s *Service) appendOp(ctx context.Context, op domain.Operation) error {
	if err := s.ops.Append(ctx, op); err != nil {
		return fmt.Errorf("append operation: %w", err)
	}
	return nil
}

func ptr[T any](v T) *T {
	return &v
}
