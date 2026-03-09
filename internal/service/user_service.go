package service

import (
	"context"
	"errors"
	"time"

	m "github.com/beastixq/marketplace/internal/model"
	"golang.org/x/crypto/bcrypt"
)

type UserRepo interface {
	GetUserByID(ctx context.Context, id int64) (u m.User, err error)
	GetUserByEmail(ctx context.Context, email string) (u m.User, err error)
	CreateUser(ctx context.Context, uc m.UserCreate) (id int64, err error)
	UpdateUser(ctx context.Context, id int64, uu m.UserUpdate) (u m.User, err error)
	ChangePasswordUser(ctx context.Context, id int64, newPassHash string) (err error)
	DeleteUserByID(ctx context.Context, id int64) (err error)
}

type SellerRepo interface {
	GetSellerByID(ctx context.Context, id int64) (s m.Seller, err error)
	GetSellerByUserID(ctx context.Context, userID int64) (s m.Seller, err error)
	GetSellerStats(ctx context.Context, sellerID int64, dateFrom time.Time, dateTo time.Time) (ss m.SellerStats, err error)
	CreateSeller(ctx context.Context, sc m.SellerCreate) (id int64, err error)
	UpdateSeller(ctx context.Context, id int64, su m.SellerUpdate) (s m.Seller, err error)
	DeleteSellerByID(ctx context.Context, id int64) (err error)
}

type AddressRepo interface {
	GetAddressByID(ctx context.Context, id int64) (a m.Address, err error)
	GetAddressesByUserID(ctx context.Context, userID int64) (ads []m.Address, err error)
	CreateAddress(ctx context.Context, ac m.AddressCreate) (id int64, err error)
	UpdateAddress(ctx context.Context, id int64, au m.AddressUpdate) (a m.Address, err error)
	DeleteAddressByID(ctx context.Context, id int64) (err error)
}

type UserService struct {
	repo UserRepo
}

func NewUserService(ur UserRepo) UserService {
	return UserService{repo: ur}
}

func (us UserService) CreateUser(ctx context.Context, uc m.UserCreate) (id int64, err error) {
	_, err = us.repo.GetUserByEmail(ctx, uc.Email)
	if err == nil {
		return 0, ErrAccountWithEmailExists
	} else if !errors.Is(err, ErrNotFound) {
		return 0, ErrGetUserByEmail
	}
	hashPass, err := bcrypt.GenerateFromPassword([]byte(uc.Password), 4)
	if err != nil {
		return 0, ErrHashingPassword
	}
	uc.Password = string(hashPass)

	id, err = us.repo.CreateUser(ctx, uc)
	if err != nil {
		return 0, ErrCreateUser
	}
	return id, nil
}

func (us UserService) UpdateUser(ctx context.Context, id int64, uu m.UserUpdate) (u m.User, err error) {
	user, err := us.repo.UpdateUser(ctx, id, uu)
	if err != nil {
		if errors.Is(err, ErrNoChangesInUpdate) {
			return m.User{}, ErrNoChangesInUpdate
		} else {
			return m.User{}, ErrUpdateUser
		}
	}
	return user, nil
}

func (us UserService) GetUserByID(ctx context.Context, id int64) (u m.User, err error) {
	user, err := us.repo.GetUserByID(ctx, id)
	if err != nil {
		return m.User{}, ErrGetUserByID
	}
	return user, nil
}

func (us UserService) GetUserByEmail(ctx context.Context, email string) (u m.User, err error) {
	user, err := us.repo.GetUserByEmail(ctx, email)
	if err != nil {
		return m.User{}, ErrGetUserByEmail
	}
	return user, nil
}

func (us UserService) DeleteUserByID(ctx context.Context, id int64) (err error) {
	if err := us.repo.DeleteUserByID(ctx, id); err != nil {
		return ErrDeleteUser
	}
	return nil
}

func (us UserService) ChangePasswordUser(ctx context.Context, id int64, newPass string) (err error) {
	newPassHash, err := bcrypt.GenerateFromPassword([]byte(newPass), 4)
	if err != nil {
		return ErrHashingPassword
	}
	if err := us.repo.ChangePasswordUser(ctx, id, string(newPassHash)); err != nil {
		return ErrChangePasswordUser
	}
	return nil
}
