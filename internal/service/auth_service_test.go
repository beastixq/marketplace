package service_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	mock_service "github.com/beastixq/marketplace/internal/mocks/service"
	m "github.com/beastixq/marketplace/internal/model"
	"github.com/beastixq/marketplace/internal/service"
)

const testSecret = "test-secret-key"
const testTokenTTL = time.Hour

func newAuthService(ctrl *gomock.Controller) (service.AuthService, *mock_service.MockAuthUserProvider, *mock_service.MockTokenBlocklist) {
	userMock := mock_service.NewMockAuthUserProvider(ctrl)
	blMock := mock_service.NewMockTokenBlocklist(ctrl)
	svc := service.NewAuthService(userMock, blMock, testSecret, testTokenTTL)
	return svc, userMock, blMock
}

func TestRegister(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc, userMock, _ := newAuthService(ctrl)
	ctx := context.Background()

	userCreate := m.UserCreate{
		Email:    someEmail,
		Password: someStrongPassword,
		FullName: someFullName,
		Phone:    &testPhone,
		Role:     m.UserRole(m.RoleBuyer),
	}

	type testCase struct {
		Description string
		Create      m.UserCreate
		MockCreate  MockCreateReturn
		MockGetByID *MockUserReturn
		ExpectToken bool
		ExpectedErr error
	}

	tCases := []testCase{
		{
			Description: "Success",
			Create:      userCreate,
			MockCreate:  MockCreateReturn{ID: someID},
			MockGetByID: &MockUserReturn{User: someUser},
			ExpectToken: true,
		},
		{
			Description: "Email already exists",
			Create:      userCreate,
			MockCreate:  MockCreateReturn{Error: service.ErrAccountWithEmailAlreadyExists},
			ExpectedErr: service.ErrAccountWithEmailAlreadyExists,
		},
		{
			Description: "CreateUser error",
			Create:      userCreate,
			MockCreate:  MockCreateReturn{Error: errors.New("some error")},
			ExpectedErr: service.ErrCreateUser,
		},
		{
			Description: "GetUserByID not found after create",
			Create:      userCreate,
			MockCreate:  MockCreateReturn{ID: someID},
			MockGetByID: &MockUserReturn{Error: service.ErrUserNotFound},
			ExpectedErr: service.ErrRegistration,
		},
		{
			Description: "GetUserByID error",
			Create:      userCreate,
			MockCreate:  MockCreateReturn{ID: someID},
			MockGetByID: &MockUserReturn{Error: errors.New("some error")},
			ExpectedErr: service.ErrGetUserByID,
		},
	}

	for _, tCase := range tCases {
		t.Run(tCase.Description, func(t *testing.T) {
			userMock.EXPECT().CreateUser(ctx, tCase.Create).Return(tCase.MockCreate.ID, tCase.MockCreate.Error)
			if tCase.MockGetByID != nil {
				userMock.EXPECT().GetUserByID(ctx, someID).Return(tCase.MockGetByID.User, tCase.MockGetByID.Error)
			}

			token, err := svc.Register(ctx, tCase.Create)
			assertError(t, err, tCase.ExpectedErr)
			if tCase.ExpectToken {
				assertTokenClaims(t, token, testSecret, someID, someRole)
			} else if token != "" {
				t.Fatalf("expected empty token, got: %v", token)
			}
		})
	}
}

func TestAuthLogin(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc, userMock, _ := newAuthService(ctrl)
	ctx := context.Background()

	type testCase struct {
		Description string
		Email       string
		Password    string
		MockReturn  MockUserReturn
		ExpectToken bool
		ExpectedErr error
	}

	tCases := []testCase{
		{
			Description: "Success",
			Email:       someEmail,
			Password:    someStrongPassword,
			MockReturn:  MockUserReturn{User: someUser},
			ExpectToken: true,
		},
		{
			Description: "User not found",
			Email:       someEmail,
			Password:    someStrongPassword,
			MockReturn:  MockUserReturn{Error: service.ErrUserNotFound},
			ExpectedErr: service.ErrUserNotFound,
		},
		{
			Description: "Wrong password",
			Email:       someEmail,
			Password:    "wrong-password",
			MockReturn:  MockUserReturn{User: someUser},
			ExpectedErr: service.ErrWrongPassword,
		},
		{
			Description: "GetUserByEmail error",
			Email:       someEmail,
			Password:    someStrongPassword,
			MockReturn:  MockUserReturn{Error: errors.New("some error")},
			ExpectedErr: service.ErrLogin,
		},
	}

	for _, tCase := range tCases {
		t.Run(tCase.Description, func(t *testing.T) {
			userMock.EXPECT().GetUserByEmail(ctx, tCase.Email).Return(tCase.MockReturn.User, tCase.MockReturn.Error)

			token, err := svc.Login(ctx, tCase.Email, tCase.Password)
			assertError(t, err, tCase.ExpectedErr)
			if tCase.ExpectToken {
				assertTokenClaims(t, token, testSecret, someID, someRole)
			} else if token != "" {
				t.Fatalf("expected empty token, got: %v", token)
			}
		})
	}
}

func TestLogout(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc, _, blMock := newAuthService(ctrl)
	ctx := context.Background()

	validJTI := uuid.New()
	validClaims := m.TokenClaims{
		UserID: someID,
		Role:   someRole,
		Exp:    time.Now().Add(time.Hour),
		JTI:    validJTI,
	}

	expiredClaims := m.TokenClaims{
		UserID: someID,
		Role:   someRole,
		Exp:    time.Now().Add(-time.Hour),
		JTI:    uuid.New(),
	}

	type testCase struct {
		Description  string
		Claims       m.TokenClaims
		MockAddError *error
		ExpectedErr  error
	}

	tCases := []testCase{
		{
			Description:  "Success",
			Claims:       validClaims,
			MockAddError: ptrErr(nil),
		},
		{
			Description: "Already expired token",
			Claims:      expiredClaims,
		},
		{
			Description:  "Blocklist error",
			Claims:       validClaims,
			MockAddError: ptrErr(errors.New("redis error")),
			ExpectedErr:  service.ErrLogout,
		},
	}

	for _, tCase := range tCases {
		t.Run(tCase.Description, func(t *testing.T) {
			if tCase.MockAddError != nil {
				expectedTTL := time.Until(tCase.Claims.Exp)
				blMock.EXPECT().Add(ctx, tCase.Claims.JTI.String(), approxDuration{expected: expectedTTL, tolerance: 5 * time.Second}).Return(*tCase.MockAddError)
			}
			err := svc.Logout(ctx, tCase.Claims)
			assertError(t, err, tCase.ExpectedErr)
		})
	}
}

func TestValidateToken(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc, _, blMock := newAuthService(ctrl)
	ctx := context.Background()

	makeToken := func(claims jwt.MapClaims, secret string, method jwt.SigningMethod) string {
		t.Helper()
		token, err := jwt.NewWithClaims(method, claims).SignedString([]byte(secret))
		if err != nil {
			t.Fatalf("failed to create test token: %v", err)
		}
		return token
	}

	validJTI := uuid.New().String()
	validClaims := jwt.MapClaims{
		"UserID": float64(someID),
		"Role":   string(someRole),
		"exp":    float64(time.Now().Add(time.Hour).Unix()),
		"jti":    validJTI,
	}

	blockedJTI := uuid.New().String()
	blockedClaims := jwt.MapClaims{
		"UserID": float64(someID),
		"Role":   string(someRole),
		"exp":    float64(time.Now().Add(time.Hour).Unix()),
		"jti":    blockedJTI,
	}

	errorJTI := uuid.New().String()
	errorClaims := jwt.MapClaims{
		"UserID": float64(someID),
		"Role":   string(someRole),
		"exp":    float64(time.Now().Add(time.Hour).Unix()),
		"jti":    errorJTI,
	}

	type testCase struct {
		Description     string
		Token           string
		ExpectedJTI     string
		MockContains    *bool
		MockContainsErr error
		ExpectedErr     error
		CheckClaims     bool
	}

	tCases := []testCase{
		{
			Description:  "Success",
			Token:        makeToken(validClaims, testSecret, jwt.SigningMethodHS256),
			ExpectedJTI:  validJTI,
			MockContains: ptrBool(false),
			CheckClaims:  true,
		},
		{
			Description: "Expired token",
			Token: makeToken(jwt.MapClaims{
				"UserID": float64(someID),
				"Role":   string(someRole),
				"exp":    float64(time.Now().Add(-time.Hour).Unix()),
				"jti":    uuid.New().String(),
			}, testSecret, jwt.SigningMethodHS256),
			ExpectedErr: service.ErrParseToken,
		},
		{
			Description: "Wrong secret",
			Token:       makeToken(validClaims, "wrong-secret", jwt.SigningMethodHS256),
			ExpectedErr: service.ErrParseToken,
		},
		{
			Description: "Malformed token",
			Token:       "not-a-jwt-token",
			ExpectedErr: service.ErrParseToken,
		},
		{
			Description:  "Blocked token",
			Token:        makeToken(blockedClaims, testSecret, jwt.SigningMethodHS256),
			ExpectedJTI:  blockedJTI,
			MockContains: ptrBool(true),
			ExpectedErr:  service.ErrTokenBlocked,
		},
		{
			Description:     "Blocklist error",
			Token:           makeToken(errorClaims, testSecret, jwt.SigningMethodHS256),
			ExpectedJTI:     errorJTI,
			MockContainsErr: errors.New("redis error"),
			ExpectedErr:     service.ErrParseToken,
		},
	}

	for _, tCase := range tCases {
		t.Run(tCase.Description, func(t *testing.T) {
			if tCase.MockContains != nil {
				blMock.EXPECT().Contains(ctx, tCase.ExpectedJTI).Return(*tCase.MockContains, nil)
			} else if tCase.MockContainsErr != nil {
				blMock.EXPECT().Contains(ctx, tCase.ExpectedJTI).Return(false, tCase.MockContainsErr)
			}

			claims, err := svc.ValidateToken(ctx, tCase.Token)
			assertError(t, err, tCase.ExpectedErr)
			if tCase.CheckClaims {
				if claims.UserID != someID {
					t.Fatalf("invalid UserID. expected: %v, got: %v", someID, claims.UserID)
				}
				if claims.Role != someRole {
					t.Fatalf("invalid Role. expected: %v, got: %v", someRole, claims.Role)
				}
				if claims.JTI.String() != validJTI {
					t.Fatalf("invalid JTI. expected: %v, got: %v", validJTI, claims.JTI.String())
				}
				if claims.Exp.IsZero() {
					t.Fatal("expected non-zero Exp")
				}
			}
		})
	}
}

func ptrBool(b bool) *bool {
	return &b
}

func assertTokenClaims(t *testing.T, token, secret string, expectedUserID int64, expectedRole m.UserRole) {
	t.Helper()
	if token == "" {
		t.Fatal("expected non-empty token")
	}
	parsed, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		t.Fatalf("failed to parse token: %v", err)
	}
	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		t.Fatal("failed to cast claims to MapClaims")
	}
	if uid, _ := claims["UserID"].(float64); int64(uid) != expectedUserID {
		t.Fatalf("token UserID: expected %v, got %v", expectedUserID, int64(uid))
	}
	if role, _ := claims["Role"].(string); m.UserRole(role) != expectedRole {
		t.Fatalf("token Role: expected %v, got %v", expectedRole, role)
	}
	if _, ok := claims["jti"]; !ok {
		t.Fatal("token missing jti claim")
	}
	if _, ok := claims["exp"]; !ok {
		t.Fatal("token missing exp claim")
	}
}

// approxDuration matches a time.Duration within ±tolerance.
type approxDuration struct {
	expected  time.Duration
	tolerance time.Duration
}

func (a approxDuration) Matches(x interface{}) bool {
	d, ok := x.(time.Duration)
	if !ok {
		return false
	}
	diff := d - a.expected
	if diff < 0 {
		diff = -diff
	}
	return diff <= a.tolerance
}

func (a approxDuration) String() string {
	return fmt.Sprintf("≈ %v (±%v)", a.expected, a.tolerance)
}
