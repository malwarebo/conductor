package db

import (
	"context"
	"database/sql"
	_ "github.com/lib/pq"
)

type DB struct {
	*sql.DB
}

func NewDB(dataSourceName string) (*DB, error) {
	db, err := sql.Open("postgres", dataSourceName)
	if err != nil {
		return nil, err
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &DB{db}, nil
}

// Transaction represents a database transaction
type Transaction struct {
	*sql.Tx
}

// BeginTx starts a new transaction
func (db *DB) BeginTx(ctx context.Context) (*Transaction, error) {
	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &Transaction{tx}, nil
}

// Commit commits the transaction
func (tx *Transaction) Commit() error {
	return tx.Tx.Commit()
}

// Rollback rolls back the transaction
func (tx *Transaction) Rollback() error {
	return tx.Tx.Rollback()
}
