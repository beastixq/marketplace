package generators

import (
	"context"
	"fmt"
	"math"
	"math/rand"

	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
)

func weightedChoice(items []string, weights []int) string {
	total := 0
	for _, w := range weights {
		total += w
	}
	r := rand.Intn(total)
	for i, w := range weights {
		r -= w
		if r < 0 {
			return items[i]
		}
	}
	return items[len(items)-1]
}

// OrderSellers maps orderID → sellerID for non-draft orders.
// Draft orders are not in the map (seller_id is NULL).
type OrderSellers map[int64]int64

func CreateOrders(tx pgx.Tx, ctx context.Context, buyerIDs, addressesIDs []int64, sellersWithProducts []int64, count int) (ordersIDs []int64, orderSellers OrderSellers, err error) {
	createdInLoop := 0
	psql := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)
	insertBuilder := psql.Insert("orders").Columns("user_id", "address_id", "seller_id", "status", "total_amount")
	ordersIDs = make([]int64, count)
	sellerPerIndex := make([]int64, count) // 0 means draft (no seller)

	statuses := []string{"draft", "pending", "paid", "shipped", "delivered", "cancelled"}
	// draft 10%, pending 15%, paid 20%, shipped 20%, delivered 25%, cancelled 10%
	weights := []int{10, 15, 20, 20, 25, 10}

	for i := range count {
		buyerID := buyerIDs[rand.Intn(len(buyerIDs))]
		status := weightedChoice(statuses, weights)

		var addressID interface{}
		var sellerID interface{}
		var totalAmount float64

		if status == "draft" {
			addressID = nil
			sellerID = nil
			totalAmount = 0
		} else {
			addressID = addressesIDs[rand.Intn(len(addressesIDs))]
			sid := sellersWithProducts[rand.Intn(len(sellersWithProducts))]
			sellerID = sid
			sellerPerIndex[i] = sid
			totalAmount = math.Round((rand.Float64()*9999+1)*100) / 100
		}

		insertBuilder = insertBuilder.Values(buyerID, addressID, sellerID, status, totalAmount)
		createdInLoop++

		if i%10 == 9 || i == count-1 {
			sql, args, err := insertBuilder.Suffix("RETURNING id").ToSql()
			if err != nil {
				return nil, nil, fmt.Errorf("Orders %s: %v", ErrToSql, err)
			}
			rows, err := tx.Query(ctx, sql, args...)
			if err != nil {
				return nil, nil, fmt.Errorf("Orders %s: %v", ErrQuery, err)
			}
			curInd := i - createdInLoop + 1
			for rows.Next() {
				err = rows.Scan(&ordersIDs[curInd])
				if err != nil {
					return nil, nil, fmt.Errorf("Orders %s: %v", ErrScan, err)
				}
				curInd++
			}
			rows.Close()
			if err = rows.Err(); err != nil {
				return nil, nil, fmt.Errorf("Orders %s: %v", ErrCloseRows, err)
			}
			insertBuilder = psql.Insert("orders").Columns("user_id", "address_id", "seller_id", "status", "total_amount")
			createdInLoop = 0
		}
	}

	orderSellers = make(OrderSellers, count)
	for i, oid := range ordersIDs {
		if sellerPerIndex[i] != 0 {
			orderSellers[oid] = sellerPerIndex[i]
		}
	}

	return ordersIDs, orderSellers, nil
}
