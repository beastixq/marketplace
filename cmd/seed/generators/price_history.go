package generators

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
)

// CreatePriceHistory generates realistic price change records for products.
// It queries actual product prices from DB and walks backward so the last
// new_price in the history matches the product's current price.
func CreatePriceHistory(tx pgx.Tx, ctx context.Context, productsIDs []int64, totalChanges int, dateFrom time.Time, dateTo time.Time) error {
	// Query actual prices for all products
	productPrices, err := queryProductPrices(tx, ctx, productsIDs)
	if err != nil {
		return fmt.Errorf("PriceHistory query prices: %v", err)
	}

	psql := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)
	insertBuilder := psql.Insert("product_price_history").
		Columns("product_id", "old_price", "new_price", "changed_at", "changed_by")
	inBatch := 0
	created := 0

	totalDuration := dateTo.Sub(dateFrom)
	changesPerProduct := distributeChanges(len(productsIDs), totalChanges)

	for i, pid := range productsIDs {
		n := changesPerProduct[i]
		if n == 0 {
			continue
		}

		currentPrice, ok := productPrices[pid]
		if !ok {
			continue
		}

		timestamps := generateTimestamps(dateFrom, totalDuration, n)

		// Walk backward from current price to generate history.
		// Build prices in reverse: start from currentPrice, apply inverse deltas.
		prices := make([]float64, n+1)
		prices[n] = currentPrice
		for j := n - 1; j >= 0; j-- {
			delta := generatePriceDelta()
			// Inverse: if new = old * (1+delta), then old = new / (1+delta)
			prices[j] = math.Round(prices[j+1]/(1+delta)*100) / 100
			if prices[j] < 1.00 {
				prices[j] = 1.00
			}
		}

		// prices[0] = oldest old_price, prices[n] = current price
		// Row j: old_price=prices[j], new_price=prices[j+1]
		for j := 0; j < n; j++ {
			insertBuilder = insertBuilder.Values(pid, prices[j], prices[j+1], timestamps[j], "seed:seller")
			inBatch++
			created++

			if inBatch >= 500 || created == totalChanges {
				sql, args, err := insertBuilder.ToSql()
				if err != nil {
					return fmt.Errorf("PriceHistory %s: %v", ErrToSql, err)
				}
				_, err = tx.Exec(ctx, sql, args...)
				if err != nil {
					return fmt.Errorf("PriceHistory %s: %v", ErrQuery, err)
				}
				insertBuilder = psql.Insert("product_price_history").
					Columns("product_id", "old_price", "new_price", "changed_at", "changed_by")
				inBatch = 0
			}
		}
	}
	return nil
}

func queryProductPrices(tx pgx.Tx, ctx context.Context, productsIDs []int64) (map[int64]float64, error) {
	psql := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)
	sql, args, err := psql.Select("id", "price").From("products").Where(squirrel.Eq{"id": productsIDs}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("%s: %v", ErrToSql, err)
	}
	rows, err := tx.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", ErrQuery, err)
	}
	defer rows.Close()

	prices := make(map[int64]float64, len(productsIDs))
	for rows.Next() {
		var id int64
		var price float64
		if err := rows.Scan(&id, &price); err != nil {
			return nil, fmt.Errorf("%s: %v", ErrScan, err)
		}
		prices[id] = price
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %v", ErrReadRows, err)
	}
	return prices, nil
}

// distributeChanges spreads totalChanges across numProducts.
// Each product gets at least 1 change, the rest is distributed randomly.
func distributeChanges(numProducts, totalChanges int) []int {
	counts := make([]int, numProducts)

	if totalChanges < numProducts {
		for i := 0; i < totalChanges; i++ {
			counts[rand.Intn(numProducts)]++
		}
		return counts
	}

	remaining := totalChanges
	for i := range counts {
		counts[i] = 1
		remaining--
	}

	for remaining > 0 {
		counts[rand.Intn(numProducts)]++
		remaining--
	}
	return counts
}

// generateTimestamps creates n sorted timestamps spread across [dateFrom, dateFrom+totalDuration].
func generateTimestamps(dateFrom time.Time, totalDuration time.Duration, n int) []time.Time {
	ts := make([]time.Time, n)
	for i := 0; i < n; i++ {
		offset := time.Duration(rand.Int63n(int64(totalDuration)))
		ts[i] = dateFrom.Add(offset)
	}
	sort.Slice(ts, func(i, j int) bool { return ts[i].Before(ts[j]) })
	return ts
}

// generatePriceDelta returns a percentage change for price.
// ~80% of the time: small change ±3%
// ~15% of the time: medium change ±8%
// ~5% of the time: large change ±15% (sales or price hikes)
func generatePriceDelta() float64 {
	r := rand.Float64()
	switch {
	case r < 0.80:
		return rand.NormFloat64() * 0.03
	case r < 0.95:
		return rand.NormFloat64() * 0.08
	default:
		return rand.NormFloat64() * 0.15
	}
}
