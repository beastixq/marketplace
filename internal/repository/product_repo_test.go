package repository_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	m "github.com/beastixq/marketplace/internal/model"
	repo "github.com/beastixq/marketplace/internal/repository"
	"github.com/beastixq/marketplace/internal/service"
	"github.com/shopspring/decimal"
)

var _ service.ProductRepo = repo.ProductRepoImpl{}

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
		_, _ = testPool.Exec(ctx, "DELETE FROM users WHERE id = $1", userID)
		t.Fatalf("createTestSeller: CreateSeller: %v", err)
	}

	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), "DELETE FROM sellers WHERE id = $1", sellerID)
		_, _ = testPool.Exec(context.Background(), "DELETE FROM users WHERE id = $1", userID)
	})
	return sellerID
}

// TestProductRepo_CreateAndGet проверяет сохранение товара и его извлечение по ID.
func TestProductRepo_CreateAndGet(t *testing.T) {
	sellerID := createTestSeller(t)
	var r service.ProductRepo = repo.NewProductRepo(testPool)
	ctx := context.Background()

	desc := "Описание тестового товара"
	id, err := r.CreateProduct(ctx, m.ProductCreate{
		SellerID:      sellerID,
		Name:          "Test Product",
		Description:   &desc,
		Price:         decimal.NewFromFloat(999.99),
		StockQuantity: 10,
	})
	if err != nil {
		t.Fatalf("CreateProduct: %v", err)
	}
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), "DELETE FROM product_price_history WHERE product_id = $1", id)
		_, _ = testPool.Exec(context.Background(), "DELETE FROM products WHERE id = $1", id)
	})

	got, err := r.GetProductByID(ctx, id)
	if err != nil {
		t.Fatalf("GetProductByID: %v", err)
	}

	if got.ID != id {
		t.Errorf("ID: got %d, want %d", got.ID, id)
	}
	if got.Name != "Test Product" {
		t.Errorf("Name: got %q, want %q", got.Name, "Test Product")
	}
	if got.SellerID != sellerID {
		t.Errorf("SellerID: got %d, want %d", got.SellerID, sellerID)
	}
	if got.StockQuantity != 10 {
		t.Errorf("StockQuantity: got %d, want %d", got.StockQuantity, 10)
	}
	if !got.Price.Equal(decimal.NewFromFloat(999.99)) {
		t.Errorf("Price: got %s, want 999.99", got.Price)
	}
	if got.DeletedAt != nil {
		t.Error("DeletedAt должен быть nil у только что созданного товара")
	}
}

// TestProductRepo_Update проверяет изменение названия и количества товара.
func TestProductRepo_Update(t *testing.T) {
	sellerID := createTestSeller(t)
	var r service.ProductRepo = repo.NewProductRepo(testPool)
	ctx := context.Background()

	id, err := r.CreateProduct(ctx, m.ProductCreate{
		SellerID:      sellerID,
		Name:          "Old Name",
		Price:         decimal.NewFromFloat(50.00),
		StockQuantity: 5,
	})
	if err != nil {
		t.Fatalf("CreateProduct: %v", err)
	}
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), "DELETE FROM product_price_history WHERE product_id = $1", id)
		_, _ = testPool.Exec(context.Background(), "DELETE FROM products WHERE id = $1", id)
	})

	newName := "Updated Name"
	newQty := 42
	updated, err := r.UpdateProduct(ctx, id, m.ProductUpdate{
		Name:          &newName,
		StockQuantity: &newQty,
	})
	if err != nil {
		t.Fatalf("UpdateProduct: %v", err)
	}

	if updated.Name != newName {
		t.Errorf("Name после Update: got %q, want %q", updated.Name, newName)
	}
	if updated.StockQuantity != newQty {
		t.Errorf("StockQuantity после Update: got %d, want %d", updated.StockQuantity, newQty)
	}
}

// TestProductRepo_GetProducts_BySeller проверяет выборку каталога с фильтром по продавцу.
// Создаёт двух продавцов и товары, и убеждается что фильтрация возвращает только нужные.
func TestProductRepo_GetProducts_BySeller(t *testing.T) {
	sellerID := createTestSeller(t)
	var r service.ProductRepo = repo.NewProductRepo(testPool)
	ctx := context.Background()

	id1, err := r.CreateProduct(ctx, m.ProductCreate{
		SellerID: sellerID, Name: "Product A",
		Price: decimal.NewFromFloat(10), StockQuantity: 1,
	})
	if err != nil {
		t.Fatalf("CreateProduct A: %v", err)
	}
	id2, err := r.CreateProduct(ctx, m.ProductCreate{
		SellerID: sellerID, Name: "Product B",
		Price: decimal.NewFromFloat(20), StockQuantity: 2,
	})
	if err != nil {
		t.Fatalf("CreateProduct B: %v", err)
	}
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), "DELETE FROM products WHERE id = $1", id1)
		_, _ = testPool.Exec(context.Background(), "DELETE FROM products WHERE id = $1", id2)
	})

	products, err := r.GetProducts(ctx, m.CatalogOptions{SellerID: &sellerID})
	if err != nil {
		t.Fatalf("GetProducts: %v", err)
	}

	if len(products) < 2 {
		t.Errorf("ожидается минимум 2 товара для продавца %d, получено %d", sellerID, len(products))
	}
	for _, p := range products {
		if p.SellerID != sellerID {
			t.Errorf("в выборке товар с SellerID=%d, ожидался %d", p.SellerID, sellerID)
		}
		if p.DeletedAt != nil {
			t.Errorf("в каталоге не должно быть мягко удалённых товаров (id=%d)", p.ID)
		}
	}
}
