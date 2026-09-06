package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"finbot/internal/domain"
	"finbot/internal/ports"
)

var _ ports.BankRepository = (*BankRepository)(nil)

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
	res, err := r.db.ExecContext(ctx, `
		INSERT INTO banks (user_id, name, name_normalized, balance_cents, include_in_total, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		userID,
		bank.Name,
		norm,
		int64(bank.Balance),
		boolToInt(bank.IncludeInTotal),
		formatTime(bank.CreatedAt),
		formatTime(bank.UpdatedAt),
	)
	if err != nil {
		if mapped := mapUnique(err); mapped != err {
			return domain.Bank{}, mapped
		}
		return domain.Bank{}, fmt.Errorf("create bank: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return domain.Bank{}, fmt.Errorf("create bank id: %w", err)
	}
	return r.GetByID(ctx, userID, id)
}

func (r *BankRepository) GetByID(ctx context.Context, userID domain.UserID, bankID int64) (domain.Bank, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, user_id, name, balance_cents, include_in_total, created_at, updated_at
		FROM banks
		WHERE id = ? AND user_id = ?`, bankID, userID)
	return scanBankRow(row, "get bank")
}

func (r *BankRepository) GetByName(ctx context.Context, userID domain.UserID, name string) (domain.Bank, error) {
	norm, err := domain.NormalizeBankName(name)
	if err != nil {
		return domain.Bank{}, fmt.Errorf("bank name: %w", err)
	}
	row := r.db.QueryRowContext(ctx, `
		SELECT id, user_id, name, balance_cents, include_in_total, created_at, updated_at
		FROM banks
		WHERE user_id = ? AND name_normalized = ?`, userID, norm)
	return scanBankRow(row, "get bank by name")
}

func (r *BankRepository) List(ctx context.Context, userID domain.UserID) (banks []domain.Bank, err error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_id, name, balance_cents, include_in_total, created_at, updated_at
		FROM banks
		WHERE user_id = ?
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
	res, err := r.db.ExecContext(ctx, `
		UPDATE banks
		SET name = ?, name_normalized = ?, balance_cents = ?, include_in_total = ?, updated_at = ?
		WHERE id = ? AND user_id = ?`,
		bank.Name,
		norm,
		int64(bank.Balance),
		boolToInt(bank.IncludeInTotal),
		formatTime(bank.UpdatedAt),
		bank.ID,
		userID,
	)
	if err != nil {
		if mapped := mapUnique(err); mapped != err {
			return domain.Bank{}, mapped
		}
		return domain.Bank{}, fmt.Errorf("update bank: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return domain.Bank{}, fmt.Errorf("update bank rows: %w", err)
	}
	if n == 0 {
		return domain.Bank{}, domain.ErrBankNotFound
	}
	return r.GetByID(ctx, userID, bank.ID)
}

func (r *BankRepository) Delete(ctx context.Context, userID domain.UserID, bankID int64) error {
	res, err := r.db.ExecContext(ctx, `
		DELETE FROM banks
		WHERE id = ? AND user_id = ?`, bankID, userID)
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
	err := r.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(balance_cents), 0)
		FROM banks
		WHERE user_id = ? AND include_in_total = 1`, userID).Scan(&cents)
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
		return domain.Bank{}, fmt.Errorf("%s: %w", op, err)
	}
	return bank, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanBank(row rowScanner) (domain.Bank, error) {
	var (
		b         domain.Bank
		cents     int64
		included  int
		createdAt string
		updatedAt string
	)
	err := row.Scan(&b.ID, &b.UserID, &b.Name, &cents, &included, &createdAt, &updatedAt)
	if err != nil {
		return domain.Bank{}, fmt.Errorf("scan bank: %w", err)
	}
	b.Balance = domain.Money(cents)
	b.IncludeInTotal = included == 1
	if b.CreatedAt, err = parseTime(createdAt); err != nil {
		return domain.Bank{}, fmt.Errorf("created_at: %w", err)
	}
	if b.UpdatedAt, err = parseTime(updatedAt); err != nil {
		return domain.Bank{}, fmt.Errorf("updated_at: %w", err)
	}
	return b, nil
}
