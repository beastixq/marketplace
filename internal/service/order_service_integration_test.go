//go:build integration

package service_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	m "github.com/beastixq/marketplace/internal/model"
	s "github.com/beastixq/marketplace/internal/service"
	"github.com/shopspring/decimal"
)

func createPendingOrder(t *testing.T, orders s.OrderService, buyer s.Actor, productID, addressID int64) int64 {
	t.Helper()
	ctx := context.Background()
	mustIntegration(t, orders.AddItemToCart(ctx, buyer, productID, 1))
	ids, err := orders.Checkout(ctx, buyer, addressID)
	mustIntegration(t, err)
	if len(ids) != 1 {
		t.Fatalf("pending orders = %v, want one", ids)
	}
	return ids[0]
}

func TestOrderService_Checkout_Integration(t *testing.T) {
	for _, sellers := range []int{1, 2} {
		name := "single seller"
		if sellers == 2 {
			name = "multiple sellers"
		}
		t.Run(name, func(t *testing.T) {
			f := newServiceIntegration(t)
			orders := f.orderService()
			ctx := context.Background()
			buyer := f.createActor(m.RoleBuyer)
			addressID := f.createAddress(buyer)
			var products []int64
			expectedSellers := make(map[int64]int64, sellers)
			for range sellers {
				actor, sellerID := f.createSeller()
				id := f.createProduct(actor, sellerID, 4)
				products = append(products, id)
				expectedSellers[id] = sellerID
				mustIntegration(t, orders.AddItemToCart(ctx, buyer, id, 1))
				if err := orders.AddItemToCart(ctx, buyer, id, 1); !errors.Is(err, s.ErrProductAlreadyInCart) {
					t.Fatalf("duplicate cart item = %v", err)
				}
				price := decimal.NewFromInt(150)
				changedBy := "integration-seller"
				_, err := f.productService().UpdateProduct(ctx, actor, id, m.ProductUpdate{Price: &price, ChangedBy: &changedBy})
				mustIntegration(t, err)
			}
			cart, err := orders.GetCart(ctx, buyer)
			mustIntegration(t, err)
			items, err := orders.GetOrderItemsByOrderID(ctx, buyer, cart.ID)
			mustIntegration(t, err)
			if len(items) != sellers {
				t.Fatalf("unique cart items = %d", len(items))
			}
			for _, item := range items {
				mustIntegration(t, orders.ChangeQuantityCartItem(ctx, buyer, item.ID, 2))
			}

			ids, err := orders.Checkout(ctx, buyer, addressID)
			mustIntegration(t, err)
			if len(ids) != sellers {
				t.Fatalf("split = %v", ids)
			}
			for _, id := range ids {
				order, err := orders.GetOrderByID(ctx, buyer, id)
				mustIntegration(t, err)
				if order.Status != m.StatusPending || order.UserID != buyer.UserID || !order.TotalAmount.Equal(decimal.NewFromInt(300)) {
					t.Fatalf("order = %+v", order)
				}
				if order.AddressID == nil || *order.AddressID != addressID {
					t.Fatalf("checkout address = %v, want %d", order.AddressID, addressID)
				}
				items, err := orders.GetOrderItemsByOrderID(ctx, buyer, id)
				mustIntegration(t, err)
				if len(items) != 1 || items[0].Quantity != 2 || !items[0].PriceAtPurchase.Equal(decimal.NewFromInt(150)) {
					t.Fatalf("purchase items = %+v", items)
				}
				wantSeller, exists := expectedSellers[items[0].ProductID]
				if !exists || order.SellerID == nil || *order.SellerID != wantSeller || items[0].OrderID != id {
					t.Fatalf("seller/product split: order=%+v, item=%+v, want seller=%d", order, items[0], wantSeller)
				}
				delete(expectedSellers, items[0].ProductID)
			}
			if len(expectedSellers) != 0 {
				t.Fatalf("products missing from checkout: %v", expectedSellers)
			}
			for _, id := range products {
				assertPersistedStock(t, f.pool, id, 4, 2)
			}
			var drafts int
			mustIntegration(t, f.pool.QueryRow(ctx, "SELECT count(*) FROM orders WHERE user_id=$1 AND status='draft'", buyer.UserID).Scan(&drafts))
			if drafts != 0 {
				t.Fatalf("drafts after checkout = %d", drafts)
			}
		})
	}
	t.Run("foreign address and partial reservation rollback", func(t *testing.T) {
		f := newServiceIntegration(t)
		orders := f.orderService()
		ctx := context.Background()
		buyer, other := f.createActor(m.RoleBuyer), f.createActor(m.RoleBuyer)
		ownAddress, foreignAddress := f.createAddress(buyer), f.createAddress(other)
		actor, sellerID := f.createSeller()
		first, last := f.createProduct(actor, sellerID, 2), f.createProduct(actor, sellerID, 1)
		mustIntegration(t, orders.AddItemToCart(ctx, buyer, first, 1))
		mustIntegration(t, orders.AddItemToCart(ctx, buyer, last, 1))
		createPendingOrder(t, orders, other, last, foreignAddress)
		cartBefore, err := orders.GetCart(ctx, buyer)
		mustIntegration(t, err)

		if _, err := orders.Checkout(ctx, buyer, foreignAddress); !errors.Is(err, s.ErrNotYourAddress) {
			t.Fatalf("foreign address = %v", err)
		}
		if _, err := orders.Checkout(ctx, buyer, ownAddress); !errors.Is(err, s.ErrInsufficientStock) {
			t.Fatalf("reservation = %v", err)
		}
		assertPersistedStock(t, f.pool, first, 2, 0)
		assertPersistedStock(t, f.pool, last, 1, 1)
		cart, err := orders.GetCart(ctx, buyer)
		mustIntegration(t, err)
		if cart.Status != m.StatusDraft {
			t.Fatalf("rollback status = %s", cart.Status)
		}
		if cart.ID != cartBefore.ID || cart.AddressID != nil || cart.SellerID != nil {
			t.Fatalf("rollback cart metadata = %+v", cart)
		}
		items, err := orders.GetOrderItemsByOrderID(ctx, buyer, cart.ID)
		mustIntegration(t, err)
		if len(items) != 2 {
			t.Fatalf("cart items after rollback = %+v", items)
		}
		for _, item := range items {
			if item.Quantity != 1 || !item.PriceAtPurchase.Equal(decimal.NewFromInt(100)) || (item.ProductID != first && item.ProductID != last) {
				t.Fatalf("cart item escaped rollback: %+v", item)
			}
		}
		var pending int
		mustIntegration(t, f.pool.QueryRow(ctx, "SELECT count(*) FROM orders WHERE user_id=$1 AND status='pending'", buyer.UserID).Scan(&pending))
		if pending != 0 {
			t.Fatalf("pending orders escaped rollback = %d", pending)
		}
	})
}

func TestOrderService_Lifecycle_Integration(t *testing.T) {
	for _, action := range []string{"cancel", "expire", "deliver"} {
		t.Run(action, func(t *testing.T) {
			f := newServiceIntegration(t)
			orders := f.orderService()
			ctx := context.Background()
			buyer := f.createActor(m.RoleBuyer)
			seller, sellerID := f.createSeller()
			productID := f.createProduct(seller, sellerID, 3)
			orderID := createPendingOrder(t, orders, buyer, productID, f.createAddress(buyer))
			assertPersistedStock(t, f.pool, productID, 3, 1)

			switch action {
			case "cancel":
				mustIntegration(t, orders.CancelOrder(ctx, buyer, orderID))
				assertPersistedStock(t, f.pool, productID, 3, 0)
			case "expire":
				mustIntegration(t, orders.ExpireOrders(ctx, time.Now().Add(time.Hour)))
				assertPersistedStock(t, f.pool, productID, 3, 0)
			case "deliver":
				mustIntegration(t, orders.PayOrder(ctx, buyer, orderID))
				mustIntegration(t, orders.ShipOrder(ctx, seller, orderID))
				assertPersistedStock(t, f.pool, productID, 2, 0)
				mustIntegration(t, orders.DeliverOrder(ctx, seller, orderID))
				if err := orders.CancelOrder(ctx, buyer, orderID); !errors.Is(err, s.ErrOrderStatusInvalid) {
					t.Fatalf("cancel delivered = %v", err)
				}
			}
			order, err := orders.GetOrderByID(ctx, buyer, orderID)
			mustIntegration(t, err)
			want := m.StatusCancelled
			if action == "deliver" {
				want = m.StatusDelivered
			}
			if order.Status != want {
				t.Fatalf("status = %s, want %s", order.Status, want)
			}
		})
	}
}

func TestOrderService_ConcurrentCheckout_Integration(t *testing.T) {
	t.Run("two buyers last unit no overselling", func(t *testing.T) {
		f := newServiceIntegration(t)
		orders := f.orderService()
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		seller, sellerID := f.createSeller()
		productID := f.createProduct(seller, sellerID, 1)
		buyers := []s.Actor{f.createActor(m.RoleBuyer), f.createActor(m.RoleBuyer)}
		addresses := []int64{f.createAddress(buyers[0]), f.createAddress(buyers[1])}
		for _, buyer := range buyers {
			mustIntegration(t, orders.AddItemToCart(ctx, buyer, productID, 1))
		}
		start := make(chan struct{})
		outcomes := make(chan error, 2)
		var wg sync.WaitGroup
		for index, buyer := range buyers {
			wg.Go(func() { <-start; _, err := orders.Checkout(ctx, buyer, addresses[index]); outcomes <- err })
		}

		close(start)
		wg.Wait()
		close(outcomes)
		passed, rejected := 0, 0
		for err := range outcomes {
			if err == nil {
				passed++
			} else if errors.Is(err, s.ErrInsufficientStock) {
				rejected++
			} else {
				t.Fatalf("concurrent checkout = %v", err)
			}
		}
		if passed != 1 || rejected != 1 {
			t.Fatalf("outcomes = %d successes, %d rejected", passed, rejected)
		}
		assertPersistedStock(t, f.pool, productID, 1, 1)
	})
}
