package database

import (
	"context"
	"database/sql"
	"fmt"
)

// TxFunc is the callback executed inside a managed transaction.
// Returning a non-nil error triggers a rollback; returning nil triggers a commit.
type TxFunc func(ctx context.Context, tx *sql.Tx) error

// WithTransaction runs fn inside a *sql.Tx.
// It commits on success and rolls back on any error or panic.
// Rollback errors are wrapped together with the original error so nothing is silently swallowed.
func WithTransaction(ctx context.Context, db *sql.DB, fn TxFunc) (err error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
		if err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				err = fmt.Errorf("rollback error: %w (original: %v)", rbErr, err)
			}
		}
	}()

	if err = fn(ctx, tx); err != nil {
		return err
	}

	return tx.Commit()
}
