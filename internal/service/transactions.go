package service

import (
	"context"
	"log/slog"
)

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
		runAfterCommitHook(ctx, hook)
	}
}

func runAfterCommitHook(ctx context.Context, hook func(context.Context)) {
	defer func() {
		if r := recover(); r != nil {
			slog.Default().Warn("after-commit hook panicked", "panic", r)
		}
	}()
	hook(ctx)
}
