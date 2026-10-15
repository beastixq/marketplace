//go:build integration

package service_test

import (
	"context"
	"errors"
	"testing"

	m "github.com/beastixq/marketplace/internal/model"
	s "github.com/beastixq/marketplace/internal/service"
)

func (f *serviceIntegration) createAddress(actor s.Actor) int64 {
	f.t.Helper()
	id, err := f.addressService().CreateAddress(context.Background(), actor, validIntegrationAddress(actor.UserID))
	mustIntegration(f.t, err)
	return id
}

func validIntegrationAddress(userID int64) m.AddressCreate {
	return m.AddressCreate{UserID: userID, City: "Test city", Street: "Test street", House: "1", ZipCode: "123456"}
}

func TestAddressService_CreateAddress_Integration(t *testing.T) {
	t.Run("owner and role boundaries", func(t *testing.T) {
		f := newServiceIntegration(t)
		buyer, other, seller := f.createActor(m.RoleBuyer), f.createActor(m.RoleBuyer), f.createActor(m.RoleSeller)
		id := f.createAddress(buyer)
		ctx := context.Background()

		addresses, err := f.addressService().GetAddressesByUserID(ctx, buyer)
		mustIntegration(t, err)
		if len(addresses) != 1 || addresses[0].ID != id || addresses[0].UserID != buyer.UserID {
			t.Fatalf("addresses = %+v", addresses)
		}
		city := "Other city"
		if _, err := f.addressService().UpdateAddress(ctx, other, id, m.AddressUpdate{City: &city}); !errors.Is(err, s.ErrNotYourAddress) {
			t.Fatalf("ownership = %v", err)
		}
		if _, err := f.addressService().CreateAddress(ctx, seller, validIntegrationAddress(seller.UserID)); !errors.Is(err, s.ErrPermissionDenied) {
			t.Fatalf("role = %v", err)
		}
	})
}
