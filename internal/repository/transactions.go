package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type txKeyType struct{}

var txKey txKeyType

func GetTxFromCtx(ctx context.Context) (pgx.Tx, bool) {
	tx, ok := ctx.Value(txKey).(pgx.Tx)
	return tx, ok
}

func SetTxCtx(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, txKey, tx)
}

type PgxTxManager struct {
	pool Database
}

func NewPgxTxManager(pool Database) PgxTxManager {
	return PgxTxManager{pool: pool}
}

func (pm PgxTxManager) WithTransaction(ctx context.Context, fn func(context.Context) error) error {
	if _, ok := GetTxFromCtx(ctx); ok {
		return fn(ctx)
	}

	return pgx.BeginFunc(ctx, pm.pool, func(tx pgx.Tx) error {
		txCtx := SetTxCtx(ctx, tx)
		return fn(txCtx)
	})
}
