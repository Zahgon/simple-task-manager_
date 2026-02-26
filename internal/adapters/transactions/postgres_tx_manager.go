package transactions

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/SilentPlaces/simple-task-manager/internal/database"
	"github.com/SilentPlaces/simple-task-manager/internal/domain/ports/transactions"
)

type PostgresTxManager struct {
	db *sql.DB
}

func NewPostgresTxManager(db *sql.DB) transactions.TxManager {
	return &PostgresTxManager{db: db}
}

func (m *PostgresTxManager) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback()

	txCtx := database.WithTx(ctx, tx)

	if err := fn(txCtx); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}
	return nil
}
