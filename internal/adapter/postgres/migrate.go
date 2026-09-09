package postgres

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"path"
	"sort"
	"strings"
	"time"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func migrate(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL
		)`); err != nil {
		return fmt.Errorf("schema_migrations: %w", err)
	}

	names, err := fs.Glob(migrationsFS, "migrations/*.sql")
	if err != nil {
		return fmt.Errorf("list migrations: %w", err)
	}
	sort.Strings(names)

	for _, name := range names {
		version := strings.TrimSuffix(path.Base(name), ".sql")
		ok, err := applied(ctx, db, version)
		if err != nil {
			return err
		}
		if ok {
			continue
		}
		body, err := migrationsFS.ReadFile(name)
		if err != nil {
			return fmt.Errorf("read %s: %w", name, err)
		}
		if err := applyMigration(ctx, db, version, string(body)); err != nil {
			return err
		}
	}
	return nil
}

func applied(ctx context.Context, db *sql.DB, version string) (bool, error) {
	var n int
	err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM schema_migrations WHERE version = $1`, version,
	).Scan(&n)
	if err != nil {
		return false, fmt.Errorf("check migration %s: %w", version, err)
	}
	return n > 0, nil
}

func applyMigration(ctx context.Context, db *sql.DB, version, script string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin %s: %w", version, err)
	}
	defer rollbackTx(tx)

	for _, stmt := range splitStatements(script) {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("migration %s: %w", version, err)
		}
	}

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO schema_migrations (version, applied_at) VALUES ($1, $2)`,
		version, time.Now().UTC(),
	); err != nil {
		return fmt.Errorf("record %s: %w", version, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit %s: %w", version, err)
	}
	return nil
}

func closeWith(db *sql.DB, err error) error {
	if cerr := db.Close(); cerr != nil {
		return fmt.Errorf("%w: close: %w", err, cerr)
	}
	return err
}

func rollbackTx(tx *sql.Tx) {
	if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
		slog.Error("rollback postgres", slog.Any("err", err))
	}
}

func splitStatements(script string) []string {
	parts := strings.Split(script, ";")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		stmt := strings.TrimSpace(part)
		if stmt == "" {
			continue
		}
		out = append(out, stmt)
	}
	return out
}
