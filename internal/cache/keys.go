package cache

import (
	"net/url"
	"sort"
	"strconv"

	m "github.com/beastixq/marketplace/internal/model"
)

const (
	productCatalogPrefix = "products:catalog:"
	categoryListPrefix   = "categories:list:"
)

// ProductByIDKey is the Redis key for a single product entity.
// Shared so every decorator that invalidates products uses the same string.
func ProductByIDKey(id int64) string {
	return "products:" + strconv.FormatInt(id, 10)
}

func ProductCatalogPrefix() string {
	return productCatalogPrefix
}

func ProductCatalogKey(opts m.CatalogOptions) string {
	values := url.Values{}
	addPagination(values, opts.Pagination)

	if len(opts.Categories) > 0 {
		categories := append([]string(nil), opts.Categories...)
		sort.Strings(categories)
		for _, category := range categories {
			values.Add("category", category)
		}
	}
	if opts.FilterName != nil && *opts.FilterName != "" {
		values.Set("filter_name", *opts.FilterName)
	}
	if opts.MaxPrice != nil {
		values.Set("max_price", opts.MaxPrice.String())
	}
	if opts.MinPrice != nil {
		values.Set("min_price", opts.MinPrice.String())
	}
	if opts.SellerID != nil {
		values.Set("seller_id", strconv.FormatInt(*opts.SellerID, 10))
	}
	if opts.SortingOrder != nil && *opts.SortingOrder != "" {
		values.Set("sort", string(*opts.SortingOrder))
	}

	return productCatalogPrefix + values.Encode()
}

func CategoryListPrefix() string {
	return categoryListPrefix
}

func CategoryListKey(opts m.CategoryListOptions) string {
	values := url.Values{}
	pg := opts.Pagination
	addPagination(values, &pg)
	if opts.OnlyRoot {
		values.Set("root", "true")
	}
	if opts.ParentID != nil {
		values.Set("parent_id", strconv.FormatInt(*opts.ParentID, 10))
	}
	if opts.Search != nil && *opts.Search != "" {
		values.Set("search", *opts.Search)
	}
	return categoryListPrefix + values.Encode()
}

func ProductReviewsPrefix(productID int64) string {
	return ProductByIDKey(productID) + ":reviews:"
}

func ProductReviewsKey(productID int64, opts m.PaginationOpts) string {
	values := url.Values{}
	values.Set("limit", strconv.Itoa(opts.Limit))
	values.Set("page", strconv.Itoa(opts.Page))
	return ProductReviewsPrefix(productID) + values.Encode()
}

func SessionKey(jti string) string {
	return "sessions:" + jti
}

func addPagination(values url.Values, pg *m.PaginationOpts) {
	if pg == nil {
		values.Set("limit", "all")
		values.Set("page", "all")
		return
	}
	values.Set("limit", strconv.Itoa(pg.Limit))
	values.Set("page", strconv.Itoa(pg.Page))
}
