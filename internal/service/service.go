package service

import (
	"time"

	"finbot/internal/ports"
)

type Service struct {
	banks         ports.BankRepository
	users         ports.UserRepository
	clock         ports.Clock
	trialDuration time.Duration
}

func New(
	banks ports.BankRepository,
	users ports.UserRepository,
	clock ports.Clock,
	trialDuration time.Duration,
) *Service {
	return &Service{
		banks:         banks,
		users:         users,
		clock:         clock,
		trialDuration: trialDuration,
	}
}
