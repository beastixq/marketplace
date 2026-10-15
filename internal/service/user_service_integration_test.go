//go:build integration

package service_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	m "github.com/beastixq/marketplace/internal/model"
	repo "github.com/beastixq/marketplace/internal/repository"
	s "github.com/beastixq/marketplace/internal/service"
)

func (f *serviceIntegration) userCreate(role m.UserRole) m.UserCreate {
	f.t.Helper()
	create := m.UserCreate{
		Email:    fmt.Sprintf("integration-%d-%d@example.test", time.Now().UnixNano(), len(f.ownedUserEmails)),
		Password: "Synthetic-password-42", FullName: "Integration account", Role: role,
	}
	// Register before CreateUser/Register: they can persist before a later step fails.
	f.ownedUserEmails = append(f.ownedUserEmails, create.Email)
	return create
}

func (f *serviceIntegration) createActor(role m.UserRole) s.Actor {
	f.t.Helper()
	id, err := f.userService().CreateUser(context.Background(), f.userCreate(role))
	mustIntegration(f.t, err)
	return s.Actor{UserID: id, Role: role}
}

func TestUserService_CreateUser_Integration(t *testing.T) {
	f := newServiceIntegration(t)
	create := f.userCreate(m.RoleBuyer)

	id, err := f.userService().CreateUser(context.Background(), create)
	mustIntegration(t, err)
	user, err := repo.NewUserRepo(f.pool).GetUserByID(context.Background(), id)
	mustIntegration(t, err)
	if user.ID != id || user.Email != create.Email || user.FullName != create.FullName || user.Role != create.Role {
		t.Fatalf("persisted account = %+v", user)
	}
	waitForIntegrationTermination(t)
}

func waitForIntegrationTermination(t *testing.T) {
	t.Helper()
	if os.Getenv("TEST_FAULT_GATE") != "integration" {
		return
	}
	// Acceptance kills the container only after a verified, committed account exists.
	mustIntegration(t, os.WriteFile("/run/integration-written", []byte("committed"), 0600))
	<-time.After(180 * time.Second)
	t.Fatal("acceptance did not terminate the integration container in 180 seconds")
}
