package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"finbot/internal/domain"
	"finbot/internal/service"
)

var _ service.UserRepository = (*UserRepository)(nil)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Get(ctx context.Context, userID domain.UserID) (domain.User, error) {
	row := conn(ctx, r.db).QueryRowContext(ctx, `
		SELECT telegram_id, username, last_activity_at, plan, trial_ends_at, discount_percent,
		       locale, referred_by, created_at, updated_at
		FROM users
		WHERE telegram_id = $1`, userID)
	user, err := scanUser(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.User{}, domain.ErrUserNotFound
	}
	if err != nil {
		return domain.User{}, fmt.Errorf("get user: %w", err)
	}
	return user, nil
}

func (r *UserRepository) Upsert(ctx context.Context, user domain.User) (domain.User, error) {
	_, err := conn(ctx, r.db).ExecContext(ctx, `
		INSERT INTO users (
			telegram_id, username, last_activity_at, plan, trial_ends_at, discount_percent,
			locale, referred_by, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (telegram_id) DO UPDATE SET
			username = excluded.username,
			last_activity_at = excluded.last_activity_at,
			updated_at = excluded.updated_at`,
		user.TelegramID,
		user.Username,
		utcTime(user.LastActivityAt),
		user.Plan,
		utcTime(user.TrialEndsAt),
		nullInt(user.DiscountPercent),
		user.Locale,
		nullUserID(user.ReferredBy),
		utcTime(user.CreatedAt),
		utcTime(user.UpdatedAt),
	)
	if err != nil {
		return domain.User{}, fmt.Errorf("upsert user: %w", err)
	}
	return r.Get(ctx, user.TelegramID)
}

func (r *UserRepository) TouchActivity(ctx context.Context, userID domain.UserID, at time.Time) error {
	res, err := conn(ctx, r.db).ExecContext(ctx, `
		UPDATE users
		SET last_activity_at = $1, updated_at = $2
		WHERE telegram_id = $3`, utcTime(at), utcTime(at), userID)
	if err != nil {
		return fmt.Errorf("touch activity: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("touch activity rows: %w", err)
	}
	if n == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}

func (r *UserRepository) SetReferredByIfEmpty(ctx context.Context, userID, referredBy domain.UserID) error {
	if referredBy == userID {
		return nil
	}
	res, err := conn(ctx, r.db).ExecContext(ctx, `
		UPDATE users
		SET referred_by = $1
		WHERE telegram_id = $2 AND referred_by IS NULL`, referredBy, userID)
	if err != nil {
		return fmt.Errorf("set referred by: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("set referred by rows: %w", err)
	}
	if n > 0 {
		return nil
	}
	if _, err := r.Get(ctx, userID); err != nil {
		return err
	}
	return nil
}

func (r *UserRepository) SetLocale(ctx context.Context, userID domain.UserID, locale string, at time.Time) error {
	res, err := conn(ctx, r.db).ExecContext(ctx, `
		UPDATE users
		SET locale = $1, updated_at = $2
		WHERE telegram_id = $3`, locale, utcTime(at), userID)
	if err != nil {
		return fmt.Errorf("set locale: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("set locale rows: %w", err)
	}
	if n == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}

func scanUser(row *sql.Row) (domain.User, error) {
	var (
		u          domain.User
		username   sql.NullString
		discount   sql.NullInt64
		referredBy sql.NullInt64
	)
	err := row.Scan(
		&u.TelegramID, &username, &u.LastActivityAt, &u.Plan, &u.TrialEndsAt, &discount,
		&u.Locale, &referredBy, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return domain.User{}, fmt.Errorf("scan user: %w", err)
	}
	u.Username = username.String
	u.DiscountPercent = intPtr(discount)
	u.ReferredBy = userIDPtr(referredBy)
	u.LastActivityAt = utcTime(u.LastActivityAt)
	u.TrialEndsAt = utcTime(u.TrialEndsAt)
	u.CreatedAt = utcTime(u.CreatedAt)
	u.UpdatedAt = utcTime(u.UpdatedAt)
	return u, nil
}
