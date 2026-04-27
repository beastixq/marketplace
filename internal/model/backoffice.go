package model

import "github.com/shopspring/decimal"

type AdminOrderListOptions struct {
	Status     *OrderStatus
	Pagination PaginationOpts
}

type PlatformStats struct {
	TotalUsers     int64
	TotalSellers   int64
	TotalProducts  int64
	TotalOrders    int64
	TotalRevenue   decimal.Decimal
	TotalReviews   int64
	UsersByRole    []RoleCount
	OrdersByStatus []StatusCount
	TopProducts    []TopProductStats
}

type RoleCount struct {
	Role  UserRole
	Count int64
}

type StatusCount struct {
	Status OrderStatus
	Count  int64
}

type TopProductStats struct {
	ID        int64
	Name      string
	Revenue   decimal.Decimal
	UnitsSold int64
}
