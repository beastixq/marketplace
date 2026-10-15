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
		f := newServiceIntegration(t)
		ctx := context.Background()
		buyer, other := f.createActor(m.RoleBuyer), f.createActor(m.RoleBuyer)
		seller, sellerID := f.createSeller()
		productID := f.createProduct(seller, sellerID, 1)
		orderID := createPendingOrder(t, f.orderService(), buyer, productID, f.createAddress(buyer))
		create := m.ReviewCreate{ProductID: productID, Rating: 5}

		if _, err := f.reviewService().CreateReview(ctx, buyer, create); !errors.Is(err, s.ErrReviewPurchaseRequired) {
			t.Fatalf("before delivery = %v", err)
		}
		mustIntegration(t, f.orderService().PayOrder(ctx, buyer, orderID))
		if _, err := f.reviewService().CreateReview(ctx, buyer, create); !errors.Is(err, s.ErrReviewPurchaseRequired) {
			t.Fatalf("paid order review = %v", err)
		}
		mustIntegration(t, f.orderService().ShipOrder(ctx, seller, orderID))
		if _, err := f.reviewService().CreateReview(ctx, buyer, create); !errors.Is(err, s.ErrReviewPurchaseRequired) {
			t.Fatalf("shipped order review = %v", err)
		}
		mustIntegration(t, f.orderService().DeliverOrder(ctx, seller, orderID))
		if _, err := f.reviewService().CreateReview(ctx, other, create); !errors.Is(err, s.ErrReviewPurchaseRequired) {
			t.Fatalf("other buyer = %v", err)
		}
		id, err := f.reviewService().CreateReview(ctx, buyer, create)
		mustIntegration(t, err)
		review, err := f.reviewService().GetReviewByID(ctx, id)
		mustIntegration(t, err)
		if review.UserID != buyer.UserID || review.ProductID != productID || review.Rating != 5 {
			t.Fatalf("review = %+v", review)
		}
		if _, err := f.reviewService().CreateReview(ctx, buyer, create); !errors.Is(err, s.ErrReviewAlreadyExists) {
			t.Fatalf("duplicate = %v", err)
		}
	})
}
