package service

import (
	"context"
	"time"

	m "github.com/beastixq/marketplace/internal/model"
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


