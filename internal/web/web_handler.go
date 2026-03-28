package web

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/beastixq/marketplace/internal/model"
	"github.com/beastixq/marketplace/internal/service"
	"github.com/beastixq/marketplace/internal/validators"
	"github.com/go-chi/chi/v5"
	"github.com/shopspring/decimal"
)

//go:embed templates/*.html
var templateFS embed.FS

//go:embed static
var staticFS embed.FS

var funcMap = template.FuncMap{
	"sub": func(a, b int) int { return a - b },
	"add": func(a, b int) int { return a + b },
	"starRange": func(n int8) []struct{} {
		return make([]struct{}, n)
	},
	"emptyStarRange": func(n int8) []struct{} {
		return make([]struct{}, 5-n)
	},
}

type WebHandler struct {
	productService  service.ProductService
	categoryService service.CategoryService
	authService     service.AuthService
	userService     service.UserService
	orderService    service.OrderService
	addressService  service.AddressService
	sellerService   service.SellerService
	reviewService   service.ReviewService
	templates       map[string]*template.Template
}

func NewWebHandler(
	productSvc service.ProductService,
	categorySvc service.CategoryService,
	authSvc service.AuthService,
	userSvc service.UserService,
	orderSvc service.OrderService,
	addressSvc service.AddressService,
	sellerSvc service.SellerService,
	reviewSvc service.ReviewService,
) *WebHandler {
	pages := []string{"catalog", "product", "login", "register", "categories", "profile", "orders", "cart", "addresses", "seller", "product-edit", "seller-profile", "order-detail", "seller-orders", "seller-products"}
	templates := make(map[string]*template.Template, len(pages))
	for _, page := range pages {
		templates[page] = template.Must(
			template.New("").Funcs(funcMap).ParseFS(
				templateFS,
				"templates/layout.html",
				"templates/"+page+".html",
			),
		)
	}
	return &WebHandler{
		productService:  productSvc,
		categoryService: categorySvc,
		authService:     authSvc,
		userService:     userSvc,
		orderService:    orderSvc,
		addressService:  addressSvc,
		sellerService:   sellerSvc,
		reviewService:   reviewSvc,
		templates:       templates,
	}
}

func (wh *WebHandler) render(w http.ResponseWriter, page string, data any) {
	tmpl, ok := wh.templates[page]
	if !ok {
		http.Error(w, "Page not found", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
	}
}

// userFromCookie extracts user info from JWT cookie for navbar display.
func (wh *WebHandler) userFromCookie(r *http.Request) *userInfo {
	cookie, err := r.Cookie("token")
	if err != nil {
		return nil
	}
	claims, err := wh.authService.ValidateToken(r.Context(), cookie.Value)
	if err != nil {
		return nil
	}
	return &userInfo{UserID: claims.UserID, Role: string(claims.Role), FullName: fmt.Sprintf("User #%d", claims.UserID)}
}

type userInfo struct {
	UserID   int64
	Role     string
	FullName string
}

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

	// Price history for the last year
	dateFrom := time.Now().AddDate(-1, 0, 0)
	dateTo := time.Now()
	priceHistory, _ := wh.productService.GetProductPriceHistory(r.Context(), id, dateFrom, dateTo)

	reviews, _ := wh.productService.GetReviewsByProductID(r.Context(), id, model.PaginationOpts{Page: 1, Limit: 50})

	seller, _ := wh.sellerService.GetSellerByID(r.Context(), product.SellerID)

	wh.render(w, "product", map[string]any{
		"Product":      product,
		"Seller":       seller,
		"PriceHistory": priceHistory,
		"Reviews":      reviews,
		"User":         wh.userFromCookie(r),
		"Notice":       r.URL.Query().Get("notice"),
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

// --- Auth ---

func (wh *WebHandler) LoginPage(w http.ResponseWriter, r *http.Request) {
	wh.render(w, "login", map[string]any{
		"Email": "",
		"Error": "",
		"User":  wh.userFromCookie(r),
	})
}

func (wh *WebHandler) LoginSubmit(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	password := r.FormValue("password")

	token, err := wh.authService.Login(r.Context(), email, password)
	if err != nil {
		wh.render(w, "login", map[string]any{
			"Email": email,
			"Error": "Invalid email or password",
			"User":  nil,
		})
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (wh *WebHandler) RegisterPage(w http.ResponseWriter, r *http.Request) {
	wh.render(w, "register", map[string]any{
		"Email":    "",
		"FullName": "",
		"Phone":    "",
		"Role":     "buyer",
		"Error":    "",
		"User":     wh.userFromCookie(r),
	})
}

func (wh *WebHandler) RegisterSubmit(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	fullName := r.FormValue("full_name")
	password := r.FormValue("password")
	phone := r.FormValue("phone")
	role := r.FormValue("role")

	renderErr := func(msg string) {
		wh.render(w, "register", map[string]any{
			"Email":    email,
			"FullName": fullName,
			"Phone":    phone,
			"Role":     role,
			"Error":    msg,
			"User":     nil,
		})
	}

	if err := validators.ValidateEmail(email); err != nil {
		renderErr(err.Error())
		return
	}
	if err := validators.ValidateFullName(fullName); err != nil {
		renderErr(err.Error())
		return
	}
	if err := validators.ValidatePassword(password); err != nil {
		renderErr(err.Error())
		return
	}
	if phone != "" {
		if err := validators.ValidatePhone(phone); err != nil {
			renderErr(err.Error())
			return
		}
	}

	uc := model.UserCreate{
		Email:    email,
		FullName: fullName,
		Password: password,
		Role:     model.UserRole(role),
	}
	if phone != "" {
		uc.Phone = &phone
	}

	token, err := wh.authService.Register(r.Context(), uc)
	if err != nil {
		wh.render(w, "register", map[string]any{
			"Email":    email,
			"FullName": fullName,
			"Phone":    phone,
			"Role":     role,
			"Error":    "Registration failed: " + err.Error(),
			"User":     nil,
		})
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (wh *WebHandler) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// --- Profile ---

func (wh *WebHandler) Profile(w http.ResponseWriter, r *http.Request) {
	user := wh.userFromCookie(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	u, err := wh.userService.GetUserByID(r.Context(), user.UserID)
	if err != nil {
		http.Error(w, "Failed to load profile", http.StatusInternalServerError)
		return
	}

	wh.render(w, "profile", map[string]any{
		"User":    user,
		"Profile": u,
		"Error":   "",
		"Success": "",
	})
}

func (wh *WebHandler) ProfileUpdate(w http.ResponseWriter, r *http.Request) {
	user := wh.userFromCookie(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	fullName := r.FormValue("full_name")
	email := r.FormValue("email")
	phone := r.FormValue("phone")

	update := model.UserUpdate{}
	if fullName != "" {
		if err := validators.ValidateFullName(fullName); err != nil {
			u, _ := wh.userService.GetUserByID(r.Context(), user.UserID)
			wh.render(w, "profile", map[string]any{
				"User":    user,
				"Profile": u,
				"Error":   err.Error(),
				"Success": "",
			})
			return
		}
		update.FullName = &fullName
	}
	if email != "" {
		if err := validators.ValidateEmail(email); err != nil {
			u, _ := wh.userService.GetUserByID(r.Context(), user.UserID)
			wh.render(w, "profile", map[string]any{
				"User":    user,
				"Profile": u,
				"Error":   err.Error(),
				"Success": "",
			})
			return
		}
		update.Email = &email
	}
	if phone != "" {
		if err := validators.ValidatePhone(phone); err != nil {
			u, _ := wh.userService.GetUserByID(r.Context(), user.UserID)
			wh.render(w, "profile", map[string]any{
				"User":    user,
				"Profile": u,
				"Error":   err.Error(),
				"Success": "",
			})
			return
		}
		update.Phone = &phone
	}

	_, err := wh.userService.UpdateUser(r.Context(), user.UserID, update)

	u, _ := wh.userService.GetUserByID(r.Context(), user.UserID)

	if err != nil {
		errMsg := "Failed to update profile"
		switch {
		case errors.Is(err, service.ErrPhoneAlreadyExists):
			errMsg = service.ErrPhoneAlreadyExists.Error()
		case errors.Is(err, service.ErrEmailAlreadyInUse):
			errMsg = service.ErrEmailAlreadyInUse.Error()
		}
		wh.render(w, "profile", map[string]any{
			"User":    user,
			"Profile": u,
			"Error":   errMsg,
			"Success": "",
		})
		return
	}

	wh.render(w, "profile", map[string]any{
		"User":    user,
		"Profile": u,
		"Error":   "",
		"Success": "Profile updated successfully",
	})
}

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

	orders, _ := wh.orderService.GetOrdersByUserID(r.Context(), user.UserID, model.PaginationOpts{})

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
	})
}

func (wh *WebHandler) OrderDetail(w http.ResponseWriter, r *http.Request) {
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

	order, err := wh.orderService.GetOrderByID(r.Context(), id)
	if err != nil || order.UserID != user.UserID {
		http.Error(w, "Order not found", http.StatusNotFound)
		return
	}

	items, _ := wh.orderService.GetOrderItemsByOrderID(r.Context(), order.ID)
	displayItems := wh.buildCartDisplay(r.Context(), items)

	var seller *model.Seller
	if order.SellerID != nil {
		s, err := wh.sellerService.GetSellerByID(r.Context(), *order.SellerID)
		if err == nil {
			seller = &s
		}
	}

	var address *model.Address
	if order.AddressID != nil {
		addresses, _ := wh.addressService.GetAddressesByUserID(r.Context(), user.UserID)
		for _, a := range addresses {
			if a.ID == *order.AddressID {
				address = &a
				break
			}
		}
	}

	wh.render(w, "order-detail", map[string]any{
		"User":    user,
		"Order":   order,
		"Items":   displayItems,
		"Seller":  seller,
		"Address": address,
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

	_ = wh.orderService.PayOrder(r.Context(), id, user.UserID)
	http.Redirect(w, r, "/orders", http.StatusSeeOther)
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

	_ = wh.orderService.CancelOrder(r.Context(), id, user.UserID)
	http.Redirect(w, r, "/orders", http.StatusSeeOther)
}

// --- Cart ---

type cartItemDisplay struct {
	model.OrderItem
	ProductName string
	TotalPrice  string
}

func (wh *WebHandler) buildCartDisplay(ctx context.Context, items []model.OrderItem) []cartItemDisplay {
	display := make([]cartItemDisplay, 0, len(items))
	for _, item := range items {
		p, _ := wh.productService.GetProductByID(ctx, item.ProductID)
		display = append(display, cartItemDisplay{
			OrderItem:   item,
			ProductName: p.Name,
			TotalPrice:  item.PriceAtPurchase.Mul(decimal.NewFromInt(int64(item.Quantity))).String(),
		})
	}
	return display
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

	cart, err := wh.orderService.GetCart(r.Context(), user.UserID)
	var items []model.OrderItem
	if err == nil {
		items, _ = wh.orderService.GetOrderItemsByOrderID(r.Context(), cart.ID)
	}

	addresses, _ := wh.addressService.GetAddressesByUserID(r.Context(), user.UserID)

	wh.render(w, "cart", map[string]any{
		"User":      user,
		"Cart":      cart,
		"Items":     wh.buildCartDisplay(r.Context(), items),
		"HasCart":    err == nil,
		"Addresses": addresses,
		"Error":     "",
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

	if err = wh.orderService.AddItemToCart(r.Context(), user.UserID, productID, quantity); err != nil {
		if errors.Is(err, service.ErrProductAlreadyInCart) {
			http.Redirect(w, r, fmt.Sprintf("/products/%d?notice=already-in-cart", productID), http.StatusSeeOther)
			return
		}
		log.Printf("CartAdd error: %v", err)
	}
	http.Redirect(w, r, "/cart", http.StatusSeeOther)
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

	_ = wh.orderService.DeleteCartItem(r.Context(), id)
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

	if err = wh.orderService.ChangeQuantityCartItem(r.Context(), id, quantity); err != nil {
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

	_, err = wh.orderService.Checkout(r.Context(), user.UserID, addressID)
	if err != nil {
		// Reload cart with error
		cart, cartErr := wh.orderService.GetCart(r.Context(), user.UserID)
		var items []model.OrderItem
		if cartErr == nil {
			items, _ = wh.orderService.GetOrderItemsByOrderID(r.Context(), cart.ID)
		}
		addresses, _ := wh.addressService.GetAddressesByUserID(r.Context(), user.UserID)
		wh.render(w, "cart", map[string]any{
			"User":      user,
			"Cart":      cart,
			"Items":     wh.buildCartDisplay(r.Context(), items),
			"HasCart":    cartErr == nil,
			"Addresses": addresses,
			"Error":     "Checkout failed: " + err.Error(),
		})
		return
	}

	http.Redirect(w, r, "/orders", http.StatusSeeOther)
}

// --- Addresses ---

func (wh *WebHandler) Addresses(w http.ResponseWriter, r *http.Request) {
	user := wh.userFromCookie(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	if user.Role == "seller" {
		http.Redirect(w, r, "/seller", http.StatusSeeOther)
		return
	}

	addresses, _ := wh.addressService.GetAddressesByUserID(r.Context(), user.UserID)

	wh.render(w, "addresses", map[string]any{
		"User":      user,
		"Addresses": addresses,
		"Error":     "",
		"Success":   "",
	})
}

func (wh *WebHandler) AddressCreate(w http.ResponseWriter, r *http.Request) {
	user := wh.userFromCookie(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	if user.Role == "seller" {
		http.Redirect(w, r, "/seller", http.StatusSeeOther)
		return
	}

	city := r.FormValue("city")
	street := r.FormValue("street")
	zipCode := r.FormValue("zip_code")
	isDefault := r.FormValue("is_default") == "on"

	_, err := wh.addressService.CreateAddress(r.Context(), model.AddressCreate{
		UserID:    user.UserID,
		City:      city,
		Street:    street,
		ZipCode:   zipCode,
		IsDefault: isDefault,
	})

	addresses, _ := wh.addressService.GetAddressesByUserID(r.Context(), user.UserID)

	if err != nil {
		wh.render(w, "addresses", map[string]any{
			"User":      user,
			"Addresses": addresses,
			"Error":     "Failed to create address: " + err.Error(),
			"Success":   "",
		})
		return
	}

	wh.render(w, "addresses", map[string]any{
		"User":      user,
		"Addresses": addresses,
		"Error":     "",
		"Success":   "Address added",
	})
}

func (wh *WebHandler) AddressDelete(w http.ResponseWriter, r *http.Request) {
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
		http.Redirect(w, r, "/addresses", http.StatusSeeOther)
		return
	}

	_ = wh.addressService.DeleteAddressByID(r.Context(), user.UserID, id)
	http.Redirect(w, r, "/addresses", http.StatusSeeOther)
}

// --- Seller Dashboard ---

func (wh *WebHandler) sellerFromUser(r *http.Request, user *userInfo) (model.Seller, bool) {
	if user == nil || user.Role != "seller" {
		return model.Seller{}, false
	}
	seller, err := wh.sellerService.GetSellerByUserID(r.Context(), user.UserID)
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

	seller, err := wh.sellerService.GetSellerByUserID(r.Context(), user.UserID)
	hasSeller := err == nil

	var stats *model.SellerStats
	if hasSeller {
		s, statsErr := wh.sellerService.GetSellerStats(r.Context(), user.UserID, seller.ID, time.Now().AddDate(-1, 0, 0), time.Now())
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

	_, err := wh.sellerService.UpdateSeller(r.Context(), user.UserID, seller.ID, su)
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

	_, err := wh.sellerService.GetSellerByUserID(r.Context(), user.UserID)
	if err != nil {
		http.Redirect(w, r, "/seller", http.StatusSeeOther)
		return
	}

	orders, _ := wh.orderService.GetSellerOrdersByUserID(r.Context(), user.UserID, model.PaginationOpts{Page: 1, Limit: 50})

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

	seller, err := wh.sellerService.GetSellerByUserID(r.Context(), user.UserID)
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
		UserID:      user.UserID,
		CompanyName: companyName,
	}
	if description != "" {
		sc.Description = &description
	}

	_, err := wh.sellerService.CreateSeller(r.Context(), sc)
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

	_, err = wh.productService.CreateProduct(r.Context(), user.UserID, pc)
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
	_, err = wh.productService.UpdateProduct(r.Context(), user.UserID, id, pu)
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

	_ = wh.productService.DeleteProductByID(r.Context(), user.UserID, id)
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

	_ = wh.orderService.ShipOrder(r.Context(), id, user.UserID)
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

	_ = wh.orderService.DeliverOrder(r.Context(), id, user.UserID)
	http.Redirect(w, r, "/seller/orders", http.StatusSeeOther)
}

// --- Review Submission ---

func (wh *WebHandler) ReviewSubmit(w http.ResponseWriter, r *http.Request) {
	user := wh.userFromCookie(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	idStr := chi.URLParam(r, "id")
	productID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid product ID", http.StatusBadRequest)
		return
	}

	ratingStr := r.FormValue("rating")
	rating, err := strconv.Atoi(ratingStr)
	if err != nil || rating < 1 || rating > 5 {
		rating = 5
	}

	comment := r.FormValue("comment")

	rc := model.ReviewCreate{
		UserID:    user.UserID,
		ProductID: productID,
		Rating:    int8(rating),
	}
	if comment != "" {
		rc.Comment = &comment
	}

	if _, err = wh.reviewService.CreateReview(r.Context(), rc); err != nil {
		log.Printf("ReviewSubmit error: %v", err)
	}

	http.Redirect(w, r, fmt.Sprintf("/products/%d", productID), http.StatusSeeOther)
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

	allProducts, _ := wh.productService.GetProducts(r.Context(), model.CatalogOptions{})
	var products []model.Product
	for _, p := range allProducts {
		if p.SellerID == seller.ID {
			products = append(products, p)
		}
	}

	wh.render(w, "seller-profile", map[string]any{
		"Seller":   seller,
		"Products": products,
		"User":     wh.userFromCookie(r),
	})
}
