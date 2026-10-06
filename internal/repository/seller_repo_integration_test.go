//go:build integration

package repository_test

import (
	"context"
	"errors"
	"testing"

	m "github.com/beastixq/marketplace/internal/model"
	repo "github.com/beastixq/marketplace/internal/repository"
	"github.com/beastixq/marketplace/internal/service"
)

var _ service.SellerRepo = (*repo.SellerRepoImpl)(nil)

func TestSellerRepo_GetSellerByID_Integration(t *testing.T) {
	requireIntegrationDatabase(t)
	t.Run("success", func(t *testing.T) {
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
			cleanupExec(t, "DELETE FROM sellers WHERE id = $1", id)
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
	})
}

func TestSellerRepo_GetSellerByUserID_Integration(t *testing.T) {
	requireIntegrationDatabase(t)
	t.Run("success", func(t *testing.T) {
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
			cleanupExec(t, "DELETE FROM sellers WHERE id = $1", id)
		})

		got, err := r.GetSellerByUserID(ctx, userID)
		if err != nil {
			t.Fatalf("GetSellerByUserID: %v", err)
		}
		if got.ID != id {
			t.Errorf("ID: got %d, want %d", got.ID, id)
		}
	})
}

func TestSellerRepo_UpdateSeller_Integration(t *testing.T) {
	requireIntegrationDatabase(t)
	t.Run("success", func(t *testing.T) {
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
			cleanupExec(t, "DELETE FROM sellers WHERE id = $1", id)
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
	})

	t.Run("not found", func(t *testing.T) {
		var r service.SellerRepo = repo.NewSellerRepo(testPool)
		ctx := context.Background()

		name := "ghost"
		_, err := r.UpdateSeller(ctx, 999999999, m.SellerUpdate{CompanyName: &name})
		if !errors.Is(err, service.ErrNotFound) {
			t.Errorf("ожидалась ErrNotFound, получено: %v", err)
		}
	})
}

func TestSellerRepo_DeleteSellerByID_Integration(t *testing.T) {
	requireIntegrationDatabase(t)
	t.Run("success", func(t *testing.T) {
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

		_, err = r.GetSellerByID(ctx, id)
		if !errors.Is(err, service.ErrNotFound) {
			t.Errorf("после удаления ожидалась ErrNotFound, получено: %v", err)
		}
	})
}
