package web

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/beastixq/marketplace/internal/model"
	"github.com/go-chi/chi/v5"
	"github.com/shopspring/decimal"
)

// --- Seller Dashboard ---

func (wh *WebHandler) sellerFromUser(r *http.Request, user *userInfo) (model.Seller, bool) {
	if user == nil || user.Role != "seller" {
		return model.Seller{}, false
	}
	seller, err := wh.sellerService.GetSellerByUserID(r.Context(), user.actor())
	if err != nil {
		return model.Seller{}, false
	}
	return seller, true
}

func (wh *WebHandler) SellerDashboard(w http.ResponseWriter, r *http.Request) {
	user := wh.userFromCookie(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	if user.Role != "seller" {
		http.Error(w, "Only sellers can access this page", http.StatusForbidden)
		return
	}

	actor := user.actor()
	seller, err := wh.sellerService.GetSellerByUserID(r.Context(), actor)
	hasSeller := err == nil

	var stats *model.SellerStats
	if hasSeller {
		s, statsErr := wh.sellerService.GetSellerStats(r.Context(), seller.ID, time.Now().AddDate(-1, 0, 0), time.Now())
		if statsErr == nil {
			stats = &s
		}
	}

	wh.render(w, "seller", map[string]any{
		"User":      user,
		"Seller":    seller,
		"HasSeller": hasSeller,
		"Stats":     stats,
		"Error":     r.URL.Query().Get("error"),
		"Success":   r.URL.Query().Get("success"),
	})
}

func (wh *WebHandler) SellerUpdate(w http.ResponseWriter, r *http.Request) {
	user := wh.userFromCookie(r)
	seller, ok := wh.sellerFromUser(r, user)
	if !ok {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	companyName := r.FormValue("company_name")
	description := r.FormValue("description")

	su := model.SellerUpdate{}
	if companyName != "" {
		su.CompanyName = &companyName
	}
	su.Description = &description

	_, err := wh.sellerService.UpdateSeller(r.Context(), user.actor(), seller.ID, su)
	if err != nil {
		http.Redirect(w, r, "/seller?error=Failed+to+update+profile", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/seller?success=Profile+updated", http.StatusSeeOther)
}

func (wh *WebHandler) SellerOrders(w http.ResponseWriter, r *http.Request) {
	user := wh.userFromCookie(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	if user.Role != "seller" {
		http.Error(w, "Only sellers can access this page", http.StatusForbidden)
		return
	}

	actor := user.actor()
	_, err := wh.sellerService.GetSellerByUserID(r.Context(), actor)
	if err != nil {
		http.Redirect(w, r, "/seller", http.StatusSeeOther)
		return
	}

	orders, _ := wh.orderService.GetSellerOrdersByUserID(r.Context(), actor, model.PaginationOpts{Page: 1, Limit: 50})

	wh.render(w, "seller-orders", map[string]any{
		"User":   user,
		"Orders": orders,
	})
}

func (wh *WebHandler) SellerProducts(w http.ResponseWriter, r *http.Request) {
	user := wh.userFromCookie(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	if user.Role != "seller" {
		http.Error(w, "Only sellers can access this page", http.StatusForbidden)
		return
	}

	actor := user.actor()
	seller, err := wh.sellerService.GetSellerByUserID(r.Context(), actor)
	if err != nil {
		http.Redirect(w, r, "/seller", http.StatusSeeOther)
		return
	}

	allProducts, _ := wh.productService.GetProducts(r.Context(), model.CatalogOptions{})
	var products []model.Product
	for _, p := range allProducts {
		if p.SellerID == seller.ID {
			products = append(products, p)
		}
	}

	wh.render(w, "seller-products", map[string]any{
		"User":     user,
		"Products": products,
	})
}

func (wh *WebHandler) SellerCreate(w http.ResponseWriter, r *http.Request) {
	user := wh.userFromCookie(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	if user.Role != "seller" {
		http.Error(w, "Only sellers can access this page", http.StatusForbidden)
		return
	}

	companyName := r.FormValue("company_name")
	description := r.FormValue("description")

	sc := model.SellerCreate{
		CompanyName: companyName,
	}
	if description != "" {
		sc.Description = &description
	}

	_, err := wh.sellerService.CreateSeller(r.Context(), user.actor(), sc)
	if err != nil {
		wh.render(w, "seller", map[string]any{
			"User":      user,
			"HasSeller": false,
			"Error":     "Failed to create seller profile: " + err.Error(),
			"Success":   "",
		})
		return
	}

	http.Redirect(w, r, "/seller", http.StatusSeeOther)
}

// --- Seller Product CRUD ---

func (wh *WebHandler) SellerProductCreate(w http.ResponseWriter, r *http.Request) {
	user := wh.userFromCookie(r)
	seller, ok := wh.sellerFromUser(r, user)
	if !ok {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	name := r.FormValue("name")
	description := r.FormValue("description")
	priceStr := r.FormValue("price")
	stockStr := r.FormValue("stock_quantity")

	price, err := decimal.NewFromString(priceStr)
	if err != nil {
		http.Redirect(w, r, "/seller", http.StatusSeeOther)
		return
	}
	stock, err := strconv.Atoi(stockStr)
	if err != nil || stock < 0 {
		stock = 0
	}

	pc := model.ProductCreate{
		SellerID:      seller.ID,
		Name:          name,
		Price:         price,
		StockQuantity: stock,
	}
	if description != "" {
		pc.Description = &description
	}

	_, err = wh.productService.CreateProduct(r.Context(), user.actor(), pc)
	if err != nil {
		log.Printf("SellerProductCreate error: %v", err)
	}
	http.Redirect(w, r, "/seller/products", http.StatusSeeOther)
}

func (wh *WebHandler) SellerProductEditPage(w http.ResponseWriter, r *http.Request) {
	user := wh.userFromCookie(r)
	seller, ok := wh.sellerFromUser(r, user)
	if !ok {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid product ID", http.StatusBadRequest)
		return
	}

	product, err := wh.productService.GetProductByID(r.Context(), id)
	if err != nil || product.SellerID != seller.ID {
		http.Error(w, "Product not found", http.StatusNotFound)
		return
	}

	wh.render(w, "product-edit", map[string]any{
		"User":    user,
		"Product": product,
		"Error":   "",
	})
}

func (wh *WebHandler) SellerProductEditSubmit(w http.ResponseWriter, r *http.Request) {
	user := wh.userFromCookie(r)
	seller, ok := wh.sellerFromUser(r, user)
	if !ok {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid product ID", http.StatusBadRequest)
		return
	}

	product, err := wh.productService.GetProductByID(r.Context(), id)
	if err != nil || product.SellerID != seller.ID {
		http.Error(w, "Product not found", http.StatusNotFound)
		return
	}

	name := r.FormValue("name")
	description := r.FormValue("description")
	priceStr := r.FormValue("price")
	stockStr := r.FormValue("stock_quantity")

	pu := model.ProductUpdate{}
	if name != "" {
		pu.Name = &name
	}
	pu.Description = &description
	if p, err := decimal.NewFromString(priceStr); err == nil {
		pu.Price = &p
	}
	if s, err := strconv.Atoi(stockStr); err == nil {
		pu.StockQuantity = &s
	}

	changedBy := fmt.Sprintf("%s:%d", user.Role, user.UserID)
	pu.ChangedBy = &changedBy
	_, err = wh.productService.UpdateProduct(r.Context(), user.actor(), id, pu)
	if err != nil {
		wh.render(w, "product-edit", map[string]any{
			"User":    user,
			"Product": product,
			"Error":   "Failed to update: " + err.Error(),
		})
		return
	}

	http.Redirect(w, r, "/seller/products", http.StatusSeeOther)
}

func (wh *WebHandler) SellerProductDelete(w http.ResponseWriter, r *http.Request) {
	user := wh.userFromCookie(r)
	seller, ok := wh.sellerFromUser(r, user)
	if !ok {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Redirect(w, r, "/seller", http.StatusSeeOther)
		return
	}

	product, err := wh.productService.GetProductByID(r.Context(), id)
	if err != nil || product.SellerID != seller.ID {
		http.Redirect(w, r, "/seller/products", http.StatusSeeOther)
		return
	}

	_ = wh.productService.DeleteProductByID(r.Context(), user.actor(), id)
	http.Redirect(w, r, "/seller/products", http.StatusSeeOther)
}

// --- Seller Order Actions ---

func (wh *WebHandler) SellerOrderShip(w http.ResponseWriter, r *http.Request) {
	user := wh.userFromCookie(r)
	if user == nil || user.Role != "seller" {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Redirect(w, r, "/seller", http.StatusSeeOther)
		return
	}

	_ = wh.orderService.ShipOrder(r.Context(), user.actor(), id)
	http.Redirect(w, r, "/seller/orders", http.StatusSeeOther)
}

func (wh *WebHandler) SellerOrderDeliver(w http.ResponseWriter, r *http.Request) {
	user := wh.userFromCookie(r)
	if user == nil || user.Role != "seller" {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Redirect(w, r, "/seller", http.StatusSeeOther)
		return
	}

	_ = wh.orderService.DeliverOrder(r.Context(), user.actor(), id)
	http.Redirect(w, r, "/seller/orders", http.StatusSeeOther)
}
