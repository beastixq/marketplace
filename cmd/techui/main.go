package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	repocomponent "github.com/beastixq/marketplace/internal/component/repository"
	servicecomponent "github.com/beastixq/marketplace/internal/component/service"
	techuicomponent "github.com/beastixq/marketplace/internal/component/techui"
	"github.com/beastixq/marketplace/internal/config"
)

func main() {
	configPath := flag.String("config", "config/config.yaml", "path to YAML config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()

	repositories, err := repocomponent.New(ctx, cfg.Database.DSN)
	if err != nil {
		fmt.Fprintf(os.Stderr, "connect database: %v\n", err)
		os.Exit(2)
	}
	defer repositories.Close()

	services := servicecomponent.New(repositories, servicecomponent.Config{
		BcryptCost:            cfg.Auth.BcryptCost,
		JWTSecret:             cfg.Auth.JWTSecret,
		TokenTTL:              cfg.Auth.JWTTTL.Std(),
		PaymentTTL:            cfg.Payment.TTL.Std(),
		PaymentGatewayBaseURL: cfg.Payment.GatewayURL,
	})
	app := techuicomponent.New(services, os.Stdin, os.Stdout)

	if err := app.Run(ctx); err != nil && !errors.Is(err, io.EOF) {
		fmt.Fprintf(os.Stderr, "techui failed: %v\n", err)
		os.Exit(3)
	}
}
