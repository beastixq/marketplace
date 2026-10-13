//go:build integration

package service_test

import (
	"context"
	"errors"
	"testing"

	m "github.com/beastixq/marketplace/internal/model"
	s "github.com/beastixq/marketplace/internal/service"
)

func TestSellerService_CreateSeller_Integration(t *testing.T) {
	t.Run("profile ownership and roles", func(t *testing.T) {
		f := newIntegrationFixture(t)
		owner, id := f.seller()
		other, buyer := f.actor(m.RoleSeller), f.actor(m.RoleBuyer)
		ctx := context.Background()

		profile, err := f.sellers.GetSellerByUserID(ctx, owner)
		mustIntegration(t, err)
		if profile.ID != id || profile.UserID != owner.UserID {
			t.Fatalf("profile = %+v", profile)
		}
		name := "Changed"
		if _, err := f.sellers.UpdateSeller(ctx, other, id, m.SellerUpdate{CompanyName: &name}); !errors.Is(err, s.ErrNotYourSeller) {
			t.Fatalf("ownership = %v", err)
		}
		if _, err := f.sellers.CreateSeller(ctx, buyer, m.SellerCreate{}); !errors.Is(err, s.ErrPermissionDenied) {
			t.Fatalf("role = %v", err)
		}
	})
}
