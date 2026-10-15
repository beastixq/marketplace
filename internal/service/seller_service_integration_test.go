//go:build integration

package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	m "github.com/beastixq/marketplace/internal/model"
	s "github.com/beastixq/marketplace/internal/service"
	"github.com/shopspring/decimal"
)

func (f *serviceIntegration) createSeller() (s.Actor, int64) {
	f.t.Helper()
	actor := f.createActor(m.RoleSeller)
	id, err := f.sellerService().CreateSeller(context.Background(), actor, m.SellerCreate{UserID: actor.UserID, CompanyName: "Integration seller"})
	mustIntegration(f.t, err)
	return actor, id
}

func TestSellerService_GetSellerStats_Integration(t *testing.T) {
	f := newServiceIntegration(t)
	ctx := context.Background()
	buyer := f.createActor(m.RoleBuyer)
	seller, sellerID := f.createSeller()
	productID := f.createProduct(seller, sellerID, 2)
	orderID := createPendingOrder(t, f.orderService(), buyer, productID, f.createAddress(buyer))
	mustIntegration(t, f.orderService().PayOrder(ctx, buyer, orderID))
	from := time.Now().UTC().Truncate(24 * time.Hour)
	to := from.AddDate(0, 0, 1)

	stats, err := f.sellerService().GetSellerStats(ctx, sellerID, from, to)
	mustIntegration(t, err)
	if stats.TotalOrders != 1 || !stats.TotalRevenue.Equal(decimal.NewFromInt(100)) || !stats.AvgOrderValue.Equal(decimal.NewFromInt(100)) || stats.TopProductName != "Integration product" {
		t.Fatalf("seller stats = %+v", stats)
	}
}

func TestSellerService_CreateSeller_Integration(t *testing.T) {
	t.Run("profile ownership and roles", func(t *testing.T) {
		f := newServiceIntegration(t)
		owner, id := f.createSeller()
		other, buyer := f.createActor(m.RoleSeller), f.createActor(m.RoleBuyer)
		ctx := context.Background()

		profile, err := f.sellerService().GetSellerByUserID(ctx, owner)
		mustIntegration(t, err)
		if profile.ID != id || profile.UserID != owner.UserID {
			t.Fatalf("profile = %+v", profile)
		}
		name := "Changed"
		if _, err := f.sellerService().UpdateSeller(ctx, other, id, m.SellerUpdate{CompanyName: &name}); !errors.Is(err, s.ErrNotYourSeller) {
			t.Fatalf("ownership = %v", err)
		}
		if _, err := f.sellerService().CreateSeller(ctx, buyer, m.SellerCreate{UserID: buyer.UserID, CompanyName: "Forbidden seller"}); !errors.Is(err, s.ErrPermissionDenied) {
			t.Fatalf("role = %v", err)
		}
	})
}
