package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/beastixq/marketplace/internal/cache"
	repocomponent "github.com/beastixq/marketplace/internal/component/repository"
	servicecomponent "github.com/beastixq/marketplace/internal/component/service"
	"github.com/beastixq/marketplace/internal/config"
	"github.com/beastixq/marketplace/internal/handler"
	"github.com/beastixq/marketplace/internal/logging"
	"github.com/beastixq/marketplace/internal/middleware"
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

	// Redis (optional cache layer)
	var rdb *redis.Client
	if cfg.Redis.Enabled {
		var err error
		rdb, err = cache.NewRedisClient(context.Background(), cfg.Redis)
		if err != nil {
			logger.Error("connect redis", "error", err)
			os.Exit(2)
		}
		defer rdb.Close()
		logger.Info("redis connected", "addr", cfg.Redis.Addr)
	}

	var cacheCfg *repocomponent.CacheConfig
	if rdb != nil {
		cacheCfg = &repocomponent.CacheConfig{Client: rdb, ProductTTL: cfg.Redis.ProductTTL.Std()}
	}

	dbCtx, dbCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer dbCancel()
	repositories, err := repocomponent.NewFromConfig(dbCtx, cfg.Database, cacheCfg)
	if err != nil {
		logger.Error("connect database", "type", cfg.Database.Type, "error", err)
		os.Exit(2)
	}
	defer repositories.Close()
	logger.Info("database connected", "type", cfg.Database.Type)

	services := servicecomponent.New(repositories, servicecomponent.Config{
		BcryptCost:            cfg.Auth.BcryptCost,
		JWTSecret:             cfg.Auth.JWTSecret,
		TokenTTL:              cfg.Auth.JWTTTL.Std(),
		PaymentTTL:            cfg.Payment.TTL.Std(),
		PaymentGatewayBaseURL: cfg.Payment.GatewayURL,
	})

	paymentTTL := cfg.Payment.TTL.Std()
	worker := svc.NewOrderExpirationWorker(
		services.Order,
		cfg.Orders.ExpirationCheckInterval.Std(),
		paymentTTL,
		logger.With("component", "order-expiration-worker"),
	)
	go worker.Run(context.Background())

	authHandler := handler.NewAuthHandler(services.Auth)
	userHandler := handler.NewUserHandler(services.User)
	sellerHandler := handler.NewSellerHandler(services.Seller, services.Order)
	addressHandler := handler.NewAddressHandler(services.Address)
	productHandler := handler.NewProductHandler(services.Product)
	favoriteHandler := handler.NewFavoriteHandler(services.Favorite)
	orderHandler := handler.NewOrderHandler(services.Order)
	paymentHandler := handler.NewPaymentHandler(services.Payment)
	categoryHandler := handler.NewCategoryHandler(services.Category)
	reviewHandler := handler.NewReviewHandler(services.Review)
	adminHandler := handler.NewAdminHandler(services.User, services.Seller)

	apiRouter := handler.NewRouter(
		logger.With("component", "http"),
		services.Auth,
		authHandler,
		userHandler,
		sellerHandler,
		addressHandler,
		productHandler,
		favoriteHandler,
		orderHandler,
		paymentHandler,
		categoryHandler,
		reviewHandler,
		adminHandler,
	)

	webHandler := web.NewWebHandler(services.Product, services.Category, services.Auth, services.User, services.Order, services.Address, services.Seller, services.Review, services.Backoffice, services.Payment, services.Favorite)
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
