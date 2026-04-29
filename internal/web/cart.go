package web

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/beastixq/marketplace/internal/model"
	"github.com/beastixq/marketplace/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/shopspring/decimal"
)

// --- Cart ---

type cartItemDisplay struct {
	model.OrderItem
	ProductName string
	UnitPrice   decimal.Decimal
	TotalPrice  string
	DeletedAt   *time.Time
}

func (wh *WebHandler) buildOrderItemsDisplay(ctx context.Context, items []model.OrderItem) []cartItemDisplay {
	display := make([]cartItemDisplay, 0, len(items))
	for _, item := range items {
		productName := fmt.Sprintf("Product #%d", item.ProductID)
		var deletedAt *time.Time
		if p, err := wh.productService.GetProductByID(ctx, item.ProductID); err == nil {
			productName = p.Name
			deletedAt = p.DeletedAt
		}
		unitPrice := item.PriceAtPurchase
		itemTotal := unitPrice.Mul(decimal.NewFromInt(int64(item.Quantity)))
		display = append(display, cartItemDisplay{
			OrderItem:   item,
			ProductName: productName,
			UnitPrice:   unitPrice,
			TotalPrice:  itemTotal.String(),
			DeletedAt:   deletedAt,
		})
	}
	return display
}

func (wh *WebHandler) buildCartDisplay(ctx context.Context, items []model.OrderItem) ([]cartItemDisplay, decimal.Decimal, bool) {
	display := make([]cartItemDisplay, 0, len(items))
	total := decimal.Zero
	hasDeletedItems := false
	for _, item := range items {
		productName := fmt.Sprintf("Product #%d", item.ProductID)
		unitPrice := item.PriceAtPurchase
		var deletedAt *time.Time
		if p, err := wh.productService.GetProductByID(ctx, item.ProductID); err == nil {
			productName = p.Name
			unitPrice = p.Price
			deletedAt = p.DeletedAt
			if deletedAt != nil {
				hasDeletedItems = true
			}
		}

		itemTotal := unitPrice.Mul(decimal.NewFromInt(int64(item.Quantity)))
		total = total.Add(itemTotal)
		display = append(display, cartItemDisplay{
			OrderItem:   item,
			ProductName: productName,
			UnitPrice:   unitPrice,
			TotalPrice:  itemTotal.String(),
			DeletedAt:   deletedAt,
		})
	}
	return display, total, hasDeletedItems
}

func (wh *WebHandler) Cart(w http.ResponseWriter, r *http.Request) {
	user := wh.userFromCookie(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	if user.Role == "seller" {
		http.Redirect(w, r, "/seller", http.StatusSeeOther)
		return
	}

	actor := user.actor()
	cart, err := wh.orderService.GetCart(r.Context(), actor)
	var items []model.OrderItem
	if err == nil {
		items, _ = wh.orderService.GetOrderItemsByOrderID(r.Context(), actor, cart.ID)
	}

	addresses, _ := wh.addressService.GetAddressesByUserID(r.Context(), actor)
	displayItems, cartTotal, hasDeletedItems := wh.buildCartDisplay(r.Context(), items)

	wh.render(w, "cart", map[string]any{
		"User":            user,
		"Cart":            cart,
		"Items":           displayItems,
		"CartTotal":       cartTotal.String(),
		"HasCart":         err == nil,
		"Addresses":       addresses,
		"Error":           "",
		"HasDeletedItems": hasDeletedItems,
	})
}

func (wh *WebHandler) CartAdd(w http.ResponseWriter, r *http.Request) {
	user := wh.userFromCookie(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	if user.Role == "seller" {
		http.Redirect(w, r, "/seller", http.StatusSeeOther)
		return
	}

	productIDStr := r.FormValue("product_id")
	productID, err := strconv.ParseInt(productIDStr, 10, 64)
	if err != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	quantityStr := r.FormValue("quantity")
	quantity, err := strconv.Atoi(quantityStr)
	if err != nil || quantity < 1 {
		quantity = 1
	}

	if err = wh.orderService.AddItemToCart(r.Context(), user.actor(), productID, quantity); err != nil {
		if errors.Is(err, service.ErrProductAlreadyInCart) {
			http.Redirect(w, r, fmt.Sprintf("/products/%d?notice=already-in-cart", productID), http.StatusSeeOther)
			return
		}
		if errors.Is(err, service.ErrProductDeleted) {
			http.Redirect(w, r, fmt.Sprintf("/products/%d?notice=product-deleted", productID), http.StatusSeeOther)
			return
		}
		log.Printf("CartAdd error: %v", err)
	}
	http.Redirect(w, r, fmt.Sprintf("/products/%d?notice=added-to-cart", productID), http.StatusSeeOther)
}

func (wh *WebHandler) CartRemoveItem(w http.ResponseWriter, r *http.Request) {
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
		http.Redirect(w, r, "/cart", http.StatusSeeOther)
		return
	}

	_ = wh.orderService.DeleteCartItem(r.Context(), user.actor(), id)
	http.Redirect(w, r, "/cart", http.StatusSeeOther)
}

func (wh *WebHandler) CartUpdateQuantity(w http.ResponseWriter, r *http.Request) {
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
		http.Redirect(w, r, "/cart", http.StatusSeeOther)
		return
	}

	quantityStr := r.FormValue("quantity")
	quantity, err := strconv.Atoi(quantityStr)
	if err != nil || quantity < 1 {
		http.Redirect(w, r, "/cart", http.StatusSeeOther)
		return
	}

	if err = wh.orderService.ChangeQuantityCartItem(r.Context(), user.actor(), id, quantity); err != nil {
		log.Printf("CartUpdateQuantity error: %v", err)
	}
	http.Redirect(w, r, "/cart", http.StatusSeeOther)
}

func (wh *WebHandler) CartCheckout(w http.ResponseWriter, r *http.Request) {
	user := wh.userFromCookie(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	if user.Role == "seller" {
		http.Redirect(w, r, "/seller", http.StatusSeeOther)
		return
	}

	addressIDStr := r.FormValue("address_id")
	addressID, err := strconv.ParseInt(addressIDStr, 10, 64)
	if err != nil {
		http.Redirect(w, r, "/cart", http.StatusSeeOther)
		return
	}

	actor := user.actor()
	_, err = wh.orderService.Checkout(r.Context(), actor, addressID)
	if err != nil {
		// Reload cart with error
		cart, cartErr := wh.orderService.GetCart(r.Context(), actor)
		var items []model.OrderItem
		if cartErr == nil {
			items, _ = wh.orderService.GetOrderItemsByOrderID(r.Context(), actor, cart.ID)
		}
		addresses, _ := wh.addressService.GetAddressesByUserID(r.Context(), actor)
		displayItems, cartTotal, hasDeletedItems := wh.buildCartDisplay(r.Context(), items)
		wh.render(w, "cart", map[string]any{
			"User":            user,
			"Cart":            cart,
			"Items":           displayItems,
			"CartTotal":       cartTotal.String(),
			"HasCart":         cartErr == nil,
			"Addresses":       addresses,
			"Error":           "Checkout failed: " + err.Error(),
			"HasDeletedItems": hasDeletedItems,
		})
		return
	}

	http.Redirect(w, r, "/orders", http.StatusSeeOther)
}
