package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"

	repocomponent "github.com/beastixq/marketplace/internal/component/repository"
	servicecomponent "github.com/beastixq/marketplace/internal/component/service"
	techuicomponent "github.com/beastixq/marketplace/internal/component/techui"
	"github.com/beastixq/marketplace/internal/config"
	"github.com/beastixq/marketplace/internal/logging"
	svc "github.com/beastixq/marketplace/internal/service"
)

func main() {
	configPath := flag.String("config", "config/config.yaml", "path to YAML config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}

	logger, logCloser, err := logging.New(cfg.Logging)
	if err != nil {
		fmt.Fprintf(os.Stderr, "init logger: %v\n", err)
		os.Exit(1)
	}
	defer logCloser.Close()
	slog.SetDefault(logger)

	logger.Info("techui starting")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	repositories, err := repocomponent.New(ctx, cfg.Database.DSN)
	if err != nil {
		logger.Error("connect database", "error", err)
		os.Exit(2)
	}
	defer repositories.Close()
	logger.Info("database connected")

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
	go worker.Run(ctx)

	app := techuicomponent.New(services, os.Stdin, os.Stdout, logger.With("component", "techui"))

	if err := app.Run(ctx); err != nil && !errors.Is(err, io.EOF) {
		logger.Error("techui failed", "error", err)
		os.Exit(3)
	}
	logger.Info("techui stopped")
}
