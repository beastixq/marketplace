package repository

import (
	"context"

	"github.com/beastixq/marketplace/internal/service"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DBTX interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func getConn(ctx context.Context, pool *pgxpool.Pool) DBTX {
	if tx, ok := service.GetTxFromCtx(ctx); ok {
		return tx
	}
	return pool
}
