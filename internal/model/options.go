package model

import "github.com/shopspring/decimal"

type SortingOrderType string

const (
	SortingOrderDesc SortingOrderType = "desc"
	SortingOrderAsc  SortingOrderType = "asc"
)

type PaginationOpts struct {
	Page  int
	Limit int
}

type UserListOptions struct {
	Search     *string
	Role       *string
	Pagination PaginationOpts
}

type CatalogOptions struct {
	Categories   []string
	MinPrice     *decimal.Decimal
	MaxPrice     *decimal.Decimal
	FilterName   *string
	SellerID     *int64
	Pagination   *PaginationOpts
	SortingOrder *SortingOrderType
}
