package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	repocomponent "github.com/beastixq/marketplace/internal/component/repository"
	servicecomponent "github.com/beastixq/marketplace/internal/component/service"
	techuicomponent "github.com/beastixq/marketplace/internal/component/techui"
)

func main() {
	ctx := context.Background()

	dbURL, ok := os.LookupEnv("DATABASE_URL")
	if !ok {
		fmt.Fprintln(os.Stderr, "DATABASE_URL is required")
		os.Exit(1)
	}

	repositories, err := repocomponent.New(ctx, dbURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "connect database: %v\n", err)
		os.Exit(2)
	}
	defer repositories.Close()

	services := servicecomponent.New(repositories, servicecomponent.Config{
		JWTSecret: os.Getenv("JWT_SECRET"),
	})
	app := techuicomponent.New(services, os.Stdin, os.Stdout)

	if err := app.Run(ctx); err != nil && !errors.Is(err, io.EOF) {
		fmt.Fprintf(os.Stderr, "techui failed: %v\n", err)
		os.Exit(3)
	}
}
