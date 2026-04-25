package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/beastixq/marketplace/internal/service"
	"github.com/go-chi/chi/v5"
)

type OrderHandler struct {
	orderService service.OrderService
}

func NewOrderHandler(orderSvc service.OrderService) OrderHandler {
	return OrderHandler{orderService: orderSvc}
}

// GET /api/v1/orders
func (oh OrderHandler) GetOrders(w http.ResponseWriter, r *http.Request) {
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

	orders, err := oh.orderService.GetOrdersByUserID(r.Context(), actor, pg)
	if err != nil {
		writeError(w, http.StatusInternalServerError, ErrInternalServer.Error())
		return
	}
	result := make([]OrderDTO, len(orders))
	for i := range orders {
		result[i] = orderDTO(orders[i])
	}
	writeJSON(w, http.StatusOK, result)
}

// GET /api/v1/orders/:id
func (oh OrderHandler) GetOrder(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFromRequest(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, ErrTokenClaimsGetFailed.Error())
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrInvalidIDParam.Error())
		return
	}

	order, err := oh.orderService.GetOrderByID(r.Context(), actor, id)
	if err != nil {
		if errors.Is(err, service.ErrOrderNotFound) {
			writeError(w, http.StatusNotFound, service.ErrOrderNotFound.Error())
			return
		}
		if errors.Is(err, service.ErrNotYourOrder) {
			writeError(w, http.StatusForbidden, service.ErrNotYourOrder.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, ErrInternalServer.Error())
		return
	}
	writeJSON(w, http.StatusOK, orderDTO(order))
}

// GET /api/v1/orders/:id/items
func (oh OrderHandler) GetOrderItems(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFromRequest(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, ErrTokenClaimsGetFailed.Error())
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrInvalidIDParam.Error())
		return
	}

	items, err := oh.orderService.GetOrderItemsByOrderID(r.Context(), actor, id)
	if err != nil {
		if errors.Is(err, service.ErrOrderNotFound) {
			writeError(w, http.StatusNotFound, service.ErrOrderNotFound.Error())
			return
		}
		if errors.Is(err, service.ErrNotYourOrder) {
			writeError(w, http.StatusForbidden, service.ErrNotYourOrder.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, ErrInternalServer.Error())
		return
	}
	result := make([]OrderItemDTO, len(items))
	for i := range items {
		result[i] = orderItemDTO(items[i])
	}
	writeJSON(w, http.StatusOK, result)
}

// GET /api/v1/cart
func (oh OrderHandler) GetCart(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFromRequest(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, ErrTokenClaimsGetFailed.Error())
		return
	}

	cart, err := oh.orderService.GetCart(r.Context(), actor)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			writeError(w, http.StatusNotFound, "Cart not found")
			return
		}
		writeError(w, http.StatusInternalServerError, ErrInternalServer.Error())
		return
	}
	writeJSON(w, http.StatusOK, orderDTO(cart))
}

type AddCartItemRequest struct {
	ProductID int64 `json:"product_id"`
	Quantity  int   `json:"quantity"`
}

func (cr AddCartItemRequest) Validate() error {
	if cr.Quantity <= 0 {
		return ErrCartQuantityInvalid
	}
	return nil
}

// POST /api/v1/cart/items
func (oh OrderHandler) AddCartItem(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFromRequest(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, ErrTokenClaimsGetFailed.Error())
		return
	}

	var req AddCartItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, ErrDecodeFailed.Error())
		return
	}
	if err := req.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := oh.orderService.AddItemToCart(r.Context(), actor, req.ProductID, req.Quantity); err != nil {
		if errors.Is(err, service.ErrQuantityTooBig) {
			writeError(w, http.StatusBadRequest, service.ErrQuantityTooBig.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, ErrInternalServer.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type UpdateCartItemRequest struct {
	Quantity int `json:"quantity"`
}

func (ur UpdateCartItemRequest) Validate() error {
	if ur.Quantity <= 0 {
		return ErrCartQuantityInvalid
	}
	return nil
}

// PATCH /api/v1/cart/items/:id
func (oh OrderHandler) UpdateCartItem(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFromRequest(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, ErrTokenClaimsGetFailed.Error())
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrInvalidIDParam.Error())
		return
	}

	var req UpdateCartItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, ErrDecodeFailed.Error())
		return
	}
	if err := req.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := oh.orderService.ChangeQuantityCartItem(r.Context(), actor, id, req.Quantity); err != nil {
		if errors.Is(err, service.ErrQuantityTooBig) {
			writeError(w, http.StatusBadRequest, service.ErrQuantityTooBig.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, ErrInternalServer.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// DELETE /api/v1/cart/items/:id
func (oh OrderHandler) DeleteCartItem(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFromRequest(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, ErrTokenClaimsGetFailed.Error())
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrInvalidIDParam.Error())
		return
	}

	if err := oh.orderService.DeleteCartItem(r.Context(), actor, id); err != nil {
		writeError(w, http.StatusInternalServerError, ErrInternalServer.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type CheckoutRequest struct {
	AddressID int64 `json:"address_id"`
}

// POST /api/v1/orders — checkout (draft → pending, split by seller)
func (oh OrderHandler) Checkout(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFromRequest(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, ErrTokenClaimsGetFailed.Error())
		return
	}

	var req CheckoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, ErrDecodeFailed.Error())
		return
	}
	if req.AddressID <= 0 {
		writeError(w, http.StatusBadRequest, ErrInvalidIDParam.Error())
		return
	}

	orderIDs, err := oh.orderService.Checkout(r.Context(), actor, req.AddressID)
	if err != nil {
		if errors.Is(err, service.ErrCartNotFound) {
			writeError(w, http.StatusNotFound, service.ErrCartNotFound.Error())
			return
		}
		if errors.Is(err, service.ErrEmptyCart) {
			writeError(w, http.StatusBadRequest, service.ErrEmptyCart.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, ErrInternalServer.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string][]int64{"order_ids": orderIDs})
}

// POST /api/v1/orders/:id/pay
func (oh OrderHandler) PayOrder(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFromRequest(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, ErrTokenClaimsGetFailed.Error())
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrInvalidIDParam.Error())
		return
	}

	if err := oh.orderService.PayOrder(r.Context(), actor, id); err != nil {
		if errors.Is(err, service.ErrOrderNotFound) {
			writeError(w, http.StatusNotFound, service.ErrOrderNotFound.Error())
			return
		}
		if errors.Is(err, service.ErrNotYourOrder) {
			writeError(w, http.StatusForbidden, service.ErrNotYourOrder.Error())
			return
		}
		if errors.Is(err, service.ErrOrderStatusInvalid) {
			writeError(w, http.StatusConflict, service.ErrOrderStatusInvalid.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, ErrInternalServer.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// POST /api/v1/orders/:id/cancel
func (oh OrderHandler) CancelOrder(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFromRequest(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, ErrTokenClaimsGetFailed.Error())
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrInvalidIDParam.Error())
		return
	}

	if err := oh.orderService.CancelOrder(r.Context(), actor, id); err != nil {
		if errors.Is(err, service.ErrOrderNotFound) {
			writeError(w, http.StatusNotFound, service.ErrOrderNotFound.Error())
			return
		}
		if errors.Is(err, service.ErrNotYourOrder) {
			writeError(w, http.StatusForbidden, service.ErrNotYourOrder.Error())
			return
		}
		if errors.Is(err, service.ErrOrderStatusInvalid) {
			writeError(w, http.StatusConflict, service.ErrOrderStatusInvalid.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, ErrInternalServer.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// POST /api/v1/orders/:id/ship — seller only
func (oh OrderHandler) ShipOrder(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFromRequest(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, ErrTokenClaimsGetFailed.Error())
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrInvalidIDParam.Error())
		return
	}

	if err := oh.orderService.ShipOrder(r.Context(), actor, id); err != nil {
		if errors.Is(err, service.ErrSellerNotFound) {
			writeError(w, http.StatusForbidden, service.ErrSellerNotFound.Error())
			return
		}
		if errors.Is(err, service.ErrOrderNotFound) {
			writeError(w, http.StatusNotFound, service.ErrOrderNotFound.Error())
			return
		}
		if errors.Is(err, service.ErrNotYourOrder) {
			writeError(w, http.StatusForbidden, service.ErrNotYourOrder.Error())
			return
		}
		if errors.Is(err, service.ErrOrderStatusInvalid) {
			writeError(w, http.StatusConflict, service.ErrOrderStatusInvalid.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, ErrInternalServer.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// POST /api/v1/orders/:id/deliver — seller only
func (oh OrderHandler) DeliverOrder(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFromRequest(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, ErrTokenClaimsGetFailed.Error())
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrInvalidIDParam.Error())
		return
	}

	if err := oh.orderService.DeliverOrder(r.Context(), actor, id); err != nil {
		if errors.Is(err, service.ErrSellerNotFound) {
			writeError(w, http.StatusForbidden, service.ErrSellerNotFound.Error())
			return
		}
		if errors.Is(err, service.ErrOrderNotFound) {
			writeError(w, http.StatusNotFound, service.ErrOrderNotFound.Error())
			return
		}
		if errors.Is(err, service.ErrNotYourOrder) {
			writeError(w, http.StatusForbidden, service.ErrNotYourOrder.Error())
			return
		}
		if errors.Is(err, service.ErrOrderStatusInvalid) {
			writeError(w, http.StatusConflict, service.ErrOrderStatusInvalid.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, ErrInternalServer.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
