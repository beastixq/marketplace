package repository_test

import (
	"context"
	"testing"

	m "github.com/beastixq/marketplace/internal/model"
	"github.com/beastixq/marketplace/internal/repository"
	"github.com/beastixq/marketplace/internal/service"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v5"
	"github.com/shopspring/decimal"
)

func TestOrderItemRepo_GetOrderItemByID(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("SELECT (.+) FROM order_items WHERE id = \\$1").WithArgs(int64(1)).WillReturnRows(orderItemRows(4))
		got, err := repository.NewOrderItemRepo(db).GetOrderItemByID(ctx, 1)
		requireErrorIs(t, err, nil)
		if got.ID != 1 {
			t.Fatalf("unexpected item: %+v", got)
		}
	})
	t.Run("scan failure", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("SELECT (.+) FROM order_items WHERE id = \\$1").WithArgs(int64(1)).WillReturnError(unitErr)
		_, err := repository.NewOrderItemRepo(db).GetOrderItemByID(ctx, 1)
		requireErrorIs(t, err, repository.ErrToScan)
	})
	t.Run("not found", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("SELECT (.+) FROM order_items WHERE id = \\$1").WithArgs(int64(1)).WillReturnError(pgx.ErrNoRows)
		_, err := repository.NewOrderItemRepo(db).GetOrderItemByID(ctx, 1)
		requireErrorIs(t, err, service.ErrNotFound)
	})
}

func TestOrderItemRepo_GetOrderItemsByOrderID(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("SELECT (.+) FROM order_items WHERE order_id = \\$1").WithArgs(int64(2)).WillReturnRows(orderItemRows(4)).RowsWillBeClosed()
		got, err := repository.NewOrderItemRepo(db).GetOrderItemsByOrderID(ctx, 2)
		requireErrorIs(t, err, nil)
		if len(got) != 1 {
			t.Fatalf("unexpected items: %+v", got)
		}
	})
	t.Run("query failure", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("SELECT (.+) FROM order_items WHERE order_id = \\$1").WithArgs(int64(2)).WillReturnError(unitErr)
		_, err := repository.NewOrderItemRepo(db).GetOrderItemsByOrderID(ctx, 2)
		requireErrorIs(t, err, repository.ErrQuery)
	})
	for _, tc := range []struct {
		name    string
		rows    *pgxmock.Rows
		wantErr error
	}{
		{name: "scan failure", rows: pgxmock.NewRows([]string{"id", "order_id", "product_id", "quantity", "price_at_purchase"}).AddRow("invalid ID", int64(2), int64(3), 4, decimal.NewFromInt(100)), wantErr: repository.ErrToScan},
		{name: "iteration failure", rows: orderItemRows(4).CloseError(unitErr), wantErr: repository.ErrRowsIteration},
	} {
		t.Run(tc.name, func(t *testing.T) {

			db := newMockPool(t)
			db.ExpectQuery("SELECT (.+) FROM order_items WHERE order_id = \\$1").WithArgs(int64(2)).WillReturnRows(tc.rows).RowsWillBeClosed()
			_, err := repository.NewOrderItemRepo(db).GetOrderItemsByOrderID(ctx, 2)
			requireErrorIs(t, err, tc.wantErr)

		})
	}
}

func TestOrderItemRepo_CreateOrderItem(t *testing.T) {
	ctx := context.Background()

	create := m.OrderItemCreate{OrderID: 2, ProductID: 3, Quantity: 4, PriceAtPurchase: decimal.NewFromInt(100)}
	t.Run("success", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("INSERT INTO order_items (.+) RETURNING id").WithArgs(create.OrderID, create.ProductID, create.Quantity, create.PriceAtPurchase).WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(int64(1)))
		id, err := repository.NewOrderItemRepo(db).CreateOrderItem(ctx, create)
		requireErrorIs(t, err, nil)
		if id != 1 {
			t.Fatalf("id: got %d, want 1", id)
		}
	})
	t.Run("scan failure", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("INSERT INTO order_items (.+) RETURNING id").WithArgs(create.OrderID, create.ProductID, create.Quantity, create.PriceAtPurchase).WillReturnError(unitErr)
		_, err := repository.NewOrderItemRepo(db).CreateOrderItem(ctx, create)
		requireErrorIs(t, err, repository.ErrToScan)
	})
	for _, tc := range []struct {
		name       string
		constraint string
		wantErr    error
	}{
		{name: "duplicate cart product", constraint: "uq_order_items_order_id_product_id", wantErr: service.ErrProductAlreadyInCart},
		{name: "unrelated unique violation", constraint: "other_constraint", wantErr: repository.ErrToScan},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := newMockPool(t)
			db.ExpectQuery("INSERT INTO order_items (.+) RETURNING id").WithArgs(create.OrderID, create.ProductID, create.Quantity, create.PriceAtPurchase).WillReturnError(&pgconn.PgError{Code: pgerrcode.UniqueViolation, ConstraintName: tc.constraint})
			r := repository.NewOrderItemRepo(db)

			_, err := r.CreateOrderItem(ctx, create)

			requireErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestOrderItemRepo_UpdateOrderItem(t *testing.T) {
	ctx := context.Background()

	quantity := 5
	t.Run("success", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("UPDATE order_items SET quantity = \\$1 WHERE id = \\$2 RETURNING (.+)").WithArgs(quantity, int64(1)).WillReturnRows(orderItemRows(quantity))
		got, err := repository.NewOrderItemRepo(db).UpdateOrderItem(ctx, 1, m.OrderItemUpdate{Quantity: &quantity})
		requireErrorIs(t, err, nil)
		if got.ID != 1 || got.Quantity != quantity {
			t.Fatalf("unexpected item: %+v", got)
		}
	})
	t.Run("empty update", func(t *testing.T) {
		_, err := repository.NewOrderItemRepo(newMockPool(t)).UpdateOrderItem(ctx, 1, m.OrderItemUpdate{})
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
			db.ExpectQuery("UPDATE order_items SET quantity = \\$1 WHERE id = \\$2 RETURNING (.+)").WithArgs(quantity, int64(1)).WillReturnError(tc.dbErr)
			r := repository.NewOrderItemRepo(db)
			_, err := r.UpdateOrderItem(ctx, 1, m.OrderItemUpdate{Quantity: &quantity})
			requireErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestOrderItemRepo_UpdateOrderItemQtyIfDraft(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("(?s)UPDATE order_items oi.+RETURNING id").WithArgs(int64(1), 5, int64(2)).WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(int64(1)))
		err := repository.NewOrderItemRepo(db).UpdateOrderItemQtyIfDraft(ctx, 1, 2, 5)
		requireErrorIs(t, err, nil)
	})
	t.Run("scan failure", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("(?s)UPDATE order_items oi.+RETURNING id").WithArgs(int64(1), 5, int64(2)).WillReturnError(unitErr)
		err := repository.NewOrderItemRepo(db).UpdateOrderItemQtyIfDraft(ctx, 1, 2, 5)
		requireErrorIs(t, err, repository.ErrToScan)
	})
	t.Run("not found", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectQuery("(?s)UPDATE order_items oi.+RETURNING id").WithArgs(int64(1), 5, int64(2)).WillReturnError(pgx.ErrNoRows)
		err := repository.NewOrderItemRepo(db).UpdateOrderItemQtyIfDraft(ctx, 1, 2, 5)
		requireErrorIs(t, err, service.ErrNotFound)
	})
}

func TestOrderItemRepo_DeleteOrderItemIfDraft(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectExec("(?s)DELETE FROM order_items oi.+").WithArgs(int64(1), int64(2)).WillReturnResult(pgxmock.NewResult("DELETE", 1))
		err := repository.NewOrderItemRepo(db).DeleteOrderItemIfDraft(ctx, 1, 2)
		requireErrorIs(t, err, nil)
	})
	t.Run("no matching draft", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectExec("(?s)DELETE FROM order_items oi.+").WithArgs(int64(1), int64(2)).WillReturnResult(pgxmock.NewResult("DELETE", 0))
		err := repository.NewOrderItemRepo(db).DeleteOrderItemIfDraft(ctx, 1, 2)
		requireErrorIs(t, err, service.ErrNotFound)
	})
}

func TestOrderItemRepo_DeleteOrderItemByID(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectExec("DELETE FROM order_items WHERE id = \\$1").WithArgs(int64(1)).WillReturnResult(pgxmock.NewResult("DELETE", 1))
		err := repository.NewOrderItemRepo(db).DeleteOrderItemByID(ctx, 1)
		requireErrorIs(t, err, nil)
	})
	t.Run("exec failure", func(t *testing.T) {
		db := newMockPool(t)
		db.ExpectExec("DELETE FROM order_items WHERE id = \\$1").WithArgs(int64(1)).WillReturnError(unitErr)
		err := repository.NewOrderItemRepo(db).DeleteOrderItemByID(ctx, 1)
		requireErrorIs(t, err, repository.ErrExec)
	})
}

func orderItemRows(quantity int) *pgxmock.Rows {
	columns := []string{"id", "order_id", "product_id", "quantity", "price_at_purchase"}
	return pgxmock.NewRows(columns).AddRow(int64(1), int64(2), int64(3), quantity, decimal.NewFromInt(100))
}
