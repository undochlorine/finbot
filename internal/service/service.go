package service

import (
	"context"
	"time"

	"finbot/internal/domain"
)

type Service struct {
	banks           BankRepository
	users           UserRepository
	ops             OperationRepository
	tx              Transactor
	clock           Clock
	trialDuration   time.Duration
	defaultCurrency string
}

func New(
	banks BankRepository,
	users UserRepository,
	ops OperationRepository,
	tx Transactor,
	clock Clock,
	cfg Config,
) *Service {
	trial, err := cfg.Trial.Parsed()
	if err != nil {
		trial = 0
	}
	return &Service{
		banks:           banks,
		users:           users,
		ops:             ops,
		tx:              tx,
		clock:           clock,
		trialDuration:   trial,
		defaultCurrency: cfg.Default.Parsed(),
	}
}

type OperationRepository interface {
	Append(ctx context.Context, op domain.Operation) error
}

type Transactor interface {
	InTx(ctx context.Context, fn func(ctx context.Context) error) error
}
