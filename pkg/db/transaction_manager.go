// pkg/db/transaction_manager.go
package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// TxController defines methods for controlling a database transaction.
// *sqlx.Tx implicitly implements this interface.
type TxController interface {
	Commit() error
	Rollback() error
}

// DBTxBeginner defines the interface for beginning transactions.
// *sqlx.DB implements this.
type DBTxBeginner interface {
	BeginTxx(ctx context.Context, opts *sql.TxOptions) (*sqlx.Tx, error)
}

// BeginTx starts a new database transaction.
// It returns a TxController interface, which *sqlx.Tx implements.
func BeginTx(ctx context.Context, dbConn DBTxBeginner) (TxController, error) {
	tx, err := dbConn.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	return tx, nil // *sqlx.Tx implicitly implements TxController
}

// CommitTx commits the transaction.
func CommitTx(tx TxController) error {
	return tx.Commit()
}

// RollbackTx rolls back the transaction.
func RollbackTx(tx TxController) {
	if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
		// Log the error, but don't return it as it's typically a deferred call
		fmt.Printf("Error rolling back transaction: %v\n", err)
	}
}
