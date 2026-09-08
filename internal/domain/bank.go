package domain

import (
	"strings"
	"time"
)

type Bank struct {
	ID             int64
	UserID         UserID
	Name           string
	Balance        Money
	IncludeInTotal bool
	Currency       string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func NormalizeBankName(name string) (string, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "", ErrInvalidBankName
	}
	return strings.ToLower(trimmed), nil
}
