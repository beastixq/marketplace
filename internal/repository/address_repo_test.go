package repository_test

import (
	"context"
	"errors"
	"testing"

	m "github.com/beastixq/marketplace/internal/model"
	repo "github.com/beastixq/marketplace/internal/repository"
	"github.com/beastixq/marketplace/internal/service"
)

var _ service.AddressRepo = repo.AddressRepoImpl{}

func TestAddressRepo_CreateAndGet(t *testing.T) {
	userID := createTestUser(t)
	var r service.AddressRepo = repo.NewAddressRepo(testPool)
	ctx := context.Background()

	id, err := r.CreateAddress(ctx, m.AddressCreate{
		UserID:    userID,
		City:      "Moscow",
		Street:    "Lenina 1",
		House:     "10",
		ZipCode:   "101000",
		IsDefault: true,
	})
	if err != nil {
		t.Fatalf("CreateAddress: %v", err)
	}
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), "DELETE FROM addresses WHERE id = $1", id)
	})

	got, err := r.GetAddressByID(ctx, id)
	if err != nil {
		t.Fatalf("GetAddressByID: %v", err)
	}
	if got.ID != id {
		t.Errorf("ID: got %d, want %d", got.ID, id)
	}
	if got.City != "Moscow" {
		t.Errorf("City: got %q, want Moscow", got.City)
	}
	if got.House != "10" {
		t.Errorf("House: got %q, want 10", got.House)
	}
	if !got.IsDefault {
		t.Error("IsDefault: got false, want true")
	}
}

func TestAddressRepo_GetByUserID(t *testing.T) {
	userID := createTestUser(t)
	var r service.AddressRepo = repo.NewAddressRepo(testPool)
	ctx := context.Background()

	// Pre-condition: у пользователя адресов нет.
	before, err := r.GetAddressesByUserID(ctx, userID)
	if err != nil {
		t.Fatalf("GetAddressesByUserID до создания: %v", err)
	}
	if len(before) != 0 {
		t.Fatalf("ожидается 0 адресов, получено %d", len(before))
	}

	id1, err := r.CreateAddress(ctx, m.AddressCreate{
		UserID: userID, City: "SPb", Street: "Nevsky", House: "10", ZipCode: "190000",
	})
	if err != nil {
		t.Fatalf("CreateAddress 1: %v", err)
	}
	id2, err := r.CreateAddress(ctx, m.AddressCreate{
		UserID: userID, City: "Kazan", Street: "Bauman", House: "5", ZipCode: "420000",
	})
	if err != nil {
		t.Fatalf("CreateAddress 2: %v", err)
	}
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), "DELETE FROM addresses WHERE id = $1", id1)
		_, _ = testPool.Exec(context.Background(), "DELETE FROM addresses WHERE id = $1", id2)
	})

	after, err := r.GetAddressesByUserID(ctx, userID)
	if err != nil {
		t.Fatalf("GetAddressesByUserID: %v", err)
	}
	if len(after) != 2 {
		t.Errorf("ожидается 2 адреса, получено %d", len(after))
	}
}

func TestAddressRepo_Update(t *testing.T) {
	userID := createTestUser(t)
	var r service.AddressRepo = repo.NewAddressRepo(testPool)
	ctx := context.Background()

	id, err := r.CreateAddress(ctx, m.AddressCreate{
		UserID: userID, City: "Old", Street: "Old St", House: "1", ZipCode: "000000",
	})
	if err != nil {
		t.Fatalf("CreateAddress: %v", err)
	}
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), "DELETE FROM addresses WHERE id = $1", id)
	})

	newCity := "New City"
	updated, err := r.UpdateAddress(ctx, id, m.AddressUpdate{City: &newCity})
	if err != nil {
		t.Fatalf("UpdateAddress: %v", err)
	}
	if updated.City != newCity {
		t.Errorf("City: got %q, want %q", updated.City, newCity)
	}

	fetched, err := r.GetAddressByID(ctx, id)
	if err != nil {
		t.Fatalf("GetAddressByID после Update: %v", err)
	}
	if fetched.City != newCity {
		t.Errorf("City в БД: got %q, want %q", fetched.City, newCity)
	}
}

func TestAddressRepo_Update_NotFound(t *testing.T) {
	var r service.AddressRepo = repo.NewAddressRepo(testPool)
	ctx := context.Background()

	city := "ghost"
	_, err := r.UpdateAddress(ctx, 999999999, m.AddressUpdate{City: &city})
	if !errors.Is(err, service.ErrNotFound) {
		t.Errorf("ожидалась ErrNotFound, получено: %v", err)
	}
}

func TestAddressRepo_Delete(t *testing.T) {
	userID := createTestUser(t)
	var r service.AddressRepo = repo.NewAddressRepo(testPool)
	ctx := context.Background()

	id, err := r.CreateAddress(ctx, m.AddressCreate{
		UserID: userID, City: "X", Street: "Y", House: "1", ZipCode: "000000",
	})
	if err != nil {
		t.Fatalf("CreateAddress: %v", err)
	}

	if err := r.DeleteAddressByID(ctx, id); err != nil {
		t.Fatalf("DeleteAddressByID: %v", err)
	}

	_, err = r.GetAddressByID(ctx, id)
	if !errors.Is(err, service.ErrNotFound) {
		t.Errorf("после удаления ожидалась ErrNotFound, получено: %v", err)
	}
}
