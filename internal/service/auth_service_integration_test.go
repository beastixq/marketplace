//go:build integration

package service_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	m "github.com/beastixq/marketplace/internal/model"
	s "github.com/beastixq/marketplace/internal/service"
	"golang.org/x/crypto/bcrypt"
)

func TestAuthService_Register_Integration(t *testing.T) {
	t.Run("persist hash login token duplicate and deactivation", func(t *testing.T) {
		f := newIntegrationFixture(t)
		ctx := context.Background()
		auth := s.NewAuthService(f.users, nil, "integration-secret", time.Hour)
		create := m.UserCreate{Email: fmt.Sprintf("auth-%d@example.test", time.Now().UnixNano()), Password: "Synthetic-password-42", FullName: "Auth account", Role: m.RoleBuyer}

		token, err := auth.Register(ctx, create)
		mustIntegration(t, err)
		user, err := f.users.GetAuthUserByEmail(ctx, create.Email)
		mustIntegration(t, err)
		f.userIDs = append(f.userIDs, user.ID)
		if user.PasswordHash == create.Password {
			t.Fatal("plaintext password persisted")
		}
		mustIntegration(t, bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(create.Password)))
		claims, err := auth.ValidateToken(ctx, token)
		mustIntegration(t, err)
		if claims.UserID != user.ID || claims.Role != m.RoleBuyer {
			t.Fatalf("unexpected identity: %+v", claims)
		}
		loginToken, err := auth.Login(ctx, create.Email, create.Password)
		mustIntegration(t, err)
		_, err = auth.ValidateToken(ctx, loginToken)
		mustIntegration(t, err)
		if _, err := auth.Register(ctx, create); !errors.Is(err, s.ErrAccountWithEmailAlreadyExists) {
			t.Fatalf("duplicate = %v", err)
		}
		if _, err := auth.Login(ctx, create.Email, "wrong-password"); !errors.Is(err, s.ErrWrongPassword) {
			t.Fatalf("password = %v", err)
		}
		mustIntegration(t, f.users.DeleteUserByID(ctx, s.Actor{UserID: user.ID, Role: m.RoleBuyer}, user.ID))
		if _, err := auth.Login(ctx, create.Email, create.Password); !errors.Is(err, s.ErrAccountDeactivated) {
			t.Fatalf("inactive login = %v", err)
		}
		if _, err := auth.ValidateToken(ctx, token); err == nil {
			t.Fatal("inactive account token accepted")
		}
	})
}
