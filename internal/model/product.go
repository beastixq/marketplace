package model

import (
	"time"

	"github.com/shopspring/decimal"
)

type Product struct {
	ID            int64
	SellerID      int64
	Name          string
	Description   *string
	Price         decimal.Decimal
	StockQuantity int
	CreatedAt     time.Time
	DeletedAt     *time.Time
}

type ProductCreate struct {
	SellerID      int64
	Name          string
	Description   *string
	Price         decimal.Decimal
	StockQuantity int
}

type ProductUpdate struct {
	SellerID      *int64
	Name          *string
	Description   *string
	Price         *decimal.Decimal
	StockQuantity *int
}

type ProductPriceHistory struct {
	ID        int64
	ProductID int64
	OldPrice  decimal.Decimal
	NewPrice  decimal.Decimal
	ChangedAt time.Time
	ChangedBy string
}
