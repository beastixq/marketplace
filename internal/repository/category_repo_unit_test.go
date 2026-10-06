package repository_test

import (
	"context"
	"testing"

	m "github.com/beastixq/marketplace/internal/model"
	"github.com/beastixq/marketplace/internal/repository"
	"github.com/beastixq/marketplace/internal/service"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v5"
)

func TestCategoryRepo_GetCategories(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("SELECT (.+) FROM categories ORDER BY (.+)").WillReturnRows(categoryRows("Electronics")).RowsWillBeClosed()
		r := repository.NewCategoryRepo(db)

		got, err := r.GetCategories(ctx, m.CategoryListOptions{})

		requireErrorIs(t, err, nil)
		if len(got) != 1 {
			t.Fatalf("unexpected categories: %+v", got)
		}
	})
	t.Run("query failure", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("SELECT (.+) FROM categories ORDER BY (.+)").WillReturnError(unitErr)
		r := repository.NewCategoryRepo(db)

		_, err := r.GetCategories(ctx, m.CategoryListOptions{})

		requireErrorIs(t, err, repository.ErrQuery)
	})
	for _, tc := range []struct {
		name    string
		rows    *pgxmock.Rows
		wantErr error
	}{
		{name: "scan failure", rows: pgxmock.NewRows([]string{"id", "parent_id", "name", "description"}).AddRow("invalid ID", nil, "Electronics", nil), wantErr: repository.ErrToScan},
		{name: "iteration failure", rows: categoryRows("Electronics").CloseError(unitErr), wantErr: repository.ErrRowsIteration},
	} {
		t.Run(tc.name, func(t *testing.T) {

			db := newMockPool(t)
			db.ExpectQuery("SELECT (.+) FROM categories ORDER BY (.+)").WillReturnRows(tc.rows).RowsWillBeClosed()
			r := repository.NewCategoryRepo(db)

			_, err := r.GetCategories(ctx, m.CategoryListOptions{})

			requireErrorIs(t, err, tc.wantErr)

		})
	}
}

func TestCategoryRepo_GetCategoryByID(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("SELECT (.+) FROM categories WHERE id = \\$1").WithArgs(int64(1)).WillReturnRows(categoryRows("Electronics"))
		r := repository.NewCategoryRepo(db)

		got, err := r.GetCategoryByID(ctx, 1)

		requireErrorIs(t, err, nil)
		if got.ID != 1 {
			t.Fatalf("unexpected category: %+v", got)
		}
	})
	t.Run("scan failure", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("SELECT (.+) FROM categories WHERE id = \\$1").WithArgs(int64(1)).WillReturnError(unitErr)
		r := repository.NewCategoryRepo(db)

		_, err := r.GetCategoryByID(ctx, 1)

		requireErrorIs(t, err, repository.ErrToScan)
	})
	t.Run("not found", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("SELECT (.+) FROM categories WHERE id = \\$1").WithArgs(int64(1)).WillReturnError(pgx.ErrNoRows)
		r := repository.NewCategoryRepo(db)

		_, err := r.GetCategoryByID(ctx, 1)

		requireErrorIs(t, err, service.ErrNotFound)
	})
}

func TestCategoryRepo_CreateCategory(t *testing.T) {
	ctx := context.Background()

	create := m.CategoryCreate{Name: "Electronics"}
	t.Run("success", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("INSERT INTO categories (.+) RETURNING id").WithArgs(pgxmock.AnyArg(), create.Name, pgxmock.AnyArg()).WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(int64(1)))
		r := repository.NewCategoryRepo(db)

		id, err := r.CreateCategory(ctx, create)

		requireErrorIs(t, err, nil)
		if id != 1 {
			t.Fatalf("id: got %d, want 1", id)
		}
	})
	t.Run("scan failure", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("INSERT INTO categories (.+) RETURNING id").WithArgs(pgxmock.AnyArg(), create.Name, pgxmock.AnyArg()).WillReturnError(unitErr)
		r := repository.NewCategoryRepo(db)

		_, err := r.CreateCategory(ctx, create)

		requireErrorIs(t, err, repository.ErrToScan)
	})
}

func TestCategoryRepo_UpdateCategory(t *testing.T) {
	ctx := context.Background()

	name := "Phones"
	t.Run("success", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("UPDATE categories SET name = \\$1 WHERE id = \\$2 RETURNING (.+)").WithArgs(name, int64(1)).WillReturnRows(categoryRows(name))
		r := repository.NewCategoryRepo(db)

		got, err := r.UpdateCategory(ctx, 1, m.CategoryUpdate{Name: &name})

		requireErrorIs(t, err, nil)
		if got.ID != 1 || got.Name != name {
			t.Fatalf("unexpected category: %+v", got)
		}
	})
	t.Run("empty update", func(t *testing.T) {
		r := repository.NewCategoryRepo(newMockPool(t))

		_, err := r.UpdateCategory(ctx, 1, m.CategoryUpdate{})

		requireErrorIs(t, err, service.ErrNoChangesInUpdate)
	})
	for _, tc := range []struct {
		name    string
		dbErr   error
		wantErr error
	}{
		{name: "query row failure", dbErr: unitErr, wantErr: repository.ErrToScan},
		{name: "not found", dbErr: pgx.ErrNoRows, wantErr: service.ErrNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := newMockPool(t)
			db.ExpectQuery("UPDATE categories SET name = \\$1 WHERE id = \\$2 RETURNING (.+)").WithArgs(name, int64(1)).WillReturnError(tc.dbErr)
			r := repository.NewCategoryRepo(db)
			_, err := r.UpdateCategory(ctx, 1, m.CategoryUpdate{Name: &name})
			requireErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestCategoryRepo_DeleteCategoryByID(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectExec("DELETE FROM categories WHERE id = \\$1").WithArgs(int64(1)).WillReturnResult(pgxmock.NewResult("DELETE", 1))
		r := repository.NewCategoryRepo(db)

		err := r.DeleteCategoryByID(ctx, 1)

		requireErrorIs(t, err, nil)
	})
	t.Run("exec failure", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectExec("DELETE FROM categories WHERE id = \\$1").WithArgs(int64(1)).WillReturnError(unitErr)
		r := repository.NewCategoryRepo(db)

		err := r.DeleteCategoryByID(ctx, 1)

		requireErrorIs(t, err, repository.ErrExec)
	})
}

func categoryRows(name string) *pgxmock.Rows {
	categoryColumns := []string{"id", "parent_id", "name", "description"}
	return pgxmock.NewRows(categoryColumns).AddRow(int64(1), nil, name, nil)
}
