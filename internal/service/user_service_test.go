package service_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
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

func assertPasswordHash(t *testing.T, hash, password string) {
	t.Helper()
	if hash == password {
		t.Fatal("repository received plaintext password")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		t.Fatalf("stored hash does not match password: %v", err)
	}
}

func testUserLookup(t *testing.T, byEmail bool, lookup func(service.UserService) (m.User, error)) {
	t.Helper()
	user := buyerMother()
	wantRepoErr := service.ErrGetUserByID
	if byEmail {
		wantRepoErr = service.ErrGetUserByEmail
	}
	tests := []struct {
		name    string
		result  m.User
		repoErr error
		wantErr error
	}{
		{name: "success", result: user},
		{name: "user not found", repoErr: service.ErrNotFound, wantErr: service.ErrUserNotFound},
		{name: "repository failure", repoErr: errors.New("lookup failed"), wantErr: wantRepoErr},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := mock_service.NewMockUserRepo(gomock.NewController(t))
			svc := service.NewUserService(repo, bcrypt.MinCost)
			if byEmail {
				repo.EXPECT().GetUserByEmail(context.Background(), user.Email).Return(tc.result, tc.repoErr)
			} else {
				repo.EXPECT().GetUserByID(context.Background(), user.ID).Return(tc.result, tc.repoErr)
			}

			got, err := lookup(svc)

			assertError(t, err, tc.wantErr)
			assertUser(t, got, tc.result)
		})
	}
}

func TestUserService_GetUserByID(t *testing.T) {
	testUserLookup(t, false, func(svc service.UserService) (m.User, error) {
		return svc.GetUserByID(context.Background(), testActor(someID, m.RoleBuyer), someID)
	})
	t.Run("cannot read another user's account", func(t *testing.T) {
		repo := mock_service.NewMockUserRepo(gomock.NewController(t))
		svc := service.NewUserService(repo, bcrypt.MinCost)

		_, err := svc.GetUserByID(context.Background(), testActor(someID, m.RoleBuyer), someID+1)

		assertError(t, err, service.ErrNotYourUser)
	})
}

func TestUserService_GetAuthUserByID(t *testing.T) {
	testUserLookup(t, false, func(svc service.UserService) (m.User, error) {
		return svc.GetAuthUserByID(context.Background(), someID)
	})
}

func TestUserService_GetUserByEmail(t *testing.T) {
	testUserLookup(t, true, func(svc service.UserService) (m.User, error) {
		return svc.GetUserByEmail(context.Background(), testActor(someID, m.RoleAdmin), someEmail)
	})
	t.Run("non-admin is forbidden", func(t *testing.T) {
		repo := mock_service.NewMockUserRepo(gomock.NewController(t))
		svc := service.NewUserService(repo, bcrypt.MinCost)

		_, err := svc.GetUserByEmail(context.Background(), testActor(someID, m.RoleAnalyst), someEmail)

		assertError(t, err, service.ErrPermissionDenied)
	})
}

func TestUserService_GetAuthUserByEmail(t *testing.T) {
	testUserLookup(t, true, func(svc service.UserService) (m.User, error) {
		return svc.GetAuthUserByEmail(context.Background(), someEmail)
	})
}

func TestUserService_GetUsers(t *testing.T) {
	tests := []struct {
		name    string
		role    m.UserRole
		repoErr error
		wantErr error
	}{
		{name: "admin lists users", role: m.RoleAdmin},
		{name: "repository failure", role: m.RoleAdmin, repoErr: errors.New("query failed"), wantErr: service.ErrGetUsers},
		{name: "buyer is forbidden", role: m.RoleBuyer, wantErr: service.ErrPermissionDenied},
		{name: "analyst is forbidden", role: m.RoleAnalyst, wantErr: service.ErrPermissionDenied},
	}
	ctx := context.Background()
	opts := m.UserListOptions{Pagination: m.PaginationOpts{Page: 2, Limit: 10}}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := mock_service.NewMockUserRepo(gomock.NewController(t))
			svc := service.NewUserService(repo, bcrypt.MinCost)
			users := []m.User{buyerMother()}
			if tc.role == m.RoleAdmin {
				repo.EXPECT().GetUsers(ctx, opts).Return(users, tc.repoErr)
			}

			got, err := svc.GetUsers(ctx, testActor(someID, tc.role), opts)

			assertError(t, err, tc.wantErr)
			if tc.wantErr == nil && !reflect.DeepEqual(got, users) {
				t.Fatalf("unexpected users: got %+v, want %+v", got, users)
			}
		})
	}
}

func TestUserService_CreateUser(t *testing.T) {
	t.Run("admin registration is forbidden", func(t *testing.T) {
		svc := service.NewUserService(mock_service.NewMockUserRepo(gomock.NewController(t)), bcrypt.MinCost)

		_, err := svc.CreateUser(context.Background(), m.UserCreate{
			Email: someEmail, Password: someStrongPassword, FullName: someFullName, Role: m.RoleAdmin,
		})

		assertError(t, err, service.ErrPermissionDenied)
	})
	ctx := context.Background()

	someUserCreate := m.UserCreate{
		Password: someStrongPassword,
		Email:    someEmail,
		FullName: someFullName,
		Phone:    &testPhone,
		Role:     someRole,
	}
	tooLongPassword := someUserCreate
	tooLongPassword.Password = strings.Repeat("a", 73)

	type testCase struct {
		Description    string
		Create         m.UserCreate
		MockGetByEmail MockUserReturn
		MockCreate     *MockCreateReturn
		ExpectedID     int64
		ExpectedErr    error
	}

	tCases := []testCase{
		{
			Description:    "Success",
			Create:         someUserCreate,
			MockGetByEmail: MockUserReturn{Error: service.ErrNotFound},
			MockCreate:     &MockCreateReturn{ID: someID},
			ExpectedID:     someID,
		},

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
		{
			Description:    "password exceeds bcrypt input limit",
			Create:         tooLongPassword,
			MockGetByEmail: MockUserReturn{Error: service.ErrNotFound},
			ExpectedErr:    service.ErrHashingPassword,
		},
	}

	for _, tCase := range tCases {
		t.Run(tCase.Description, func(t *testing.T) {
			mock := mock_service.NewMockUserRepo(gomock.NewController(t))
			svc := service.NewUserService(mock, bcrypt.MinCost)

			mock.EXPECT().GetUserByEmail(ctx, tCase.Create.Email).Return(tCase.MockGetByEmail.User, tCase.MockGetByEmail.Error)
			if tCase.MockCreate != nil {
				mock.EXPECT().CreateUser(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, create m.UserCreate) (int64, error) {
					assertPasswordHash(t, create.Password, tCase.Create.Password)
					want := tCase.Create
					want.Password = create.Password
					if !reflect.DeepEqual(create, want) {
						t.Fatalf("unexpected create fields: got %+v, want %+v", create, want)
					}
					return tCase.MockCreate.ID, tCase.MockCreate.Error
				})
			}
			id, err := svc.CreateUser(ctx, tCase.Create)
			assertError(t, err, tCase.ExpectedErr)
			if id != tCase.ExpectedID {
				t.Fatalf("invalid id. expected: %v, got: %v", tCase.ExpectedID, id)
			}
		})
	}
}

func TestUserService_UpdateUser(t *testing.T) {
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
			mock := mock_service.NewMockUserRepo(gomock.NewController(t))
			svc := service.NewUserService(mock, bcrypt.MinCost)

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

func TestUserService_ChangePasswordUser(t *testing.T) {
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
		{
			Description:    "Success",
			UserID:         someID,
			OldPassword:    someStrongPassword,
			NewPassword:    "newStrongPassword123!",
			MockGetByID:    MockUserReturn{User: someUser},
			MockChangePass: ptrErr(nil),
		},

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
		{
			Description: "password exceeds bcrypt input limit",
			UserID:      someID,
			OldPassword: someStrongPassword,
			NewPassword: strings.Repeat("a", 73),
			MockGetByID: MockUserReturn{User: someUser},
			ExpectedErr: service.ErrHashingPassword,
		},
	}

	for _, tCase := range tCases {
		t.Run(tCase.Description, func(t *testing.T) {
			mock := mock_service.NewMockUserRepo(gomock.NewController(t))
			svc := service.NewUserService(mock, bcrypt.MinCost)

			mock.EXPECT().GetUserByID(ctx, tCase.UserID).Return(tCase.MockGetByID.User, tCase.MockGetByID.Error)
			if tCase.MockChangePass != nil {
				mock.EXPECT().ChangePasswordUser(ctx, tCase.UserID, gomock.Any()).DoAndReturn(func(_ context.Context, _ int64, hash string) error {
					assertPasswordHash(t, hash, tCase.NewPassword)
					return *tCase.MockChangePass
				})
			}
			err := svc.ChangePasswordUser(ctx, testActor(tCase.UserID, m.RoleBuyer), tCase.OldPassword, tCase.NewPassword)
			assertError(t, err, tCase.ExpectedErr)
		})
	}
}

func TestUserService_DeleteUserByID(t *testing.T) {
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
			mock := mock_service.NewMockUserRepo(gomock.NewController(t))
			svc := service.NewUserService(mock, bcrypt.MinCost)

			mock.EXPECT().DeleteUserByID(ctx, tCase.UserID).Return(tCase.MockError)
			err := svc.DeleteUserByID(ctx, testActor(tCase.UserID, m.RoleBuyer), tCase.UserID)
			assertError(t, err, tCase.ExpectedErr)
		})
	}
}

func ptrErr(err error) *error {
	return &err
}
