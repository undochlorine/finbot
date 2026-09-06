package service

import (
	"testing"
	"time"

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

func newBankSvc(t *testing.T) (*Service, *mocks.MockBankRepository, *mocks.MockClock) {
	t.Helper()
	banks := mocks.NewMockBankRepository(t)
	clock := mocks.NewMockClock(t)
	return New(banks, mocks.NewMockUserRepository(t), clock, 168*time.Hour), banks, clock
}

func sampleBank(user domain.UserID, id int64, name string, bal domain.Money, include bool) domain.Bank {
	now := fixedNow()
	return domain.Bank{
		ID:             id,
		UserID:         user,
		Name:           name,
		Balance:        bal,
		IncludeInTotal: include,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}
