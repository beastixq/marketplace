package repository_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v5"
)

var (
	unitNow = time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
	unitErr = errors.New("database failure")
)

func newMockPool(t *testing.T) pgxmock.PgxPoolIface {
	t.Helper()
	db, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("create pgx mock: %v", err)
	}
	t.Cleanup(func() {
		if err := db.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet database expectations: %v", err)
		}
		db.Close()
	})
	return db
}

func beginMockTransaction(t *testing.T, db pgxmock.PgxPoolIface, ctx context.Context) pgx.Tx {
	t.Helper()
	db.ExpectBegin()
	tx, err := db.Begin(ctx)
	if err != nil {
		t.Fatalf("begin transaction fixture: %v", err)
	}
	t.Cleanup(func() {
		db.ExpectRollback()
		if err := tx.Rollback(ctx); err != nil {
			t.Errorf("rollback transaction fixture: %v", err)
		}
	})
	return tx
}

func requireErrorIs(t *testing.T, got, want error) {
	t.Helper()
	if !errors.Is(got, want) {
		t.Fatalf("error: got %v, want %v", got, want)
	}
}

func ptr[T any](value T) *T { return &value }
