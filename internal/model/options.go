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

type CatalogOptions struct {
	Categories   []string
	MinPrice     *decimal.Decimal
	MaxPrice     *decimal.Decimal
	FilterName   *string
	Pagination   *PaginationOpts
	SortingOrder *SortingOrderType
}
