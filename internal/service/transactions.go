package service

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type txKeyType struct{}

var txKey txKeyType

func GetTxFromCtx(ctx context.Context) (tx pgx.Tx, ok bool) {
	if tx, ok = ctx.Value(txKey).(pgx.Tx); ok {
		return tx, true
	}
	return tx, false
}

func SetTxCtx(old context.Context, tx pgx.Tx) (new context.Context) {
	return context.WithValue(old, txKey, tx)
}

type TxManager interface {
	WithTransaction(ctx context.Context, fn func(context.Context) error) error
}
