package repository_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	m "github.com/beastixq/marketplace/internal/model"
	repo "github.com/beastixq/marketplace/internal/repository"
	"github.com/beastixq/marketplace/internal/service"
)

var _ service.CategoryRepo = repo.CategoryRepoImpl{}

func TestCategoryRepo_CreateAndGet(t *testing.T) {
	var r service.CategoryRepo = repo.NewCategoryRepo(testPool)
	ctx := context.Background()

	name := fmt.Sprintf("cat_%d", time.Now().UnixNano())
	id, err := r.CreateCategory(ctx, m.CategoryCreate{Name: name})
	if err != nil {
		t.Fatalf("CreateCategory: %v", err)
	}
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), "DELETE FROM categories WHERE id = $1", id)
	})

	got, err := r.GetCategoryByID(ctx, id)
	if err != nil {
		t.Fatalf("GetCategoryByID: %v", err)
	}
	if got.ID != id {
		t.Errorf("ID: got %d, want %d", got.ID, id)
	}
	if got.Name != name {
		t.Errorf("Name: got %q, want %q", got.Name, name)
	}
}

func TestCategoryRepo_GetCategories_Pagination(t *testing.T) {
	var r service.CategoryRepo = repo.NewCategoryRepo(testPool)
	ctx := context.Background()

	suffix := time.Now().UnixNano()
	ids := make([]int64, 0, 3)
	for i := range 3 {
		id, err := r.CreateCategory(ctx, m.CategoryCreate{
			Name: fmt.Sprintf("cat_%d_%d", suffix, i),
		})
		if err != nil {
			t.Fatalf("CreateCategory %d: %v", i, err)
		}
		ids = append(ids, id)
	}
	t.Cleanup(func() {
		for _, id := range ids {
			_, _ = testPool.Exec(context.Background(), "DELETE FROM categories WHERE id = $1", id)
		}
	})

	got, err := r.GetCategories(ctx, m.CategoryListOptions{Pagination: m.PaginationOpts{Page: 1, Limit: 100}})
	if err != nil {
		t.Fatalf("GetCategories: %v", err)
	}
	if len(got) < 3 {
		t.Errorf("ожидается минимум 3 категории, получено %d", len(got))
	}
}

func TestCategoryRepo_Update(t *testing.T) {
	var r service.CategoryRepo = repo.NewCategoryRepo(testPool)
	ctx := context.Background()

	id, err := r.CreateCategory(ctx, m.CategoryCreate{
		Name: fmt.Sprintf("upd_old_%d", time.Now().UnixNano()),
	})
	if err != nil {
		t.Fatalf("CreateCategory: %v", err)
	}
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), "DELETE FROM categories WHERE id = $1", id)
	})

	newName := fmt.Sprintf("upd_new_%d", time.Now().UnixNano())
	updated, err := r.UpdateCategory(ctx, id, m.CategoryUpdate{Name: &newName})
	if err != nil {
		t.Fatalf("UpdateCategory: %v", err)
	}
	if updated.Name != newName {
		t.Errorf("Name: got %q, want %q", updated.Name, newName)
	}

	fetched, err := r.GetCategoryByID(ctx, id)
	if err != nil {
		t.Fatalf("GetCategoryByID после Update: %v", err)
	}
	if fetched.Name != newName {
		t.Errorf("Name в БД: got %q, want %q", fetched.Name, newName)
	}
}

func TestCategoryRepo_GetCategories_Limit(t *testing.T) {
	var r service.CategoryRepo = repo.NewCategoryRepo(testPool)
	ctx := context.Background()

	suffix := time.Now().UnixNano()
	ids := make([]int64, 3)
	for i := range 3 {
		id, err := r.CreateCategory(ctx, m.CategoryCreate{
			Name: fmt.Sprintf("limit_%d_%d", suffix, i),
		})
		if err != nil {
			t.Fatalf("CreateCategory %d: %v", i, err)
		}
		ids[i] = id
	}
	t.Cleanup(func() {
		for _, id := range ids {
			_, _ = testPool.Exec(context.Background(), "DELETE FROM categories WHERE id = $1", id)
		}
	})

	got, err := r.GetCategories(ctx, m.CategoryListOptions{Pagination: m.PaginationOpts{Page: 1, Limit: 2}})
	if err != nil {
		t.Fatalf("GetCategories: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("ожидается ровно 2, получено %d", len(got))
	}
	for _, c := range got {
		if c.ID == ids[2] {
			t.Errorf("третий элемент id=%d не должен попасть в страницу с limit=2", ids[2])
		}
	}
}

func TestCategoryRepo_GetCategories_Page2(t *testing.T) {
	var r service.CategoryRepo = repo.NewCategoryRepo(testPool)
	ctx := context.Background()

	suffix := time.Now().UnixNano()
	ids := make([]int64, 3)
	for i := range 3 {
		id, err := r.CreateCategory(ctx, m.CategoryCreate{
			Name: fmt.Sprintf("page_%d_%d", suffix, i),
		})
		if err != nil {
			t.Fatalf("CreateCategory %d: %v", i, err)
		}
		ids[i] = id
	}
	t.Cleanup(func() {
		for _, id := range ids {
			_, _ = testPool.Exec(context.Background(), "DELETE FROM categories WHERE id = $1", id)
		}
	})

	page1, err := r.GetCategories(ctx, m.CategoryListOptions{Pagination: m.PaginationOpts{Page: 1, Limit: 2}})
	if err != nil {
		t.Fatalf("GetCategories page1: %v", err)
	}
	if len(page1) != 2 {
		t.Fatalf("page1: ожидается 2, получено %d", len(page1))
	}

	page2, err := r.GetCategories(ctx, m.CategoryListOptions{Pagination: m.PaginationOpts{Page: 2, Limit: 2}})
	if err != nil {
		t.Fatalf("GetCategories page2: %v", err)
	}
	if len(page2) < 1 {
		t.Errorf("page2: ожидается минимум 1, получено %d", len(page2))
	}

	// Страницы не пересекаются.
	page1IDs := map[int64]bool{}
	for _, c := range page1 {
		page1IDs[c.ID] = true
	}
	for _, c := range page2 {
		if page1IDs[c.ID] {
			t.Errorf("категория id=%d присутствует на обеих страницах", c.ID)
		}
	}
}

func TestCategoryRepo_Create_DuplicateName(t *testing.T) {
	var r service.CategoryRepo = repo.NewCategoryRepo(testPool)
	ctx := context.Background()

	name := fmt.Sprintf("dup_%d", time.Now().UnixNano())
	id, err := r.CreateCategory(ctx, m.CategoryCreate{Name: name})
	if err != nil {
		t.Fatalf("CreateCategory первый: %v", err)
	}
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), "DELETE FROM categories WHERE id = $1", id)
	})

	_, err = r.CreateCategory(ctx, m.CategoryCreate{Name: name})
	if err == nil {
		t.Error("CreateCategory с дублирующим именем должен вернуть ошибку")
	}
}

func TestCategoryRepo_GetByID_NotFound(t *testing.T) {
	var r service.CategoryRepo = repo.NewCategoryRepo(testPool)
	ctx := context.Background()

	_, err := r.GetCategoryByID(ctx, 999999999)
	if !errors.Is(err, service.ErrNotFound) {
		t.Errorf("ожидалась ErrNotFound, получено: %v", err)
	}
}

func TestCategoryRepo_Update_NotFound(t *testing.T) {
	var r service.CategoryRepo = repo.NewCategoryRepo(testPool)
	ctx := context.Background()

	name := "ghost"
	_, err := r.UpdateCategory(ctx, 999999999, m.CategoryUpdate{Name: &name})
	if !errors.Is(err, service.ErrNotFound) {
		t.Errorf("ожидалась ErrNotFound, получено: %v", err)
	}
}

func TestCategoryRepo_Delete(t *testing.T) {
	var r service.CategoryRepo = repo.NewCategoryRepo(testPool)
	ctx := context.Background()

	id, err := r.CreateCategory(ctx, m.CategoryCreate{
		Name: fmt.Sprintf("del_%d", time.Now().UnixNano()),
	})
	if err != nil {
		t.Fatalf("CreateCategory: %v", err)
	}

	if err := r.DeleteCategoryByID(ctx, id); err != nil {
		t.Fatalf("DeleteCategoryByID: %v", err)
	}

	_, err = r.GetCategoryByID(ctx, id)
	if !errors.Is(err, service.ErrNotFound) {
		t.Errorf("после удаления ожидалась ErrNotFound, получено: %v", err)
	}
}
