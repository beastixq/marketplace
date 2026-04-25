package service

import (
	"context"
	"errors"
	"fmt"

	m "github.com/beastixq/marketplace/internal/model"
)

//go:generate mockgen -package mock_service -destination ../mocks/service/mock_address_repo.go github.com/beastixq/marketplace/internal/service AddressRepo
type AddressRepo interface {
	GetAddressByID(ctx context.Context, id int64) (a m.Address, err error)
	GetAddressesByUserID(ctx context.Context, userID int64) (ads []m.Address, err error)
	CreateAddress(ctx context.Context, ac m.AddressCreate) (id int64, err error)
	UpdateAddress(ctx context.Context, id int64, au m.AddressUpdate) (a m.Address, err error)
	DeleteAddressByID(ctx context.Context, id int64) (err error)
}

type AddressService struct {
	addrRepo AddressRepo
}

func NewAddressService(addressRepo AddressRepo) AddressService {
	return AddressService{addrRepo: addressRepo}
}

func (as AddressService) GetAddressesByUserID(ctx context.Context, actor Actor) (addrs []m.Address, err error) {
	addrs, err = as.addrRepo.GetAddressesByUserID(ctx, actor.UserID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrGetAddressesByUserID, err)
	}
	return addrs, nil
}

func (as AddressService) CreateAddress(ctx context.Context, actor Actor, ac m.AddressCreate) (id int64, err error) {
	ac.UserID = actor.UserID
	id, err = as.addrRepo.CreateAddress(ctx, ac)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrCreateAddress, err)
	}
	return id, nil
}

func (as AddressService) UpdateAddress(ctx context.Context, actor Actor, id int64, au m.AddressUpdate) (a m.Address, err error) {
	a, err = as.addrRepo.GetAddressByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return m.Address{}, ErrAddressNotFound
		}
		return m.Address{}, fmt.Errorf("%w: %v", ErrGetAddressByID, err)
	}
	if a.UserID != actor.UserID {
		return m.Address{}, ErrNotYourAddress
	}

	a, err = as.addrRepo.UpdateAddress(ctx, id, au)
	if err != nil {
		return m.Address{}, fmt.Errorf("%w: %v", ErrUpdateAddress, err)
	}
	return a, nil
}

func (as AddressService) DeleteAddressByID(ctx context.Context, actor Actor, id int64) (err error) {
	a, err := as.addrRepo.GetAddressByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return ErrAddressNotFound
		}
		return fmt.Errorf("%w: %v", ErrGetAddressByID, err)
	}
	if a.UserID != actor.UserID {
		return ErrNotYourAddress
	}

	err = as.addrRepo.DeleteAddressByID(ctx, id)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDeleteAddressByID, err)
	}
	return nil
}
