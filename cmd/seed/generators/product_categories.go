package generators

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
)

func CreateProductCategories(tx pgx.Tx, ctx context.Context, productsIDs, categoriesIDs []int64) error {
	psql := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)
	insertBuilder := psql.Insert("product_categories").Columns("product_id", "category_id")
	createdInLoop := 0

	seen := make(map[[2]int64]bool)

	for _, productID := range productsIDs {
		// Each product gets 1-3 categories
		numCategories := rand.Intn(3) + 1
		for range numCategories {
			categoryID := categoriesIDs[rand.Intn(len(categoriesIDs))]
			key := [2]int64{productID, categoryID}
			if seen[key] {
				continue
			}
			seen[key] = true

			insertBuilder = insertBuilder.Values(productID, categoryID)
			createdInLoop++

			if createdInLoop%10 == 0 {
				sql, args, err := insertBuilder.ToSql()
				if err != nil {
					return fmt.Errorf("ProductCategories %s: %v", ErrToSql, err)
				}
				_, err = tx.Exec(ctx, sql, args...)
				if err != nil {
					return fmt.Errorf("ProductCategories %s: %v", ErrQuery, err)
				}
				insertBuilder = psql.Insert("product_categories").Columns("product_id", "category_id")
				createdInLoop = 0
			}
		}
	}

	// Flush remaining
	if createdInLoop > 0 {
		sql, args, err := insertBuilder.ToSql()
		if err != nil {
			return fmt.Errorf("ProductCategories %s: %v", ErrToSql, err)
		}
		_, err = tx.Exec(ctx, sql, args...)
		if err != nil {
			return fmt.Errorf("ProductCategories %s: %v", ErrQuery, err)
		}
	}

	return nil
}
