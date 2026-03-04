package service

import (
	"context"

	m "github.com/beastixq/marketplace/internal/model"
)

type OrderItemRepo interface {
	GetOrderItemByID(ctx context.Context, id int64) (oi m.OrderItem, err error)
	GetOrderItemsByOrderID(ctx context.Context, orderID int64) (ois []m.OrderItem, err error)
	CreateOrderItem(ctx context.Context, oic m.OrderItemCreate) (id int64, err error)
	UpdateOrderItem(ctx context.Context, id int64, oiu m.OrderItemUpdate) (oi m.OrderItem, err error)
	DeleteOrderItemByID(ctx context.Context, id int64) (err error)
}

type OrderRepo interface {
	GetOrderByID(ctx context.Context, id int64) (o m.Order, err error)
	GetOrdersByUserID(ctx context.Context, userID int64) (orders []m.Order, err error)
	CreateOrder(ctx context.Context, oc m.OrderCreate) (id int64, err error)
	UpdateOrder(ctx context.Context, id int64, ou m.OrderUpdate) (o m.Order, err error)
	DeleteOrderByID(ctx context.Context, id int64) (err error)
}
