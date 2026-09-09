package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"finbot/internal/service"
)

var _ service.Transactor = (*Transactor)(nil)

const maxTxAttempts = 3

type txKey struct{}

type dbConn interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type Transactor struct {
	db *sql.DB
}

func NewTransactor(db *sql.DB) *Transactor {
	return &Transactor{db: db}
}

func (t *Transactor) InTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return runWithRetry(maxTxAttempts, func() error {
		return t.oneTx(ctx, fn)
	})
}

func (t *Transactor) oneTx(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := t.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer rollbackTx(tx)

	if err := fn(context.WithValue(ctx, txKey{}, tx)); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

func runWithRetry(attempts int, fn func() error) error {
	var err error
	for range attempts {
		err = fn()
		if err == nil || !retryableSQLState(err) {
			return err
		}
	}
	return err
}

func conn(ctx context.Context, db *sql.DB) dbConn {
	if tx, ok := ctx.Value(txKey{}).(*sql.Tx); ok {
		return tx
	}
	return db
}

func forUpdate(ctx context.Context) string {
	if _, ok := ctx.Value(txKey{}).(*sql.Tx); ok {
		return " FOR UPDATE"
	}
	return ""
}
