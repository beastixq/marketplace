package repository_test

import (
	"context"
	"testing"

	"github.com/beastixq/marketplace/internal/repository"
)

func TestPgxTxManager_WithTransaction(t *testing.T) {
	ctx := context.Background()

	t.Run("commits successful callback", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectBegin()
		db.ExpectCommit()
		db.ExpectRollback()
		called := false

		err := repository.NewPgxTxManager(db).WithTransaction(ctx, func(txCtx context.Context) error {
			called = true
			if _, ok := repository.GetTxFromCtx(txCtx); !ok {
				t.Fatal("transaction is missing from callback context")
			}
			return nil
		})

		requireErrorIs(t, err, nil)
		if !called {
			t.Fatal("callback was not called")
		}
	})

	t.Run("rolls back callback failure", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectBegin()
		db.ExpectRollback()
		db.ExpectRollback()

		err := repository.NewPgxTxManager(db).WithTransaction(ctx, func(context.Context) error { return unitErr })

		requireErrorIs(t, err, unitErr)
	})
	for _, tc := range []struct {
		name      string
		beginErr  error
		commitErr error
		wantCalls int
	}{
		{name: "begin failure does not call callback", beginErr: unitErr},
		{name: "commit failure is returned", commitErr: unitErr, wantCalls: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := newMockPool(t)
			db.ExpectBegin().WillReturnError(tc.beginErr)
			if tc.beginErr == nil {
				db.ExpectCommit().WillReturnError(tc.commitErr)
				db.ExpectRollback()
			}
			calls := 0
			manager := repository.NewPgxTxManager(db)

			err := manager.WithTransaction(ctx, func(context.Context) error {
				calls++
				return nil
			})

			requireErrorIs(t, err, unitErr)
			if calls != tc.wantCalls {
				t.Fatalf("callback calls: got %d, want %d", calls, tc.wantCalls)
			}
		})
	}
	for _, tc := range []struct {
		name        string
		callbackErr error
	}{
		{name: "existing transaction reused"},
		{name: "existing transaction propagates callback error", callbackErr: unitErr},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := newMockPool(t)
			tx := beginMockTransaction(t, db, ctx)
			txCtx := repository.SetTxCtx(ctx, tx)
			manager := repository.NewPgxTxManager(db)

			err := manager.WithTransaction(txCtx, func(gotCtx context.Context) error {
				gotTx, ok := repository.GetTxFromCtx(gotCtx)
				if !ok || gotTx != tx {
					t.Fatal("callback did not receive the existing transaction")
				}
				return tc.callbackErr
			})

			requireErrorIs(t, err, tc.callbackErr)
		})
	}
}
