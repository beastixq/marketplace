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
	r.Post("/orders/{id}/pay", wh.OrderPay)
	r.Post("/orders/{id}/cancel", wh.OrderCancel)
	r.Get("/cart", wh.Cart)
	r.Post("/cart/add", wh.CartAdd)
	r.Post("/cart/items/{id}/remove", wh.CartRemoveItem)
	r.Post("/cart/checkout", wh.CartCheckout)

	return r
}
