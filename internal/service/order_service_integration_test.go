//go:build integration

package service_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	m "github.com/beastixq/marketplace/internal/model"
	repo "github.com/beastixq/marketplace/internal/repository"
	s "github.com/beastixq/marketplace/internal/service"
	"github.com/shopspring/decimal"
)

func TestOrderService_Checkout_Integration(t *testing.T) {
	for _, sellers := range []int{1, 2} {
		name := "single seller"
		if sellers == 2 {
			name = "multiple sellers"
		}
		t.Run(name, func(t *testing.T) {
			f := newIntegrationFixture(t)
			ctx := context.Background()
			buyer := f.actor(m.RoleBuyer)
			addressID := f.address(buyer)
			var products []int64
			expectedSellers := make(map[int64]int64, sellers)
			for range sellers {
				actor, sellerID := f.seller()
				id := f.product(actor, sellerID, 4)
				products = append(products, id)
				expectedSellers[id] = sellerID
				mustIntegration(t, f.orders.AddItemToCart(ctx, buyer, id, 1))
				if err := f.orders.AddItemToCart(ctx, buyer, id, 1); !errors.Is(err, s.ErrProductAlreadyInCart) {
					t.Fatalf("duplicate cart item = %v", err)
				}
				price := decimal.NewFromInt(150)
				changedBy := "integration-seller"
				_, err := f.products.UpdateProduct(ctx, actor, id, m.ProductUpdate{Price: &price, ChangedBy: &changedBy})
				mustIntegration(t, err)
			}
			cart, err := f.orders.GetCart(ctx, buyer)
			mustIntegration(t, err)
			items, err := f.orders.GetOrderItemsByOrderID(ctx, buyer, cart.ID)
			mustIntegration(t, err)
			if len(items) != sellers {
				t.Fatalf("unique cart items = %d", len(items))
			}
			for _, item := range items {
				mustIntegration(t, f.orders.ChangeQuantityCartItem(ctx, buyer, item.ID, 2))
			}

			ids, err := f.orders.Checkout(ctx, buyer, addressID)
			mustIntegration(t, err)
			if len(ids) != sellers {
				t.Fatalf("split = %v", ids)
			}
			for _, id := range ids {
				order, err := f.orders.GetOrderByID(ctx, buyer, id)
				mustIntegration(t, err)
				if order.Status != m.StatusPending || order.UserID != buyer.UserID || !order.TotalAmount.Equal(decimal.NewFromInt(300)) {
					t.Fatalf("order = %+v", order)
				}
				if order.AddressID == nil || *order.AddressID != addressID {
					t.Fatalf("checkout address = %v, want %d", order.AddressID, addressID)
				}
				items, err := f.orders.GetOrderItemsByOrderID(ctx, buyer, id)
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
				f.stock(id, 4, 2)
			}
			var drafts int
			mustIntegration(t, f.pool.QueryRow(ctx, "SELECT count(*) FROM orders WHERE user_id=$1 AND status='draft'", buyer.UserID).Scan(&drafts))
			if drafts != 0 {
				t.Fatalf("drafts after checkout = %d", drafts)
			}
		})
	}
	t.Run("foreign address and partial reservation rollback", func(t *testing.T) {
		f := newIntegrationFixture(t)
		ctx := context.Background()
		buyer, other := f.actor(m.RoleBuyer), f.actor(m.RoleBuyer)
		ownAddress, foreignAddress := f.address(buyer), f.address(other)
		actor, sellerID := f.seller()
		first, last := f.product(actor, sellerID, 2), f.product(actor, sellerID, 1)
		mustIntegration(t, f.orders.AddItemToCart(ctx, buyer, first, 1))
		mustIntegration(t, f.orders.AddItemToCart(ctx, buyer, last, 1))
		mustIntegration(t, repo.NewProductRepo(f.pool).ChangeStockAndReserved(ctx, last, 0, 1))

		if _, err := f.orders.Checkout(ctx, buyer, foreignAddress); !errors.Is(err, s.ErrNotYourAddress) {
			t.Fatalf("foreign address = %v", err)
		}
		if _, err := f.orders.Checkout(ctx, buyer, ownAddress); !errors.Is(err, s.ErrInsufficientStock) {
			t.Fatalf("reservation = %v", err)
		}
		f.stock(first, 2, 0)
		f.stock(last, 1, 1)
		cart, err := f.orders.GetCart(ctx, buyer)
		mustIntegration(t, err)
		if cart.Status != m.StatusDraft {
			t.Fatalf("rollback status = %s", cart.Status)
		}
	})
}

func TestOrderService_Lifecycle_Integration(t *testing.T) {
	for _, action := range []string{"cancel", "expire", "deliver"} {
		t.Run(action, func(t *testing.T) {
			f := newIntegrationFixture(t)
			ctx := context.Background()
			buyer := f.actor(m.RoleBuyer)
			seller, sellerID := f.seller()
			productID := f.product(seller, sellerID, 3)
			orderID := f.checkout(buyer, productID, f.address(buyer))
			f.stock(productID, 3, 1)

			switch action {
			case "cancel":
				mustIntegration(t, f.orders.CancelOrder(ctx, buyer, orderID))
				f.stock(productID, 3, 0)
			case "expire":
				mustIntegration(t, f.orders.ExpireOrders(ctx, time.Now().Add(time.Hour)))
				f.stock(productID, 3, 0)
			case "deliver":
				mustIntegration(t, f.orders.PayOrder(ctx, buyer, orderID))
				mustIntegration(t, f.orders.ShipOrder(ctx, seller, orderID))
				f.stock(productID, 2, 0)
				mustIntegration(t, f.orders.DeliverOrder(ctx, seller, orderID))
				if err := f.orders.CancelOrder(ctx, buyer, orderID); !errors.Is(err, s.ErrOrderStatusInvalid) {
					t.Fatalf("cancel delivered = %v", err)
				}
			}
			order, err := f.orders.GetOrderByID(ctx, buyer, orderID)
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
		f := newIntegrationFixture(t)
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		seller, sellerID := f.seller()
		productID := f.product(seller, sellerID, 1)
		buyers := []s.Actor{f.actor(m.RoleBuyer), f.actor(m.RoleBuyer)}
		addresses := []int64{f.address(buyers[0]), f.address(buyers[1])}
		for _, buyer := range buyers {
			mustIntegration(t, f.orders.AddItemToCart(ctx, buyer, productID, 1))
		}
		start := make(chan struct{})
		outcomes := make(chan error, 2)
		var wg sync.WaitGroup
		for index, buyer := range buyers {
			wg.Go(func() { <-start; _, err := f.orders.Checkout(ctx, buyer, addresses[index]); outcomes <- err })
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
		f.stock(productID, 1, 1)
	})
}
