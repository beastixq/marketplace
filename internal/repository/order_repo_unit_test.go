package repository_test

import (
	"context"
	"testing"

	m "github.com/beastixq/marketplace/internal/model"
	"github.com/beastixq/marketplace/internal/repository"
	"github.com/beastixq/marketplace/internal/service"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v5"
	"github.com/shopspring/decimal"
)

func TestOrderRepo_GetOrderByID(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("SELECT (.+) FROM orders WHERE id = \\$1").WithArgs(int64(1)).WillReturnRows(orderRows(m.StatusPending))
		got, err := repository.NewOrderRepo(db).GetOrderByID(ctx, 1)
		requireErrorIs(t, err, nil)
		if got.ID != 1 {
			t.Fatalf("unexpected order: %+v", got)
		}
	})
	t.Run("scan failure", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("SELECT (.+) FROM orders WHERE id = \\$1").WithArgs(int64(1)).WillReturnError(unitErr)
		_, err := repository.NewOrderRepo(db).GetOrderByID(ctx, 1)
		requireErrorIs(t, err, repository.ErrToScan)
	})
	t.Run("not found", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("SELECT (.+) FROM orders WHERE id = \\$1").WithArgs(int64(1)).WillReturnError(pgx.ErrNoRows)
		_, err := repository.NewOrderRepo(db).GetOrderByID(ctx, 1)
		requireErrorIs(t, err, service.ErrNotFound)
	})
}

func TestOrderRepo_GetOrdersByUserID(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("SELECT (.+) FROM orders WHERE user_id = \\$1 ORDER BY created_at DESC").WithArgs(int64(2)).WillReturnRows(orderRows(m.StatusPending)).RowsWillBeClosed()
		got, err := repository.NewOrderRepo(db).GetOrdersByUserID(ctx, 2, m.PaginationOpts{})
		requireErrorIs(t, err, nil)
		if len(got) != 1 {
			t.Fatalf("unexpected orders: %+v", got)
		}
	})
	t.Run("query failure", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("SELECT (.+) FROM orders WHERE user_id = \\$1 ORDER BY created_at DESC").WithArgs(int64(2)).WillReturnError(unitErr)
		_, err := repository.NewOrderRepo(db).GetOrdersByUserID(ctx, 2, m.PaginationOpts{})
		requireErrorIs(t, err, repository.ErrQuery)
	})
	for _, tc := range []struct {
		name    string
		rows    *pgxmock.Rows
		wantErr error
	}{
		{name: "scan failure", rows: pgxmock.NewRows([]string{"id", "user_id", "address_id", "seller_id", "status", "total_amount", "created_at", "updated_at"}).AddRow("invalid ID", int64(2), nil, nil, string(m.StatusPending), decimal.NewFromInt(100), unitNow, unitNow), wantErr: repository.ErrToScan},
		{name: "iteration failure", rows: orderRows(m.StatusPending).CloseError(unitErr), wantErr: repository.ErrRowsIteration},
	} {
		t.Run(tc.name, func(t *testing.T) {

			db := newMockPool(t)
			db.ExpectQuery("SELECT (.+) FROM orders WHERE user_id = \\$1 ORDER BY created_at DESC").WithArgs(int64(2)).WillReturnRows(tc.rows).RowsWillBeClosed()
			_, err := repository.NewOrderRepo(db).GetOrdersByUserID(ctx, 2, m.PaginationOpts{})
			requireErrorIs(t, err, tc.wantErr)

		})
	}
}

func TestOrderRepo_GetSellerOrdersBySellerID(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("SELECT (.+) FROM orders WHERE seller_id = \\$1 ORDER BY created_at DESC").WithArgs(int64(3)).WillReturnRows(orderRows(m.StatusPending)).RowsWillBeClosed()
		got, err := repository.NewOrderRepo(db).GetSellerOrdersBySellerID(ctx, 3, m.PaginationOpts{})
		requireErrorIs(t, err, nil)
		if len(got) != 1 {
			t.Fatalf("unexpected orders: %+v", got)
		}
	})
	t.Run("query failure", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("SELECT (.+) FROM orders WHERE seller_id = \\$1 ORDER BY created_at DESC").WithArgs(int64(3)).WillReturnError(unitErr)
		_, err := repository.NewOrderRepo(db).GetSellerOrdersBySellerID(ctx, 3, m.PaginationOpts{})
		requireErrorIs(t, err, repository.ErrQuery)
	})
	for _, tc := range []struct {
		name    string
		rows    *pgxmock.Rows
		wantErr error
	}{
		{name: "scan failure", rows: pgxmock.NewRows([]string{"id", "user_id", "address_id", "seller_id", "status", "total_amount", "created_at", "updated_at"}).AddRow("invalid ID", int64(2), nil, nil, string(m.StatusPending), decimal.NewFromInt(100), unitNow, unitNow), wantErr: repository.ErrToScan},
		{name: "iteration failure", rows: orderRows(m.StatusPending).CloseError(unitErr), wantErr: repository.ErrRowsIteration},
	} {
		t.Run(tc.name, func(t *testing.T) {

			db := newMockPool(t)
			db.ExpectQuery("SELECT (.+) FROM orders WHERE seller_id = \\$1 ORDER BY created_at DESC").WithArgs(int64(3)).WillReturnRows(tc.rows).RowsWillBeClosed()
			_, err := repository.NewOrderRepo(db).GetSellerOrdersBySellerID(ctx, 3, m.PaginationOpts{})
			requireErrorIs(t, err, tc.wantErr)

		})
	}
}

func TestOrderRepo_GetExpiredPendingOrders(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("SELECT (.+) FROM orders WHERE status = \\$1 AND created_at <= \\$2").WithArgs("pending", unitNow).WillReturnRows(orderRows(m.StatusPending)).RowsWillBeClosed()
		got, err := repository.NewOrderRepo(db).GetExpiredPendingOrders(ctx, unitNow)
		requireErrorIs(t, err, nil)
		if len(got) != 1 {
			t.Fatalf("unexpected orders: %+v", got)
		}
	})
	t.Run("query failure", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("SELECT (.+) FROM orders WHERE status = \\$1 AND created_at <= \\$2").WithArgs("pending", unitNow).WillReturnError(unitErr)
		_, err := repository.NewOrderRepo(db).GetExpiredPendingOrders(ctx, unitNow)
		requireErrorIs(t, err, repository.ErrQuery)
	})
	for _, tc := range []struct {
		name    string
		rows    *pgxmock.Rows
		wantErr error
	}{
		{name: "scan failure", rows: pgxmock.NewRows([]string{"id", "user_id", "address_id", "seller_id", "status", "total_amount", "created_at", "updated_at"}).AddRow("invalid ID", int64(2), nil, nil, string(m.StatusPending), decimal.NewFromInt(100), unitNow, unitNow), wantErr: repository.ErrToScan},
		{name: "iteration failure", rows: orderRows(m.StatusPending).CloseError(unitErr), wantErr: repository.ErrRowsIteration},
	} {
		t.Run(tc.name, func(t *testing.T) {

			db := newMockPool(t)
			db.ExpectQuery("SELECT (.+) FROM orders WHERE status = \\$1 AND created_at <= \\$2").WithArgs("pending", unitNow).WillReturnRows(tc.rows).RowsWillBeClosed()
			_, err := repository.NewOrderRepo(db).GetExpiredPendingOrders(ctx, unitNow)
			requireErrorIs(t, err, tc.wantErr)

		})
	}
}

func TestOrderRepo_CreateOrder(t *testing.T) {
	ctx := context.Background()

	create := m.OrderCreate{UserID: 2, SellerID: ptr(int64(3)), Status: m.StatusDraft, TotalAmount: decimal.Zero}
	t.Run("success", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("INSERT INTO orders (.+) RETURNING id").WithArgs(create.UserID, pgxmock.AnyArg(), pgxmock.AnyArg(), create.Status, create.TotalAmount).WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(int64(1)))
		id, err := repository.NewOrderRepo(db).CreateOrder(ctx, create)
		requireErrorIs(t, err, nil)
		if id != 1 {
			t.Fatalf("id: got %d, want 1", id)
		}
	})
	t.Run("scan failure", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("INSERT INTO orders (.+) RETURNING id").WithArgs(create.UserID, pgxmock.AnyArg(), pgxmock.AnyArg(), create.Status, create.TotalAmount).WillReturnError(unitErr)
		_, err := repository.NewOrderRepo(db).CreateOrder(ctx, create)
		requireErrorIs(t, err, repository.ErrToScan)
	})
}

func TestOrderRepo_UpdateOrder(t *testing.T) {
	ctx := context.Background()

	status := m.StatusPending
	t.Run("success", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("UPDATE orders SET status = \\$1 WHERE id = \\$2 RETURNING (.+)").WithArgs(status, int64(1)).WillReturnRows(orderRows(status))
		got, err := repository.NewOrderRepo(db).UpdateOrder(ctx, 1, m.OrderUpdate{Status: &status})
		requireErrorIs(t, err, nil)
		if got.ID != 1 || got.Status != status {
			t.Fatalf("unexpected order: %+v", got)
		}
	})
	t.Run("empty update", func(t *testing.T) {
		_, err := repository.NewOrderRepo(newMockPool(t)).UpdateOrder(ctx, 1, m.OrderUpdate{})
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
			db.ExpectQuery("UPDATE orders SET status = \\$1 WHERE id = \\$2 RETURNING (.+)").WithArgs(status, int64(1)).WillReturnError(tc.dbErr)
			r := repository.NewOrderRepo(db)
			_, err := r.UpdateOrder(ctx, 1, m.OrderUpdate{Status: &status})
			requireErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestOrderRepo_UpdateOrderStatus(t *testing.T) {
	ctx := context.Background()

	from := []m.OrderStatus{m.StatusPending}
	t.Run("success", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectExec("UPDATE orders SET status = \\$1, updated_at = NOW\\(\\) WHERE id = \\$2 AND status IN \\(\\$3\\)").WithArgs(m.StatusPaid, int64(1), m.StatusPending).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		err := repository.NewOrderRepo(db).UpdateOrderStatus(ctx, 1, from, m.StatusPaid)
		requireErrorIs(t, err, nil)
	})
	t.Run("concurrent transition", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectExec("UPDATE orders SET status = \\$1, updated_at = NOW\\(\\) WHERE id = \\$2 AND status IN \\(\\$3\\)").WithArgs(m.StatusPaid, int64(1), m.StatusPending).WillReturnResult(pgxmock.NewResult("UPDATE", 0))
		err := repository.NewOrderRepo(db).UpdateOrderStatus(ctx, 1, from, m.StatusPaid)
		requireErrorIs(t, err, service.ErrNotFound)
	})
}

func TestOrderRepo_DeleteOrderByID(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectExec("DELETE FROM orders WHERE id = \\$1").WithArgs(int64(1)).WillReturnResult(pgxmock.NewResult("DELETE", 1))
		err := repository.NewOrderRepo(db).DeleteOrderByID(ctx, 1)
		requireErrorIs(t, err, nil)
	})
	t.Run("exec failure", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectExec("DELETE FROM orders WHERE id = \\$1").WithArgs(int64(1)).WillReturnError(unitErr)
		err := repository.NewOrderRepo(db).DeleteOrderByID(ctx, 1)
		requireErrorIs(t, err, repository.ErrExec)
	})
}

func TestOrderRepo_LockUserCart(t *testing.T) {
	ctx := context.Background()

	t.Run("success in transaction", func(t *testing.T) {
		db := newMockPool(t)
		tx := beginMockTransaction(t, db, ctx)
		db.ExpectExec("SELECT pg_advisory_xact_lock\\(\\$1\\)").WithArgs(int64(1_000_000_000_002)).WillReturnResult(pgxmock.NewResult("SELECT", 1))

		err := repository.NewOrderRepo(db).LockUserCart(repository.SetTxCtx(ctx, tx), 2)

		requireErrorIs(t, err, nil)
	})
	t.Run("without transaction", func(t *testing.T) {
		err := repository.NewOrderRepo(newMockPool(t)).LockUserCart(ctx, 2)
		requireErrorIs(t, err, service.ErrMustBeInTransaction)
	})
}

func orderRows(status m.OrderStatus) *pgxmock.Rows {
	columns := []string{"id", "user_id", "address_id", "seller_id", "status", "total_amount", "created_at", "updated_at"}
	return pgxmock.NewRows(columns).AddRow(int64(1), int64(2), nil, nil, string(status), decimal.NewFromInt(100), unitNow, unitNow)
}
