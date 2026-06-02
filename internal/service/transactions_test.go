package service

import (
	"context"
	"testing"
)

func TestAfterCommitHooksRunRecoversPanicsAndContinues(t *testing.T) {
	ctx, hooks := WithAfterCommitHooks(context.Background())
	calls := make([]string, 0, 2)

	AddAfterCommit(ctx, func(context.Context) {
		calls = append(calls, "first")
		panic("boom")
	})
	AddAfterCommit(ctx, func(context.Context) {
		calls = append(calls, "second")
	})

	hooks.Run(ctx)

	if len(calls) != 2 {
		t.Fatalf("hook calls: got %v, want two calls", calls)
	}
	if calls[0] != "first" || calls[1] != "second" {
		t.Fatalf("hook calls: got %v, want [first second]", calls)
	}
}
