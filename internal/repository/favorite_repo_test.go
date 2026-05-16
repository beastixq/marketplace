package repository_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	m "github.com/beastixq/marketplace/internal/model"
	repo "github.com/beastixq/marketplace/internal/repository"
	"github.com/beastixq/marketplace/internal/service"
	"github.com/shopspring/decimal"
)

var _ service.FavoriteRepo = (*repo.FavoriteRepoImpl)(nil)

func createTestBuyer(t *testing.T) int64 {
	t.Helper()
	ctx := context.Background()
	userRepo := repo.NewUserRepo(testPool)

	userID, err := userRepo.CreateUser(ctx, m.UserCreate{
		Email:    fmt.Sprintf("buyer_%d@example.com", time.Now().UnixNano()),
		Password: "hashed_password",
		FullName: "Buyer User",
		Role:     m.RoleBuyer,
	})
	if err != nil {
		t.Fatalf("createTestBuyer: CreateUser: %v", err)
	}

	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), "DELETE FROM users WHERE id = $1", userID)
	})
	return userID
}

func createFavoriteTestProduct(t *testing.T, productRepo service.ProductRepo, name string) int64 {
	t.Helper()
	sellerID := createTestSeller(t)
	productID, err := productRepo.CreateProduct(context.Background(), m.ProductCreate{
		SellerID:      sellerID,
		Name:          name,
		Price:         decimal.NewFromInt(100),
		StockQuantity: 5,
	})
	if err != nil {
		t.Fatalf("CreateProduct: %v", err)
	}
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), "DELETE FROM product_price_history WHERE product_id = $1", productID)
		_, _ = testPool.Exec(context.Background(), "DELETE FROM products WHERE id = $1", productID)
	})
	return productID
}

func TestFavoriteRepo_AddFavorite(t *testing.T) {
	ctx := context.Background()
	userID := createTestBuyer(t)
	productRepo := repo.NewProductRepo(testPool)
	productID := createFavoriteTestProduct(t, productRepo, "Favorite active product")
	favoriteRepo := repo.NewFavoriteRepo(testPool)

	created, err := favoriteRepo.AddFavorite(ctx, userID, productID)
	if err != nil {
		t.Fatalf("AddFavorite: %v", err)
	}
	if !created {
		t.Fatal("AddFavorite created: got false, want true")
	}

	created, err = favoriteRepo.AddFavorite(ctx, userID, productID)
	if err != nil {
		t.Fatalf("AddFavorite duplicate: %v", err)
	}
	if created {
		t.Fatal("AddFavorite duplicate created: got true, want false")
	}

	missingID := productID + 999999
	_, err = favoriteRepo.AddFavorite(ctx, userID, missingID)
	if !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("AddFavorite missing product: got %v, want %v", err, service.ErrNotFound)
	}
	isFavorite, err := favoriteRepo.IsFavorite(ctx, userID, missingID)
	if err != nil {
		t.Fatalf("IsFavorite missing product: %v", err)
	}
	if isFavorite {
		t.Fatal("missing product favorite row exists")
	}

	// Repo does not enforce product-visibility policy. Soft-deleting a product
	// does not remove the FK target, so AddFavorite succeeds. The service
	// layer is responsible for rejecting favorites of soft-deleted products
	// before calling the repo; list/state reads filter by deleted_at.
	deletedProductID := createFavoriteTestProduct(t, productRepo, "Favorite deleted product")
	if err = productRepo.DeleteProductByID(ctx, deletedProductID); err != nil {
		t.Fatalf("DeleteProductByID: %v", err)
	}
	created, err = favoriteRepo.AddFavorite(ctx, userID, deletedProductID)
	if err != nil {
		t.Fatalf("AddFavorite deleted product: %v", err)
	}
	if !created {
		t.Fatal("AddFavorite deleted product created: got false, want true")
	}
}

func TestFavoriteRepo_DeleteListAndCheck(t *testing.T) {
	ctx := context.Background()
	userID := createTestBuyer(t)
	productRepo := repo.NewProductRepo(testPool)
	favoriteRepo := repo.NewFavoriteRepo(testPool)
	firstID := createFavoriteTestProduct(t, productRepo, "Favorite older product")
	secondID := createFavoriteTestProduct(t, productRepo, "Favorite newer product")
	deletedID := createFavoriteTestProduct(t, productRepo, "Favorite hidden product")

	if _, err := favoriteRepo.AddFavorite(ctx, userID, firstID); err != nil {
		t.Fatalf("AddFavorite first: %v", err)
	}
	if _, err := favoriteRepo.AddFavorite(ctx, userID, secondID); err != nil {
		t.Fatalf("AddFavorite second: %v", err)
	}
	if _, err := favoriteRepo.AddFavorite(ctx, userID, deletedID); err != nil {
		t.Fatalf("AddFavorite deleted candidate: %v", err)
	}
	if _, err := testPool.Exec(ctx, "UPDATE product_favorites SET created_at = now() - interval '1 hour' WHERE user_id = $1 AND product_id = $2", userID, firstID); err != nil {
		t.Fatalf("update favorite created_at: %v", err)
	}
	if err := productRepo.DeleteProductByID(ctx, deletedID); err != nil {
		t.Fatalf("DeleteProductByID: %v", err)
	}

	products, err := favoriteRepo.ListFavoriteProductsByUserID(ctx, userID, m.PaginationOpts{})
	if err != nil {
		t.Fatalf("ListFavoriteProductsByUserID: %v", err)
	}
	if len(products) != 2 {
		t.Fatalf("products len: got %d, want 2", len(products))
	}
	if products[0].ID != secondID || products[1].ID != firstID {
		t.Fatalf("favorite order: got [%d %d], want [%d %d]", products[0].ID, products[1].ID, secondID, firstID)
	}
	for _, product := range products {
		if product.ID == deletedID {
			t.Fatal("deleted product returned in favorite list")
		}
	}

	isFavorite, err := favoriteRepo.IsFavorite(ctx, userID, secondID)
	if err != nil {
		t.Fatalf("IsFavorite: %v", err)
	}
	if !isFavorite {
		t.Fatal("IsFavorite: got false, want true")
	}

	if err = favoriteRepo.DeleteFavorite(ctx, userID, secondID); err != nil {
		t.Fatalf("DeleteFavorite: %v", err)
	}
	if err = favoriteRepo.DeleteFavorite(ctx, userID, secondID); err != nil {
		t.Fatalf("DeleteFavorite idempotent: %v", err)
	}
	isFavorite, err = favoriteRepo.IsFavorite(ctx, userID, secondID)
	if err != nil {
		t.Fatalf("IsFavorite after delete: %v", err)
	}
	if isFavorite {
		t.Fatal("IsFavorite after delete: got true, want false")
	}
}
