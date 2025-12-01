package stores

import (
	"context"

	"gorm.io/gorm"
)

type contextKey string

const TxKey contextKey = "tx"

type BaseStore struct {
	db *gorm.DB
}

func (s *BaseStore) GetDB(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value(TxKey).(*gorm.DB); ok {
		return tx
	}
	return s.db.WithContext(ctx)
}

func (s *BaseStore) WithTransaction(ctx context.Context, fn func(context.Context) error) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txCtx := context.WithValue(ctx, TxKey, tx)
		return fn(txCtx)
	})
}

