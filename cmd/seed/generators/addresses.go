package generators

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/Masterminds/squirrel"
	"github.com/go-faker/faker/v4"
	"github.com/jackc/pgx/v5"
)

type realAddress struct {
	city    string
	street  string
	zipCode string
}

var realAddresses = []realAddress{
	{"Moscow", "Tverskaya st., 15", "125009"},
	{"Moscow", "Arbat st., 24", "119002"},
	{"Moscow", "Leninsky prospect, 42", "119991"},
	{"Moscow", "Kutuzovsky prospect, 30", "121165"},
	{"Moscow", "Baumanskaya st., 50", "105005"},
	{"Moscow", "Bolshaya Sadovaya st., 10", "123001"},
	{"Moscow", "Noviy Arbat st., 21", "119019"},
	{"Moscow", "Pokrovka st., 17", "101000"},
	{"Moscow", "Myasnitskaya st., 40", "101990"},
	{"Moscow", "Maroseyka st., 12", "101000"},
}

var russianCities = []string{
	"Moscow", "Saint Petersburg", "Novosibirsk", "Yekaterinburg", "Kazan",
	"Nizhny Novgorod", "Chelyabinsk", "Samara", "Omsk", "Rostov-on-Don",
	"Ufa", "Krasnoyarsk", "Voronezh", "Perm", "Volgograd",
}

func CreateAddresses(tx pgx.Tx, ctx context.Context, userIDs []int64, count int) (addressesIDs []int64, err error) {
	createdInLoop := 0
	psql := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)
	insertBuilder := psql.Insert("addresses").Columns("user_id", "city", "street", "zip_code", "is_default")
	addressesIDs = make([]int64, count)

	hasDefault := make(map[int64]bool)

	for i := range count {
		userID := userIDs[rand.Intn(len(userIDs))]

		var city, street, zipCode string
		if i < len(realAddresses) {
			ra := realAddresses[i]
			city = ra.city
			street = ra.street
			zipCode = ra.zipCode
		} else {
			city = russianCities[rand.Intn(len(russianCities))]
			street = faker.Word() + " st., " + fmt.Sprintf("%d", rand.Intn(200)+1)
			zipCode = fmt.Sprintf("%06d", rand.Intn(999999))
		}

		isDefault := !hasDefault[userID]
		if isDefault {
			hasDefault[userID] = true
		}

		insertBuilder = insertBuilder.Values(userID, city, street, zipCode, isDefault)
		createdInLoop++

		if i%10 == 9 || i == count-1 {
			sql, args, err := insertBuilder.Suffix("RETURNING id").ToSql()
			if err != nil {
				return nil, fmt.Errorf("Addresses %s: %v", ErrToSql, err)
			}
			rows, err := tx.Query(ctx, sql, args...)
			if err != nil {
				return nil, fmt.Errorf("Addresses %s: %v", ErrQuery, err)
			}
			curInd := i - createdInLoop + 1
			for rows.Next() {
				err = rows.Scan(&addressesIDs[curInd])
				if err != nil {
					return nil, fmt.Errorf("Addresses %s: %v", ErrScan, err)
				}
				curInd++
			}
			rows.Close()
			if err = rows.Err(); err != nil {
				return nil, fmt.Errorf("Addresses %s: %v", ErrCloseRows, err)
			}
			insertBuilder = psql.Insert("addresses").Columns("user_id", "city", "street", "zip_code", "is_default")
			createdInLoop = 0
		}
	}
	return addressesIDs, nil
}
