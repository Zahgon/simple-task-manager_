package database

import (
	"context"
	"database/sql"
)

// Row is the minimal interface for a query row. Allows adapters to wrap *sql.Row
// (e.g. for metrics) while keeping compatibility with standard usage.
type Row interface {
	Scan(dest ...any) error
}

type Executor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) Row
}

type txKey struct{}

func WithTx(ctx context.Context, tx *sql.Tx) context.Context {
	return context.WithValue(ctx, txKey{}, tx)
}

func GetExecutor(ctx context.Context, db *sql.DB) Executor {
	if tx, ok := ctx.Value(txKey{}).(*sql.Tx); ok {
		return &txExecutor{tx}
	}
	return &dbExecutor{db}
}

// dbExecutor and txExecutor adapt *sql.DB and *sql.Tx to Executor (returning Row).
type dbExecutor struct{ *sql.DB }
type txExecutor struct{ *sql.Tx }

func (d *dbExecutor) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return d.DB.ExecContext(ctx, query, args...)
}
func (d *dbExecutor) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return d.DB.QueryContext(ctx, query, args...)
}
func (d *dbExecutor) QueryRowContext(ctx context.Context, query string, args ...any) Row {
	return d.DB.QueryRowContext(ctx, query, args...)
}

func (t *txExecutor) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return t.Tx.ExecContext(ctx, query, args...)
}
func (t *txExecutor) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return t.Tx.QueryContext(ctx, query, args...)
}
func (t *txExecutor) QueryRowContext(ctx context.Context, query string, args ...any) Row {
	return t.Tx.QueryRowContext(ctx, query, args...)
}
