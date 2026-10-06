//go:build integration

package repository_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	m "github.com/beastixq/marketplace/internal/model"
	repo "github.com/beastixq/marketplace/internal/repository"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

var testPool *pgxpool.Pool

func cleanupExec(t *testing.T, query string, args ...any) {
	t.Helper()
	if _, err := testPool.Exec(context.Background(), query, args...); err != nil {
		t.Errorf("clean integration fixture: %v", err)
	}
}

func init() {
	setupIntegrationDatabase = setupPostgresIntegrationDatabase
}

func setupPostgresIntegrationDatabase() (func(), error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return func() {}, nil
	}
	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		return nil, err
	}
	testPool = pool
	return pool.Close, nil
}

func requireIntegrationDatabase(t *testing.T) {
	t.Helper()
	if testPool == nil {
		t.Skip("set DATABASE_URL to run PostgreSQL integration tests")
	}
}

func createTestCategory(t *testing.T, create m.CategoryCreate) int64 {
	t.Helper()
	id, err := repo.NewCategoryRepo(testPool).CreateCategory(context.Background(), create)
	if err != nil {
		t.Fatalf("create category fixture: %v", err)
	}
	cleanupCategory(t, id)
	return id
}

func cleanupCategory(t *testing.T, id int64) {
	t.Helper()
	t.Cleanup(func() {
		_, err := testPool.Exec(context.Background(), "DELETE FROM categories WHERE id = $1", id)
		if err != nil {
			t.Errorf("delete category fixture: %v", err)
		}
	})
}

func createTestOrder(t *testing.T, buyerID, sellerID int64) int64 {
	t.Helper()
	ctx := context.Background()
	r := repo.NewOrderRepo(testPool)
	id, err := r.CreateOrder(ctx, m.OrderCreate{
		UserID: buyerID, SellerID: &sellerID,
		Status: m.StatusDraft, TotalAmount: decimal.NewFromFloat(0),
	})
	if err != nil {
		t.Fatalf("createTestOrder: %v", err)
	}
	t.Cleanup(func() {
		cleanupExec(t, "DELETE FROM orders WHERE id = $1", id)
	})
	return id
}

func createTestSeller(t *testing.T) int64 {
	t.Helper()
	ctx := context.Background()
	userRepo := repo.NewUserRepo(testPool)
	sellerRepo := repo.NewSellerRepo(testPool)

	userID, err := userRepo.CreateUser(ctx, m.UserCreate{
		Email:    fmt.Sprintf("seller_%d@example.com", time.Now().UnixNano()),
		Password: "hashed_password",
		FullName: "Seller User",
		Role:     m.RoleSeller,
	})
	if err != nil {
		t.Fatalf("createTestSeller: CreateUser: %v", err)
	}

	sellerID, err := sellerRepo.CreateSeller(ctx, m.SellerCreate{
		UserID:      userID,
		CompanyName: "Test Company LLC",
	})
	if err != nil {
		cleanupExec(t, "DELETE FROM users WHERE id = $1", userID)
		t.Fatalf("createTestSeller: CreateSeller: %v", err)
	}

	t.Cleanup(func() {
		cleanupExec(t, "DELETE FROM sellers WHERE id = $1", sellerID)
		cleanupExec(t, "DELETE FROM users WHERE id = $1", userID)
	})
	return sellerID
}

func createTestProduct(t *testing.T, sellerID int64) int64 {
	t.Helper()
	ctx := context.Background()
	r := repo.NewProductRepo(testPool)
	id, err := r.CreateProduct(ctx, m.ProductCreate{
		SellerID:      sellerID,
		Name:          "Review Test Product",
		Price:         decimal.NewFromFloat(100),
		StockQuantity: 1,
	})
	if err != nil {
		t.Fatalf("createTestProduct: %v", err)
	}
	t.Cleanup(func() {
		cleanupExec(t, "DELETE FROM product_price_history WHERE product_id = $1", id)
		cleanupExec(t, "DELETE FROM products WHERE id = $1", id)
	})
	return id
}

func createSellerUser(t *testing.T) int64 {
	t.Helper()
	ctx := context.Background()
	r := repo.NewUserRepo(testPool)
	id, err := r.CreateUser(ctx, m.UserCreate{
		Email:    fmt.Sprintf("selleruser_%d@example.com", time.Now().UnixNano()),
		Password: "hashed_password",
		FullName: "Seller Owner",
		Role:     m.RoleSeller,
	})
	if err != nil {
		t.Fatalf("createSellerUser: %v", err)
	}
	t.Cleanup(func() {
		cleanupExec(t, "DELETE FROM users WHERE id = $1", id)
	})
	return id
}

func createTestUser(t *testing.T) int64 {
	t.Helper()
	ctx := context.Background()
	r := repo.NewUserRepo(testPool)

	id, err := r.CreateUser(ctx, m.UserCreate{
		Email:    fmt.Sprintf("testuser_%d@example.com", time.Now().UnixNano()),
		Password: "hashed_password",
		FullName: "Test User",
		Role:     m.RoleBuyer,
	})
	if err != nil {
		t.Fatalf("createTestUser: %v", err)
	}
	t.Cleanup(func() {
		cleanupExec(t, "DELETE FROM users WHERE id = $1", id)
	})
	return id
}
