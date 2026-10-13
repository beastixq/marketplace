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

func (f *integrationFixture) category() int64 {
	f.t.Helper()
	id, err := s.NewCategoryService(repo.NewCategoryRepo(f.pool)).CreateCategory(context.Background(), s.Actor{Role: m.RoleAdmin}, m.CategoryCreate{Name: fmt.Sprintf("integration-category-%d", time.Now().UnixNano())})
	mustIntegration(f.t, err)
	f.categoryIDs = append(f.categoryIDs, id)
	return id
}

func TestCategoryService_CreateCategory_Integration(t *testing.T) {
	t.Run("admin persists and buyer cannot mutate", func(t *testing.T) {
		f := newIntegrationFixture(t)
		categories := s.NewCategoryService(repo.NewCategoryRepo(f.pool))
		id := f.category()

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
