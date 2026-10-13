//go:build integration

package service_test

import (
	"context"
	"errors"
	"testing"

	m "github.com/beastixq/marketplace/internal/model"
	s "github.com/beastixq/marketplace/internal/service"
)

func TestAddressService_CreateAddress_Integration(t *testing.T) {
	t.Run("owner and role boundaries", func(t *testing.T) {
		f := newIntegrationFixture(t)
		buyer, other, seller := f.actor(m.RoleBuyer), f.actor(m.RoleBuyer), f.actor(m.RoleSeller)
		id := f.address(buyer)
		ctx := context.Background()

		addresses, err := f.addresses.GetAddressesByUserID(ctx, buyer)
		mustIntegration(t, err)
		if len(addresses) != 1 || addresses[0].ID != id || addresses[0].UserID != buyer.UserID {
			t.Fatalf("addresses = %+v", addresses)
		}
		city := "Other city"
		if _, err := f.addresses.UpdateAddress(ctx, other, id, m.AddressUpdate{City: &city}); !errors.Is(err, s.ErrNotYourAddress) {
			t.Fatalf("ownership = %v", err)
		}
		if _, err := f.addresses.CreateAddress(ctx, seller, m.AddressCreate{}); !errors.Is(err, s.ErrPermissionDenied) {
			t.Fatalf("role = %v", err)
		}
	})
}
