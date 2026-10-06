package repository_test

import (
	"fmt"
	"os"
	"testing"
)

// One setup symbol supports gobco, which type-checks tagged test files together.
var setupIntegrationDatabase = func() (func(), error) {
	return func() {}, nil
}

func TestMain(m *testing.M) {
	cleanup, err := setupIntegrationDatabase()
	if err != nil {
		fmt.Fprintf(os.Stderr, "initialize integration database: %v\n", err)
		os.Exit(1)
	}
	code := m.Run()
	cleanup()
	os.Exit(code)
}
