package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"finbot/internal/domain"
	"finbot/internal/service"
)

var _ service.OperationRepository = (*OperationRepository)(nil)

type OperationRepository struct {
	db *sql.DB
}

func NewOperationRepository(db *sql.DB) *OperationRepository {
	return &OperationRepository{db: db}
}

func (r *OperationRepository) Append(ctx context.Context, op domain.Operation) error {
	_, err := conn(ctx, r.db).ExecContext(ctx, `
		INSERT INTO operations (
			user_id, bank_id, type, amount_cents, balance_after_cents, meta, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		op.UserID,
		nullBankID(op.BankID),
		string(op.Type),
		int64(op.Amount),
		nullMoney(op.BalanceAfter),
		nullString(op.Meta),
		utcTime(op.CreatedAt),
	)
	if err != nil {
		return fmt.Errorf("append operation: %w", err)
	}
	return nil
}
