package web

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/beastixq/marketplace/internal/model"
	"github.com/go-chi/chi/v5"
)

// --- Orders ---

func (wh *WebHandler) Orders(w http.ResponseWriter, r *http.Request) {
	user := wh.userFromCookie(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	if user.Role == "seller" {
		http.Redirect(w, r, "/seller", http.StatusSeeOther)
		return
	}

	orders, _ := wh.orderService.GetOrdersByUserID(r.Context(), user.actor(), model.PaginationOpts{})

	// Split into current (pending, paid, shipped) and completed (delivered, cancelled), skip drafts
	var currentOrders, completedOrders []model.Order
	for _, o := range orders {
		switch o.Status {
		case model.StatusPending, model.StatusPaid, model.StatusShipped:
			currentOrders = append(currentOrders, o)
		case model.StatusDelivered, model.StatusCancelled:
			completedOrders = append(completedOrders, o)
		}
	}

	tab := r.URL.Query().Get("tab")
	if tab != "completed" {
		tab = "current"
	}

	wh.render(w, "orders", map[string]any{
		"User":            user,
		"CurrentOrders":   currentOrders,
		"CompletedOrders": completedOrders,
		"Tab":             tab,
		"Payment":         r.URL.Query().Get("payment"),
		"PaymentError":    r.URL.Query().Get("payment_error"),
	})
}

func (wh *WebHandler) OrderDetail(w http.ResponseWriter, r *http.Request) {
	user := wh.userFromCookie(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid order ID", http.StatusBadRequest)
		return
	}

	actor := user.actor()
	order, err := wh.orderService.GetOrderByID(r.Context(), actor, id)
	if err != nil {
		http.Error(w, "Order not found", http.StatusNotFound)
		return
	}

	items, _ := wh.orderService.GetOrderItemsByOrderID(r.Context(), actor, order.ID)
	displayItems := wh.buildOrderItemsDisplay(r.Context(), items)

	var seller *model.Seller
	if order.SellerID != nil {
		s, err := wh.sellerService.GetSellerByID(r.Context(), *order.SellerID)
		if err == nil {
			seller = &s
		}
	}

	var address *model.Address
	if user.Role == "buyer" && order.AddressID != nil {
		addresses, _ := wh.addressService.GetAddressesByUserID(r.Context(), actor)
		for _, a := range addresses {
			if a.ID == *order.AddressID {
				address = &a
				break
			}
		}
	}

	wh.render(w, "order-detail", map[string]any{
		"User":         user,
		"Order":        order,
		"Items":        displayItems,
		"Seller":       seller,
		"Address":      address,
		"Payment":      r.URL.Query().Get("payment"),
		"PaymentError": r.URL.Query().Get("payment_error"),
	})
}

func (wh *WebHandler) OrderPay(w http.ResponseWriter, r *http.Request) {
	user := wh.userFromCookie(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	if user.Role == "seller" {
		http.Redirect(w, r, "/seller", http.StatusSeeOther)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid order ID", http.StatusBadRequest)
		return
	}

	paymentURL, _, err := wh.paymentService.GetOrderPaymentURL(r.Context(), user.actor(), id)
	if err != nil {
		http.Redirect(w, r, fmt.Sprintf("/orders/%d?payment=failed&payment_error=%s", id, url.QueryEscape(err.Error())), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, paymentURL, http.StatusSeeOther)
}

func (wh *WebHandler) OrderCancel(w http.ResponseWriter, r *http.Request) {
	user := wh.userFromCookie(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	if user.Role == "seller" {
		http.Redirect(w, r, "/seller", http.StatusSeeOther)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid order ID", http.StatusBadRequest)
		return
	}

	_ = wh.orderService.CancelOrder(r.Context(), user.actor(), id)
	http.Redirect(w, r, "/orders", http.StatusSeeOther)
}
