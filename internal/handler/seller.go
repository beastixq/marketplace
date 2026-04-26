package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/beastixq/marketplace/internal/model"
	"github.com/beastixq/marketplace/internal/service"
	"github.com/go-chi/chi/v5"
)

type SellerHandler struct {
	sellerService service.SellerService
	orderService  service.OrderService
}

func NewSellerHandler(sellerSvc service.SellerService, orderSvc service.OrderService) SellerHandler {
	return SellerHandler{
		sellerService: sellerSvc,
		orderService:  orderSvc,
	}
}

// GET /api/v1/sellers/:id
func (sh SellerHandler) GetSellerByID(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrInvalidIDParam.Error())
		return
	}
	seller, err := sh.sellerService.GetSellerByID(r.Context(), id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, sellerDTO(seller))
}

// POST /api/v1/sellers
func (sh SellerHandler) CreateSeller(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFromRequest(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, ErrTokenClaimsGetFailed.Error())
		return
	}

	var req CreateSellerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, ErrDecodeFailed.Error())
		return
	}
	if err := req.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	id, err := sh.sellerService.CreateSeller(r.Context(), actor, model.SellerCreate{
		CompanyName: req.CompanyName,
		Description: req.Description,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]int64{"id": id})
}

// PATCH /api/v1/sellers/:id
func (sh SellerHandler) UpdateSeller(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFromRequest(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, ErrTokenClaimsGetFailed.Error())
		return
	}

	sellerID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrInvalidIDParam.Error())
		return
	}

	var req UpdateSellerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, ErrDecodeFailed.Error())
		return
	}
	if err := req.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	seller, err := sh.sellerService.UpdateSeller(r.Context(), actor, sellerID, model.SellerUpdate{
		CompanyName: req.CompanyName,
		Description: req.Description,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, sellerDTO(seller))
}

// DELETE /api/v1/sellers/:id
func (sh SellerHandler) DeleteSeller(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFromRequest(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, ErrTokenClaimsGetFailed.Error())
		return
	}

	sellerID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrInvalidIDParam.Error())
		return
	}

	if err := sh.sellerService.DeleteSellerByID(r.Context(), actor, sellerID); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GET /api/v1/sellers/:id/stats
func (sh SellerHandler) GetSellerStats(w http.ResponseWriter, r *http.Request) {
	sellerID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrInvalidIDParam.Error())
		return
	}

	dateFromStr := r.URL.Query().Get("date_from")
	if dateFromStr == "" {
		writeError(w, http.StatusBadRequest, ErrDateFromRequired.Error())
		return
	}
	dateFrom, err := time.Parse(time.DateOnly, dateFromStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrInvalidDateFormat.Error())
		return
	}

	dateToStr := r.URL.Query().Get("date_to")
	if dateToStr == "" {
		writeError(w, http.StatusBadRequest, ErrDateToRequired.Error())
		return
	}
	dateTo, err := time.Parse(time.DateOnly, dateToStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrInvalidDateFormat.Error())
		return
	}

	stats, err := sh.sellerService.GetSellerStats(r.Context(), sellerID, dateFrom, dateTo)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, sellerStatsDTO(stats))
}

// GET /api/v1/sellers/me/orders
func (sh SellerHandler) GetSellerOrders(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFromRequest(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, ErrTokenClaimsGetFailed.Error())
		return
	}

	pg, err := parsePagination(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	ordersService, err := sh.orderService.GetSellerOrdersByUserID(r.Context(), actor, pg)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	orders := make([]OrderDTO, len(ordersService))
	for i, orderService := range ordersService {
		orders[i] = orderDTO(orderService)
	}

	writeJSON(w, http.StatusOK, orders)
}

// --- Request DTOs ---

type CreateSellerRequest struct {
	CompanyName string  `json:"company_name"`
	Description *string `json:"description"`
}

func (cr CreateSellerRequest) Validate() error {
	if cr.CompanyName == "" {
		return ErrCompanyNameRequired
	}
	return nil
}

type UpdateSellerRequest struct {
	CompanyName *string `json:"company_name"`
	Description *string `json:"description"`
}

func (ur UpdateSellerRequest) Validate() error {
	if ur.CompanyName == nil && ur.Description == nil {
		return ErrUpdateSellerAllNil
	}
	return nil
}
