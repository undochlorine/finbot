package postgres

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	"finbot/internal/domain"
)

func TestMapUnique(t *testing.T) {
	unique := &pgconn.PgError{Code: "23505"}
	tests := []struct {
		name string
		err  error
		want error
	}{
		{name: "unique violation", err: unique, want: domain.ErrBankNameTaken},
		{name: "wrapped unique", err: fmt.Errorf("insert: %w", unique), want: domain.ErrBankNameTaken},
		{name: "deadlock", err: &pgconn.PgError{Code: "40P01"}},
		{name: "plain", err: errors.New("boom")},
		{name: "nil", err: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapUnique(tt.err)
			if tt.want != nil {
				if !errors.Is(got, tt.want) {
					t.Fatalf("got %v, want %v", got, tt.want)
				}
				return
			}
			if got != tt.err {
				t.Fatalf("got %v, want original %v", got, tt.err)
			}
		})
	}
}

func TestRetryableSQLState(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "deadlock", err: &pgconn.PgError{Code: "40P01"}, want: true},
		{name: "serialization", err: &pgconn.PgError{Code: "40001"}, want: true},
		{name: "wrapped deadlock", err: fmt.Errorf("get: %w", &pgconn.PgError{Code: "40P01"}), want: true},
		{name: "unique", err: &pgconn.PgError{Code: "23505"}},
		{name: "plain", err: errors.New("boom")},
		{name: "nil"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := retryableSQLState(tt.err); got != tt.want {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}
