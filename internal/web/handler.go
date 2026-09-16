package web

import (
	"embed"
	"fmt"
	"html/template"
	"net/http"

	"github.com/beastixq/marketplace/internal/middleware"
	"github.com/beastixq/marketplace/internal/model"
	"github.com/beastixq/marketplace/internal/service"
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
	"ratingValue": func(rating *float32) string {
		if rating == nil {
			return ""
		}
		return fmt.Sprintf("%.1f", *rating)
	},
}

type WebHandler struct {
	productService    service.ProductService
	categoryService   service.CategoryService
	authService       service.AuthService
	userService       service.UserService
	orderService      service.OrderService
	addressService    service.AddressService
	sellerService     service.SellerService
	reviewService     service.ReviewService
	backofficeService service.BackofficeService
	paymentService    *service.PaymentService
	templates         map[string]*template.Template
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
	backofficeSvc service.BackofficeService,
	paymentSvc *service.PaymentService,
) *WebHandler {
	pages := []string{"catalog", "product", "login", "register", "categories", "profile", "orders", "cart", "addresses", "seller", "product-edit", "seller-profile", "order-detail", "seller-orders", "seller-products", "admin-users", "admin-user-edit", "admin-categories", "admin-orders", "analyst", "payment"}
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
		productService:    productSvc,
		categoryService:   categorySvc,
		authService:       authSvc,
		userService:       userSvc,
		orderService:      orderSvc,
		addressService:    addressSvc,
		sellerService:     sellerSvc,
		reviewService:     reviewSvc,
		backofficeService: backofficeSvc,
		paymentService:    paymentSvc,
		templates:         templates,
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
	middleware.PublishActor(r.Context(), claims)
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
