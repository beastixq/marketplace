package model

import (
	"time"

	"github.com/shopspring/decimal"
)

type Product struct {
	ID               int64
	SellerID         int64
	Name             string
	Description      *string
	Price            decimal.Decimal
	StockQuantity    int
	ReservedQuantity int
	Rating           *float32
	CreatedAt        time.Time
	DeletedAt        *time.Time
}

func (p Product) AvailableQuantity() int {
	return p.StockQuantity - p.ReservedQuantity
}

type ProductCreate struct {
	SellerID      int64
	Name          string
	Description   *string
	Price         decimal.Decimal
	StockQuantity int
	CategoryIDs   []int64
}

type ProductUpdate struct {
	SellerID      *int64
	Name          *string
	Description   *string
	Price         *decimal.Decimal
	StockQuantity *int
	CategoryIDs   *[]int64
	ChangedBy     *string
}

type ProductPriceHistory struct {
	ID        int64
	ProductID int64
	OldPrice  decimal.Decimal
	NewPrice  decimal.Decimal
	ChangedAt time.Time
	ChangedBy string
}
