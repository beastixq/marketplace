//go:build integration

package service_test

import (
	"context"
	"errors"
	"testing"

	m "github.com/beastixq/marketplace/internal/model"
	s "github.com/beastixq/marketplace/internal/service"
)

func TestReviewService_CreateReview_Integration(t *testing.T) {
	t.Run("delivered owner only and duplicate rejected", func(t *testing.T) {
		f := newIntegrationFixture(t)
		ctx := context.Background()
		buyer, other := f.actor(m.RoleBuyer), f.actor(m.RoleBuyer)
		seller, sellerID := f.seller()
		productID := f.product(seller, sellerID, 1)
		orderID := f.checkout(buyer, productID, f.address(buyer))
		create := m.ReviewCreate{ProductID: productID, Rating: 5}

		if _, err := f.reviews.CreateReview(ctx, buyer, create); !errors.Is(err, s.ErrReviewPurchaseRequired) {
			t.Fatalf("before delivery = %v", err)
		}
		mustIntegration(t, f.orders.PayOrder(ctx, buyer, orderID))
		mustIntegration(t, f.orders.ShipOrder(ctx, seller, orderID))
		mustIntegration(t, f.orders.DeliverOrder(ctx, seller, orderID))
		if _, err := f.reviews.CreateReview(ctx, other, create); !errors.Is(err, s.ErrReviewPurchaseRequired) {
			t.Fatalf("other buyer = %v", err)
		}
		id, err := f.reviews.CreateReview(ctx, buyer, create)
		mustIntegration(t, err)
		review, err := f.reviews.GetReviewByID(ctx, id)
		mustIntegration(t, err)
		if review.UserID != buyer.UserID || review.ProductID != productID || review.Rating != 5 {
			t.Fatalf("review = %+v", review)
		}
		if _, err := f.reviews.CreateReview(ctx, buyer, create); !errors.Is(err, s.ErrReviewAlreadyExists) {
			t.Fatalf("duplicate = %v", err)
		}
	})
}
