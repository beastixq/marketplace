package testsupport

import "context"

type PassThroughTxManager struct{}

func (PassThroughTxManager) WithTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}
