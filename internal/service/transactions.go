package service

import "context"

type TxManager interface {
	WithTransaction(ctx context.Context, fn func(context.Context) error) error
}
