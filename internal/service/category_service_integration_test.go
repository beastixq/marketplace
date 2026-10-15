//go:build integration

package service_test

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

func (f *serviceIntegration) createCategory() int64 {
	f.t.Helper()
	name := fmt.Sprintf("integration-category-%d", time.Now().UnixNano())
	f.t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if _, err := f.pool.Exec(ctx, "DELETE FROM categories WHERE name = $1", name); err != nil {
			f.t.Errorf("clean owned category: %v", err)
		}
	})
	id, err := s.NewCategoryService(repo.NewCategoryRepo(f.pool)).CreateCategory(context.Background(), s.Actor{Role: m.RoleAdmin}, m.CategoryCreate{Name: name})
	mustIntegration(f.t, err)
	return id
}

func TestCategoryService_CreateCategory_Integration(t *testing.T) {
	t.Run("admin persists and buyer cannot mutate", func(t *testing.T) {
		f := newServiceIntegration(t)
		categories := s.NewCategoryService(repo.NewCategoryRepo(f.pool))
		id := f.createCategory()

		category, err := categories.GetCategoryByID(context.Background(), id)
		mustIntegration(t, err)
		if category.ID != id || category.Name == "" {
			t.Fatalf("category = %+v", category)
		}
		if _, err := categories.CreateCategory(context.Background(), s.Actor{Role: m.RoleBuyer}, m.CategoryCreate{Name: "Forbidden"}); !errors.Is(err, s.ErrPermissionDenied) {
			t.Fatalf("role = %v", err)
		}
	})
}
