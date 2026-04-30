package generators

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
)

type seedOrderItem struct {
	orderID   int64
	productID int64
	quantity  int
	price     float64
}

func CreateOrderItems(tx pgx.Tx, ctx context.Context, ordersIDs, productsIDs []int64, seedOrders SeedOrders, productsBySeller map[int64][]int64, targetCount int) error {
	productPrices, err := queryProductPrices(tx, ctx, productsIDs)
	if err != nil {
		return fmt.Errorf("OrderItems query product prices: %v", err)
	}

	maxItemSlots := 0
	for _, orderID := range ordersIDs {
		maxItemSlots += len(productPoolForSeedOrder(seedOrders[orderID], productsIDs, productsBySeller))
	}
	if targetCount < len(ordersIDs) {
		targetCount = len(ordersIDs)
	}
	if targetCount > maxItemSlots {
		targetCount = maxItemSlots
	}

	items := make([]seedOrderItem, 0, targetCount)
	usedByOrder := make(map[int64]map[int64]struct{}, len(ordersIDs))

	for _, orderID := range ordersIDs {
		added, err := appendRandomSeedOrderItem(&items, usedByOrder, orderID, seedOrders[orderID], productsIDs, productsBySeller, productPrices)
		if err != nil {
			return err
		}
		if !added && !seedOrders[orderID].IsDraft() {
			return fmt.Errorf("OrderItems no products for non-draft order %d", orderID)
		}
	}

	availableOrderIDs := make([]int64, 0, len(ordersIDs))
	for _, orderID := range ordersIDs {
		if hasSeedOrderItemCapacity(usedByOrder, orderID, seedOrders[orderID], productsIDs, productsBySeller) {
			availableOrderIDs = append(availableOrderIDs, orderID)
		}
	}
	for len(items) < targetCount && len(availableOrderIDs) > 0 {
		idx := rand.Intn(len(availableOrderIDs))
		orderID := availableOrderIDs[idx]
		added, err := appendRandomSeedOrderItem(&items, usedByOrder, orderID, seedOrders[orderID], productsIDs, productsBySeller, productPrices)
		if err != nil {
			return err
		}
		if !added || !hasSeedOrderItemCapacity(usedByOrder, orderID, seedOrders[orderID], productsIDs, productsBySeller) {
			availableOrderIDs[idx] = availableOrderIDs[len(availableOrderIDs)-1]
			availableOrderIDs = availableOrderIDs[:len(availableOrderIDs)-1]
		}
	}

	if err := insertSeedOrderItems(tx, ctx, items); err != nil {
		return err
	}
	if err := syncSeedOrderTotals(tx, ctx, ordersIDs); err != nil {
		return err
	}
	if err := syncSeedProductReservations(tx, ctx, productsIDs); err != nil {
		return err
	}
	return nil
}

func appendRandomSeedOrderItem(items *[]seedOrderItem, usedByOrder map[int64]map[int64]struct{}, orderID int64, order SeedOrder, productsIDs []int64, productsBySeller map[int64][]int64, productPrices map[int64]float64) (bool, error) {
	pool := productPoolForSeedOrder(order, productsIDs, productsBySeller)
	if len(pool) == 0 {
		return false, nil
	}

	used := usedByOrder[orderID]
	if used == nil {
		used = make(map[int64]struct{})
		usedByOrder[orderID] = used
	}
	if len(used) >= len(pool) {
		return false, nil
	}

	start := rand.Intn(len(pool))
	var productID int64
	for offset := 0; offset < len(pool); offset++ {
		candidate := pool[(start+offset)%len(pool)]
		if _, ok := used[candidate]; !ok {
			productID = candidate
			break
		}
	}
	if productID == 0 {
		return false, nil
	}

	price, ok := productPrices[productID]
	if !ok {
		return false, fmt.Errorf("OrderItems product %d has no price", productID)
	}

	used[productID] = struct{}{}
	*items = append(*items, seedOrderItem{
		orderID:   orderID,
		productID: productID,
		quantity:  rand.Intn(10) + 1,
		price:     price,
	})
	return true, nil
}

func productPoolForSeedOrder(order SeedOrder, productsIDs []int64, productsBySeller map[int64][]int64) []int64 {
	if !order.IsDraft() {
		return productsBySeller[order.SellerID]
	}
	return productsIDs
}

func hasSeedOrderItemCapacity(usedByOrder map[int64]map[int64]struct{}, orderID int64, order SeedOrder, productsIDs []int64, productsBySeller map[int64][]int64) bool {
	return len(usedByOrder[orderID]) < len(productPoolForSeedOrder(order, productsIDs, productsBySeller))
}

func insertSeedOrderItems(tx pgx.Tx, ctx context.Context, items []seedOrderItem) error {
	if len(items) == 0 {
		return nil
	}

	psql := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)
	insertBuilder := psql.
		Insert("order_items").
		Columns("order_id", "product_id", "quantity", "price_at_purchase")
	inBatch := 0

	for i, item := range items {
		insertBuilder = insertBuilder.Values(item.orderID, item.productID, item.quantity, item.price)
		inBatch++

		if inBatch >= 500 || i == len(items)-1 {
			sql, args, err := insertBuilder.ToSql()
			if err != nil {
				return fmt.Errorf("OrderItems %s: %v", ErrToSql, err)
			}
			_, err = tx.Exec(ctx, sql, args...)
			if err != nil {
				return fmt.Errorf("OrderItems %s: %v", ErrQuery, err)
			}
			insertBuilder = psql.Insert("order_items").Columns("order_id", "product_id", "quantity", "price_at_purchase")
			inBatch = 0
		}
	}
	return nil
}

func syncSeedOrderTotals(tx pgx.Tx, ctx context.Context, orderIDs []int64) error {
	if len(orderIDs) == 0 {
		return nil
	}

	const sql = `
UPDATE orders o
SET total_amount = totals.total_amount
FROM (
    SELECT o2.id, COALESCE(SUM(oi.quantity * oi.price_at_purchase), 0) AS total_amount
    FROM orders o2
    LEFT JOIN order_items oi ON oi.order_id = o2.id
    WHERE o2.id = ANY($1::bigint[])
    GROUP BY o2.id
) totals
WHERE o.id = totals.id`
	if _, err := tx.Exec(ctx, sql, orderIDs); err != nil {
		return fmt.Errorf("OrderItems sync order totals %s: %v", ErrQuery, err)
	}
	return nil
}

func syncSeedProductReservations(tx pgx.Tx, ctx context.Context, productIDs []int64) error {
	if len(productIDs) == 0 {
		return nil
	}

	const sql = `
UPDATE products p
SET reserved_quantity = totals.reserved_quantity,
    stock_quantity = GREATEST(p.stock_quantity, totals.reserved_quantity)
FROM (
    SELECT p2.id,
           COALESCE(SUM(oi.quantity) FILTER (WHERE o.status IN ('pending', 'paid')), 0)::integer AS reserved_quantity
    FROM products p2
    LEFT JOIN order_items oi ON oi.product_id = p2.id
    LEFT JOIN orders o ON o.id = oi.order_id
    WHERE p2.id = ANY($1::bigint[])
    GROUP BY p2.id
) totals
WHERE p.id = totals.id`
	if _, err := tx.Exec(ctx, sql, productIDs); err != nil {
		return fmt.Errorf("OrderItems sync product reservations %s: %v", ErrQuery, err)
	}
	return nil
}
