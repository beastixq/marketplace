package service_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	mock_service "github.com/beastixq/marketplace/internal/mocks/service"
	m "github.com/beastixq/marketplace/internal/model"
	"github.com/beastixq/marketplace/internal/service"
	"go.uber.org/mock/gomock"
	"golang.org/x/crypto/bcrypt"
)

const someID int64 = 42
const someStrongPassword = "xy1szo87vcnawkz&zj1SZ1NZC"
const someStrongPasswordHash = "$2a$04$6TTOHTypNV5UN4zlH94M1Oc20yvkekVJPJ2fYBGnQuiAcCWcHSga6"
const someFullName = "Some Fullname"
const someEmail = "some@email.com"

var someTime = time.UnixMilli(1771837932 * 1000)
var someRole = m.UserRole(m.RoleSeller)
var testPhone = "+71231231231"

var someUser = m.User{
	ID:           someID,
	Email:        someEmail,
	PasswordHash: someStrongPasswordHash,
	FullName:     someFullName,
	Phone:        &testPhone,
	Role:         someRole,
	CreatedAt:    someTime,
	DeletedAt:    nil,
}

// MockUserReturn is a shared mock return type for repo methods returning (m.User, error).
type MockUserReturn struct {
	User  m.User
	Error error
}

type MockCreateReturn struct {
	ID    int64
	Error error
}

func assertError(t *testing.T, got, want error) {
	t.Helper()
	if !errors.Is(got, want) {
		t.Fatalf("invalid error. expected: %v, got: %v", want, got)
	}
}

func assertUser(t *testing.T, got, want m.User) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("invalid user. expected: %v, got: %v", want, got)
	}
}

func TestGetUserByID(t *testing.T) {
	mock := mock_service.NewMockUserRepo(gomock.NewController(t))
	svc := service.NewUserService(mock, bcrypt.MinCost)
	ctx := context.Background()

	type testCase struct {
		Description  string
		UserID       int64
		MockReturn   MockUserReturn
		ExpectedUser m.User
		ExpectedErr  error
	}

	tCases := []testCase{
		// positives
		{
			Description:  "Success",
			UserID:       someID,
			MockReturn:   MockUserReturn{User: someUser},
			ExpectedUser: someUser,
		},
		// negatives
		{
			Description: "User not found",
			UserID:      123,
			MockReturn:  MockUserReturn{Error: service.ErrNotFound},
			ExpectedErr: service.ErrUserNotFound,
		},
		{
			Description: "Repo error",
			UserID:      999,
			MockReturn:  MockUserReturn{Error: errors.New("some repo error")},
			ExpectedErr: service.ErrGetUserByID,
		},
	}

	for _, tCase := range tCases {
		t.Run(tCase.Description, func(t *testing.T) {
			mock.EXPECT().GetUserByID(ctx, tCase.UserID).Return(tCase.MockReturn.User, tCase.MockReturn.Error)
			user, err := svc.GetUserByID(ctx, testActor(tCase.UserID, m.RoleBuyer), tCase.UserID)
			assertError(t, err, tCase.ExpectedErr)
			assertUser(t, user, tCase.ExpectedUser)
		})
	}
}

func TestGetUserByEmail(t *testing.T) {
	mock := mock_service.NewMockUserRepo(gomock.NewController(t))
	svc := service.NewUserService(mock, bcrypt.MinCost)
	ctx := context.Background()

	type testCase struct {
		Description  string
		Email        string
		MockReturn   MockUserReturn
		ExpectedUser m.User
		ExpectedErr  error
	}

	tCases := []testCase{
		// positives
		{
			Description:  "Success",
			Email:        someEmail,
			MockReturn:   MockUserReturn{User: someUser},
			ExpectedUser: someUser,
		},
		// negatives
		{
			Description: "User not found",
			Email:       someEmail,
			MockReturn:  MockUserReturn{Error: service.ErrNotFound},
			ExpectedErr: service.ErrUserNotFound,
		},
		{
			Description: "Repo error",
			Email:       someEmail,
			MockReturn:  MockUserReturn{Error: errors.New("some repo error")},
			ExpectedErr: service.ErrGetUserByEmail,
		},
	}

	for _, tCase := range tCases {
		t.Run(tCase.Description, func(t *testing.T) {
			mock.EXPECT().GetUserByEmail(ctx, tCase.Email).Return(tCase.MockReturn.User, tCase.MockReturn.Error)
			user, err := svc.GetUserByEmail(ctx, tCase.Email)
			assertError(t, err, tCase.ExpectedErr)
			assertUser(t, user, tCase.ExpectedUser)
		})
	}
}

func TestCreateUser(t *testing.T) {
	mock := mock_service.NewMockUserRepo(gomock.NewController(t))
	svc := service.NewUserService(mock, bcrypt.MinCost)
	ctx := context.Background()

	someUserCreate := m.UserCreate{
		Password: someStrongPasswordHash,
		Email:    someEmail,
		FullName: someFullName,
		Phone:    &testPhone,
		Role:     someRole,
	}

	type testCase struct {
		Description    string
		Create         m.UserCreate
		MockGetByEmail MockUserReturn
		MockCreate     *MockCreateReturn
		ExpectedID     int64
		ExpectedErr    error
	}

	tCases := []testCase{
		// positives
		{
			Description:    "Success",
			Create:         someUserCreate,
			MockGetByEmail: MockUserReturn{Error: service.ErrNotFound},
			MockCreate:     &MockCreateReturn{ID: someID},
			ExpectedID:     someID,
		},
		// negatives
		{
			Description:    "User already exists",
			Create:         someUserCreate,
			MockGetByEmail: MockUserReturn{User: someUser},
			ExpectedErr:    service.ErrAccountWithEmailAlreadyExists,
		},
		{
			Description:    "GetUserByEmail fails",
			Create:         someUserCreate,
			MockGetByEmail: MockUserReturn{Error: errors.New("some error")},
			ExpectedErr:    service.ErrGetUserByEmail,
		},
		{
			Description:    "CreateUser fails",
			Create:         someUserCreate,
			MockGetByEmail: MockUserReturn{Error: service.ErrNotFound},
			MockCreate:     &MockCreateReturn{Error: errors.New("some error")},
			ExpectedErr:    service.ErrCreateUser,
		},
	}

	for _, tCase := range tCases {
		t.Run(tCase.Description, func(t *testing.T) {
			mock.EXPECT().GetUserByEmail(ctx, tCase.Create.Email).Return(tCase.MockGetByEmail.User, tCase.MockGetByEmail.Error)
			if tCase.MockCreate != nil {
				mock.EXPECT().CreateUser(ctx, gomock.Any()).Return(tCase.MockCreate.ID, tCase.MockCreate.Error)
			}
			id, err := svc.CreateUser(ctx, tCase.Create)
			assertError(t, err, tCase.ExpectedErr)
			if id != tCase.ExpectedID {
				t.Fatalf("invalid id. expected: %v, got: %v", tCase.ExpectedID, id)
			}
		})
	}
}

func TestUpdateUser(t *testing.T) {
	mock := mock_service.NewMockUserRepo(gomock.NewController(t))
	svc := service.NewUserService(mock, bcrypt.MinCost)
	ctx := context.Background()

	newEmail := "newemail@gmail.com"
	newFullName := "NEW fullname"
	newPhone := "+70987654321"
	newRole := m.UserRole(m.RoleAdmin)

	fullUpdate := m.UserUpdate{
		Email:    &newEmail,
		FullName: &newFullName,
		Phone:    &newPhone,
		Role:     &newRole,
	}

	partialUpdate := m.UserUpdate{
		FullName: &newFullName,
	}

	fullyUpdated := m.User{
		ID:           someUser.ID,
		PasswordHash: someUser.PasswordHash,
		CreatedAt:    someUser.CreatedAt,
		DeletedAt:    someUser.DeletedAt,
		Email:        *fullUpdate.Email,
		Phone:        fullUpdate.Phone,
		FullName:     *fullUpdate.FullName,
		Role:         *fullUpdate.Role,
	}

	partiallyUpdated := m.User{
		ID:           someUser.ID,
		Email:        someUser.Email,
		PasswordHash: someUser.PasswordHash,
		FullName:     *partialUpdate.FullName,
		Phone:        someUser.Phone,
		Role:         someUser.Role,
		CreatedAt:    someUser.CreatedAt,
		DeletedAt:    someUser.DeletedAt,
	}

	type testCase struct {
		Description  string
		UserID       int64
		Update       m.UserUpdate
		MockGetByID  MockUserReturn
		MockUpdate   *MockUserReturn
		ExpectedUser m.User
		ExpectedErr  error
	}

	tCases := []testCase{
		// positives
		{
			Description:  "Success full update",
			UserID:       someUser.ID,
			Update:       fullUpdate,
			MockGetByID:  MockUserReturn{User: someUser},
			MockUpdate:   &MockUserReturn{User: fullyUpdated},
			ExpectedUser: fullyUpdated,
		},
		{
			Description:  "Success partial update",
			UserID:       someUser.ID,
			Update:       partialUpdate,
			MockGetByID:  MockUserReturn{User: someUser},
			MockUpdate:   &MockUserReturn{User: partiallyUpdated},
			ExpectedUser: partiallyUpdated,
		},
		// negatives
		{
			Description: "No changes in update",
			UserID:      someUser.ID,
			Update:      m.UserUpdate{},
			MockGetByID: MockUserReturn{User: someUser},
			MockUpdate:  &MockUserReturn{Error: service.ErrNoChangesInUpdate},
			ExpectedErr: service.ErrNoChangesInUpdate,
		},
		{
			Description: "GetByID returns NotFound",
			UserID:      999,
			Update:      fullUpdate,
			MockGetByID: MockUserReturn{Error: service.ErrNotFound},
			ExpectedErr: service.ErrUserNotFound,
		},
		{
			Description: "GetByID returns error",
			UserID:      someUser.ID,
			Update:      fullUpdate,
			MockGetByID: MockUserReturn{Error: errors.New("some repo error")},
			ExpectedErr: service.ErrGetUserByID,
		},
		{
			Description: "UpdateUser returns error",
			UserID:      someUser.ID,
			Update:      fullUpdate,
			MockGetByID: MockUserReturn{User: someUser},
			MockUpdate:  &MockUserReturn{Error: errors.New("some repo error")},
			ExpectedErr: service.ErrUpdateUser,
		},
	}

	for _, tCase := range tCases {
		t.Run(tCase.Description, func(t *testing.T) {
			mock.EXPECT().GetUserByID(ctx, tCase.UserID).Return(tCase.MockGetByID.User, tCase.MockGetByID.Error)
			if tCase.MockUpdate != nil {
				mock.EXPECT().UpdateUser(ctx, tCase.UserID, tCase.Update).Return(tCase.MockUpdate.User, tCase.MockUpdate.Error)
			}
			user, err := svc.UpdateUser(ctx, testActor(someID, m.RoleAdmin), tCase.UserID, tCase.Update)
			assertError(t, err, tCase.ExpectedErr)
			assertUser(t, user, tCase.ExpectedUser)
		})
	}
}

func TestChangePasswordUser(t *testing.T) {
	mock := mock_service.NewMockUserRepo(gomock.NewController(t))
	svc := service.NewUserService(mock, bcrypt.MinCost)
	ctx := context.Background()

	type testCase struct {
		Description    string
		UserID         int64
		OldPassword    string
		NewPassword    string
		MockGetByID    MockUserReturn
		MockChangePass *error
		ExpectedErr    error
	}

	tCases := []testCase{
		// positives
		{
			Description:    "Success",
			UserID:         someID,
			OldPassword:    someStrongPassword,
			NewPassword:    "newStrongPassword123!",
			MockGetByID:    MockUserReturn{User: someUser},
			MockChangePass: ptrErr(nil),
		},
		// negatives
		{
			Description: "User not found",
			UserID:      999,
			OldPassword: someStrongPassword,
			NewPassword: "newStrongPassword123!",
			MockGetByID: MockUserReturn{Error: service.ErrNotFound},
			ExpectedErr: service.ErrUserNotFound,
		},
		{
			Description: "GetByID repo error",
			UserID:      someID,
			OldPassword: someStrongPassword,
			NewPassword: "newStrongPassword123!",
			MockGetByID: MockUserReturn{Error: errors.New("some repo error")},
			ExpectedErr: service.ErrGetUserByID,
		},
		{
			Description: "Wrong old password",
			UserID:      someID,
			OldPassword: "wrongPassword",
			NewPassword: "newStrongPassword123!",
			MockGetByID: MockUserReturn{User: someUser},
			ExpectedErr: service.ErrWrongPassword,
		},
		{
			Description:    "ChangePasswordUser repo error",
			UserID:         someID,
			OldPassword:    someStrongPassword,
			NewPassword:    "newStrongPassword123!",
			MockGetByID:    MockUserReturn{User: someUser},
			MockChangePass: ptrErr(errors.New("some repo error")),
			ExpectedErr:    service.ErrChangePasswordUser,
		},
	}

	for _, tCase := range tCases {
		t.Run(tCase.Description, func(t *testing.T) {
			mock.EXPECT().GetUserByID(ctx, tCase.UserID).Return(tCase.MockGetByID.User, tCase.MockGetByID.Error)
			if tCase.MockChangePass != nil {
				mock.EXPECT().ChangePasswordUser(ctx, tCase.UserID, gomock.Any()).Return(*tCase.MockChangePass)
			}
			err := svc.ChangePasswordUser(ctx, testActor(tCase.UserID, m.RoleBuyer), tCase.OldPassword, tCase.NewPassword)
			assertError(t, err, tCase.ExpectedErr)
		})
	}
}

func TestDeleteUserByID(t *testing.T) {
	mock := mock_service.NewMockUserRepo(gomock.NewController(t))
	svc := service.NewUserService(mock, bcrypt.MinCost)
	ctx := context.Background()

	type testCase struct {
		Description string
		UserID      int64
		MockError   error
		ExpectedErr error
	}

	tCases := []testCase{
		{
			Description: "Success",
			UserID:      someID,
		},
		{
			Description: "Repo error",
			UserID:      someID,
			MockError:   errors.New("some repo error"),
			ExpectedErr: service.ErrDeleteUser,
		},
	}

	for _, tCase := range tCases {
		t.Run(tCase.Description, func(t *testing.T) {
			mock.EXPECT().DeleteUserByID(ctx, tCase.UserID).Return(tCase.MockError)
			err := svc.DeleteUserByID(ctx, testActor(tCase.UserID, m.RoleBuyer), tCase.UserID)
			assertError(t, err, tCase.ExpectedErr)
		})
	}
}

func ptrErr(err error) *error {
	return &err
}
