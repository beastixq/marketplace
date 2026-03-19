package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	_ "github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/beastixq/marketplace/internal/handler"
	repo "github.com/beastixq/marketplace/internal/repository"
	svc "github.com/beastixq/marketplace/internal/service"
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

	userService := svc.NewUserService(userRepo)
	sellerService := svc.NewSellerService(sellerRepo)
	addressService := svc.NewAddressService(addressRepo)
	reviewService := svc.NewReviewService(reviewRepo)
	productService := svc.NewProductService(productRepo, reviewRepo, sellerRepo)
	orderService := svc.NewOrderService(orderRepo, orderItemRepo, productRepo, sellerRepo)
	categoryService := svc.NewCategoryService(categoryRepo)
	authService := svc.NewAuthService(userService, "TODO_SECRET", 24*time.Hour)

	authHandler := handler.NewAuthHandler(authService)
	userHandler := handler.NewUserHandler(userService)
	sellerHandler := handler.NewSellerHandler(sellerService)
	addressHandler := handler.NewAddressHandler(addressService)
	productHandler := handler.NewProductHandler(productService)
	orderHandler := handler.NewOrderHandler(orderService)
	categoryHandler := handler.NewCategoryHandler(categoryService)
	reviewHandler := handler.NewReviewHandler(reviewService)
	adminHandler := handler.NewAdminHandler(userService, sellerService)

	router := handler.NewRouter(
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

	http.ListenAndServe(":8080", router)
}
