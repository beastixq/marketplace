package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	payment "github.com/beastixq/marketplace/internal/adapter/payment"
	"github.com/beastixq/marketplace/internal/config"
	"github.com/beastixq/marketplace/internal/handler"
	repo "github.com/beastixq/marketplace/internal/repository"
	svc "github.com/beastixq/marketplace/internal/service"
	"github.com/beastixq/marketplace/internal/web"
)

func main() {
	configPath := flag.String("config", "config/config.yaml", "path to YAML config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	fmt.Println("Marketplace server starting")

	pool, err := pgxpool.New(context.Background(), cfg.Database.DSN)
	if err != nil {
		log.Fatalf("connect database: %v", err)
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

	userService := svc.NewUserService(userRepo, cfg.Auth.BcryptCost)
	sellerService := svc.NewSellerService(sellerRepo)
	addressService := svc.NewAddressService(addressRepo)
	reviewService := svc.NewReviewService(reviewRepo)
	productService := svc.NewProductService(productRepo, reviewRepo, sellerRepo)
	orderService := svc.NewOrderService(orderRepo, orderItemRepo, productRepo, sellerRepo, txManager)
	categoryService := svc.NewCategoryService(categoryRepo)
	// TODO: replace with Redis TokenBlocklist implementation
	authService := svc.NewAuthService(userService, nil, cfg.Auth.JWTSecret, cfg.Auth.JWTTTL.Std())

	paymentTTL := cfg.Payment.TTL.Std()
	gateway := payment.NewMockBankGateway(cfg.Payment.GatewayURL)
	paymentService := svc.NewPaymentService(orderRepo, gateway, paymentTTL)

	logger := slog.Default()
	worker := svc.NewOrderExpirationWorker(orderService, cfg.Orders.ExpirationCheckInterval.Std(), paymentTTL, logger)
	go worker.Run(context.Background())

	authHandler := handler.NewAuthHandler(authService)
	userHandler := handler.NewUserHandler(userService)
	sellerHandler := handler.NewSellerHandler(sellerService, orderService)
	addressHandler := handler.NewAddressHandler(addressService)
	productHandler := handler.NewProductHandler(productService)
	orderHandler := handler.NewOrderHandler(orderService)
	paymentHandler := handler.NewPaymentHandler(paymentService)
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
		paymentHandler,
		categoryHandler,
		reviewHandler,
		adminHandler,
	)

	webHandler := web.NewWebHandler(productService, categoryService, authService, userService, orderService, addressService, sellerService, reviewService, pool)
	webRouter := web.NewWebRouter(webHandler)

	// API routes already include /api/v1/ prefix, so mount both at root.
	// chi matches the most specific route, so no conflicts between
	// /api/v1/* (API) and /* (web).
	mux := http.NewServeMux()
	mux.Handle("/api/", apiRouter)
	mux.Handle("/", webRouter)

	fmt.Printf("Listening on %s\n", cfg.Server.Addr)
	if err := http.ListenAndServe(cfg.Server.Addr, mux); err != nil {
		log.Fatalf("http server: %v", err)
	}
}
