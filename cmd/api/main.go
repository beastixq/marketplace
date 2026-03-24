package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/beastixq/marketplace/internal/handler"
	repo "github.com/beastixq/marketplace/internal/repository"
	svc "github.com/beastixq/marketplace/internal/service"
	"github.com/beastixq/marketplace/internal/web"
)

func main() {
	fmt.Println("Marketplace server starting")

	dbURL, ok := os.LookupEnv("DATABASE_URL")
	if !ok {
		os.Exit(1)
	}
	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		os.Exit(2)
	}

	userRepo := repo.NewUserRepo(pool)
	sellerRepo := repo.NewSellerRepo(pool)
	addressRepo := repo.NewAddressRepo(pool)
	reviewRepo := repo.NewReviewRepo(pool)
	productRepo := repo.NewProductRepo(pool)
	orderRepo := repo.NewOrderRepo(pool)
	orderItemRepo := repo.NewOrderItemRepo(pool)
	categoryRepo := repo.NewCategoryRepo(pool)

	userService := svc.NewUserService(userRepo, 10)
	sellerService := svc.NewSellerService(sellerRepo)
	addressService := svc.NewAddressService(addressRepo)
	reviewService := svc.NewReviewService(reviewRepo)
	productService := svc.NewProductService(productRepo, reviewRepo, sellerRepo)
	orderService := svc.NewOrderService(orderRepo, orderItemRepo, productRepo, sellerRepo)
	categoryService := svc.NewCategoryService(categoryRepo)
	// TODO: replace with Redis TokenBlocklist implementation
	authService := svc.NewAuthService(userService, nil, "TODO_SECRET", 24*time.Hour)

	authHandler := handler.NewAuthHandler(authService)
	userHandler := handler.NewUserHandler(userService)
	sellerHandler := handler.NewSellerHandler(sellerService)
	addressHandler := handler.NewAddressHandler(addressService)
	productHandler := handler.NewProductHandler(productService)
	orderHandler := handler.NewOrderHandler(orderService)
	categoryHandler := handler.NewCategoryHandler(categoryService)
	reviewHandler := handler.NewReviewHandler(reviewService)
	adminHandler := handler.NewAdminHandler(userService, sellerService)

	apiRouter := handler.NewRouter(
		authService,
		authHandler,
		userHandler,
		sellerHandler,
		addressHandler,
		productHandler,
		orderHandler,
		categoryHandler,
		reviewHandler,
		adminHandler,
	)

	webHandler := web.NewWebHandler(productService, categoryService, authService, userService, orderService, addressService)
	webRouter := web.NewWebRouter(webHandler)

	// API routes already include /api/v1/ prefix, so mount both at root.
	// chi matches the most specific route, so no conflicts between
	// /api/v1/* (API) and /* (web).
	mux := http.NewServeMux()
	mux.Handle("/api/", apiRouter)
	mux.Handle("/", webRouter)

	fmt.Println("Listening on :8080")
	http.ListenAndServe(":8080", mux)
}
