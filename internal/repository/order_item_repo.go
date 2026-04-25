package repository

import (
	"context"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	m "github.com/beastixq/marketplace/internal/model"
	"github.com/beastixq/marketplace/internal/service"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OrderItemRepoImpl struct {
	pool *pgxpool.Pool
}

func NewOrderItemRepo(pool *pgxpool.Pool) OrderItemRepoImpl {
	return OrderItemRepoImpl{pool: pool}
}

func (oir OrderItemRepoImpl) GetOrderItemByID(ctx context.Context, id int64) (oi m.OrderItem, err error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	sql, args, err := psql.Select("id", "order_id", "product_id", "quantity", "price_at_purchase").From("order_items").Where(sq.Eq{"id": id}).ToSql()
	if err != nil {
		return m.OrderItem{}, fmt.Errorf("%w: %v", ErrToSql, err)
	}
	row := getConn(ctx, oir.pool).QueryRow(ctx, sql, args...)
	var ordItem orderItemRow
	if err = row.Scan(&ordItem.ID, &ordItem.OrderID, &ordItem.ProductID, &ordItem.Quantity, &ordItem.PriceAtPurchase); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return m.OrderItem{}, service.ErrNotFound
		}
		return m.OrderItem{}, fmt.Errorf("%w: %v", ErrToScan, err)
	}
	return ordItem.toModel(), nil
}

func (oir OrderItemRepoImpl) GetOrderItemsByOrderID(ctx context.Context, orderID int64) (ois []m.OrderItem, err error) {
	oisql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	sql, args, err := oisql.Select("id", "order_id", "product_id", "quantity", "price_at_purchase").From("order_items").Where(sq.Eq{"order_id": orderID}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrToSql, err)
	}
	rows, err := getConn(ctx, oir.pool).Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrQuery, err)
	}
	defer rows.Close()
	ois = make([]m.OrderItem, 0)
	var prow orderItemRow
	for rows.Next() {
		err = rows.Scan(&prow.ID, &prow.OrderID, &prow.ProductID, &prow.Quantity, &prow.PriceAtPurchase)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrToScan, err)
		}
		ois = append(ois, prow.toModel())
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRowsIteration, err)
	}
	return ois, nil
}

func (oir OrderItemRepoImpl) CreateOrderItem(ctx context.Context, oic m.OrderItemCreate) (id int64, err error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	sql, args, err := psql.Insert("order_items").Columns("order_id", "product_id", "quantity", "price_at_purchase").Values(oic.OrderID, oic.ProductID, oic.Quantity, oic.PriceAtPurchase).Suffix("RETURNING id").ToSql()
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrToSql, err)
	}
	row := getConn(ctx, oir.pool).QueryRow(ctx, sql, args...)
	if err = row.Scan(&id); err != nil {
		return 0, fmt.Errorf("%w: %v", ErrToScan, err)
	}
	return id, nil
}

func (oir OrderItemRepoImpl) UpdateOrderItem(ctx context.Context, id int64, oiu m.OrderItemUpdate) (oi m.OrderItem, err error) {
	if oiu.OrderID == nil && oiu.ProductID == nil && oiu.Quantity == nil && oiu.PriceAtPurchase == nil {
		return m.OrderItem{}, service.ErrNoChangesInUpdate
	}

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	ub := psql.Update("order_items").Where(sq.Eq{"id": id})
	if oiu.OrderID != nil {
		ub = ub.Set("order_id", *oiu.OrderID)
	}
	if oiu.ProductID != nil {
		ub = ub.Set("product_id", *oiu.ProductID)
	}
	if oiu.Quantity != nil {
		ub = ub.Set("quantity", *oiu.Quantity)
	}
	if oiu.PriceAtPurchase != nil {
		ub = ub.Set("price_at_purchase", *oiu.PriceAtPurchase)
	}
	ub = ub.Suffix("RETURNING id, order_id, product_id, quantity, price_at_purchase")
	sql, args, err := ub.ToSql()
	if err != nil {
		return m.OrderItem{}, fmt.Errorf("%w: %v", ErrToSql, err)
	}
	row := getConn(ctx, oir.pool).QueryRow(ctx, sql, args...)
	var ordItem orderItemRow
	if err = row.Scan(&ordItem.ID, &ordItem.OrderID, &ordItem.ProductID, &ordItem.Quantity, &ordItem.PriceAtPurchase); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return m.OrderItem{}, service.ErrNotFound
		}
		return m.OrderItem{}, fmt.Errorf("%w: %v", ErrToScan, err)
	}
	return ordItem.toModel(), nil
}

// UpdateOrderItemQtyIfDraft updates an order item's quantity only when the
// owning order belongs to userID and is still in draft status. The conditional
// EXISTS clause enforces ownership + status invariants atomically at SQL level.
func (oir OrderItemRepoImpl) UpdateOrderItemQtyIfDraft(ctx context.Context, id int64, userID int64, qty int) error {
	const sql = `
UPDATE order_items oi
SET quantity = $2
WHERE oi.id = $1
  AND EXISTS (
    SELECT 1 FROM orders o
    WHERE o.id = oi.order_id
      AND o.user_id = $3
      AND o.status = 'draft'
  )
RETURNING id`
	var updatedID int64
	if err := getConn(ctx, oir.pool).QueryRow(ctx, sql, id, qty, userID).Scan(&updatedID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return service.ErrNotFound
		}
		return fmt.Errorf("%w: %v", ErrToScan, err)
	}
	return nil
}

// DeleteOrderItemIfDraft deletes an order item only when the owning order
// belongs to userID and is still in draft status.
func (oir OrderItemRepoImpl) DeleteOrderItemIfDraft(ctx context.Context, id int64, userID int64) error {
	const sql = `
DELETE FROM order_items oi
WHERE oi.id = $1
  AND EXISTS (
    SELECT 1 FROM orders o
    WHERE o.id = oi.order_id
      AND o.user_id = $2
      AND o.status = 'draft'
  )`
	res, err := getConn(ctx, oir.pool).Exec(ctx, sql, id, userID)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrExec, err)
	}
	if res.RowsAffected() == 0 {
		return service.ErrNotFound
	}
	return nil
}

func (oir OrderItemRepoImpl) DeleteOrderItemByID(ctx context.Context, id int64) (err error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	sql, args, err := psql.Delete("order_items").Where(sq.Eq{"id": id}).ToSql()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrToSql, err)
	}
	if _, err = getConn(ctx, oir.pool).Exec(ctx, sql, args...); err != nil {
		return fmt.Errorf("%w: %v", ErrExec, err)
	}
	return nil
}
