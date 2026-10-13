//go:build integration

package repository_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	m "github.com/beastixq/marketplace/internal/model"
	repo "github.com/beastixq/marketplace/internal/repository"
	s "github.com/beastixq/marketplace/internal/service"
)

func TestPgxTxManager_WithTransaction_Integration(t *testing.T) {
	requireIntegrationDatabase(t)
	for _, rollback := range []bool{false, true} {
		name := "commit"
		if rollback {
			name = "rollback"
		}
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			tx := repo.NewPgxTxManager(testPool)
			categories := repo.NewCategoryRepo(testPool)
			failure := errors.New("abort composed operation")
			var id int64

			err := tx.WithTransaction(ctx, func(ctx context.Context) error {
				var err error
				id, err = categories.CreateCategory(ctx, m.CategoryCreate{Name: fmt.Sprintf("tx-%d", time.Now().UnixNano())})
				if err != nil {
					return err
				}
				return tx.WithTransaction(ctx, func(context.Context) error {
					if rollback {
						return failure
					}
					return nil
				})
			})
			if rollback {
				if !errors.Is(err, failure) {
					t.Fatalf("rollback = %v", err)
				}
				if _, err := categories.GetCategoryByID(ctx, id); !errors.Is(err, s.ErrNotFound) {
					t.Fatalf("rolled-back row = %v", err)
				}
			} else {
				cleanupCategory(t, id)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := categories.GetCategoryByID(ctx, id); err != nil {
					t.Fatalf("committed row = %v", err)
				}
			}
		})
	}
}
