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
	"github.com/jackc/pgx/v5/pgxpool"
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
	dbPool          *pgxpool.Pool
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
	dbPool *pgxpool.Pool,
) *WebHandler {
	pages := []string{"catalog", "product", "login", "register", "categories", "profile", "orders", "cart", "addresses", "seller", "product-edit", "seller-profile", "order-detail", "seller-orders", "seller-products", "admin-users", "admin-user-edit", "admin-categories", "admin-orders", "analyst"}
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
		dbPool:          dbPool,
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

func (u *userInfo) actor() service.Actor {
	return service.Actor{UserID: u.UserID, Role: model.UserRole(u.Role)}
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

	// Build user names map for reviews
	reviewUserNames := make(map[int64]string, len(reviews))
	for _, rv := range reviews {
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
		"Product":         product,
		"Seller":          seller,
		"PriceHistory":    priceHistory,
		"PriceRange":      rangeParam,
		"Reviews":         reviews,
		"ReviewUserNames": reviewUserNames,
		"User":            wh.userFromCookie(r),
		"Notice":          r.URL.Query().Get("notice"),
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

	actor := user.actor()
	u, err := wh.userService.GetUserByID(r.Context(), actor, user.UserID)
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
			u, _ := wh.userService.GetUserByID(r.Context(), user.actor(), user.UserID)
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
			u, _ := wh.userService.GetUserByID(r.Context(), user.actor(), user.UserID)
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
			u, _ := wh.userService.GetUserByID(r.Context(), user.actor(), user.UserID)
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

	actor := user.actor()
	_, err := wh.userService.UpdateUser(r.Context(), actor, user.UserID, update)

	u, _ := wh.userService.GetUserByID(r.Context(), actor, user.UserID)

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

	actor := user.actor()
	order, err := wh.orderService.GetOrderByID(r.Context(), actor, id)
	if err != nil {
		http.Error(w, "Order not found", http.StatusNotFound)
		return
	}

	items, _ := wh.orderService.GetOrderItemsByOrderID(r.Context(), actor, order.ID)
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
		addresses, _ := wh.addressService.GetAddressesByUserID(r.Context(), actor)
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

	_ = wh.orderService.PayOrder(r.Context(), user.actor(), id)
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

	_ = wh.orderService.CancelOrder(r.Context(), user.actor(), id)
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

	actor := user.actor()
	cart, err := wh.orderService.GetCart(r.Context(), actor)
	var items []model.OrderItem
	if err == nil {
		items, _ = wh.orderService.GetOrderItemsByOrderID(r.Context(), actor, cart.ID)
	}

	addresses, _ := wh.addressService.GetAddressesByUserID(r.Context(), actor)

	wh.render(w, "cart", map[string]any{
		"User":      user,
		"Cart":      cart,
		"Items":     wh.buildCartDisplay(r.Context(), items),
		"HasCart":   err == nil,
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

	if err = wh.orderService.AddItemToCart(r.Context(), user.actor(), productID, quantity); err != nil {
		if errors.Is(err, service.ErrProductAlreadyInCart) {
			http.Redirect(w, r, fmt.Sprintf("/products/%d?notice=already-in-cart", productID), http.StatusSeeOther)
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
		wh.render(w, "cart", map[string]any{
			"User":      user,
			"Cart":      cart,
			"Items":     wh.buildCartDisplay(r.Context(), items),
			"HasCart":   cartErr == nil,
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

	addresses, _ := wh.addressService.GetAddressesByUserID(r.Context(), user.actor())

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

	actor := user.actor()
	_, err := wh.addressService.CreateAddress(r.Context(), actor, model.AddressCreate{
		City:      city,
		Street:    street,
		ZipCode:   zipCode,
		IsDefault: isDefault,
	})

	addresses, _ := wh.addressService.GetAddressesByUserID(r.Context(), actor)

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

func (wh *WebHandler) AddressSetDefault(w http.ResponseWriter, r *http.Request) {
	user := wh.userFromCookie(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Redirect(w, r, "/addresses", http.StatusSeeOther)
		return
	}

	isDefault := true
	_, err = wh.addressService.UpdateAddress(r.Context(), user.actor(), id, model.AddressUpdate{IsDefault: &isDefault})
	if err != nil {
		log.Printf("AddressSetDefault error: %v", err)
	}
	http.Redirect(w, r, "/addresses", http.StatusSeeOther)
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

	_ = wh.addressService.DeleteAddressByID(r.Context(), user.actor(), id)
	http.Redirect(w, r, "/addresses", http.StatusSeeOther)
}

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
		ProductID: productID,
		Rating:    int8(rating),
	}
	if comment != "" {
		rc.Comment = &comment
	}

	if _, err = wh.reviewService.CreateReview(r.Context(), user.actor(), rc); err != nil {
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

// --- Admin Pages ---

func (wh *WebHandler) requireRole(w http.ResponseWriter, r *http.Request, roles ...string) *userInfo {
	user := wh.userFromCookie(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return nil
	}
	for _, role := range roles {
		if user.Role == role {
			return user
		}
	}
	http.Error(w, "Forbidden", http.StatusForbidden)
	return nil
}

func (wh *WebHandler) AdminUsers(w http.ResponseWriter, r *http.Request) {
	user := wh.requireRole(w, r, "admin")
	if user == nil {
		return
	}

	pageStr := r.URL.Query().Get("page")
	page, _ := strconv.Atoi(pageStr)
	if page < 1 {
		page = 1
	}
	const perPage = 50

	search := r.URL.Query().Get("search")
	role := r.URL.Query().Get("role")

	opts := model.UserListOptions{
		Pagination: model.PaginationOpts{Page: page, Limit: perPage},
	}
	if search != "" {
		opts.Search = &search
	}
	if role != "" {
		opts.Role = &role
	}

	users, _ := wh.userService.GetUsers(r.Context(), user.actor(), opts)

	wh.render(w, "admin-users", map[string]any{
		"User":       user,
		"Users":      users,
		"Pagination": map[string]int{"Page": page},
		"HasMore":    len(users) == perPage,
		"Search":     search,
		"Role":       role,
		"Success":    r.URL.Query().Get("success"),
		"Error":      r.URL.Query().Get("error"),
	})
}

func (wh *WebHandler) AdminUserEdit(w http.ResponseWriter, r *http.Request) {
	user := wh.requireRole(w, r, "admin")
	if user == nil {
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
		return
	}

	editUser, err := wh.userService.GetUserByID(r.Context(), user.actor(), id)
	if err != nil {
		http.Redirect(w, r, "/admin/users?error=User+not+found", http.StatusSeeOther)
		return
	}

	wh.render(w, "admin-user-edit", map[string]any{
		"User":     user,
		"EditUser": editUser,
		"Success":  r.URL.Query().Get("success"),
		"Error":    r.URL.Query().Get("error"),
	})
}

func (wh *WebHandler) AdminUserEditSubmit(w http.ResponseWriter, r *http.Request) {
	user := wh.requireRole(w, r, "admin")
	if user == nil {
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
		return
	}

	email := r.FormValue("email")
	fullName := r.FormValue("full_name")
	phone := r.FormValue("phone")
	roleStr := r.FormValue("role")
	role := model.UserRole(roleStr)

	uu := model.UserUpdate{
		Email:    &email,
		FullName: &fullName,
		Role:     &role,
	}
	if phone != "" {
		uu.Phone = &phone
	}

	_, err = wh.userService.UpdateUser(r.Context(), user.actor(), id, uu)
	if err != nil {
		http.Redirect(w, r, fmt.Sprintf("/admin/users/%d?error=%s", id, url.QueryEscape(err.Error())), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/admin/users/%d?success=User+updated", id), http.StatusSeeOther)
}

func (wh *WebHandler) AdminUserDelete(w http.ResponseWriter, r *http.Request) {
	user := wh.requireRole(w, r, "admin")
	if user == nil {
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
		return
	}

	if err = wh.userService.DeleteUserByID(r.Context(), user.actor(), id); err != nil {
		http.Redirect(w, r, fmt.Sprintf("/admin/users/%d?error=%s", id, url.QueryEscape(err.Error())), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/admin/users?success=User+deleted", http.StatusSeeOther)
}

func (wh *WebHandler) AdminCategories(w http.ResponseWriter, r *http.Request) {
	user := wh.requireRole(w, r, "admin")
	if user == nil {
		return
	}

	categories, _ := wh.categoryService.GetCategories(r.Context(), model.PaginationOpts{Page: 1, Limit: 500})

	wh.render(w, "admin-categories", map[string]any{
		"User":       user,
		"Categories": categories,
		"Success":    r.URL.Query().Get("success"),
		"Error":      r.URL.Query().Get("error"),
	})
}

func (wh *WebHandler) AdminCategoryCreate(w http.ResponseWriter, r *http.Request) {
	user := wh.requireRole(w, r, "admin")
	if user == nil {
		return
	}

	name := r.FormValue("name")
	description := r.FormValue("description")

	cc := model.CategoryCreate{Name: name}
	if description != "" {
		cc.Description = &description
	}

	_, err := wh.categoryService.CreateCategory(r.Context(), user.actor(), cc)
	if err != nil {
		http.Redirect(w, r, "/admin/categories?error="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/admin/categories?success=Category+created", http.StatusSeeOther)
}

func (wh *WebHandler) AdminCategoryDelete(w http.ResponseWriter, r *http.Request) {
	user := wh.requireRole(w, r, "admin")
	if user == nil {
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Redirect(w, r, "/admin/categories", http.StatusSeeOther)
		return
	}

	if err = wh.categoryService.DeleteCategoryByID(r.Context(), user.actor(), id); err != nil {
		http.Redirect(w, r, "/admin/categories?error="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/admin/categories?success=Category+deleted", http.StatusSeeOther)
}

func (wh *WebHandler) AdminOrders(w http.ResponseWriter, r *http.Request) {
	user := wh.requireRole(w, r, "admin")
	if user == nil {
		return
	}

	pageStr := r.URL.Query().Get("page")
	page, _ := strconv.Atoi(pageStr)
	if page < 1 {
		page = 1
	}
	const perPage = 50
	statusFilter := r.URL.Query().Get("status")

	ctx := r.Context()
	query := `SELECT id, user_id, address_id, seller_id, status, total_amount, created_at, updated_at
		FROM orders WHERE status != 'draft'`
	args := []any{}
	if statusFilter != "" {
		query += " AND status = $1"
		args = append(args, statusFilter)
	}
	query += " ORDER BY created_at DESC LIMIT $" + strconv.Itoa(len(args)+1) + " OFFSET $" + strconv.Itoa(len(args)+2)
	args = append(args, perPage, (page-1)*perPage)

	rows, err := wh.dbPool.Query(ctx, query, args...)
	var orders []model.Order
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var o model.Order
			if rows.Scan(&o.ID, &o.UserID, &o.AddressID, &o.SellerID, &o.Status, &o.TotalAmount, &o.CreatedAt, &o.UpdatedAt) == nil {
				orders = append(orders, o)
			}
		}
	}

	wh.render(w, "admin-orders", map[string]any{
		"User":    user,
		"Orders":  orders,
		"Page":    page,
		"HasMore": len(orders) == perPage,
		"Status":  statusFilter,
		"Success": r.URL.Query().Get("success"),
	})
}

// Admin moderation: delete product (bypasses ownership check)
func (wh *WebHandler) AdminProductDelete(w http.ResponseWriter, r *http.Request) {
	user := wh.requireRole(w, r, "admin")
	if user == nil {
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	_, err = wh.dbPool.Exec(r.Context(), "UPDATE products SET deleted_at = NOW() WHERE id = $1", id)
	if err != nil {
		log.Printf("AdminProductDelete error: %v", err)
	}
	http.Redirect(w, r, fmt.Sprintf("/products/%d", id), http.StatusSeeOther)
}

// Admin moderation: delete seller
func (wh *WebHandler) AdminSellerDelete(w http.ResponseWriter, r *http.Request) {
	user := wh.requireRole(w, r, "admin")
	if user == nil {
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
		return
	}

	_, err = wh.dbPool.Exec(r.Context(), "DELETE FROM sellers WHERE id = $1", id)
	if err != nil {
		log.Printf("AdminSellerDelete error: %v", err)
	}
	http.Redirect(w, r, "/admin/users?success=Seller+deleted", http.StatusSeeOther)
}

// Admin moderation: delete review
func (wh *WebHandler) AdminReviewDelete(w http.ResponseWriter, r *http.Request) {
	user := wh.requireRole(w, r, "admin")
	if user == nil {
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Get product ID for redirect
	var productID int64
	wh.dbPool.QueryRow(r.Context(), "SELECT product_id FROM reviews WHERE id = $1", id).Scan(&productID)

	_, err = wh.dbPool.Exec(r.Context(), "DELETE FROM reviews WHERE id = $1", id)
	if err != nil {
		log.Printf("AdminReviewDelete error: %v", err)
	}

	if productID > 0 {
		http.Redirect(w, r, fmt.Sprintf("/products/%d", productID), http.StatusSeeOther)
	} else {
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

// --- Analyst Dashboard ---

type platformStats struct {
	TotalUsers     int
	TotalSellers   int
	TotalProducts  int
	TotalOrders    int
	TotalRevenue   string
	TotalReviews   int
	UsersByRole    []roleCount
	OrdersByStatus []statusCount
	TopProducts    []topProduct
}

type roleCount struct {
	Role  string
	Count int
}

type statusCount struct {
	Status string
	Count  int
}

type topProduct struct {
	ID        int64
	Name      string
	Revenue   string
	UnitsSold int
}

func (wh *WebHandler) AnalystDashboard(w http.ResponseWriter, r *http.Request) {
	user := wh.requireRole(w, r, "analyst", "admin")
	if user == nil {
		return
	}

	ctx := r.Context()
	stats := platformStats{}

	// Total counts
	wh.dbPool.QueryRow(ctx, "SELECT count(*) FROM users WHERE deleted_at IS NULL").Scan(&stats.TotalUsers)
	wh.dbPool.QueryRow(ctx, "SELECT count(*) FROM sellers").Scan(&stats.TotalSellers)
	wh.dbPool.QueryRow(ctx, "SELECT count(*) FROM products WHERE deleted_at IS NULL").Scan(&stats.TotalProducts)
	wh.dbPool.QueryRow(ctx, "SELECT count(*) FROM orders WHERE status != 'draft'").Scan(&stats.TotalOrders)
	wh.dbPool.QueryRow(ctx, "SELECT count(*) FROM reviews").Scan(&stats.TotalReviews)
	wh.dbPool.QueryRow(ctx, "SELECT COALESCE(SUM(total_amount), 0) FROM orders WHERE status IN ('paid','shipped','delivered')").Scan(&stats.TotalRevenue)

	// Users by role
	rows, err := wh.dbPool.Query(ctx, "SELECT role, count(*) FROM users WHERE deleted_at IS NULL GROUP BY role ORDER BY count(*) DESC")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var rc roleCount
			if rows.Scan(&rc.Role, &rc.Count) == nil {
				stats.UsersByRole = append(stats.UsersByRole, rc)
			}
		}
	}

	// Orders by status
	rows2, err := wh.dbPool.Query(ctx, "SELECT status, count(*) FROM orders GROUP BY status ORDER BY count(*) DESC")
	if err == nil {
		defer rows2.Close()
		for rows2.Next() {
			var sc statusCount
			if rows2.Scan(&sc.Status, &sc.Count) == nil {
				stats.OrdersByStatus = append(stats.OrdersByStatus, sc)
			}
		}
	}

	// Top 10 products by revenue
	rows3, err := wh.dbPool.Query(ctx, `
		SELECT p.id, p.name, COALESCE(SUM(oi.price_at_purchase * oi.quantity), 0) as revenue, COALESCE(SUM(oi.quantity), 0) as units
		FROM products p
		JOIN order_items oi ON oi.product_id = p.id
		JOIN orders o ON o.id = oi.order_id AND o.status IN ('paid','shipped','delivered')
		GROUP BY p.id, p.name
		ORDER BY revenue DESC
		LIMIT 10
	`)
	if err == nil {
		defer rows3.Close()
		for rows3.Next() {
			var tp topProduct
			if rows3.Scan(&tp.ID, &tp.Name, &tp.Revenue, &tp.UnitsSold) == nil {
				stats.TopProducts = append(stats.TopProducts, tp)
			}
		}
	}

	wh.render(w, "analyst", map[string]any{
		"User":  user,
		"Stats": stats,
	})
}
