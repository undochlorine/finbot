package domain

import "time"

type OperationType string

const (
	OperationAdd      OperationType = "add"
	OperationSpend    OperationType = "spend"
	OperationSet      OperationType = "set"
	OperationDelete   OperationType = "delete"
	OperationTransfer OperationType = "transfer"
)

type Operation struct {
	ID           int64
	UserID       UserID
	BankID       *int64
	Type         OperationType
	Amount       Money
	BalanceAfter *Money
	Meta         string
	CreatedAt    time.Time
}
