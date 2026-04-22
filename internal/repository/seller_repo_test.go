package repository_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	m "github.com/beastixq/marketplace/internal/model"
	repo "github.com/beastixq/marketplace/internal/repository"
	"github.com/beastixq/marketplace/internal/service"
)

var _ service.SellerRepo = repo.SellerRepoImpl{}

func createSellerUser(t *testing.T) int64 {
	t.Helper()
	ctx := context.Background()
	r := repo.NewUserRepo(testPool)
	id, err := r.CreateUser(ctx, m.UserCreate{
		Email:    fmt.Sprintf("selleruser_%d@example.com", time.Now().UnixNano()),
		Password: "hashed_password",
		FullName: "Seller Owner",
		Role:     m.RoleSeller,
	})
	if err != nil {
		t.Fatalf("createSellerUser: %v", err)
	}
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), "DELETE FROM users WHERE id = $1", id)
	})
	return id
}

func TestSellerRepo_CreateAndGet(t *testing.T) {
	userID := createSellerUser(t)
	var r service.SellerRepo = repo.NewSellerRepo(testPool)
	ctx := context.Background()

	_, err := r.GetSellerByUserID(ctx, userID)
	if !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("до создания GetSellerByUserID должен вернуть ErrNotFound, получено: %v", err)
	}

	companyName := "Acme Ltd"
	id, err := r.CreateSeller(ctx, m.SellerCreate{
		UserID:      userID,
		CompanyName: companyName,
	})
	if err != nil {
		t.Fatalf("CreateSeller: %v", err)
	}
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), "DELETE FROM sellers WHERE id = $1", id)
	})

	got, err := r.GetSellerByID(ctx, id)
	if err != nil {
		t.Fatalf("GetSellerByID: %v", err)
	}
	if got.ID != id {
		t.Errorf("ID: got %d, want %d", got.ID, id)
	}
	if got.UserID != userID {
		t.Errorf("UserID: got %d, want %d", got.UserID, userID)
	}
	if got.CompanyName != companyName {
		t.Errorf("CompanyName: got %q, want %q", got.CompanyName, companyName)
	}
}

func TestSellerRepo_GetByUserID(t *testing.T) {
	userID := createSellerUser(t)
	var r service.SellerRepo = repo.NewSellerRepo(testPool)
	ctx := context.Background()

	id, err := r.CreateSeller(ctx, m.SellerCreate{
		UserID:      userID,
		CompanyName: "Lookup Test",
	})
	if err != nil {
		t.Fatalf("CreateSeller: %v", err)
	}
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), "DELETE FROM sellers WHERE id = $1", id)
	})

	got, err := r.GetSellerByUserID(ctx, userID)
	if err != nil {
		t.Fatalf("GetSellerByUserID: %v", err)
	}
	if got.ID != id {
		t.Errorf("ID: got %d, want %d", got.ID, id)
	}
}

func TestSellerRepo_Update(t *testing.T) {
	userID := createSellerUser(t)
	var r service.SellerRepo = repo.NewSellerRepo(testPool)
	ctx := context.Background()

	id, err := r.CreateSeller(ctx, m.SellerCreate{
		UserID:      userID,
		CompanyName: "Old Company",
	})
	if err != nil {
		t.Fatalf("CreateSeller: %v", err)
	}
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), "DELETE FROM sellers WHERE id = $1", id)
	})

	newName := "New Company"
	newDesc := "New description"
	updated, err := r.UpdateSeller(ctx, id, m.SellerUpdate{
		CompanyName: &newName,
		Description: &newDesc,
	})
	if err != nil {
		t.Fatalf("UpdateSeller: %v", err)
	}
	if updated.CompanyName != newName {
		t.Errorf("CompanyName: got %q, want %q", updated.CompanyName, newName)
	}
	if updated.Description == nil || *updated.Description != newDesc {
		t.Errorf("Description: got %v, want %q", updated.Description, newDesc)
	}

	fetched, err := r.GetSellerByID(ctx, id)
	if err != nil {
		t.Fatalf("GetSellerByID после Update: %v", err)
	}
	if fetched.CompanyName != newName {
		t.Errorf("CompanyName в БД: got %q, want %q", fetched.CompanyName, newName)
	}
	if fetched.Description == nil || *fetched.Description != newDesc {
		t.Errorf("Description в БД: got %v, want %q", fetched.Description, newDesc)
	}
}

func TestSellerRepo_Update_NotFound(t *testing.T) {
	var r service.SellerRepo = repo.NewSellerRepo(testPool)
	ctx := context.Background()

	name := "ghost"
	_, err := r.UpdateSeller(ctx, 999999999, m.SellerUpdate{CompanyName: &name})
	if !errors.Is(err, service.ErrNotFound) {
		t.Errorf("ожидалась ErrNotFound, получено: %v", err)
	}
}

func TestSellerRepo_Delete(t *testing.T) {
	userID := createSellerUser(t)
	var r service.SellerRepo = repo.NewSellerRepo(testPool)
	ctx := context.Background()

	id, err := r.CreateSeller(ctx, m.SellerCreate{
		UserID:      userID,
		CompanyName: "To Delete",
	})
	if err != nil {
		t.Fatalf("CreateSeller: %v", err)
	}

	if err := r.DeleteSellerByID(ctx, id); err != nil {
		t.Fatalf("DeleteSellerByID: %v", err)
	}

	// Post-condition: жёсткое удаление, запись отсутствует.
	_, err = r.GetSellerByID(ctx, id)
	if !errors.Is(err, service.ErrNotFound) {
		t.Errorf("после удаления ожидалась ErrNotFound, получено: %v", err)
	}
}
