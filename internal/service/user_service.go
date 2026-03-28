package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	m "github.com/beastixq/marketplace/internal/model"

	"golang.org/x/crypto/bcrypt"
)

//go:generate mockgen -package mock_service -destination ../mocks/service/mock_user_repo.go github.com/beastixq/marketplace/internal/service UserRepo
type UserRepo interface {
	GetUserByID(ctx context.Context, id int64) (u m.User, err error)
	GetUserByEmail(ctx context.Context, email string) (u m.User, err error)
	CreateUser(ctx context.Context, uc m.UserCreate) (id int64, err error)
	UpdateUser(ctx context.Context, id int64, uu m.UserUpdate) (u m.User, err error)
	ChangePasswordUser(ctx context.Context, id int64, newPassHash string) (err error)
	DeleteUserByID(ctx context.Context, id int64) (err error)
}

type UserService struct {
	repo       UserRepo
	bcryptCost int
}

func NewUserService(ur UserRepo, bcryptCost int) UserService {
	return UserService{repo: ur, bcryptCost: bcryptCost}
}

func (us UserService) CreateUser(ctx context.Context, uc m.UserCreate) (id int64, err error) {
	_, err = us.repo.GetUserByEmail(ctx, uc.Email)
	if err == nil {
		return 0, ErrAccountWithEmailAlreadyExists
	}
	if !errors.Is(err, ErrNotFound) {
		return 0, fmt.Errorf("%w: %v", ErrGetUserByEmail, err)
	}
	hashPass, err := bcrypt.GenerateFromPassword([]byte(uc.Password), us.bcryptCost)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrHashingPassword, err)
	}
	uc.Password = string(hashPass)

	id, err = us.repo.CreateUser(ctx, uc)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrCreateUser, err)
	}
	return id, nil
}

func (us UserService) UpdateUser(ctx context.Context, id int64, uu m.UserUpdate) (u m.User, err error) {
	_, err = us.repo.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return m.User{}, ErrUserNotFound
		}
		return m.User{}, fmt.Errorf("%w: %v", ErrGetUserByID, err)
	}
	user, err := us.repo.UpdateUser(ctx, id, uu)
	if err != nil {
		if errors.Is(err, ErrNoChangesInUpdate) {
			return m.User{}, ErrNoChangesInUpdate
		}
		errStr := err.Error()
		switch {
		case strings.Contains(errStr, "users_phone_key"):
			return m.User{}, ErrPhoneAlreadyExists
		case strings.Contains(errStr, "users_email_key"):
			return m.User{}, ErrEmailAlreadyInUse
		}
		return m.User{}, fmt.Errorf("%w: %v", ErrUpdateUser, err)
	}
	return user, nil
}

func (us UserService) GetUserByID(ctx context.Context, id int64) (u m.User, err error) {
	user, err := us.repo.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return m.User{}, ErrUserNotFound
		}
		return m.User{}, fmt.Errorf("%w: %v", ErrGetUserByID, err)
	}
	return user, nil
}

func (us UserService) GetUserByEmail(ctx context.Context, email string) (u m.User, err error) {
	user, err := us.repo.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return m.User{}, ErrUserNotFound
		}
		return m.User{}, fmt.Errorf("%w: %v", ErrGetUserByEmail, err)
	}
	return user, nil
}

func (us UserService) DeleteUserByID(ctx context.Context, id int64) (err error) {
	if err := us.repo.DeleteUserByID(ctx, id); err != nil {
		return fmt.Errorf("%w: %v", ErrDeleteUser, err)
	}
	return nil
}

func (us UserService) ChangePasswordUser(ctx context.Context, id int64, oldPass, newPass string) (err error) {
	user, err := us.repo.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return ErrUserNotFound
		}
		return fmt.Errorf("%w: %v", ErrGetUserByID, err)
	}
	if err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(oldPass)); err != nil {
		return ErrWrongPassword
	}

	newPassHash, err := bcrypt.GenerateFromPassword([]byte(newPass), us.bcryptCost)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrHashingPassword, err)
	}
	if err := us.repo.ChangePasswordUser(ctx, id, string(newPassHash)); err != nil {
		return fmt.Errorf("%w: %v", ErrChangePasswordUser, err)
	}
	return nil
}
