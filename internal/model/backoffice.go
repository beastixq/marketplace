package model

import (
	"time"

	"github.com/shopspring/decimal"
)

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

type ReportPeriod string

const (
	ReportPeriodDay   ReportPeriod = "day"
	ReportPeriodWeek  ReportPeriod = "week"
	ReportPeriodMonth ReportPeriod = "month"
)

type ReportOptions struct {
	DateFrom *time.Time
	DateTo   *time.Time
	Period   ReportPeriod
	Limit    int
}

type OrderDynamicsPoint struct {
	PeriodStart time.Time
	OrdersCount int64
	Revenue     decimal.Decimal
}

type CategorySalesStats struct {
	CategoryID   int64
	CategoryName string
	OrdersCount  int64
	UnitsSold    int64
	Revenue      decimal.Decimal
}
