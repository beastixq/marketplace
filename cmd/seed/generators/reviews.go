package generators

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
)

var realReviewComments = []string{
	"Great product, exactly as described. Fast delivery!",
	"Good quality for the price. Would recommend.",
	"Decent product but packaging could be better.",
	"Excellent! Exceeded my expectations.",
	"Average quality, nothing special.",
	"Very satisfied with this purchase. Will buy again.",
	"Product arrived damaged, but seller resolved it quickly.",
	"Perfect gift idea. My friend loved it.",
	"Not worth the money honestly. Expected more.",
	"Solid build quality. Using it daily for a month now.",
}

var fakerReviewTemplates = []string{
	"Good product. ",
	"Nice quality. ",
	"Fast shipping. ",
	"As expected. ",
	"Satisfied. ",
	"Could be better. ",
	"Great value. ",
	"Recommended. ",
	"Works well. ",
	"Happy with purchase. ",
}

func CreateReviews(tx pgx.Tx, ctx context.Context, buyerIDs, productsIDs []int64, count int) error {
	psql := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)
	insertBuilder := psql.Insert("reviews").Columns("user_id", "product_id", "rating", "comment")
	createdInLoop := 0
	created := 0

	seen := make(map[[2]int64]bool)

	for created < count {
		userID := buyerIDs[rand.Intn(len(buyerIDs))]
		productID := productsIDs[rand.Intn(len(productsIDs))]
		key := [2]int64{userID, productID}
		if seen[key] {
			continue
		}
		seen[key] = true

		rating := rand.Intn(5) + 1

		var comment string
		if created < len(realReviewComments) {
			comment = realReviewComments[created]
		} else {
			comment = fakerReviewTemplates[rand.Intn(len(fakerReviewTemplates))]
		}

		insertBuilder = insertBuilder.Values(userID, productID, rating, comment)
		createdInLoop++
		created++

		if createdInLoop%10 == 0 || created == count {
			sql, args, err := insertBuilder.ToSql()
			if err != nil {
				return fmt.Errorf("Reviews %s: %v", ErrToSql, err)
			}
			_, err = tx.Exec(ctx, sql, args...)
			if err != nil {
				return fmt.Errorf("Reviews %s: %v", ErrQuery, err)
			}
			insertBuilder = psql.Insert("reviews").Columns("user_id", "product_id", "rating", "comment")
			createdInLoop = 0
		}
	}
	return nil
}
