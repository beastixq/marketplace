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

var _ service.ProductRepo = repo.ProductRepoImpl{}
var _ service.ProductCategoryRepo = repo.ProductRepoImpl{}

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

	fetched, err := r.GetProductByID(ctx, id)
	if err != nil {
		t.Fatalf("GetProductByID после Update: %v", err)
	}
	if fetched.Name != newName {
		t.Errorf("Name в БД: got %q, want %q", fetched.Name, newName)
	}
	if fetched.StockQuantity != newQty {
		t.Errorf("StockQuantity в БД: got %d, want %d", fetched.StockQuantity, newQty)
	}
}

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

func TestProductRepo_GetProducts_Limit(t *testing.T) {
	sellerID := createTestSeller(t)
	var r service.ProductRepo = repo.NewProductRepo(testPool)
	ctx := context.Background()

	ids := make([]int64, 3)
	for i := range 3 {
		id, err := r.CreateProduct(ctx, m.ProductCreate{
			SellerID: sellerID, Name: fmt.Sprintf("lim_%d_%d", time.Now().UnixNano(), i),
			Price: decimal.NewFromFloat(10), StockQuantity: 1,
		})
		if err != nil {
			t.Fatalf("CreateProduct %d: %v", i, err)
		}
		ids[i] = id
	}
	t.Cleanup(func() {
		for _, id := range ids {
			_, _ = testPool.Exec(context.Background(), "DELETE FROM products WHERE id = $1", id)
		}
	})

	limit := 2
	got, err := r.GetProducts(ctx, m.CatalogOptions{
		SellerID:   &sellerID,
		Pagination: &m.PaginationOpts{Page: 1, Limit: limit},
	})
	if err != nil {
		t.Fatalf("GetProducts: %v", err)
	}
	if len(got) != limit {
		t.Errorf("ожидается ровно %d, получено %d", limit, len(got))
	}
	for _, p := range got {
		if p.ID == ids[2] {
			t.Errorf("третий товар id=%d не должен попасть в страницу с limit=2", ids[2])
		}
	}
}

func TestProductRepo_Update_NotFound(t *testing.T) {
	var r service.ProductRepo = repo.NewProductRepo(testPool)
	ctx := context.Background()

	name := "ghost"
	_, err := r.UpdateProduct(ctx, 999999999, m.ProductUpdate{Name: &name})
	if !errors.Is(err, service.ErrNotFound) {
		t.Errorf("ожидалась ErrNotFound, получено: %v", err)
	}
}

// Soft delete: DeletedAt nil→non-nil, запись остаётся в БД но уходит из каталога.
func TestProductRepo_Delete(t *testing.T) {
	sellerID := createTestSeller(t)
	var r service.ProductRepo = repo.NewProductRepo(testPool)
	ctx := context.Background()

	id, err := r.CreateProduct(ctx, m.ProductCreate{
		SellerID:      sellerID,
		Name:          "To Delete",
		Price:         decimal.NewFromFloat(100),
		StockQuantity: 1,
	})
	if err != nil {
		t.Fatalf("CreateProduct: %v", err)
	}
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), "DELETE FROM product_price_history WHERE product_id = $1", id)
		_, _ = testPool.Exec(context.Background(), "DELETE FROM products WHERE id = $1", id)
	})

	// Pre-condition.
	before, err := r.GetProductByID(ctx, id)
	if err != nil {
		t.Fatalf("GetProductByID до удаления: %v", err)
	}
	if before.DeletedAt != nil {
		t.Fatal("некорректные данные теста: DeletedAt не nil до удаления")
	}

	// Act.
	if err := r.DeleteProductByID(ctx, id); err != nil {
		t.Fatalf("DeleteProductByID: %v", err)
	}

	// Post-condition: DeletedAt установлен.
	after, err := r.GetProductByID(ctx, id)
	if err != nil {
		t.Fatalf("GetProductByID после удаления: %v", err)
	}
	if after.DeletedAt == nil {
		t.Error("DeletedAt должен быть установлен после DeleteProductByID")
	}

	// Post-condition: товар не попадает в выборку каталога.
	products, err := r.GetProducts(ctx, m.CatalogOptions{SellerID: &sellerID})
	if err != nil {
		t.Fatalf("GetProducts: %v", err)
	}
	for _, p := range products {
		if p.ID == id {
			t.Errorf("удалённый товар id=%d присутствует в каталоге", id)
		}
	}
}

func TestProductRepo_PriceHistory(t *testing.T) {
	sellerID := createTestSeller(t)
	var r service.ProductRepo = repo.NewProductRepo(testPool)
	ctx := context.Background()

	id, err := r.CreateProduct(ctx, m.ProductCreate{
		SellerID:      sellerID,
		Name:          "Price History Test",
		Price:         decimal.NewFromFloat(100),
		StockQuantity: 1,
	})
	if err != nil {
		t.Fatalf("CreateProduct: %v", err)
	}
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), "DELETE FROM product_price_history WHERE product_id = $1", id)
		_, _ = testPool.Exec(context.Background(), "DELETE FROM products WHERE id = $1", id)
	})

	// Pre-condition: истории нет.
	from := time.Now().Add(-time.Hour)
	to := time.Now().Add(time.Hour)
	historyBefore, err := r.GetProductPriceHistory(ctx, id, from, to)
	if err != nil {
		t.Fatalf("GetProductPriceHistory до обновления цены: %v", err)
	}
	if len(historyBefore) != 0 {
		t.Fatalf("история пуста до изменения цены, получено %d записей", len(historyBefore))
	}

	// Act: меняем цену с ChangedBy (триггер запишет в product_price_history).
	newPrice := decimal.NewFromFloat(250)
	changedBy := "test-actor"
	if _, err := r.UpdateProduct(ctx, id, m.ProductUpdate{
		Price:     &newPrice,
		ChangedBy: &changedBy,
	}); err != nil {
		t.Fatalf("UpdateProduct: %v", err)
	}

	// Post-condition: в истории одна запись с правильными ценами.
	historyAfter, err := r.GetProductPriceHistory(ctx, id, from, to)
	if err != nil {
		t.Fatalf("GetProductPriceHistory после обновления цены: %v", err)
	}
	if len(historyAfter) != 1 {
		t.Fatalf("ожидается 1 запись истории, получено %d", len(historyAfter))
	}
	h := historyAfter[0]
	if !h.OldPrice.Equal(decimal.NewFromFloat(100)) {
		t.Errorf("OldPrice: got %s, want 100", h.OldPrice)
	}
	if !h.NewPrice.Equal(newPrice) {
		t.Errorf("NewPrice: got %s, want %s", h.NewPrice, newPrice)
	}
	if h.ChangedBy != changedBy {
		t.Errorf("ChangedBy: got %q, want %q", h.ChangedBy, changedBy)
	}
}
