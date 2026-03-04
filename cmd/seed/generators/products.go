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

type realProduct struct {
	name  string
	desc  string
	price float64
	stock int
}

var realProducts = []realProduct{
	{"iPhone 15 Pro", "Apple smartphone with A17 Pro chip, titanium design", 89990.00, 150},
	{"Samsung Galaxy S24 Ultra", "Samsung flagship with AI features and S Pen", 79990.00, 200},
	{"MacBook Air M3", "Lightweight laptop with Apple M3 chip, 15-inch display", 129990.00, 75},
	{"Sony WH-1000XM5", "Premium wireless noise-cancelling headphones", 29990.00, 300},
	{"PlayStation 5 Slim", "Next-gen gaming console, 1TB SSD", 49990.00, 100},
	{"Kindle Paperwhite 2024", "E-reader with 6.8 inch display, adjustable warm light", 12990.00, 500},
	{"Nike Air Max 90", "Classic running shoes, mesh and leather upper", 11990.00, 400},
	{"Dyson V15 Detect", "Cordless vacuum cleaner with laser dust detection", 54990.00, 60},
	{"LEGO Technic Porsche 911", "Building set, 1580 pieces, detailed replica", 14990.00, 250},
	{"JBL Charge 5", "Portable Bluetooth speaker, IP67 waterproof", 9990.00, 350},
}

func CreateProducts(tx pgx.Tx, ctx context.Context, sellersIDs []int64, count int) (productsIDs []int64, err error) {
	createdInLoop := 0
	psql := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)
	insertBuilder := psql.Insert("products").Columns("seller_id", "name", "description", "price", "stock_quantity")
	productsIDs = make([]int64, count)

	for i := range count {
		var name, desc string
		var price float64
		var stock int

		if i < len(realProducts) {
			rp := realProducts[i]
			name = rp.name
			desc = rp.desc
			price = rp.price
			stock = rp.stock
		} else {
			name = faker.Word() + " " + faker.Word()
			desc = faker.Sentence()
			price = math.Round((rand.Float64()*999+1)*100) / 100
			stock = rand.Intn(1000)
		}

		sellerID := sellersIDs[rand.Intn(len(sellersIDs))]
		insertBuilder = insertBuilder.Values(sellerID, name, desc, price, stock)
		createdInLoop++

		if i%10 == 9 || i == count-1 {
			sql, args, err := insertBuilder.Suffix("RETURNING id").ToSql()
			if err != nil {
				return nil, fmt.Errorf("Products %s: %v", ErrToSql, err)
			}
			rows, err := tx.Query(ctx, sql, args...)
			if err != nil {
				return nil, fmt.Errorf("Products %s: %v", ErrQuery, err)
			}
			curInd := i - createdInLoop + 1
			for rows.Next() {
				err = rows.Scan(&productsIDs[curInd])
				if err != nil {
					return nil, fmt.Errorf("Products %s: %v", ErrScan, err)
				}
				curInd++
			}
			rows.Close()
			if err = rows.Err(); err != nil {
				return nil, fmt.Errorf("Products %s: %v", ErrCloseRows, err)
			}
			insertBuilder = psql.Insert("products").Columns("seller_id", "name", "description", "price", "stock_quantity")
			createdInLoop = 0
		}
	}
	return productsIDs, nil
}
