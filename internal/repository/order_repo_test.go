package repository_test

import (
	"context"
	"errors"
	"testing"
	"time"

	m "github.com/beastixq/marketplace/internal/model"
	repo "github.com/beastixq/marketplace/internal/repository"
	"github.com/beastixq/marketplace/internal/service"
	"github.com/shopspring/decimal"
)

var _ service.OrderRepo = repo.OrderRepoImpl{}

func TestOrderRepo_CreateAndGet(t *testing.T) {
	buyerID := createTestUser(t)
	sellerID := createTestSeller(t)
	var r service.OrderRepo = repo.NewOrderRepo(testPool)
	ctx := context.Background()

	id, err := r.CreateOrder(ctx, m.OrderCreate{
		UserID:      buyerID,
		SellerID:    &sellerID,
		Status:      m.StatusDraft,
		TotalAmount: decimal.NewFromFloat(199.99),
	})
	if err != nil {
		t.Fatalf("CreateOrder: %v", err)
	}
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), "DELETE FROM orders WHERE id = $1", id)
	})

	got, err := r.GetOrderByID(ctx, id)
	if err != nil {
		t.Fatalf("GetOrderByID: %v", err)
	}
	if got.ID != id {
		t.Errorf("ID: got %d, want %d", got.ID, id)
	}
	if got.UserID != buyerID {
		t.Errorf("UserID: got %d, want %d", got.UserID, buyerID)
	}
	if got.Status != m.StatusDraft {
		t.Errorf("Status: got %q, want %q", got.Status, m.StatusDraft)
	}
	if !got.TotalAmount.Equal(decimal.NewFromFloat(199.99)) {
		t.Errorf("TotalAmount: got %s, want 199.99", got.TotalAmount)
	}
}

func TestOrderRepo_GetByUserID(t *testing.T) {
	buyerID := createTestUser(t)
	sellerID := createTestSeller(t)
	var r service.OrderRepo = repo.NewOrderRepo(testPool)
	ctx := context.Background()

	before, err := r.GetOrdersByUserID(ctx, buyerID, m.PaginationOpts{Page: 1, Limit: 10})
	if err != nil {
		t.Fatalf("GetOrdersByUserID до создания: %v", err)
	}
	if len(before) != 0 {
		t.Fatalf("ожидается 0 заказов, получено %d", len(before))
	}

	id, err := r.CreateOrder(ctx, m.OrderCreate{
		UserID: buyerID, SellerID: &sellerID,
		Status: m.StatusPending, TotalAmount: decimal.NewFromFloat(50),
	})
	if err != nil {
		t.Fatalf("CreateOrder: %v", err)
	}
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), "DELETE FROM orders WHERE id = $1", id)
	})

	after, err := r.GetOrdersByUserID(ctx, buyerID, m.PaginationOpts{Page: 1, Limit: 10})
	if err != nil {
		t.Fatalf("GetOrdersByUserID: %v", err)
	}
	if len(after) != 1 {
		t.Errorf("ожидается 1 заказ, получено %d", len(after))
	}
}

func TestOrderRepo_GetSellerOrders(t *testing.T) {
	buyerID := createTestUser(t)
	sellerID := createTestSeller(t)
	var r service.OrderRepo = repo.NewOrderRepo(testPool)
	ctx := context.Background()

	id, err := r.CreateOrder(ctx, m.OrderCreate{
		UserID: buyerID, SellerID: &sellerID,
		Status: m.StatusPaid, TotalAmount: decimal.NewFromFloat(75),
	})
	if err != nil {
		t.Fatalf("CreateOrder: %v", err)
	}
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), "DELETE FROM orders WHERE id = $1", id)
	})

	orders, err := r.GetSellerOrdersBySellerID(ctx, sellerID, m.PaginationOpts{Page: 1, Limit: 10})
	if err != nil {
		t.Fatalf("GetSellerOrdersBySellerID: %v", err)
	}
	if len(orders) != 1 {
		t.Errorf("ожидается 1 заказ, получено %d", len(orders))
	}
	if len(orders) > 0 && (orders[0].SellerID == nil || *orders[0].SellerID != sellerID) {
		t.Errorf("заказ с чужим SellerID: %v", orders[0].SellerID)
	}
}

func TestOrderRepo_Update(t *testing.T) {
	buyerID := createTestUser(t)
	sellerID := createTestSeller(t)
	var r service.OrderRepo = repo.NewOrderRepo(testPool)
	ctx := context.Background()

	id, err := r.CreateOrder(ctx, m.OrderCreate{
		UserID: buyerID, SellerID: &sellerID,
		Status: m.StatusPending, TotalAmount: decimal.NewFromFloat(100),
	})
	if err != nil {
		t.Fatalf("CreateOrder: %v", err)
	}
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), "DELETE FROM orders WHERE id = $1", id)
	})

	newStatus := m.StatusShipped
	updated, err := r.UpdateOrder(ctx, id, m.OrderUpdate{Status: &newStatus})
	if err != nil {
		t.Fatalf("UpdateOrder: %v", err)
	}
	if updated.Status != newStatus {
		t.Errorf("Status: got %q, want %q", updated.Status, newStatus)
	}

	fetched, err := r.GetOrderByID(ctx, id)
	if err != nil {
		t.Fatalf("GetOrderByID после Update: %v", err)
	}
	if fetched.Status != newStatus {
		t.Errorf("Status в БД: got %q, want %q", fetched.Status, newStatus)
	}
}

func TestOrderRepo_Update_NotFound(t *testing.T) {
	var r service.OrderRepo = repo.NewOrderRepo(testPool)
	ctx := context.Background()

	status := m.StatusShipped
	_, err := r.UpdateOrder(ctx, 999999999, m.OrderUpdate{Status: &status})
	if !errors.Is(err, service.ErrNotFound) {
		t.Errorf("ожидалась ErrNotFound, получено: %v", err)
	}
}

func TestOrderRepo_Delete(t *testing.T) {
	buyerID := createTestUser(t)
	sellerID := createTestSeller(t)
	var r service.OrderRepo = repo.NewOrderRepo(testPool)
	ctx := context.Background()

	id, err := r.CreateOrder(ctx, m.OrderCreate{
		UserID: buyerID, SellerID: &sellerID,
		Status: m.StatusDraft, TotalAmount: decimal.NewFromFloat(10),
	})
	if err != nil {
		t.Fatalf("CreateOrder: %v", err)
	}

	if err := r.DeleteOrderByID(ctx, id); err != nil {
		t.Fatalf("DeleteOrderByID: %v", err)
	}

	_, err = r.GetOrderByID(ctx, id)
	if !errors.Is(err, service.ErrNotFound) {
		t.Errorf("после удаления ожидалась ErrNotFound, получено: %v", err)
	}
}

func TestOrderRepo_GetExpiredPendingOrders(t *testing.T) {
	buyerID := createTestUser(t)
	sellerID := createTestSeller(t)
	var r service.OrderRepo = repo.NewOrderRepo(testPool)
	ctx := context.Background()

	id, err := r.CreateOrder(ctx, m.OrderCreate{
		UserID: buyerID, SellerID: &sellerID,
		Status: m.StatusPending, TotalAmount: decimal.NewFromFloat(1),
	})
	if err != nil {
		t.Fatalf("CreateOrder: %v", err)
	}
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), "DELETE FROM orders WHERE id = $1", id)
	})

	// deadline в будущем → наш pending заказ попадает в результат
	orders, err := r.GetExpiredPendingOrders(ctx, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("GetExpiredPendingOrders: %v", err)
	}
	found := false
	for _, o := range orders {
		if o.ID == id {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("order %d not found in expired pending orders", id)
	}
}
