package repository_test

import (
	"context"
	"testing"
	"time"

	m "github.com/beastixq/marketplace/internal/model"
	"github.com/beastixq/marketplace/internal/repository"
	"github.com/beastixq/marketplace/internal/service"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v5"
	"github.com/shopspring/decimal"
)

func TestProductRepo_GetProducts(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("SELECT DISTINCT (.+) FROM products WHERE products.deleted_at IS NULL").WillReturnRows(productRows("Product")).RowsWillBeClosed()
		got, err := repository.NewProductRepo(db).GetProducts(ctx, m.CatalogOptions{})
		requireErrorIs(t, err, nil)
		if len(got) != 1 || got[0].ID != 1 {
			t.Fatalf("unexpected products: %+v", got)
		}
	})
	t.Run("query failure", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("SELECT DISTINCT (.+) FROM products WHERE products.deleted_at IS NULL").WillReturnError(unitErr)
		_, err := repository.NewProductRepo(db).GetProducts(ctx, m.CatalogOptions{})
		requireErrorIs(t, err, repository.ErrQuery)
	})
	for _, tc := range []struct {
		name    string
		rows    *pgxmock.Rows
		wantErr error
	}{
		{name: "scan failure", rows: pgxmock.NewRows([]string{"id", "seller_id", "name", "description", "price", "stock_quantity", "reserved_quantity", "rating", "created_at", "deleted_at"}).AddRow("invalid ID", int64(2), "Product", nil, decimal.NewFromInt(100), 10, 2, nil, unitNow, nil), wantErr: repository.ErrToScan},
		{name: "iteration failure", rows: productRows("Product").CloseError(unitErr), wantErr: repository.ErrRowsIteration},
	} {
		t.Run(tc.name, func(t *testing.T) {

			db := newMockPool(t)
			db.ExpectQuery("SELECT DISTINCT (.+) FROM products WHERE products.deleted_at IS NULL").WillReturnRows(tc.rows).RowsWillBeClosed()
			_, err := repository.NewProductRepo(db).GetProducts(ctx, m.CatalogOptions{})
			requireErrorIs(t, err, tc.wantErr)

		})
	}
}

func TestProductRepo_GetProductByID(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("SELECT (.+) FROM products WHERE id = \\$1").WithArgs(int64(1)).WillReturnRows(productRows("Product"))
		got, err := repository.NewProductRepo(db).GetProductByID(ctx, 1)
		requireErrorIs(t, err, nil)
		if got.ID != 1 {
			t.Fatalf("unexpected product: %+v", got)
		}
	})
	t.Run("scan failure", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("SELECT (.+) FROM products WHERE id = \\$1").WithArgs(int64(1)).WillReturnError(unitErr)
		_, err := repository.NewProductRepo(db).GetProductByID(ctx, 1)
		requireErrorIs(t, err, repository.ErrToScan)
	})
	t.Run("not found", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("SELECT (.+) FROM products WHERE id = \\$1").WithArgs(int64(1)).WillReturnError(pgx.ErrNoRows)
		_, err := repository.NewProductRepo(db).GetProductByID(ctx, 1)
		requireErrorIs(t, err, service.ErrNotFound)
	})
}

func TestProductRepo_GetProductByIDForUpdate(t *testing.T) {
	ctx := context.Background()

	t.Run("success in transaction", func(t *testing.T) {
		db := newMockPool(t)
		tx := beginMockTransaction(t, db, ctx)
		db.ExpectQuery("SELECT (.+) FROM products WHERE id = \\$1 FOR UPDATE").WithArgs(int64(1)).WillReturnRows(productRows("Product"))

		got, err := repository.NewProductRepo(db).GetProductByIDForUpdate(repository.SetTxCtx(ctx, tx), 1)

		requireErrorIs(t, err, nil)
		if got.ID != 1 {
			t.Fatalf("unexpected product: %+v", got)
		}
	})
	t.Run("without transaction", func(t *testing.T) {
		_, err := repository.NewProductRepo(newMockPool(t)).GetProductByIDForUpdate(ctx, 1)
		requireErrorIs(t, err, service.ErrMustBeInTransaction)
	})
	for _, tc := range []struct {
		name    string
		dbErr   error
		wantErr error
	}{
		{name: "not found", dbErr: pgx.ErrNoRows, wantErr: service.ErrNotFound},
		{name: "query row failure", dbErr: unitErr, wantErr: repository.ErrToScan},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := newMockPool(t)
			tx := beginMockTransaction(t, db, ctx)
			db.ExpectQuery("SELECT (.+) FROM products WHERE id = \\$1 FOR UPDATE").WithArgs(int64(1)).WillReturnError(tc.dbErr)
			r := repository.NewProductRepo(db)

			_, err := r.GetProductByIDForUpdate(repository.SetTxCtx(ctx, tx), 1)

			requireErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestProductRepo_GetProductPriceHistory(t *testing.T) {
	ctx := context.Background()

	from, to := unitNow.Add(-time.Hour), unitNow
	t.Run("success", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("SELECT (.+) FROM product_price_history WHERE product_id = \\$1 AND (.+) ORDER BY changed_at ASC, id ASC").WithArgs(int64(2), from, to).WillReturnRows(productHistoryRows()).RowsWillBeClosed()
		got, err := repository.NewProductRepo(db).GetProductPriceHistory(ctx, 2, from, to)
		requireErrorIs(t, err, nil)
		if len(got) != 1 {
			t.Fatalf("unexpected history: %+v", got)
		}
	})
	t.Run("query failure", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("SELECT (.+) FROM product_price_history WHERE product_id = \\$1 AND (.+) ORDER BY changed_at ASC, id ASC").WithArgs(int64(2), from, to).WillReturnError(unitErr)
		_, err := repository.NewProductRepo(db).GetProductPriceHistory(ctx, 2, from, to)
		requireErrorIs(t, err, repository.ErrQuery)
	})
	for _, tc := range []struct {
		name    string
		rows    *pgxmock.Rows
		wantErr error
	}{
		{name: "scan failure", rows: pgxmock.NewRows([]string{"id", "product_id", "old_price", "new_price", "changed_at", "changed_by"}).AddRow("invalid ID", int64(2), decimal.NewFromInt(90), decimal.NewFromInt(100), unitNow, "seller:2"), wantErr: repository.ErrToScan},
		{name: "iteration failure", rows: productHistoryRows().CloseError(unitErr), wantErr: repository.ErrRowsIteration},
	} {
		t.Run(tc.name, func(t *testing.T) {

			db := newMockPool(t)
			db.ExpectQuery("SELECT (.+) FROM product_price_history WHERE product_id = \\$1 AND (.+) ORDER BY changed_at ASC, id ASC").WithArgs(int64(2), from, to).WillReturnRows(tc.rows).RowsWillBeClosed()
			_, err := repository.NewProductRepo(db).GetProductPriceHistory(ctx, 2, from, to)
			requireErrorIs(t, err, tc.wantErr)

		})
	}
}

func TestProductRepo_GetProductCategories(t *testing.T) {
	ctx := context.Background()

	categoryRows := func() *pgxmock.Rows {
		return pgxmock.NewRows([]string{"id", "parent_id", "name", "description"}).AddRow(int64(1), nil, "Category", nil)
	}
	t.Run("success", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("SELECT (.+) FROM categories c JOIN product_categories pc (.+) WHERE pc.product_id = \\$1 ORDER BY (.+)").WithArgs(int64(2)).WillReturnRows(categoryRows()).RowsWillBeClosed()
		got, err := repository.NewProductRepo(db).GetProductCategories(ctx, 2)
		requireErrorIs(t, err, nil)
		if len(got) != 1 {
			t.Fatalf("unexpected categories: %+v", got)
		}
	})
	t.Run("query failure", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("SELECT (.+) FROM categories c JOIN product_categories pc (.+) WHERE pc.product_id = \\$1 ORDER BY (.+)").WithArgs(int64(2)).WillReturnError(unitErr)
		_, err := repository.NewProductRepo(db).GetProductCategories(ctx, 2)
		requireErrorIs(t, err, repository.ErrQuery)
	})
	for _, tc := range []struct {
		name    string
		rows    *pgxmock.Rows
		wantErr error
	}{
		{name: "scan failure", rows: pgxmock.NewRows([]string{"id", "parent_id", "name", "description"}).AddRow("invalid ID", nil, "Category", nil), wantErr: repository.ErrToScan},
		{name: "iteration failure", rows: categoryRows().CloseError(unitErr), wantErr: repository.ErrRowsIteration},
	} {
		t.Run(tc.name, func(t *testing.T) {

			db := newMockPool(t)
			db.ExpectQuery("SELECT (.+) FROM categories c JOIN product_categories pc (.+) WHERE pc.product_id = \\$1 ORDER BY (.+)").WithArgs(int64(2)).WillReturnRows(tc.rows).RowsWillBeClosed()
			_, err := repository.NewProductRepo(db).GetProductCategories(ctx, 2)
			requireErrorIs(t, err, tc.wantErr)

		})
	}
}

func TestProductRepo_CreateProduct(t *testing.T) {
	ctx := context.Background()

	create := m.ProductCreate{SellerID: 2, Name: "Product", Price: decimal.NewFromInt(100), StockQuantity: 10}
	t.Run("success", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("INSERT INTO products (.+) RETURNING id").WithArgs(create.SellerID, create.Name, pgxmock.AnyArg(), create.Price, create.StockQuantity).WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(int64(1)))
		id, err := repository.NewProductRepo(db).CreateProduct(ctx, create)
		requireErrorIs(t, err, nil)
		if id != 1 {
			t.Fatalf("id: got %d, want 1", id)
		}
	})
	t.Run("scan failure", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("INSERT INTO products (.+) RETURNING id").WithArgs(create.SellerID, create.Name, pgxmock.AnyArg(), create.Price, create.StockQuantity).WillReturnError(unitErr)
		_, err := repository.NewProductRepo(db).CreateProduct(ctx, create)
		requireErrorIs(t, err, repository.ErrToScan)
	})
}

func TestProductRepo_ReplaceProductCategories(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectBegin()
		db.ExpectExec("DELETE FROM product_categories WHERE product_id = \\$1").WithArgs(int64(2)).WillReturnResult(pgxmock.NewResult("DELETE", 2))
		db.ExpectExec("INSERT INTO product_categories (.+) VALUES (.+)").WithArgs(int64(2), int64(1), int64(2), int64(3)).WillReturnResult(pgxmock.NewResult("INSERT", 2))
		db.ExpectCommit()
		db.ExpectRollback()

		err := repository.NewProductRepo(db).ReplaceProductCategories(ctx, 2, []int64{1, 1, 3})

		requireErrorIs(t, err, nil)
	})
	t.Run("delete failure rolls back", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectBegin()
		db.ExpectExec("DELETE FROM product_categories WHERE product_id = \\$1").WithArgs(int64(2)).WillReturnError(unitErr)
		db.ExpectRollback()
		db.ExpectRollback()

		err := repository.NewProductRepo(db).ReplaceProductCategories(ctx, 2, []int64{1})

		requireErrorIs(t, err, repository.ErrExec)
	})
	for _, tc := range []struct {
		name       string
		constraint string
		wantErr    error
	}{
		{name: "missing product rolls back", constraint: "fk_product_categories_product_id", wantErr: service.ErrProductNotFound},
		{name: "missing category rolls back", constraint: "fk_product_categories_category_id", wantErr: service.ErrCategoryNotFound},
		{name: "unknown foreign key is not reclassified", constraint: "unexpected_constraint", wantErr: repository.ErrExec},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := newMockPool(t)
			db.ExpectBegin()
			db.ExpectExec("DELETE FROM product_categories WHERE product_id = \\$1").WithArgs(int64(2)).WillReturnResult(pgxmock.NewResult("DELETE", 1))
			db.ExpectExec("INSERT INTO product_categories (.+) VALUES (.+)").WithArgs(int64(2), int64(1)).WillReturnError(&pgconn.PgError{Code: pgerrcode.ForeignKeyViolation, ConstraintName: tc.constraint})
			db.ExpectRollback()
			db.ExpectRollback()
			r := repository.NewProductRepo(db)

			err := r.ReplaceProductCategories(ctx, 2, []int64{1})

			requireErrorIs(t, err, tc.wantErr)
		})
	}
	for _, tc := range []struct {
		name      string
		beginErr  error
		commitErr error
	}{
		{name: "begin failure", beginErr: unitErr},
		{name: "commit failure", commitErr: unitErr},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := newMockPool(t)
			db.ExpectBegin().WillReturnError(tc.beginErr)
			if tc.beginErr == nil {
				db.ExpectExec("DELETE FROM product_categories WHERE product_id = \\$1").WithArgs(int64(2)).WillReturnResult(pgxmock.NewResult("DELETE", 1))
				db.ExpectCommit().WillReturnError(tc.commitErr)
				db.ExpectRollback()
			}
			r := repository.NewProductRepo(db)

			err := r.ReplaceProductCategories(ctx, 2, nil)

			requireErrorIs(t, err, unitErr)
		})
	}
	t.Run("clears categories in existing transaction", func(t *testing.T) {
		db := newMockPool(t)
		tx := beginMockTransaction(t, db, ctx)
		db.ExpectExec("DELETE FROM product_categories WHERE product_id = \\$1").WithArgs(int64(2)).WillReturnResult(pgxmock.NewResult("DELETE", 2))
		r := repository.NewProductRepo(db)

		err := r.ReplaceProductCategories(repository.SetTxCtx(ctx, tx), 2, nil)

		requireErrorIs(t, err, nil)
	})
}

func TestProductRepo_UpdateProduct(t *testing.T) {
	ctx := context.Background()

	name := "Updated"
	t.Run("success", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("UPDATE products SET name = \\$1 WHERE id = \\$2 RETURNING (.+)").WithArgs(name, int64(1)).WillReturnRows(productRows(name))
		got, err := repository.NewProductRepo(db).UpdateProduct(ctx, 1, m.ProductUpdate{Name: &name})
		requireErrorIs(t, err, nil)
		if got.ID != 1 || got.Name != name {
			t.Fatalf("unexpected product: %+v", got)
		}
	})
	t.Run("empty update", func(t *testing.T) {
		_, err := repository.NewProductRepo(newMockPool(t)).UpdateProduct(ctx, 1, m.ProductUpdate{})
		requireErrorIs(t, err, service.ErrNoChangesInUpdate)
	})
	for _, tc := range []struct {
		name    string
		dbErr   error
		wantErr error
	}{
		{name: "query row failure", dbErr: unitErr, wantErr: repository.ErrToScan},
		{name: "not found", dbErr: pgx.ErrNoRows, wantErr: service.ErrNotFound},
		{name: "unrelated check violation", dbErr: &pgconn.PgError{Code: pgerrcode.CheckViolation, ConstraintName: "other_constraint"}, wantErr: repository.ErrToScan},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := newMockPool(t)
			db.ExpectQuery("UPDATE products SET name = \\$1 WHERE id = \\$2 RETURNING (.+)").WithArgs(name, int64(1)).WillReturnError(tc.dbErr)
			r := repository.NewProductRepo(db)
			_, err := r.UpdateProduct(ctx, 1, m.ProductUpdate{Name: &name})
			requireErrorIs(t, err, tc.wantErr)
		})
	}
	t.Run("stock below reserved", func(t *testing.T) {
		stock := 1
		db := newMockPool(t)
		db.ExpectQuery("UPDATE products SET stock_quantity = \\$1 WHERE id = \\$2 RETURNING (.+)").WithArgs(stock, int64(1)).WillReturnError(&pgconn.PgError{Code: pgerrcode.CheckViolation, ConstraintName: "chk_products_reserved_le_stock"})
		r := repository.NewProductRepo(db)

		_, err := r.UpdateProduct(ctx, 1, m.ProductUpdate{StockQuantity: &stock})

		requireErrorIs(t, err, service.ErrStockBelowReserved)
	})
}

func TestProductRepo_ChangeStockAndReserved(t *testing.T) {
	ctx := context.Background()

	query := "UPDATE products SET stock_quantity = stock_quantity \\+ \\$1, reserved_quantity = reserved_quantity \\+ \\$2 WHERE id = \\$3"
	t.Run("success", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectExec(query).WithArgs(-1, -1, int64(1)).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		err := repository.NewProductRepo(db).ChangeStockAndReserved(ctx, 1, -1, -1)
		requireErrorIs(t, err, nil)
	})
	t.Run("missing product", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectExec(query).WithArgs(-1, -1, int64(1)).WillReturnResult(pgxmock.NewResult("UPDATE", 0))
		err := repository.NewProductRepo(db).ChangeStockAndReserved(ctx, 1, -1, -1)
		requireErrorIs(t, err, service.ErrNotFound)
	})
	for _, tc := range []struct {
		name    string
		dbErr   error
		wantErr error
	}{
		{name: "stock invariant violation", dbErr: &pgconn.PgError{Code: pgerrcode.CheckViolation, ConstraintName: "chk_products_reserved_le_stock"}, wantErr: service.ErrStockInvariantViolated},
		{name: "exec failure", dbErr: unitErr, wantErr: repository.ErrExec},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := newMockPool(t)
			db.ExpectExec(query).WithArgs(-1, -1, int64(1)).WillReturnError(tc.dbErr)
			r := repository.NewProductRepo(db)

			err := r.ChangeStockAndReserved(ctx, 1, -1, -1)

			requireErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestProductRepo_DeleteProductByID(t *testing.T) {
	ctx := context.Background()

	query := "UPDATE products SET deleted_at = NOW\\(\\) WHERE id = \\$1"
	t.Run("success", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectExec(query).WithArgs(int64(1)).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		err := repository.NewProductRepo(db).DeleteProductByID(ctx, 1)
		requireErrorIs(t, err, nil)
	})
	t.Run("exec failure", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectExec(query).WithArgs(int64(1)).WillReturnError(unitErr)
		err := repository.NewProductRepo(db).DeleteProductByID(ctx, 1)
		requireErrorIs(t, err, repository.ErrExec)
	})
}

func productRows(name string) *pgxmock.Rows {
	columns := []string{"id", "seller_id", "name", "description", "price", "stock_quantity", "reserved_quantity", "rating", "created_at", "deleted_at"}
	return pgxmock.NewRows(columns).AddRow(int64(1), int64(2), name, nil, decimal.NewFromInt(100), 10, 2, nil, unitNow, nil)
}

func productHistoryRows() *pgxmock.Rows {
	return pgxmock.NewRows([]string{"id", "product_id", "old_price", "new_price", "changed_at", "changed_by"}).AddRow(int64(1), int64(2), decimal.NewFromInt(90), decimal.NewFromInt(100), unitNow, "seller:2")
}
