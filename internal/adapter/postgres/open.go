package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

const (
	MaxOpenConns    = 10
	MaxIdleConns    = 5
	ConnMaxLifetime = 30 * time.Minute
	ConnMaxIdleTime = 5 * time.Minute
)

func Open(ctx context.Context, url string) (*sql.DB, error) {
	url = strings.TrimSpace(url)
	if url == "" {
		return nil, fmt.Errorf("postgres url is required")
	}

	cfg, err := pgx.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("parse postgres url: %w", err)
	}
	db := stdlib.OpenDB(*cfg, stdlib.OptionAfterConnect(setSessionTimeouts))
	db.SetMaxOpenConns(MaxOpenConns)
	db.SetMaxIdleConns(MaxIdleConns)
	db.SetConnMaxLifetime(ConnMaxLifetime)
	db.SetConnMaxIdleTime(ConnMaxIdleTime)

	if err := db.PingContext(ctx); err != nil {
		return nil, closeWith(db, fmt.Errorf("ping postgres: %w", err))
	}
	if err := migrate(ctx, db); err != nil {
		return nil, closeWith(db, err)
	}
	return db, nil
}

func setSessionTimeouts(ctx context.Context, conn *pgx.Conn) error {
	stmts := []string{
		"SET statement_timeout = '5s'",
		"SET lock_timeout = '2s'",
		"SET idle_in_transaction_session_timeout = '10s'",
	}
	for _, stmt := range stmts {
		if _, err := conn.Exec(ctx, stmt); err != nil {
			return fmt.Errorf("%s: %w", stmt, err)
		}
	}
	return nil
}
