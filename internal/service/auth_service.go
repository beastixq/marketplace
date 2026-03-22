package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	m "github.com/beastixq/marketplace/internal/model"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type AuthService struct {
	userService UserService
	secret      string
	tokenTTL    time.Duration
	// redis
}

func NewAuthService(userService UserService, secret string, tokenTTL time.Duration) AuthService {
	return AuthService{
		userService: userService,
		secret:      secret,
		tokenTTL:    tokenTTL}
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
	userID, err := as.userService.CreateUser(ctx, uc)
	if err != nil {
		if errors.Is(err, ErrAccountWithEmailAlreadyExists) {
			return "", ErrAccountWithEmailAlreadyExists
		}
		return "", fmt.Errorf("%w: %v", ErrCreateUser, err)
	}

	user, err := as.userService.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
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
	user, err := as.userService.Login(ctx, email, password)
	if err != nil {
		if errors.Is(err, ErrWrongPassword) {
			return "", ErrWrongPassword
		}
		if errors.Is(err, ErrUserNotFound) {
			return "", ErrUserNotFound
		}
		return "", fmt.Errorf("%w: %v", ErrLogin, err)
	}

	token, err = as.generateToken(user)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrGenerateToken, err)
	}

	return token, nil
}

func (as AuthService) Logout(ctx context.Context, token string) error {
	// TODO: redis blocklist
	panic("not implemented yet")
}

func (as AuthService) ValidateToken(token string) (claims m.TokenClaims, err error) {
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
	return claims, nil
}
