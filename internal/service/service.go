package service

import "time"

type Service struct {
	banks         BankRepository
	users         UserRepository
	clock         Clock
	trialDuration time.Duration
}

func New(
	banks BankRepository,
	users UserRepository,
	clock Clock,
	trialDuration time.Duration,
) *Service {
	return &Service{
		banks:         banks,
		users:         users,
		clock:         clock,
		trialDuration: trialDuration,
	}
}
