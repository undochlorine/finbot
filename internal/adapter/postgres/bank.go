package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"finbot/internal/domain"
	"finbot/internal/service"
)

var _ service.BankRepository = (*BankRepository)(nil)

const selectBank = `
		SELECT id, user_id, name, balance_cents, include_in_total, currency, created_at, updated_at
		FROM banks`

type BankRepository struct {
	db *sql.DB
}

func NewBankRepository(db *sql.DB) *BankRepository {
	return &BankRepository{db: db}
}

func (r *BankRepository) Create(ctx context.Context, userID domain.UserID, bank domain.Bank) (domain.Bank, error) {
	norm, err := domain.NormalizeBankName(bank.Name)
	if err != nil {
		return domain.Bank{}, fmt.Errorf("bank name: %w", err)
	}
	row := conn(ctx, r.db).QueryRowContext(ctx, `
		INSERT INTO banks (user_id, name, name_normalized, balance_cents, include_in_total, currency, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, COALESCE(NULLIF($6, ''), 'USD'), $7, $8)
		RETURNING id, user_id, name, balance_cents, include_in_total, currency, created_at, updated_at`,
		userID,
		bank.Name,
		norm,
		int64(bank.Balance),
		bank.IncludeInTotal,
		bank.Currency,
		utcTime(bank.CreatedAt),
		utcTime(bank.UpdatedAt),
	)
	created, err := scanBank(row)
	if err != nil {
		if mapped := mapUnique(err); mapped != err {
			return domain.Bank{}, mapped
		}
		return domain.Bank{}, fmt.Errorf("create bank: %w", err)
	}
	return created, nil
}

func (r *BankRepository) GetByID(ctx context.Context, userID domain.UserID, bankID int64) (domain.Bank, error) {
	row := conn(ctx, r.db).QueryRowContext(ctx, selectBank+`
		WHERE id = $1 AND user_id = $2`+forUpdate(ctx), bankID, userID)
	return scanBankRow(row, "get bank")
}

func (r *BankRepository) GetByName(ctx context.Context, userID domain.UserID, name string) (domain.Bank, error) {
	norm, err := domain.NormalizeBankName(name)
	if err != nil {
		return domain.Bank{}, fmt.Errorf("bank name: %w", err)
	}
	row := conn(ctx, r.db).QueryRowContext(ctx, selectBank+`
		WHERE user_id = $1 AND name_normalized = $2`+forUpdate(ctx), userID, norm)
	return scanBankRow(row, "get bank by name")
}

func (r *BankRepository) List(ctx context.Context, userID domain.UserID) (banks []domain.Bank, err error) {
	rows, err := conn(ctx, r.db).QueryContext(ctx, selectBank+`
		WHERE user_id = $1
		ORDER BY name_normalized`, userID)
	if err != nil {
		return nil, fmt.Errorf("list banks: %w", err)
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("list banks close: %w", cerr)
		}
	}()

	banks = make([]domain.Bank, 0)
	for rows.Next() {
		var bank domain.Bank
		bank, err = scanBank(rows)
		if err != nil {
			return nil, fmt.Errorf("list banks: %w", err)
		}
		banks = append(banks, bank)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("list banks: %w", err)
	}
	return banks, nil
}

func (r *BankRepository) Update(ctx context.Context, userID domain.UserID, bank domain.Bank) (domain.Bank, error) {
	norm, err := domain.NormalizeBankName(bank.Name)
	if err != nil {
		return domain.Bank{}, fmt.Errorf("bank name: %w", err)
	}
	row := conn(ctx, r.db).QueryRowContext(ctx, `
		UPDATE banks
		SET name = $1, name_normalized = $2, balance_cents = $3, include_in_total = $4, updated_at = $5
		WHERE id = $6 AND user_id = $7
		RETURNING id, user_id, name, balance_cents, include_in_total, currency, created_at, updated_at`,
		bank.Name,
		norm,
		int64(bank.Balance),
		bank.IncludeInTotal,
		utcTime(bank.UpdatedAt),
		bank.ID,
		userID,
	)
	updated, err := scanBank(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Bank{}, domain.ErrBankNotFound
	}
	if err != nil {
		if mapped := mapUnique(err); mapped != err {
			return domain.Bank{}, mapped
		}
		return domain.Bank{}, fmt.Errorf("update bank: %w", err)
	}
	return updated, nil
}

func (r *BankRepository) Delete(ctx context.Context, userID domain.UserID, bankID int64) error {
	res, err := conn(ctx, r.db).ExecContext(ctx, `
		DELETE FROM banks
		WHERE id = $1 AND user_id = $2`, bankID, userID)
	if err != nil {
		return fmt.Errorf("delete bank: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete bank rows: %w", err)
	}
	if n == 0 {
		return domain.ErrBankNotFound
	}
	return nil
}

func (r *BankRepository) TotalIncluded(ctx context.Context, userID domain.UserID) (domain.Money, error) {
	var cents int64
	err := conn(ctx, r.db).QueryRowContext(ctx, `
		SELECT COALESCE(SUM(balance_cents), 0)
		FROM banks
		WHERE user_id = $1 AND include_in_total = TRUE`, userID).Scan(&cents)
	if err != nil {
		return 0, fmt.Errorf("total included: %w", err)
	}
	return domain.Money(cents), nil
}

func scanBankRow(row *sql.Row, op string) (domain.Bank, error) {
	bank, err := scanBank(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Bank{}, domain.ErrBankNotFound
	}
	if err != nil {
		return domain.Bank{}, scanErr(op, err)
	}
	return bank, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanBank(row rowScanner) (domain.Bank, error) {
	var (
		b     domain.Bank
		cents int64
	)
	err := row.Scan(&b.ID, &b.UserID, &b.Name, &cents, &b.IncludeInTotal, &b.Currency, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		return domain.Bank{}, fmt.Errorf("scan bank: %w", err)
	}
	b.Balance = domain.Money(cents)
	b.CreatedAt = utcTime(b.CreatedAt)
	b.UpdatedAt = utcTime(b.UpdatedAt)
	return b, nil
}
