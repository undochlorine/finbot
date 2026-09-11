package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	"finbot/internal/domain"
	"finbot/internal/service/mocks"
)

const (
	userA domain.UserID = 1
	userB domain.UserID = 2
)

func fixedNow() time.Time {
	return time.Date(2026, time.September, 6, 12, 0, 0, 0, time.UTC)
}

func newBankSvc(t *testing.T) (*Service, *mocks.MockBankRepository, *mocks.MockOperationRepository, *mocks.MockClock) {
	t.Helper()
	banks := mocks.NewMockBankRepository(t)
	ops := mocks.NewMockOperationRepository(t)
	clock := mocks.NewMockClock(t)
	return New(banks, mocks.NewMockUserRepository(t), ops, passthroughTx(t), clock, testCfg(168*time.Hour, "USD")), banks, ops, clock
}

func testCfg(trial time.Duration, currency string) Config {
	return Config{
		Trial:   TrialConfig{Duration: trial.String()},
		Default: DefaultConfig{Currency: currency},
	}
}

func passthroughTx(t *testing.T) *mocks.MockTransactor {
	t.Helper()
	tx := mocks.NewMockTransactor(t)
	tx.EXPECT().InTx(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Maybe()
	return tx
}

func sampleBank(user domain.UserID, id int64, name string, bal domain.Money, include bool) domain.Bank {
	now := fixedNow()
	return domain.Bank{
		ID:             id,
		UserID:         user,
		Name:           name,
		Balance:        bal,
		IncludeInTotal: include,
		Currency:       "USD",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

func matchOp(
	user domain.UserID,
	typ domain.OperationType,
	amount domain.Money,
	bankID int64,
	after *domain.Money,
	meta string,
	at time.Time,
) any {
	return mock.MatchedBy(func(op domain.Operation) bool {
		if op.UserID != user || op.Type != typ || op.Amount != amount || op.Meta != meta || !op.CreatedAt.Equal(at) {
			return false
		}
		if op.BankID == nil || *op.BankID != bankID {
			return false
		}
		if after == nil {
			return op.BalanceAfter == nil
		}
		return op.BalanceAfter != nil && *op.BalanceAfter == *after
	})
}
