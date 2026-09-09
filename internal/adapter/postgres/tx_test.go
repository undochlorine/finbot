package postgres

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestRunWithRetry(t *testing.T) {
	deadlock := &pgconn.PgError{Code: "40P01"}
	other := errors.New("boom")

	t.Run("retries deadlock then succeeds", func(t *testing.T) {
		var n int
		err := runWithRetry(maxTxAttempts, func() error {
			n++
			if n < maxTxAttempts {
				return deadlock
			}
			return nil
		})
		if err != nil {
			t.Fatalf("got %v", err)
		}
		if n != maxTxAttempts {
			t.Fatalf("attempts %d, want %d", n, maxTxAttempts)
		}
	})

	t.Run("does not retry other errors", func(t *testing.T) {
		var n int
		err := runWithRetry(maxTxAttempts, func() error {
			n++
			return other
		})
		if !errors.Is(err, other) {
			t.Fatalf("got %v, want %v", err, other)
		}
		if n != 1 {
			t.Fatalf("attempts %d, want 1", n)
		}
	})

	t.Run("returns last retryable error", func(t *testing.T) {
		err := runWithRetry(maxTxAttempts, func() error {
			return fmt.Errorf("tx: %w", deadlock)
		})
		if !retryableSQLState(err) {
			t.Fatalf("got %v", err)
		}
	})
}
