package domain

import "errors"

var (
	ErrBankNotFound     = errors.New("bank not found")
	ErrBankNameTaken    = errors.New("bank name already exists")
	ErrInvalidAmount    = errors.New("invalid amount")
	ErrInvalidBankName  = errors.New("invalid bank name")
	ErrSameBank         = errors.New("same bank")
	ErrCurrencyMismatch = errors.New("currency mismatch")
	ErrUserNotFound     = errors.New("user not found")
)
