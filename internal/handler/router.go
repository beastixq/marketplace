package handler

import (
	"net/http"

	"github.com/beastixq/marketplace/internal/middleware"
	"github.com/beastixq/marketplace/internal/model"
	"github.com/beastixq/marketplace/internal/service"
	"github.com/go-chi/chi/v5"
)

func NewRouter(
	authService service.AuthService,
	authHandler AuthHandler,
	userHandler UserHandler,
	sellerHandler SellerHandler,
	addressHandler AddressHandler,
	productHandler ProductHandler,
	orderHandler OrderHandler,
	paymentHandler PaymentHandler,
	categoryHandler CategoryHandler,
	reviewHandler ReviewHandler,
	adminHandler AdminHandler,
) http.Handler {
	r := chi.NewRouter()

	// Public routes — no auth
	r.Post("/api/v1/auth/register", authHandler.Register)
	r.Post("/api/v1/auth/login", authHandler.Login)

	// Payment callback (called by bank, no auth)
	r.Post("/api/v1/payments/callback/mock-bank", paymentHandler.MockBankCallback)

	// Public read-only
	r.Get("/api/v1/products", productHandler.GetCatalog)
	r.Get("/api/v1/products/{id}", productHandler.GetProduct)
	r.Get("/api/v1/products/{id}/price-history", productHandler.GetProductPriceHistory)
	r.Get("/api/v1/products/{id}/reviews", productHandler.GetProductReviews)
	r.Get("/api/v1/categories", categoryHandler.GetCategories)
	r.Get("/api/v1/sellers/{id}", sellerHandler.GetSellerByID)
	r.Get("/api/v1/sellers/{id}/stats", sellerHandler.GetSellerStats)

	// Authenticated routes
	r.Group(func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(authService))

		// Auth
		r.Post("/api/v1/auth/logout", authHandler.Logout)

		// Users
		r.Get("/api/v1/users/me", userHandler.GetMyProfile)
		r.Patch("/api/v1/users/me", userHandler.UpdateMyProfile)
		r.Delete("/api/v1/users/me", userHandler.DeleteMyAccount)
		r.Patch("/api/v1/users/me/password", userHandler.ChangePassword)

		// Buyer only
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireRole(model.RoleBuyer))

			// Addresses
			r.Get("/api/v1/addresses", addressHandler.GetAddresses)
			r.Post("/api/v1/addresses", addressHandler.CreateAddress)
			r.Patch("/api/v1/addresses/{id}", addressHandler.UpdateAddress)
			r.Delete("/api/v1/addresses/{id}", addressHandler.DeleteAddress)

			// Cart
			r.Get("/api/v1/cart", orderHandler.GetCart)
			r.Post("/api/v1/cart/items", orderHandler.AddCartItem)
			r.Patch("/api/v1/cart/items/{id}", orderHandler.UpdateCartItem)
			r.Delete("/api/v1/cart/items/{id}", orderHandler.DeleteCartItem)

			// Orders (buyer)
			r.Get("/api/v1/orders", orderHandler.GetOrders)
			r.Get("/api/v1/orders/{id}", orderHandler.GetOrder)
			r.Get("/api/v1/orders/{id}/items", orderHandler.GetOrderItems)
			r.Post("/api/v1/orders", orderHandler.Checkout)
			r.Post("/api/v1/orders/{id}/payment-link", paymentHandler.GetPaymentLink)
			r.Post("/api/v1/orders/{id}/cancel", orderHandler.CancelOrder)

			// Reviews
			r.Post("/api/v1/reviews", reviewHandler.CreateReview)
			r.Patch("/api/v1/reviews/{id}", reviewHandler.UpdateReview)
			r.Delete("/api/v1/reviews/{id}", reviewHandler.DeleteReview)
		})

		// Seller only
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireRole(model.RoleSeller))

			// Sellers
			r.Post("/api/v1/sellers", sellerHandler.CreateSeller)
			r.Patch("/api/v1/sellers/{id}", sellerHandler.UpdateSeller)
			r.Delete("/api/v1/sellers/{id}", sellerHandler.DeleteSeller)
			r.Get("/api/v1/sellers/me/orders", sellerHandler.GetSellerOrders)

			// Products (seller creates/updates/deletes)
			r.Post("/api/v1/products", productHandler.CreateProduct)
			r.Patch("/api/v1/products/{id}", productHandler.UpdateProduct)
			r.Delete("/api/v1/products/{id}", productHandler.DeleteProduct)

			// Orders (seller)
			r.Post("/api/v1/orders/{id}/ship", orderHandler.ShipOrder)
			r.Post("/api/v1/orders/{id}/deliver", orderHandler.DeliverOrder)
		})

		// Admin only
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireRole(model.RoleAdmin))

			r.Get("/api/v1/admin/users", adminHandler.GetUsers)
			r.Get("/api/v1/admin/users/{id}", adminHandler.GetUser)
			r.Patch("/api/v1/admin/users/{id}", adminHandler.UpdateUser)
			r.Delete("/api/v1/admin/users/{id}", adminHandler.DeleteUser)
			r.Delete("/api/v1/admin/sellers/{id}", adminHandler.DeleteSeller)

			r.Post("/api/v1/categories", categoryHandler.CreateCategory)
			r.Patch("/api/v1/categories/{id}", categoryHandler.UpdateCategory)
			r.Delete("/api/v1/categories/{id}", categoryHandler.DeleteCategory)
		})
	})

	return r
}
