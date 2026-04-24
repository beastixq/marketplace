package model

import (
	"time"

	"github.com/shopspring/decimal"
)

type OrderStatus string

const (
	StatusDraft     OrderStatus = "draft"
	StatusPending   OrderStatus = "pending"
	StatusPaid      OrderStatus = "paid"
	StatusShipped   OrderStatus = "shipped"
	StatusDelivered OrderStatus = "delivered"
	StatusCancelled OrderStatus = "cancelled"
)

type OrderItem struct {
	ID              int64
	OrderID         int64
	ProductID       int64
	Quantity        int
	PriceAtPurchase decimal.Decimal
}

type OrderItemCreate struct {
	OrderID         int64
	ProductID       int64
	Quantity        int
	PriceAtPurchase decimal.Decimal
}

type OrderItemUpdate struct {
	OrderID         *int64
	ProductID       *int64
	Quantity        *int
	PriceAtPurchase *decimal.Decimal
}

type Order struct {
	ID        int64
	UserID    int64
	AddressID *int64
	SellerID  *int64

	Status      OrderStatus
	TotalAmount decimal.Decimal
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
type OrderCreate struct {
	UserID      int64
	AddressID   *int64
	SellerID    *int64
	Status      OrderStatus
	TotalAmount decimal.Decimal
}
type OrderUpdate struct {
	UserID      *int64
	SellerID    *int64
	AddressID   *int64
	Status      *OrderStatus
	TotalAmount *decimal.Decimal
}
