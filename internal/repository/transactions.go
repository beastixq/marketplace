package repository

import (
	"context"

	"github.com/beastixq/marketplace/internal/service"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PgxTxManager struct {
	pool *pgxpool.Pool
}

func NewPgxTxManager(pool *pgxpool.Pool) PgxTxManager {
	return PgxTxManager{pool: pool}
}

func (pm PgxTxManager) WithTransaction(ctx context.Context, fn func(context.Context) error) error {
	if _, ok := service.GetTxFromCtx(ctx); ok {
		return fn(ctx)
	}

	return pgx.BeginFunc(ctx, pm.pool, func(tx pgx.Tx) error {
		txCtx := service.SetTxCtx(ctx, tx)
		return fn(txCtx)
	})
}
