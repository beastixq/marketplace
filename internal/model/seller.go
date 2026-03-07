package model

import (
	"time"

	"github.com/shopspring/decimal"
)

type Seller struct {
	ID          int64
	UserID      int64
	CompanyName string
	Description *string
	Rating      *float32
	CreatedAt   time.Time
}

type SellerCreate struct {
	UserID      int64
	CompanyName string
	Description *string
	Rating      *float32
}

type SellerUpdate struct {
	UserID      *int64
	CompanyName *string
	Description *string
	Rating      *float32
}

type SellerStats struct {
	TotalOrders    int64
	TotalRevenue   decimal.Decimal
	AvgOrderValue  decimal.Decimal
	TopProductName string
}
