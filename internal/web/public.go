package web

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/beastixq/marketplace/internal/model"
	"github.com/go-chi/chi/v5"
	"github.com/shopspring/decimal"
)

// --- Catalog ---

type catalogData struct {
	Products         []model.Product
	Categories       []model.Category
	FilterName       string
	MinPrice         string
	MaxPrice         string
	SelectedCategory string
	SortingOrder     string
	Page             int
	TotalPages       int
	User             *userInfo
}

func (cd catalogData) PaginationURL(page int) string {
	v := url.Values{}
	if cd.FilterName != "" {
		v.Set("filter_name", cd.FilterName)
	}
	if cd.MinPrice != "" {
		v.Set("min_price", cd.MinPrice)
	}
	if cd.MaxPrice != "" {
		v.Set("max_price", cd.MaxPrice)
	}
	if cd.SelectedCategory != "" {
		v.Set("category", cd.SelectedCategory)
	}
	if cd.SortingOrder != "" {
		v.Set("sorting_order", cd.SortingOrder)
	}
	v.Set("page", strconv.Itoa(page))
	return "/?" + v.Encode()
}

func (cd catalogData) PageRange() []int {
	pages := make([]int, 0, cd.TotalPages)
	for i := 1; i <= cd.TotalPages; i++ {
		pages = append(pages, i)
	}
	return pages
}

func (wh *WebHandler) Catalog(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	page, _ := strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = 1
	}
	const perPage = 12

	opts := model.CatalogOptions{
		Pagination: &model.PaginationOpts{Page: page, Limit: perPage},
	}

	filterName := q.Get("filter_name")
	if filterName != "" {
		opts.FilterName = &filterName
	}

	minPriceStr := q.Get("min_price")
	if minPriceStr != "" {
		if mp, err := decimal.NewFromString(minPriceStr); err == nil {
			opts.MinPrice = &mp
		}
	}

	maxPriceStr := q.Get("max_price")
	if maxPriceStr != "" {
		if mp, err := decimal.NewFromString(maxPriceStr); err == nil {
			opts.MaxPrice = &mp
		}
	}

	category := q.Get("category")
	if category != "" {
		opts.Categories = []string{category}
	}

	sortingOrder := q.Get("sorting_order")
	if sortingOrder == "asc" || sortingOrder == "desc" {
		so := model.SortingOrderType(sortingOrder)
		opts.SortingOrder = &so
	}

	products, _ := wh.productService.GetProducts(r.Context(), opts)

	categories, _ := wh.categoryService.GetCategories(r.Context(), model.PaginationOpts{Page: 1, Limit: 100})

	// TODO(human): implement total count in service/repo for proper pagination
	totalPages := 1
	if len(products) == perPage {
		totalPages = page + 1
	}

	data := catalogData{
		Products:         products,
		Categories:       categories,
		FilterName:       filterName,
		MinPrice:         minPriceStr,
		MaxPrice:         maxPriceStr,
		SelectedCategory: category,
		SortingOrder:     sortingOrder,
		Page:             page,
		TotalPages:       totalPages,
		User:             wh.userFromCookie(r),
	}

	wh.render(w, "catalog", data)
}

// --- Product Detail ---

func (wh *WebHandler) ProductDetail(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid product ID", http.StatusBadRequest)
		return
	}

	product, err := wh.productService.GetProductByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Product not found", http.StatusNotFound)
		return
	}

	// Price history with configurable date range
	dateTo := time.Now()
	rangeParam := r.URL.Query().Get("range")
	if rangeParam == "" {
		rangeParam = "3m"
	}
	var dateFrom time.Time
	switch rangeParam {
	case "1w":
		dateFrom = dateTo.AddDate(0, 0, -7)
	case "1m":
		dateFrom = dateTo.AddDate(0, -1, 0)
	case "3m":
		dateFrom = dateTo.AddDate(0, -3, 0)
	case "6m":
		dateFrom = dateTo.AddDate(0, -6, 0)
	case "1y":
		dateFrom = dateTo.AddDate(-1, 0, 0)
	case "all":
		dateFrom = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	default:
		dateFrom = dateTo.AddDate(0, -3, 0)
	}
	priceHistory, _ := wh.productService.GetProductPriceHistory(r.Context(), id, dateFrom, dateTo)

	reviews, _ := wh.productService.GetReviewsByProductID(r.Context(), id, model.PaginationOpts{Page: 1, Limit: 50})
	user := wh.userFromCookie(r)

	// Build user names map for reviews
	reviewUserNames := make(map[int64]string, len(reviews))
	var currentUserReview *model.Review
	for _, rv := range reviews {
		if user != nil && user.Role == "buyer" && rv.UserID == user.UserID {
			review := rv
			currentUserReview = &review
		}
		if _, ok := reviewUserNames[rv.UserID]; !ok {
			u, err := wh.userService.GetAuthUserByID(r.Context(), rv.UserID)
			if err == nil {
				reviewUserNames[rv.UserID] = u.FullName
			} else {
				reviewUserNames[rv.UserID] = fmt.Sprintf("User #%d", rv.UserID)
			}
		}
	}

	seller, _ := wh.sellerService.GetSellerByID(r.Context(), product.SellerID)

	wh.render(w, "product", map[string]any{
		"Product":           product,
		"Seller":            seller,
		"PriceHistory":      priceHistory,
		"PriceRange":        rangeParam,
		"Reviews":           reviews,
		"ReviewUserNames":   reviewUserNames,
		"CurrentUserReview": currentUserReview,
		"User":              user,
		"Notice":            r.URL.Query().Get("notice"),
		"ReviewError":       r.URL.Query().Get("review_error"),
	})
}

// --- Categories ---

func (wh *WebHandler) Categories(w http.ResponseWriter, r *http.Request) {
	categories, _ := wh.categoryService.GetCategories(r.Context(), model.PaginationOpts{Page: 1, Limit: 100})

	wh.render(w, "categories", map[string]any{
		"Categories": categories,
		"User":       wh.userFromCookie(r),
	})
}

// --- Public Seller Profile ---

func (wh *WebHandler) SellerProfile(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid seller ID", http.StatusBadRequest)
		return
	}

	seller, err := wh.sellerService.GetSellerByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Seller not found", http.StatusNotFound)
		return
	}

	sellerID := &seller.ID
	allProducts, _ := wh.productService.GetProducts(r.Context(), model.CatalogOptions{SellerID: sellerID})

	user := wh.userFromCookie(r)
	var stats *model.SellerStats
	if s, err := wh.sellerService.GetSellerStats(r.Context(), seller.ID, time.Now().AddDate(-1, 0, 0), time.Now()); err == nil {
		stats = &s
	}

	wh.render(w, "seller-profile", map[string]any{
		"Seller":   seller,
		"Products": allProducts,
		"Stats":    stats,
		"User":     user,
	})
}
