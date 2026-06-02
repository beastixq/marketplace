package service

import "context"

type TxManager interface {
	WithTransaction(ctx context.Context, fn func(context.Context) error) error
}

type afterCommitKeyType struct{}

var afterCommitKey afterCommitKeyType

type AfterCommitHooks struct {
	hooks []func(context.Context)
}

func WithAfterCommitHooks(ctx context.Context) (context.Context, *AfterCommitHooks) {
	if existing, ok := ctx.Value(afterCommitKey).(*AfterCommitHooks); ok {
		return ctx, existing
	}
	hooks := &AfterCommitHooks{}
	return context.WithValue(ctx, afterCommitKey, hooks), hooks
}

func AddAfterCommit(ctx context.Context, hook func(context.Context)) bool {
	hooks, ok := ctx.Value(afterCommitKey).(*AfterCommitHooks)
	if !ok {
		return false
	}
	hooks.hooks = append(hooks.hooks, hook)
	return true
}

func (h *AfterCommitHooks) Run(ctx context.Context) {
	if h == nil {
		return
	}
	for _, hook := range h.hooks {
		hook(ctx)
	}
}
