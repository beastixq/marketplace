package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type DBTX interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Database is the minimal PostgreSQL connection contract used by repository
// implementations. Both pgxpool.Pool and pgxmock pools satisfy it.
type Database interface {
	DBTX
	Begin(ctx context.Context) (pgx.Tx, error)
}

func getConn(ctx context.Context, db DBTX) DBTX {
	if tx, ok := GetTxFromCtx(ctx); ok {
		return tx
	}
	return db
}
