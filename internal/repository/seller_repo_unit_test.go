package repository_test

import (
	"context"
	"testing"
	"time"

	m "github.com/beastixq/marketplace/internal/model"
	"github.com/beastixq/marketplace/internal/repository"
	"github.com/beastixq/marketplace/internal/service"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v5"
	"github.com/shopspring/decimal"
)

func TestSellerRepo_GetSellerByID(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("SELECT (.+) FROM sellers WHERE id = \\$1").WithArgs(int64(1)).WillReturnRows(sellerRows("Shop"))
		got, err := repository.NewSellerRepo(db).GetSellerByID(ctx, 1)
		requireErrorIs(t, err, nil)
		if got.ID != 1 {
			t.Fatalf("unexpected seller: %+v", got)
		}
	})
	t.Run("scan failure", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("SELECT (.+) FROM sellers WHERE id = \\$1").WithArgs(int64(1)).WillReturnError(unitErr)
		_, err := repository.NewSellerRepo(db).GetSellerByID(ctx, 1)
		requireErrorIs(t, err, repository.ErrToScan)
	})
	t.Run("not found", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("SELECT (.+) FROM sellers WHERE id = \\$1").WithArgs(int64(1)).WillReturnError(pgx.ErrNoRows)
		_, err := repository.NewSellerRepo(db).GetSellerByID(ctx, 1)
		requireErrorIs(t, err, service.ErrNotFound)
	})
}

func TestSellerRepo_GetSellerByUserID(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("SELECT (.+) FROM sellers WHERE user_id = \\$1").WithArgs(int64(2)).WillReturnRows(sellerRows("Shop"))
		got, err := repository.NewSellerRepo(db).GetSellerByUserID(ctx, 2)
		requireErrorIs(t, err, nil)
		if got.ID != 1 {
			t.Fatalf("unexpected seller: %+v", got)
		}
	})
	t.Run("scan failure", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("SELECT (.+) FROM sellers WHERE user_id = \\$1").WithArgs(int64(2)).WillReturnError(unitErr)
		_, err := repository.NewSellerRepo(db).GetSellerByUserID(ctx, 2)
		requireErrorIs(t, err, repository.ErrToScan)
	})
	t.Run("not found", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("SELECT (.+) FROM sellers WHERE user_id = \\$1").WithArgs(int64(2)).WillReturnError(pgx.ErrNoRows)
		_, err := repository.NewSellerRepo(db).GetSellerByUserID(ctx, 2)
		requireErrorIs(t, err, service.ErrNotFound)
	})
}

func TestSellerRepo_GetSellerStats(t *testing.T) {
	ctx := context.Background()

	from, to := unitNow.Add(-time.Hour), unitNow
	t.Run("success", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("SELECT (.+) FROM get_seller_statistics").WithArgs(int64(1), from, to).WillReturnRows(pgxmock.NewRows([]string{"total_orders", "total_revenue", "avg_order_value", "top_product_name"}).AddRow(int64(2), decimal.NewFromInt(200), decimal.NewFromInt(100), "Product"))
		got, err := repository.NewSellerRepo(db).GetSellerStats(ctx, 1, from, to)
		requireErrorIs(t, err, nil)
		if got.TotalOrders != 2 {
			t.Fatalf("unexpected stats: %+v", got)
		}
	})
	t.Run("scan failure", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("SELECT (.+) FROM get_seller_statistics").WithArgs(int64(1), from, to).WillReturnError(unitErr)
		_, err := repository.NewSellerRepo(db).GetSellerStats(ctx, 1, from, to)
		requireErrorIs(t, err, repository.ErrToScan)
	})
	t.Run("not found", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("SELECT (.+) FROM get_seller_statistics").WithArgs(int64(1), from, to).WillReturnError(pgx.ErrNoRows)
		_, err := repository.NewSellerRepo(db).GetSellerStats(ctx, 1, from, to)
		requireErrorIs(t, err, service.ErrNotFound)
	})
}

func TestSellerRepo_CreateSeller(t *testing.T) {
	ctx := context.Background()

	create := m.SellerCreate{UserID: 2, CompanyName: "Shop"}
	t.Run("success", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("INSERT INTO sellers (.+) RETURNING id").WithArgs(create.UserID, create.CompanyName, pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(int64(1)))
		id, err := repository.NewSellerRepo(db).CreateSeller(ctx, create)
		requireErrorIs(t, err, nil)
		if id != 1 {
			t.Fatalf("id: got %d, want 1", id)
		}
	})
	t.Run("scan failure", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("INSERT INTO sellers (.+) RETURNING id").WithArgs(create.UserID, create.CompanyName, pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnError(unitErr)
		_, err := repository.NewSellerRepo(db).CreateSeller(ctx, create)
		requireErrorIs(t, err, repository.ErrToScan)
	})
}

func TestSellerRepo_UpdateSeller(t *testing.T) {
	ctx := context.Background()

	name := "New Shop"
	t.Run("success", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("UPDATE sellers SET company_name = \\$1 WHERE id = \\$2 RETURNING (.+)").WithArgs(name, int64(1)).WillReturnRows(sellerRows(name))
		got, err := repository.NewSellerRepo(db).UpdateSeller(ctx, 1, m.SellerUpdate{CompanyName: &name})
		requireErrorIs(t, err, nil)
		if got.ID != 1 || got.CompanyName != name {
			t.Fatalf("unexpected seller: %+v", got)
		}
	})
	t.Run("empty update", func(t *testing.T) {
		_, err := repository.NewSellerRepo(newMockPool(t)).UpdateSeller(ctx, 1, m.SellerUpdate{})
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
			db.ExpectQuery("UPDATE sellers SET company_name = \\$1 WHERE id = \\$2 RETURNING (.+)").WithArgs(name, int64(1)).WillReturnError(tc.dbErr)
			r := repository.NewSellerRepo(db)
			_, err := r.UpdateSeller(ctx, 1, m.SellerUpdate{CompanyName: &name})
			requireErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestSellerRepo_DeleteSellerByID(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectExec("DELETE FROM sellers WHERE id = \\$1").WithArgs(int64(1)).WillReturnResult(pgxmock.NewResult("DELETE", 1))
		err := repository.NewSellerRepo(db).DeleteSellerByID(ctx, 1)
		requireErrorIs(t, err, nil)
	})
	t.Run("exec failure", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectExec("DELETE FROM sellers WHERE id = \\$1").WithArgs(int64(1)).WillReturnError(unitErr)
		err := repository.NewSellerRepo(db).DeleteSellerByID(ctx, 1)
		requireErrorIs(t, err, repository.ErrExec)
	})
}

func sellerRows(company string) *pgxmock.Rows {
	columns := []string{"id", "user_id", "company_name", "description", "rating", "created_at"}
	return pgxmock.NewRows(columns).AddRow(int64(1), int64(2), company, nil, nil, unitNow)
}
