package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	m "github.com/beastixq/marketplace/internal/model"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// AuthUserProvider is a subset of UserService methods required by AuthService.
// Follows ISP: AuthService only depends on what it actually uses.
//
//go:generate mockgen -package mock_service -destination ../mocks/service/mock_auth_user_provider.go github.com/beastixq/marketplace/internal/service AuthUserProvider
type AuthUserProvider interface {
	CreateUser(ctx context.Context, uc m.UserCreate) (id int64, err error)
	GetAuthUserByID(ctx context.Context, id int64) (u m.User, err error)
	GetAuthUserByEmail(ctx context.Context, email string) (u m.User, err error)
}

// TokenBlocklist stores revoked token JTIs (e.g. Redis SET with TTL).
//
//go:generate mockgen -package mock_service -destination ../mocks/service/mock_token_blocklist.go github.com/beastixq/marketplace/internal/service TokenBlocklist
type TokenBlocklist interface {
	Add(ctx context.Context, jti string, exp time.Duration) error
	Contains(ctx context.Context, jti string) (bool, error)
}

type AuthService struct {
	userProvider AuthUserProvider
	blocklist    TokenBlocklist
	secret       string
	tokenTTL     time.Duration
}

func NewAuthService(userProvider AuthUserProvider, blocklist TokenBlocklist, secret string, tokenTTL time.Duration) AuthService {
	return AuthService{
		userProvider: userProvider,
		blocklist:    blocklist,
		secret:       secret,
		tokenTTL:     tokenTTL}
}

type jwtClaims struct {
	jwt.RegisteredClaims
	UserID int64
	Role   string
}

func (as AuthService) generateToken(user m.User) (token string, err error) {
	claims := jwtClaims{
		UserID: user.ID,
		Role:   string(user.Role),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(as.tokenTTL)),
			ID:        uuid.New().String(),
		},
	}
	token, err = jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(as.secret))
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrSign, err)
	}
	return token, nil
}

func (as AuthService) Register(ctx context.Context, uc m.UserCreate) (token string, err error) {
	userID, err := as.userProvider.CreateUser(ctx, uc)
	if err != nil {
		if errors.Is(err, ErrAccountWithEmailAlreadyExists) {
			return "", ErrAccountWithEmailAlreadyExists
		}
		if errors.Is(err, ErrPermissionDenied) {
			return "", ErrPermissionDenied
		}
		return "", fmt.Errorf("%w: %v", ErrCreateUser, err)
	}

	user, err := as.userProvider.GetAuthUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return "", ErrRegistration
		}
		return "", fmt.Errorf("%w: %v", ErrGetUserByID, err)
	}

	token, err = as.generateToken(user)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrRegistration, err)
	}
	return token, nil
}

func (as AuthService) Login(ctx context.Context, email, password string) (token string, err error) {
	user, err := as.userProvider.GetAuthUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return "", ErrUserNotFound
		}
		return "", fmt.Errorf("%w: %v", ErrLogin, err)
	}
	if user.DeletedAt != nil {
		return "", ErrAccountDeactivated
	}

	if err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", ErrWrongPassword
	}

	token, err = as.generateToken(user)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrGenerateToken, err)
	}

	return token, nil
}

func (as AuthService) Logout(ctx context.Context, claims m.TokenClaims) error {
	if as.blocklist == nil {
		return nil
	}
	remaining := time.Until(claims.Exp)
	if remaining <= 0 {
		return nil
	}
	if err := as.blocklist.Add(ctx, claims.JTI.String(), remaining); err != nil {
		return fmt.Errorf("%w: %v", ErrLogout, err)
	}
	return nil
}

func (as AuthService) ValidateToken(ctx context.Context, token string) (claims m.TokenClaims, err error) {
	t, err := jwt.ParseWithClaims(token, &jwtClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrWrongSigningMethod
		}
		return []byte(as.secret), nil
	})
	if err != nil {
		return m.TokenClaims{}, fmt.Errorf("%w: %v", ErrParseToken, err)
	}

	c, ok := t.Claims.(*jwtClaims)
	if !ok || !t.Valid {
		return m.TokenClaims{}, ErrParseToken
	}

	claims = m.TokenClaims{
		UserID: c.UserID,
		Role:   m.UserRole(c.Role),
		Exp:    c.ExpiresAt.Time,
		JTI:    uuid.MustParse(c.ID),
	}

	if as.blocklist != nil {
		blocked, err := as.blocklist.Contains(ctx, claims.JTI.String())
		if err != nil {
			return m.TokenClaims{}, fmt.Errorf("%w: %v", ErrParseToken, err)
		}
		if blocked {
			return m.TokenClaims{}, ErrTokenBlocked
		}
	}

	// Reject tokens whose owner was soft-deleted after the token was issued.
	user, err := as.userProvider.GetAuthUserByID(ctx, claims.UserID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return m.TokenClaims{}, ErrAccountDeactivated
		}
		return m.TokenClaims{}, fmt.Errorf("%w: %v", ErrParseToken, err)
	}
	if user.DeletedAt != nil {
		return m.TokenClaims{}, ErrAccountDeactivated
	}

	return claims, nil
}
