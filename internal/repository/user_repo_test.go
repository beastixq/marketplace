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

	fetched, err := r.GetUserByID(ctx, id)
	if err != nil {
		t.Fatalf("GetUserByID после Update: %v", err)
	}
	if fetched.FullName != newName {
		t.Errorf("FullName в БД: got %q, want %q", fetched.FullName, newName)
	}
	if fetched.Email != newEmail {
		t.Errorf("Email в БД: got %q, want %q", fetched.Email, newEmail)
	}
}

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

func TestUserRepo_Update_NotFound(t *testing.T) {
	var r service.UserRepo = repo.NewUserRepo(testPool)
	ctx := context.Background()

	name := "ghost"
	_, err := r.UpdateUser(ctx, 999999999, m.UserUpdate{FullName: &name})
	if !errors.Is(err, service.ErrNotFound) {
		t.Errorf("ожидалась ErrNotFound, получено: %v", err)
	}
}

func TestUserRepo_GetUserByEmail(t *testing.T) {
	var r service.UserRepo = repo.NewUserRepo(testPool)
	ctx := context.Background()
	email := fmt.Sprintf("byemail_%d@example.com", time.Now().UnixNano())

	// 1. Pre-condition: пользователя с таким email нет в БД.
	_, err := r.GetUserByEmail(ctx, email)
	if err == nil {
		t.Fatalf("до создания GetUserByEmail(%q) должен вернуть ошибку, но вернул успех", email)
	}
	if !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("ожидалась ErrNotFound, получена: %v", err)
	}

	// 2. Act: создаём пользователя.
	id, err := r.CreateUser(ctx, m.UserCreate{
		Email:    email,
		Password: "hashed_password",
		FullName: "ByEmail User",
		Role:     m.RoleBuyer,
	})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), "DELETE FROM users WHERE id = $1", id)
	})

	// 3. Post-condition: GetUserByEmail возвращает именно созданного пользователя.
	got, err := r.GetUserByEmail(ctx, email)
	if err != nil {
		t.Fatalf("GetUserByEmail после создания: %v", err)
	}
	if got.ID != id {
		t.Errorf("ID: got %d, want %d", got.ID, id)
	}
	if got.Email != email {
		t.Errorf("Email: got %q, want %q", got.Email, email)
	}
}

func TestUserRepo_ChangePassword(t *testing.T) {
	r := repo.NewUserRepo(testPool)
	ctx := context.Background()
	id := createTestUser(t)

	// Pre-condition: читаем текущий хеш.
	before, err := r.GetUserByID(ctx, id)
	if err != nil {
		t.Fatalf("GetUserByID до смены пароля: %v", err)
	}
	newHash := "new_hashed_password_value"
	if before.PasswordHash == newHash {
		t.Fatal("тестовые данные некорректны: хеши совпадают до смены")
	}

	// Act.
	if err := r.ChangePasswordUser(ctx, id, newHash); err != nil {
		t.Fatalf("ChangePasswordUser: %v", err)
	}

	// Post-condition: хеш изменён.
	after, err := r.GetUserByID(ctx, id)
	if err != nil {
		t.Fatalf("GetUserByID после смены пароля: %v", err)
	}
	if after.PasswordHash != newHash {
		t.Errorf("PasswordHash после смены: got %q, want %q", after.PasswordHash, newHash)
	}
}
