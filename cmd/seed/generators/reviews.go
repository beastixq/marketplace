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
	// Pre-generate unique (buyer, product) pairs by assigning
	// each buyer a shuffled slice of product indices.
	// This guarantees uniqueness without retries.
	maxPossible := len(buyerIDs) * len(productsIDs)
	if count > maxPossible {
		count = maxPossible
	}

	// Build all possible pairs by giving each buyer random products
	type pair struct {
		userID    int64
		productID int64
	}
	pairs := make([]pair, 0, count)

	// How many reviews per buyer (spread evenly, remainder distributed)
	perBuyer := count / len(buyerIDs)
	remainder := count % len(buyerIDs)

	for i, buyerID := range buyerIDs {
		n := perBuyer
		if i < remainder {
			n++
		}
		if n == 0 {
			continue
		}
		// Shuffle product indices and take first n
		perm := rand.Perm(len(productsIDs))
		for j := 0; j < n; j++ {
			pairs = append(pairs, pair{buyerID, productsIDs[perm[j]]})
		}
	}

	// Shuffle pairs so inserts aren't grouped by buyer
	rand.Shuffle(len(pairs), func(i, j int) { pairs[i], pairs[j] = pairs[j], pairs[i] })

	const batchSize = 500
	psql := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)
	insertBuilder := psql.Insert("reviews").Columns("user_id", "product_id", "rating", "comment")
	inBatch := 0

	for i, p := range pairs {
		rating := rand.Intn(5) + 1
		var comment string
		if i < len(realReviewComments) {
			comment = realReviewComments[i]
		} else {
			comment = fakerReviewTemplates[rand.Intn(len(fakerReviewTemplates))]
		}

		insertBuilder = insertBuilder.Values(p.userID, p.productID, rating, comment)
		inBatch++

		if inBatch >= batchSize || i == len(pairs)-1 {
			sql, args, err := insertBuilder.ToSql()
			if err != nil {
				return fmt.Errorf("Reviews %s: %v", ErrToSql, err)
			}
			_, err = tx.Exec(ctx, sql, args...)
			if err != nil {
				return fmt.Errorf("Reviews %s: %v", ErrQuery, err)
			}
			insertBuilder = psql.Insert("reviews").Columns("user_id", "product_id", "rating", "comment")
			inBatch = 0
		}
	}
	return nil
}
