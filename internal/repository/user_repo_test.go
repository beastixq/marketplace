package repository_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	m "github.com/beastixq/marketplace/internal/model"
	repo "github.com/beastixq/marketplace/internal/repository"
	"github.com/beastixq/marketplace/internal/service"
)

var _ service.UserRepo = repo.UserRepoImpl{}

func createTestUser(t *testing.T) int64 {
	t.Helper()
	ctx := context.Background()
	r := repo.NewUserRepo(testPool)

	id, err := r.CreateUser(ctx, m.UserCreate{
		Email:    fmt.Sprintf("testuser_%d@example.com", time.Now().UnixNano()),
		Password: "hashed_password",
		FullName: "Test User",
		Role:     m.RoleBuyer,
	})
	if err != nil {
		t.Fatalf("createTestUser: %v", err)
	}
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), "DELETE FROM users WHERE id = $1", id)
	})
	return id
}

// TestUserRepo_CreateAndGet проверяет сохранение пользователя и его извлечение по ID.
func TestUserRepo_CreateAndGet(t *testing.T) {
	var r service.UserRepo = repo.NewUserRepo(testPool)
	ctx := context.Background()

	email := fmt.Sprintf("testuser_%d@example.com", time.Now().UnixNano())
	id, err := r.CreateUser(ctx, m.UserCreate{
		Email:    email,
		Password: "hashed_password",
		FullName: "Ivan Ivanov",
		Role:     m.RoleBuyer,
	})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), "DELETE FROM users WHERE id = $1", id)
	})

	got, err := r.GetUserByID(ctx, id)
	if err != nil {
		t.Fatalf("GetUserByID: %v", err)
	}

	if got.ID != id {
		t.Errorf("ID: got %d, want %d", got.ID, id)
	}
	if got.Email != email {
		t.Errorf("Email: got %q, want %q", got.Email, email)
	}
	if got.FullName != "Ivan Ivanov" {
		t.Errorf("FullName: got %q, want %q", got.FullName, "Ivan Ivanov")
	}
	if got.Role != m.RoleBuyer {
		t.Errorf("Role: got %q, want %q", got.Role, m.RoleBuyer)
	}
	if got.DeletedAt != nil {
		t.Error("DeletedAt должен быть nil у только что созданного пользователя")
	}
}

// TestUserRepo_Update проверяет изменение полей пользователя.
func TestUserRepo_Update(t *testing.T) {
	var r service.UserRepo = repo.NewUserRepo(testPool)
	ctx := context.Background()

	id := createTestUser(t)

	newName := "Updated Name"
	newEmail := fmt.Sprintf("updated_%d@example.com", time.Now().UnixNano())
	updated, err := r.UpdateUser(ctx, id, m.UserUpdate{
		FullName: &newName,
		Email:    &newEmail,
	})
	if err != nil {
		t.Fatalf("UpdateUser: %v", err)
	}

	if updated.FullName != newName {
		t.Errorf("FullName после Update: got %q, want %q", updated.FullName, newName)
	}
	if updated.Email != newEmail {
		t.Errorf("Email после Update: got %q, want %q", updated.Email, newEmail)
	}
}

// TestUserRepo_SoftDelete проверяет, что DeleteUserByID выполняет мягкое удаление
// (deleted_at становится ненулевым, запись остаётся в базе).
func TestUserRepo_SoftDelete(t *testing.T) {
	var r service.UserRepo = repo.NewUserRepo(testPool)
	ctx := context.Background()

	id := createTestUser(t)

	if err := r.DeleteUserByID(ctx, id); err != nil {
		t.Fatalf("DeleteUserByID: %v", err)
	}

	got, err := r.GetUserByID(ctx, id)
	if err != nil {
		t.Fatalf("GetUserByID после удаления: %v", err)
	}
	if got.DeletedAt == nil {
		t.Error("DeletedAt должен быть установлен после мягкого удаления")
	}
}
