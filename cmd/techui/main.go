package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	payment "github.com/beastixq/marketplace/internal/adapter/payment"
	repo "github.com/beastixq/marketplace/internal/repository"
	svc "github.com/beastixq/marketplace/internal/service"
	"github.com/beastixq/marketplace/internal/techui"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	ctx := context.Background()

	dbURL, ok := os.LookupEnv("DATABASE_URL")
	if !ok {
		fmt.Fprintln(os.Stderr, "DATABASE_URL is required")
		os.Exit(1)
	}

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "connect database: %v\n", err)
		os.Exit(2)
	}
	defer pool.Close()

	userRepo := repo.NewUserRepo(pool)
	sellerRepo := repo.NewSellerRepo(pool)
	addressRepo := repo.NewAddressRepo(pool)
	reviewRepo := repo.NewReviewRepo(pool)
	productRepo := repo.NewProductRepo(pool)
	orderRepo := repo.NewOrderRepo(pool)
	orderItemRepo := repo.NewOrderItemRepo(pool)
	categoryRepo := repo.NewCategoryRepo(pool)
	txManager := repo.NewPgxTxManager(pool)

	userService := svc.NewUserService(userRepo, 10)
	sellerService := svc.NewSellerService(sellerRepo)
	addressService := svc.NewAddressService(addressRepo)
	reviewService := svc.NewReviewService(reviewRepo)
	productService := svc.NewProductService(productRepo, reviewRepo, sellerRepo)
	orderService := svc.NewOrderService(orderRepo, orderItemRepo, productRepo, productRepo, sellerRepo, txManager)
	categoryService := svc.NewCategoryService(categoryRepo)
	authService := svc.NewAuthService(userService, nil, "TODO_SECRET", 24*time.Hour)
	paymentService := svc.NewPaymentService(orderRepo, payment.NewMockBankGateway("http://localhost:8080"), 15*time.Minute)

	app := techui.New(techui.ServicePorts{
		Auth:               authService,
		UserProfile:        userService,
		UserAdministration: userService,
		SellerProfile:      sellerService,
		SellerStatistics:   sellerService,
		ProductCatalog:     productService,
		ProductDetails:     productService,
		ProductManagement:  productService,
		CategoryBrowser:    categoryService,
		CategoryManagement: categoryService,
		AddressBook:        addressService,
		Cart:               orderService,
		BuyerOrders:        orderService,
		SellerOrders:       orderService,
		Payments:           paymentService,
		ReviewManagement:   reviewService,
	}, os.Stdin, os.Stdout)

	if err := app.Run(ctx); err != nil && !errors.Is(err, io.EOF) {
		fmt.Fprintf(os.Stderr, "techui failed: %v\n", err)
		os.Exit(3)
	}
}
