package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strconv"
	"time"

	"github.com/beastixq/marketplace/internal/middleware"
	"github.com/beastixq/marketplace/internal/model"
	"github.com/beastixq/marketplace/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/shopspring/decimal"
)

type ProductHandler struct {
	productService service.ProductService
}

func NewProductHandler(productSvc service.ProductService) ProductHandler {
	return ProductHandler{productService: productSvc}
}

// GET /api/v1/products - get catalog
func (ph ProductHandler) GetCatalog(w http.ResponseWriter, r *http.Request) {

	var minPrice, maxPrice *decimal.Decimal
	var paginationOpts *model.PaginationOpts
	var filterName *string
	var sortingOrder *model.SortingOrderType
	if r.URL.Query().Has("min_price") {
		mp, err := decimal.NewFromString(r.URL.Query().Get("min_price"))
		if err != nil {
			writeError(w, http.StatusBadRequest, ErrInvalidMinPriceOption.Error())
			return
		}
		minPrice = &mp
	}
	if r.URL.Query().Has("max_price") {
		mp, err := decimal.NewFromString(r.URL.Query().Get("max_price"))
		if err != nil {
			writeError(w, http.StatusBadRequest, ErrInvalidMaxPriceOption.Error())
			return
		}
		maxPrice = &mp
	}
	if minPrice != nil && maxPrice != nil && maxPrice.LessThan(*minPrice) {
		writeError(w, http.StatusBadRequest, ErrMaxIsLessThanMin.Error())
		return
	}
	fn := r.URL.Query().Get("filter_name")
	if fn != "" {
		filterName = &fn
	}
	if r.URL.Query().Has("limit") && !r.URL.Query().Has("page") {
		writeError(w, http.StatusBadRequest, ErrNoPageInPaginationOptions.Error())
		return
	}
	if r.URL.Query().Has("page") {
		if !r.URL.Query().Has("limit") {
			writeError(w, http.StatusBadRequest, ErrNoLimitInPaginationOptions.Error())
			return
		}
		page, err := strconv.Atoi(r.URL.Query().Get("page"))
		if err != nil || page <= 0 {
			writeError(w, http.StatusBadRequest, ErrInvalidPagePaginationOption.Error())
			return
		}
		limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
		if err != nil || limit <= 0 {
			writeError(w, http.StatusBadRequest, ErrInvalidLimitPaginationOption.Error())
			return
		}
		p := model.PaginationOpts{Page: page, Limit: limit}
		paginationOpts = &p
	}
	so := model.SortingOrderType(r.URL.Query().Get("sorting_order"))
	if so != "" && !slices.Contains([]model.SortingOrderType{model.SortingOrderAsc, model.SortingOrderDesc}, so) {
		writeError(w, http.StatusBadRequest, ErrInvalidSortingOrder.Error())
		return
	}
	if so != "" {
		sortingOrder = &so
	}
	categories := r.URL.Query()["categories"]

	options := model.CatalogOptions{
		MinPrice:     minPrice,
		MaxPrice:     maxPrice,
		FilterName:   filterName,
		Pagination:   paginationOpts,
		SortingOrder: sortingOrder,
		Categories:   categories,
	}
	productsService, err := ph.productService.GetProducts(r.Context(), options)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			writeError(w, http.StatusNotFound, service.ErrNotFound.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, ErrInternalServer.Error())
		return
	}
	products := make([]Product, len(productsService))
	for i, productService := range productsService {
		products[i] = productFromService(productService)
	}
	writeJSON(w, http.StatusOK, products)
}

// GET /api/v1/products/:id
func (ph ProductHandler) GetProduct(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrInvalidIDParam.Error())
		return
	}
	product, err := ph.productService.GetProductByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			writeError(w, http.StatusNotFound, service.ErrNotFound.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, ErrInternalServer.Error())
		return
	}
	writeJSON(w, http.StatusOK, productFromService(product))
}

// GET /api/v1/products/:id/price-history
func (ph ProductHandler) GetProductPriceHistory(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrInvalidIDParam.Error())
		return
	}
	dateFromStr := r.URL.Query().Get("date_from")
	if dateFromStr == "" {
		writeError(w, http.StatusBadRequest, ErrDateFromRequired.Error())
		return
	}
	dateToStr := r.URL.Query().Get("date_to")
	if dateToStr == "" {
		writeError(w, http.StatusBadRequest, ErrDateToRequired.Error())
		return
	}
	dateFrom, err := time.Parse(time.DateOnly, dateFromStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrInvalidDateFormat.Error())
		return
	}
	dateTo, err := time.Parse(time.DateOnly, dateToStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrInvalidDateFormat.Error())
		return
	}
	if dateFrom.After(dateTo) {
		writeError(w, http.StatusBadRequest, ErrDateFromMustBeBeforeDateTo.Error())
		return
	}

	priceHistoryService, err := ph.productService.GetProductPriceHistory(r.Context(), id, dateFrom, dateTo)
	if err != nil {
		if errors.Is(err, service.ErrProductNotFound) {
			writeError(w, http.StatusNotFound, service.ErrProductNotFound.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, ErrInternalServer.Error())
		return
	}
	priceHistory := make([]ProductPriceHistory, len(priceHistoryService))
	for i := range priceHistory {
		priceHistory[i] = productPriceHistoryFromService(priceHistoryService[i])
	}
	writeJSON(w, http.StatusOK, priceHistory)
}

// GET /api/v1/products/:id/reviews
func (ph ProductHandler) GetProductReviews(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrInvalidIDParam.Error())
		return
	}

	if r.URL.Query().Has("limit") && !r.URL.Query().Has("page") {
		writeError(w, http.StatusBadRequest, ErrNoPageInPaginationOptions.Error())
		return
	}
	if r.URL.Query().Has("page") && !r.URL.Query().Has("limit") {
		writeError(w, http.StatusBadRequest, ErrNoLimitInPaginationOptions.Error())
		return
	}

	var opts model.PaginationOpts
	if r.URL.Query().Has("page") {
		page, err := strconv.Atoi(r.URL.Query().Get("page"))
		if err != nil || page <= 0 {
			writeError(w, http.StatusBadRequest, ErrInvalidPagePaginationOption.Error())
			return
		}
		limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
		if err != nil || limit <= 0 {
			writeError(w, http.StatusBadRequest, ErrInvalidLimitPaginationOption.Error())
			return
		}
		opts = model.PaginationOpts{Page: page, Limit: limit}
	}

	reviewsService, err := ph.productService.GetReviewsByProductID(r.Context(), id, opts)
	if err != nil {
		if errors.Is(err, service.ErrProductNotFound) {
			writeError(w, http.StatusNotFound, service.ErrProductNotFound.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, ErrInternalServer.Error())
		return
	}
	reviews := make([]Review, len(reviewsService))
	for i := range reviews {
		reviews[i] = reviewFromService(reviewsService[i])
	}
	writeJSON(w, http.StatusOK, reviews)
}

type CreateProductRequest struct {
	SellerID      int64           `json:"seller_id"`
	Name          string          `json:"name"`
	Description   *string         `json:"description"`
	Price         decimal.Decimal `json:"price"`
	StockQuantity int             `json:"stock_quantity"`
}

func (cr CreateProductRequest) Validate() error {
	if cr.Name == "" {
		return ErrProductNameRequired
	}
	if cr.Price.IsNegative() || cr.Price.IsZero() {
		return ErrProductPriceInvalid
	}
	if cr.StockQuantity < 0 {
		return ErrProductStockInvalid
	}
	return nil
}

// POST /api/v1/products
func (ph ProductHandler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromCtx(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, ErrTokenClaimsGetFailed.Error())
		return
	}

	var req CreateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, ErrDecodeFailed.Error())
		return
	}
	if err := req.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	id, err := ph.productService.CreateProduct(r.Context(), claims.UserID, model.ProductCreate{
		SellerID:      req.SellerID,
		Name:          req.Name,
		Description:   req.Description,
		Price:         req.Price,
		StockQuantity: req.StockQuantity,
	})
	if err != nil {
		if errors.Is(err, service.ErrSellerNotFound) {
			writeError(w, http.StatusNotFound, service.ErrSellerNotFound.Error())
			return
		}
		if errors.Is(err, service.ErrNotYourSeller) {
			writeError(w, http.StatusForbidden, service.ErrNotYourSeller.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, ErrInternalServer.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]int64{"id": id})
}

type UpdateProductRequest struct {
	Name          *string          `json:"name"`
	Description   *string          `json:"description"`
	Price         *decimal.Decimal `json:"price"`
	StockQuantity *int             `json:"stock_quantity"`
}

func (ur UpdateProductRequest) Validate() error {
	if ur.Name == nil && ur.Description == nil && ur.Price == nil && ur.StockQuantity == nil {
		return ErrUpdateProductAllNil
	}
	if ur.Name != nil && *ur.Name == "" {
		return ErrProductNameRequired
	}
	if ur.Price != nil && (ur.Price.IsNegative() || ur.Price.IsZero()) {
		return ErrProductPriceInvalid
	}
	if ur.StockQuantity != nil && *ur.StockQuantity < 0 {
		return ErrProductStockInvalid
	}
	return nil
}

// PATCH /api/v1/products/:id
func (ph ProductHandler) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromCtx(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, ErrTokenClaimsGetFailed.Error())
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrInvalidIDParam.Error())
		return
	}

	var req UpdateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, ErrDecodeFailed.Error())
		return
	}
	if err := req.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	product, err := ph.productService.UpdateProduct(r.Context(), claims.UserID, id, model.ProductUpdate{
		Name:          req.Name,
		Description:   req.Description,
		Price:         req.Price,
		StockQuantity: req.StockQuantity,
	})
	if err != nil {
		if errors.Is(err, service.ErrProductNotFound) {
			writeError(w, http.StatusNotFound, service.ErrProductNotFound.Error())
			return
		}
		if errors.Is(err, service.ErrNotYourSeller) {
			writeError(w, http.StatusForbidden, service.ErrNotYourSeller.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, ErrInternalServer.Error())
		return
	}
	writeJSON(w, http.StatusOK, productFromService(product))
}

// DELETE /api/v1/products/:id
func (ph ProductHandler) DeleteProduct(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromCtx(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, ErrTokenClaimsGetFailed.Error())
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrInvalidIDParam.Error())
		return
	}

	if err := ph.productService.DeleteProductByID(r.Context(), claims.UserID, id); err != nil {
		if errors.Is(err, service.ErrProductNotFound) {
			writeError(w, http.StatusNotFound, service.ErrProductNotFound.Error())
			return
		}
		if errors.Is(err, service.ErrNotYourSeller) {
			writeError(w, http.StatusForbidden, service.ErrNotYourSeller.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, ErrInternalServer.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
