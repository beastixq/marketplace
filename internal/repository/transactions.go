package repository

import (
	"context"

	"github.com/beastixq/marketplace/internal/service"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
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
	pool *pgxpool.Pool
}

func NewPgxTxManager(pool *pgxpool.Pool) PgxTxManager {
	return PgxTxManager{pool: pool}
}

func (pm PgxTxManager) WithTransaction(ctx context.Context, fn func(context.Context) error) error {
	if _, ok := GetTxFromCtx(ctx); ok {
		return fn(ctx)
	}

	hookCtx, hooks := service.WithAfterCommitHooks(ctx)
	if err := pgx.BeginFunc(hookCtx, pm.pool, func(tx pgx.Tx) error {
		txCtx := SetTxCtx(hookCtx, tx)
		return fn(txCtx)
	}); err != nil {
		return err
	}
	hooks.Run(context.WithoutCancel(ctx))
	return nil
}
