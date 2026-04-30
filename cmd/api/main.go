package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	payment "github.com/beastixq/marketplace/internal/adapter/payment"
	"github.com/beastixq/marketplace/internal/config"
	"github.com/beastixq/marketplace/internal/handler"
	"github.com/beastixq/marketplace/internal/logging"
	"github.com/beastixq/marketplace/internal/middleware"
	repo "github.com/beastixq/marketplace/internal/repository"
	svc "github.com/beastixq/marketplace/internal/service"
	"github.com/beastixq/marketplace/internal/web"
)

func main() {
	configPath := flag.String("config", "config/config.yaml", "path to YAML config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		// Logger not yet initialized; fall back to stderr.
		fatal("load config", err)
	}

	logger, logCloser, err := logging.New(cfg.Logging)
	if err != nil {
		fatal("init logger", err)
	}
	defer logCloser.Close()
	slog.SetDefault(logger)

	logger.Info("marketplace api starting", "addr", cfg.Server.Addr, "log_level", cfg.Logging.Level)

	dbCtx, dbCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer dbCancel()
	pool, err := pgxpool.New(dbCtx, cfg.Database.DSN)
	if err != nil {
		logger.Error("connect database", "error", err)
		os.Exit(2)
	}
	defer pool.Close()
	if err = pool.Ping(dbCtx); err != nil {
		logger.Error("ping database", "error", err)
		os.Exit(2)
	}
	logger.Info("database connected")

	userRepo := repo.NewUserRepo(pool)
	sellerRepo := repo.NewSellerRepo(pool)
	addressRepo := repo.NewAddressRepo(pool)
	reviewRepo := repo.NewReviewRepo(pool)
	productRepo := repo.NewProductRepo(pool)
	orderRepo := repo.NewOrderRepo(pool)
	orderItemRepo := repo.NewOrderItemRepo(pool)
	categoryRepo := repo.NewCategoryRepo(pool)
	backofficeRepo := repo.NewBackofficeRepo(pool)
	txManager := repo.NewPgxTxManager(pool)

	userService := svc.NewUserService(userRepo, cfg.Auth.BcryptCost)
	sellerService := svc.NewSellerService(sellerRepo)
	addressService := svc.NewAddressService(addressRepo)
	reviewService := svc.NewReviewService(reviewRepo, reviewRepo, productRepo)
	productService := svc.NewProductService(productRepo, reviewRepo, sellerRepo)
	orderService := svc.NewOrderService(orderRepo, orderItemRepo, productRepo, addressRepo, sellerRepo, txManager)
	categoryService := svc.NewCategoryService(categoryRepo)
	backofficeService := svc.NewBackofficeService(backofficeRepo)
	// TODO: replace with Redis TokenBlocklist implementation
	authService := svc.NewAuthService(userService, nil, cfg.Auth.JWTSecret, cfg.Auth.JWTTTL.Std())

	paymentTTL := cfg.Payment.TTL.Std()
	gateway := payment.NewMockBankGateway(cfg.Payment.GatewayURL)
	paymentService := svc.NewPaymentService(orderRepo, gateway, paymentTTL)

	worker := svc.NewOrderExpirationWorker(
		orderService,
		cfg.Orders.ExpirationCheckInterval.Std(),
		paymentTTL,
		logger.With("component", "order-expiration-worker"),
	)
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
		logger.With("component", "http"),
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

	webHandler := web.NewWebHandler(productService, categoryService, authService, userService, orderService, addressService, sellerService, reviewService, backofficeService, paymentService)
	webRouter := web.NewWebRouter(webHandler)
	webLogger := logger.With("component", "web")
	webHandlerWithLogs := middleware.ActorHolder()(
		middleware.RequestLogger(webLogger)(
			middleware.Recoverer(webLogger)(webRouter),
		),
	)

	// API routes already include /api/v1/ prefix, so mount both at root.
	// chi matches the most specific route, so no conflicts between
	// /api/v1/* (API) and /* (web).
	mux := http.NewServeMux()
	mux.Handle("/api/", apiRouter)
	mux.Handle("/", webHandlerWithLogs)

	logger.Info("listening", "addr", cfg.Server.Addr)
	if err := http.ListenAndServe(cfg.Server.Addr, mux); err != nil {
		logger.Error("http server stopped", "error", err)
		os.Exit(3)
	}
}

func fatal(stage string, err error) {
	_, _ = os.Stderr.WriteString(stage + ": " + err.Error() + "\n")
	os.Exit(1)
}
