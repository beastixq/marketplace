package generators

import (
	"context"
	"fmt"
	"math"
	"math/rand"

	"github.com/Masterminds/squirrel"
	"github.com/go-faker/faker/v4"
	"github.com/jackc/pgx/v5"
)

// sellersIDs, err := CreateSellers(tx, ctx, usersIDs[3], sellersCount)

func CreateSellers(tx pgx.Tx, ctx context.Context, usersSellersIDs []int64) (sellersIDs []int64, err error) {
	createdInLoop := 0
	psql := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)
	insertBuilder := psql.Insert("sellers").Columns("user_id", "company_name", "description", "rating")
	count := len(usersSellersIDs)
	sellersIDs = make([]int64, count)

	for i := range count {
		comName := faker.Word() + " " + faker.Word() + " LLC"
		desc := faker.Sentence()
		rating := math.Round((rand.Float64()*5)*100) / 100 // 0.00 — 5.00

		insertBuilder = insertBuilder.Values(usersSellersIDs[i], comName, desc, rating)
		createdInLoop++

		if i%10 == 9 || i == count-1 {
			sql, args, err := insertBuilder.Suffix("RETURNING id").ToSql()
			if err != nil {
				return nil, fmt.Errorf("Sellers %s: %v\n", ErrToSql, err)
			}
			rows, err := tx.Query(ctx, sql, args...)
			if err != nil {
				return nil, fmt.Errorf("Sellers %s: %v\n", ErrQuery, err)
			}
			curInd := i - createdInLoop + 1
			for rows.Next() {
				err = rows.Scan(&sellersIDs[curInd])
				if err != nil {
					return nil, fmt.Errorf("Sellers %s: %v\n", ErrScan, err)
				}
				curInd++
			}
			rows.Close()
			if err = rows.Err(); err != nil {
				return nil, fmt.Errorf("Sellers %s: %v\n", ErrCloseRows, err)
			}
			insertBuilder = psql.Insert("sellers").Columns("user_id", "company_name", "description", "rating")
			createdInLoop = 0
		}
	}
	return sellersIDs, nil
}
