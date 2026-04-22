package repository_test

import (
	"context"
	"errors"
	"testing"

	m "github.com/beastixq/marketplace/internal/model"
	repo "github.com/beastixq/marketplace/internal/repository"
	"github.com/beastixq/marketplace/internal/service"
	"github.com/shopspring/decimal"
)

var _ service.OrderItemRepo = repo.OrderItemRepoImpl{}

// createTestOrder создаёт draft-заказ для переданных buyer/seller, возвращает ID.
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
		_, _ = testPool.Exec(context.Background(), "DELETE FROM orders WHERE id = $1", id)
	})
	return id
}

func TestOrderItemRepo_CreateAndGet(t *testing.T) {
	buyerID := createTestUser(t)
	sellerID := createTestSeller(t)
	productID := createTestProduct(t, sellerID)
	orderID := createTestOrder(t, buyerID, sellerID)
	var r service.OrderItemRepo = repo.NewOrderItemRepo(testPool)
	ctx := context.Background()

	id, err := r.CreateOrderItem(ctx, m.OrderItemCreate{
		OrderID:         orderID,
		ProductID:       productID,
		Quantity:        3,
		PriceAtPurchase: decimal.NewFromFloat(50),
	})
	if err != nil {
		t.Fatalf("CreateOrderItem: %v", err)
	}
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), "DELETE FROM order_items WHERE id = $1", id)
	})

	got, err := r.GetOrderItemByID(ctx, id)
	if err != nil {
		t.Fatalf("GetOrderItemByID: %v", err)
	}
	if got.OrderID != orderID {
		t.Errorf("OrderID: got %d, want %d", got.OrderID, orderID)
	}
	if got.ProductID != productID {
		t.Errorf("ProductID: got %d, want %d", got.ProductID, productID)
	}
	if got.Quantity != 3 {
		t.Errorf("Quantity: got %d, want 3", got.Quantity)
	}
	if !got.PriceAtPurchase.Equal(decimal.NewFromFloat(50)) {
		t.Errorf("PriceAtPurchase: got %s, want 50", got.PriceAtPurchase)
	}
}

func TestOrderItemRepo_GetByOrderID(t *testing.T) {
	buyerID := createTestUser(t)
	sellerID := createTestSeller(t)
	productID := createTestProduct(t, sellerID)
	orderID := createTestOrder(t, buyerID, sellerID)
	var r service.OrderItemRepo = repo.NewOrderItemRepo(testPool)
	ctx := context.Background()

	before, err := r.GetOrderItemsByOrderID(ctx, orderID)
	if err != nil {
		t.Fatalf("GetOrderItemsByOrderID до создания: %v", err)
	}
	if len(before) != 0 {
		t.Fatalf("ожидается 0 позиций, получено %d", len(before))
	}

	id, err := r.CreateOrderItem(ctx, m.OrderItemCreate{
		OrderID: orderID, ProductID: productID, Quantity: 1,
		PriceAtPurchase: decimal.NewFromFloat(10),
	})
	if err != nil {
		t.Fatalf("CreateOrderItem: %v", err)
	}
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), "DELETE FROM order_items WHERE id = $1", id)
	})

	after, err := r.GetOrderItemsByOrderID(ctx, orderID)
	if err != nil {
		t.Fatalf("GetOrderItemsByOrderID: %v", err)
	}
	if len(after) != 1 {
		t.Errorf("ожидается 1 позиция, получено %d", len(after))
	}
}

func TestOrderItemRepo_Update(t *testing.T) {
	buyerID := createTestUser(t)
	sellerID := createTestSeller(t)
	productID := createTestProduct(t, sellerID)
	orderID := createTestOrder(t, buyerID, sellerID)
	var r service.OrderItemRepo = repo.NewOrderItemRepo(testPool)
	ctx := context.Background()

	id, err := r.CreateOrderItem(ctx, m.OrderItemCreate{
		OrderID: orderID, ProductID: productID, Quantity: 1,
		PriceAtPurchase: decimal.NewFromFloat(10),
	})
	if err != nil {
		t.Fatalf("CreateOrderItem: %v", err)
	}
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), "DELETE FROM order_items WHERE id = $1", id)
	})

	newQty := 7
	updated, err := r.UpdateOrderItem(ctx, id, m.OrderItemUpdate{Quantity: &newQty})
	if err != nil {
		t.Fatalf("UpdateOrderItem: %v", err)
	}
	if updated.Quantity != newQty {
		t.Errorf("Quantity: got %d, want %d", updated.Quantity, newQty)
	}

	fetched, err := r.GetOrderItemByID(ctx, id)
	if err != nil {
		t.Fatalf("GetOrderItemByID после Update: %v", err)
	}
	if fetched.Quantity != newQty {
		t.Errorf("Quantity в БД: got %d, want %d", fetched.Quantity, newQty)
	}
}

func TestOrderItemRepo_Update_NotFound(t *testing.T) {
	var r service.OrderItemRepo = repo.NewOrderItemRepo(testPool)
	ctx := context.Background()

	qty := 99
	_, err := r.UpdateOrderItem(ctx, 999999999, m.OrderItemUpdate{Quantity: &qty})
	if !errors.Is(err, service.ErrNotFound) {
		t.Errorf("ожидалась ErrNotFound, получено: %v", err)
	}
}

func TestOrderItemRepo_Delete(t *testing.T) {
	buyerID := createTestUser(t)
	sellerID := createTestSeller(t)
	productID := createTestProduct(t, sellerID)
	orderID := createTestOrder(t, buyerID, sellerID)
	var r service.OrderItemRepo = repo.NewOrderItemRepo(testPool)
	ctx := context.Background()

	id, err := r.CreateOrderItem(ctx, m.OrderItemCreate{
		OrderID: orderID, ProductID: productID, Quantity: 1,
		PriceAtPurchase: decimal.NewFromFloat(10),
	})
	if err != nil {
		t.Fatalf("CreateOrderItem: %v", err)
	}

	if err := r.DeleteOrderItemByID(ctx, id); err != nil {
		t.Fatalf("DeleteOrderItemByID: %v", err)
	}

	_, err = r.GetOrderItemByID(ctx, id)
	if !errors.Is(err, service.ErrNotFound) {
		t.Errorf("после удаления ожидалась ErrNotFound, получено: %v", err)
	}
}
