//go:build integration

package repository_test

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	m "github.com/beastixq/marketplace/internal/model"
	"github.com/beastixq/marketplace/internal/repository"
	"github.com/beastixq/marketplace/internal/service"
)

var _ service.CategoryRepo = (*repository.CategoryRepoImpl)(nil)

func TestCategoryRepo_GetCategoryByID_Integration(t *testing.T) {
	requireIntegrationDatabase(t)
	t.Run("success", func(t *testing.T) {
		name := fmt.Sprintf("category_%d", time.Now().UnixNano())
		id := createTestCategory(t, m.CategoryCreate{Name: name})
		r := repository.NewCategoryRepo(testPool)

		got, err := r.GetCategoryByID(context.Background(), id)

		requireErrorIs(t, err, nil)
		if got.ID != id || got.Name != name || got.ParentID != nil || got.Description != nil {
			t.Fatalf("unexpected category: %+v", got)
		}
	})
	t.Run("not found", func(t *testing.T) {
		r := repository.NewCategoryRepo(testPool)

		_, err := r.GetCategoryByID(context.Background(), 0)

		requireErrorIs(t, err, service.ErrNotFound)
	})
}

func TestCategoryRepo_GetCategories_Integration(t *testing.T) {
	requireIntegrationDatabase(t)
	for _, tc := range []struct {
		name        string
		page        int
		limit       int
		wantIndexes []int
	}{
		{name: "all matching categories", page: 1, limit: 100, wantIndexes: []int{0, 1, 2}},
		{name: "first page limit", page: 1, limit: 2, wantIndexes: []int{0, 1}},
		{name: "second page offset", page: 2, limit: 2, wantIndexes: []int{2}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			prefix := fmt.Sprintf("pagination_%d_", time.Now().UnixNano())
			ids := make([]int64, 3)
			for i := range ids {
				ids[i] = createTestCategory(t, m.CategoryCreate{Name: fmt.Sprintf("%s%d", prefix, i)})
			}
			opts := m.CategoryListOptions{Search: &prefix, Pagination: m.PaginationOpts{Page: tc.page, Limit: tc.limit}}
			r := repository.NewCategoryRepo(testPool)

			got, err := r.GetCategories(context.Background(), opts)

			requireErrorIs(t, err, nil)
			gotIDs := make([]int64, len(got))
			for i, c := range got {
				gotIDs[i] = c.ID
			}
			wantIDs := make([]int64, len(tc.wantIndexes))
			for i, index := range tc.wantIndexes {
				wantIDs[i] = ids[index]
			}
			if !reflect.DeepEqual(gotIDs, wantIDs) {
				t.Fatalf("category IDs: got %v, want %v", gotIDs, wantIDs)
			}
		})
	}
}

func TestCategoryRepo_CreateCategory_Integration(t *testing.T) {
	requireIntegrationDatabase(t)
	t.Run("success", func(t *testing.T) {
		name := fmt.Sprintf("create_%d", time.Now().UnixNano())
		r := repository.NewCategoryRepo(testPool)

		id, err := r.CreateCategory(context.Background(), m.CategoryCreate{Name: name})

		requireErrorIs(t, err, nil)
		cleanupCategory(t, id)
		var stored string
		err = testPool.QueryRow(context.Background(), "SELECT name FROM categories WHERE id = $1", id).Scan(&stored)
		requireErrorIs(t, err, nil)
		if stored != name {
			t.Fatalf("stored name: got %q, want %q", stored, name)
		}
	})
	t.Run("duplicate name", func(t *testing.T) {
		name := fmt.Sprintf("duplicate_%d", time.Now().UnixNano())
		createTestCategory(t, m.CategoryCreate{Name: name})
		r := repository.NewCategoryRepo(testPool)

		_, err := r.CreateCategory(context.Background(), m.CategoryCreate{Name: name})

		requireErrorIs(t, err, repository.ErrToScan)
	})
}

func TestCategoryRepo_UpdateCategory_Integration(t *testing.T) {
	requireIntegrationDatabase(t)
	t.Run("success", func(t *testing.T) {
		id := createTestCategory(t, m.CategoryCreate{Name: fmt.Sprintf("old_%d", time.Now().UnixNano())})
		name := fmt.Sprintf("new_%d", time.Now().UnixNano())
		r := repository.NewCategoryRepo(testPool)

		got, err := r.UpdateCategory(context.Background(), id, m.CategoryUpdate{Name: &name})

		requireErrorIs(t, err, nil)
		if got.ID != id || got.Name != name {
			t.Fatalf("unexpected updated category: %+v", got)
		}
		var stored string
		err = testPool.QueryRow(context.Background(), "SELECT name FROM categories WHERE id = $1", id).Scan(&stored)
		requireErrorIs(t, err, nil)
		if stored != name {
			t.Fatalf("stored name: got %q, want %q", stored, name)
		}
	})
	t.Run("not found", func(t *testing.T) {
		name := "missing"
		r := repository.NewCategoryRepo(testPool)

		_, err := r.UpdateCategory(context.Background(), 0, m.CategoryUpdate{Name: &name})

		requireErrorIs(t, err, service.ErrNotFound)
	})
}

func TestCategoryRepo_DeleteCategoryByID_Integration(t *testing.T) {
	requireIntegrationDatabase(t)
	t.Run("success", func(t *testing.T) {
		id := createTestCategory(t, m.CategoryCreate{Name: fmt.Sprintf("delete_%d", time.Now().UnixNano())})
		r := repository.NewCategoryRepo(testPool)

		err := r.DeleteCategoryByID(context.Background(), id)

		requireErrorIs(t, err, nil)
		var count int
		err = testPool.QueryRow(context.Background(), "SELECT count(*) FROM categories WHERE id = $1", id).Scan(&count)
		requireErrorIs(t, err, nil)
		if count != 0 {
			t.Fatalf("deleted category still exists: count %d", count)
		}
	})
}
