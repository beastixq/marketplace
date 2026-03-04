package generators

import (
	"context"
	"fmt"
	"math"
	"math/rand"

	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
)

func CreateOrderItems(tx pgx.Tx, ctx context.Context, ordersIDs, productsIDs []int64, count int) error {
	createdInLoop := 0
	psql := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)
	insertBuilder := psql.Insert("order_items").Columns("order_id", "product_id", "quantity", "price_at_purchase")

	for i := range count {
		orderID := ordersIDs[rand.Intn(len(ordersIDs))]
		productID := productsIDs[rand.Intn(len(productsIDs))]
		quantity := rand.Intn(10) + 1
		price := math.Round((rand.Float64()*999+1)*100) / 100 // 1.00 — 1000.00

		insertBuilder = insertBuilder.Values(orderID, productID, quantity, price)
		createdInLoop++

		if i%10 == 9 || i == count-1 {
			sql, args, err := insertBuilder.ToSql()
			if err != nil {
				return fmt.Errorf("OrderItems %s: %v", ErrToSql, err)
			}
			_, err = tx.Exec(ctx, sql, args...)
			if err != nil {
				return fmt.Errorf("OrderItems %s: %v", ErrQuery, err)
			}
			insertBuilder = psql.Insert("order_items").Columns("order_id", "product_id", "quantity", "price_at_purchase")
			createdInLoop = 0
		}
	}
	return nil
}
