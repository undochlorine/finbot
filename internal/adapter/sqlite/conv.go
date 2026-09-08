package sqlite

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	moderncsqlite "modernc.org/sqlite"
	sqlite3lib "modernc.org/sqlite/lib"

	"finbot/internal/domain"
)

func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

func parseTime(raw string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse time: %w", err)
	}
	return t.UTC(), nil
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func nullInt(p *int) sql.NullInt64 {
	if p == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(*p), Valid: true}
}

func intPtr(n sql.NullInt64) *int {
	if !n.Valid {
		return nil
	}
	v := int(n.Int64)
	return &v
}

func nullUserID(p *domain.UserID) sql.NullInt64 {
	if p == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(*p), Valid: true}
}

func userIDPtr(n sql.NullInt64) *domain.UserID {
	if !n.Valid {
		return nil
	}
	v := domain.UserID(n.Int64)
	return &v
}

func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func nullMoney(p *domain.Money) sql.NullInt64 {
	if p == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(*p), Valid: true}
}

func nullBankID(p *int64) sql.NullInt64 {
	if p == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *p, Valid: true}
}

func isUniqueConstraint(err error) bool {
	var se *moderncsqlite.Error
	if !errors.As(err, &se) {
		return false
	}
	if se.Code() == sqlite3lib.SQLITE_CONSTRAINT_UNIQUE {
		return true
	}
	return se.Code() == sqlite3lib.SQLITE_CONSTRAINT && strings.Contains(se.Error(), "UNIQUE")
}

func mapUnique(err error) error {
	if isUniqueConstraint(err) {
		return domain.ErrBankNameTaken
	}
	return err
}
