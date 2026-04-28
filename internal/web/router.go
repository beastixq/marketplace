package web

import (
	"io/fs"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func NewWebRouter(wh *WebHandler) http.Handler {
	r := chi.NewRouter()

	// Static files
	staticSub, _ := fs.Sub(staticFS, "static")
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.FS(staticSub))))

	// Public pages
	r.Get("/", wh.Catalog)
	r.Get("/products/{id}", wh.ProductDetail)
	r.Get("/categories", wh.Categories)
	r.Get("/sellers/{id}", wh.SellerProfile)

	// Auth
	r.Get("/login", wh.LoginPage)
	r.Post("/login", wh.LoginSubmit)
	r.Get("/register", wh.RegisterPage)
	r.Post("/register", wh.RegisterSubmit)
	r.Get("/logout", wh.Logout)

	// Authenticated pages
	r.Get("/profile", wh.Profile)
	r.Post("/profile", wh.ProfileUpdate)
	r.Get("/orders", wh.Orders)
	r.Get("/orders/{id}", wh.OrderDetail)
	r.Post("/orders/{id}/pay", wh.OrderPay)
	r.Post("/orders/{id}/cancel", wh.OrderCancel)
	r.Get("/mock_bank/payment", wh.MockBankPaymentPage)
	r.Post("/mock_bank/payment", wh.MockBankPaymentSubmit)
	r.Get("/cart", wh.Cart)
	r.Post("/cart/add", wh.CartAdd)
	r.Post("/cart/items/{id}/update", wh.CartUpdateQuantity)
	r.Post("/cart/items/{id}/remove", wh.CartRemoveItem)
	r.Post("/cart/checkout", wh.CartCheckout)

	// Addresses
	r.Get("/addresses", wh.Addresses)
	r.Post("/addresses", wh.AddressCreate)
	r.Post("/addresses/{id}/default", wh.AddressSetDefault)
	r.Post("/addresses/{id}/delete", wh.AddressDelete)

	// Seller
	r.Get("/seller", wh.SellerDashboard)
	r.Post("/seller", wh.SellerCreate)
	r.Post("/seller/update", wh.SellerUpdate)
	r.Get("/seller/orders", wh.SellerOrders)
	r.Get("/seller/products", wh.SellerProducts)
	r.Post("/seller/products", wh.SellerProductCreate)
	r.Get("/seller/products/{id}/edit", wh.SellerProductEditPage)
	r.Post("/seller/products/{id}/edit", wh.SellerProductEditSubmit)
	r.Post("/seller/products/{id}/delete", wh.SellerProductDelete)
	r.Post("/seller/orders/{id}/ship", wh.SellerOrderShip)
	r.Post("/seller/orders/{id}/deliver", wh.SellerOrderDeliver)

	// Reviews
	r.Post("/products/{id}/review", wh.ReviewSubmit)
	r.Post("/reviews/{id}/edit", wh.ReviewUpdate)
	r.Post("/reviews/{id}/delete", wh.ReviewDelete)

	// Admin
	r.Get("/admin/users", wh.AdminUsers)
	r.Get("/admin/users/{id}", wh.AdminUserEdit)
	r.Post("/admin/users/{id}", wh.AdminUserEditSubmit)
	r.Post("/admin/users/{id}/delete", wh.AdminUserDelete)
	r.Get("/admin/categories", wh.AdminCategories)
	r.Post("/admin/categories", wh.AdminCategoryCreate)
	r.Post("/admin/categories/{id}", wh.AdminCategoryUpdate)
	r.Post("/admin/categories/{id}/delete", wh.AdminCategoryDelete)
	r.Get("/admin/orders", wh.AdminOrders)
	r.Post("/admin/products/{id}/delete", wh.AdminProductDelete)
	r.Post("/admin/sellers/{id}/delete", wh.AdminSellerDelete)
	r.Post("/admin/reviews/{id}/delete", wh.AdminReviewDelete)

	// Analyst
	r.Get("/analyst", wh.AnalystDashboard)

	return r
}
