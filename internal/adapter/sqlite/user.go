package sqlite

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
		WHERE telegram_id = ?`, userID)
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
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (telegram_id) DO UPDATE SET
			username = excluded.username,
			last_activity_at = excluded.last_activity_at,
			updated_at = excluded.updated_at`,
		user.TelegramID,
		user.Username,
		formatTime(user.LastActivityAt),
		user.Plan,
		formatTime(user.TrialEndsAt),
		nullInt(user.DiscountPercent),
		user.Locale,
		nullUserID(user.ReferredBy),
		formatTime(user.CreatedAt),
		formatTime(user.UpdatedAt),
	)
	if err != nil {
		return domain.User{}, fmt.Errorf("upsert user: %w", err)
	}
	return r.Get(ctx, user.TelegramID)
}

func (r *UserRepository) TouchActivity(ctx context.Context, userID domain.UserID, at time.Time) error {
	res, err := conn(ctx, r.db).ExecContext(ctx, `
		UPDATE users
		SET last_activity_at = ?, updated_at = ?
		WHERE telegram_id = ?`, formatTime(at), formatTime(at), userID)
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
		SET referred_by = ?
		WHERE telegram_id = ? AND referred_by IS NULL`, referredBy, userID)
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

func scanUser(row *sql.Row) (domain.User, error) {
	var (
		u          domain.User
		username   sql.NullString
		activity   string
		trialEnds  string
		discount   sql.NullInt64
		referredBy sql.NullInt64
		createdAt  string
		updatedAt  string
	)
	err := row.Scan(
		&u.TelegramID, &username, &activity, &u.Plan, &trialEnds, &discount,
		&u.Locale, &referredBy, &createdAt, &updatedAt,
	)
	if err != nil {
		return domain.User{}, fmt.Errorf("scan user: %w", err)
	}
	u.Username = username.String
	u.DiscountPercent = intPtr(discount)
	u.ReferredBy = userIDPtr(referredBy)
	if u.LastActivityAt, err = parseTime(activity); err != nil {
		return domain.User{}, fmt.Errorf("last_activity_at: %w", err)
	}
	if u.TrialEndsAt, err = parseTime(trialEnds); err != nil {
		return domain.User{}, fmt.Errorf("trial_ends_at: %w", err)
	}
	if u.CreatedAt, err = parseTime(createdAt); err != nil {
		return domain.User{}, fmt.Errorf("created_at: %w", err)
	}
	if u.UpdatedAt, err = parseTime(updatedAt); err != nil {
		return domain.User{}, fmt.Errorf("updated_at: %w", err)
	}
	return u, nil
}
