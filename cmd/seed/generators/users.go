package generators

import (
	"context"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/go-faker/faker/v4"
	"github.com/go-faker/faker/v4/pkg/options"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

const hashCost = 4

func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), hashCost)
	return string(bytes), err
}

func CreateUsers(tx pgx.Tx, ctx context.Context, adminsCount, analystsCount, buyersCount, sellersCount int) (users [4][]int64, err error) {
	psql := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)
	insertBuilder := psql.Insert("users").Columns("email", "password_hash", "full_name", "phone", "role")
	roles := [4]string{"admin", "analyst", "buyer", "seller"}
	rolesCounts := [4]int{adminsCount, analystsCount, buyersCount, sellersCount}
	for ri := range len(roles) {
		users[ri] = make([]int64, rolesCounts[ri])
	}

	for ri, role := range roles {
		createdInLoop := 0
		count := rolesCounts[ri]
		for i := range count {
			email := faker.Email(options.WithGenerateUniqueValues(true))
			password := faker.Password()
			hashPass, err := hashPassword(password)
			if err != nil {
				return [4][]int64{}, fmt.Errorf("%s: %v\n", ErrHashPassword.Error(), err)
			}
			fullName := faker.FirstName() + " " + faker.LastName()
			phone := faker.Phonenumber(options.WithGenerateUniqueValues(true))
			insertBuilder = insertBuilder.Values(email, hashPass, fullName, phone, role)
			createdInLoop++

			if i%10 == 9 || i == count-1 {
				sql, args, err := insertBuilder.Suffix("RETURNING id").ToSql()
				if err != nil {
					return [4][]int64{}, fmt.Errorf("Users %s: %v\n", ErrToSql, err)
				}
				rows, err := tx.Query(ctx, sql, args...)
				if err != nil {
					return [4][]int64{}, fmt.Errorf("Users %s: %v\n", ErrQuery, err)
				}
				curInd := i - createdInLoop + 1
				for rows.Next() {
					err = rows.Scan(&users[ri][curInd])
					if err != nil {
						return [4][]int64{}, fmt.Errorf("Users %s: %v\n", ErrScan, err)
					}
					curInd++
				}
				rows.Close()
				if err = rows.Err(); err != nil {
					return users, fmt.Errorf("Users %s: %v\n", ErrCloseRows, err)
				}
				insertBuilder = psql.Insert("users").Columns("email", "password_hash", "full_name", "phone", "role")
				createdInLoop = 0
			}
		}
	}
	return users, nil
}
