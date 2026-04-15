package repository_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

var testPool *pgxpool.Pool

func TestMain(m *testing.M) {
	dbURL, ok := os.LookupEnv("DATABASE_URL")
	if !ok {
		fmt.Fprintln(os.Stderr, "DATABASE_URL not set, skipping repository integration tests")
		os.Exit(0)
	}

	var err error
	testPool, err = pgxpool.New(context.Background(), dbURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer testPool.Close()

	os.Exit(m.Run())
}
