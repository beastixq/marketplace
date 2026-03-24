package service_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	mock_service "github.com/beastixq/marketplace/internal/mocks/service"
	m "github.com/beastixq/marketplace/internal/model"
	"github.com/beastixq/marketplace/internal/service"
	"go.uber.org/mock/gomock"
)

var someAddressUserID int64 = 100

var someAddress = m.Address{
	ID:        someID,
	UserID:    someAddressUserID,
	City:      "Moscow",
	Street:    "Baumanskaya 5",
	ZipCode:   "105005",
	IsDefault: true,
	CreatedAt: someTime,
}

type MockAddressReturn struct {
	Address m.Address
	Error   error
}

type MockAddressListReturn struct {
	Addresses []m.Address
	Error     error
}

func assertAddress(t *testing.T, got, want m.Address) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("invalid address. expected: %v, got: %v", want, got)
	}
}

func TestGetAddressesByUserID(t *testing.T) {
	mock := mock_service.NewMockAddressRepo(gomock.NewController(t))
	svc := service.NewAddressService(mock)
	ctx := context.Background()

	addresses := []m.Address{someAddress}

	type testCase struct {
		Description       string
		UserID            int64
		MockReturn        MockAddressListReturn
		ExpectedAddresses []m.Address
		ExpectedErr       error
	}

	tCases := []testCase{
		{
			Description:       "Success",
			UserID:            someAddressUserID,
			MockReturn:        MockAddressListReturn{Addresses: addresses},
			ExpectedAddresses: addresses,
		},
		{
			Description: "Success empty list",
			UserID:      someAddressUserID,
			MockReturn:  MockAddressListReturn{},
		},
		{
			Description: "Repo error",
			UserID:      someAddressUserID,
			MockReturn:  MockAddressListReturn{Error: errors.New("some repo error")},
			ExpectedErr: service.ErrGetAddressesByUserID,
		},
	}

	for _, tCase := range tCases {
		t.Run(tCase.Description, func(t *testing.T) {
			mock.EXPECT().GetAddressesByUserID(ctx, tCase.UserID).Return(tCase.MockReturn.Addresses, tCase.MockReturn.Error)
			addrs, err := svc.GetAddressesByUserID(ctx, tCase.UserID)
			assertError(t, err, tCase.ExpectedErr)
			if !reflect.DeepEqual(addrs, tCase.ExpectedAddresses) {
				t.Fatalf("invalid addresses. expected: %v, got: %v", tCase.ExpectedAddresses, addrs)
			}
		})
	}
}

func TestCreateAddress(t *testing.T) {
	mock := mock_service.NewMockAddressRepo(gomock.NewController(t))
	svc := service.NewAddressService(mock)
	ctx := context.Background()

	someAddressCreate := m.AddressCreate{
		UserID:    someAddressUserID,
		City:      "Moscow",
		Street:    "Baumanskaya 5",
		ZipCode:   "105005",
		IsDefault: true,
	}

	type testCase struct {
		Description string
		Create      m.AddressCreate
		MockReturn  MockCreateReturn
		ExpectedID  int64
		ExpectedErr error
	}

	tCases := []testCase{
		{
			Description: "Success",
			Create:      someAddressCreate,
			MockReturn:  MockCreateReturn{ID: someID},
			ExpectedID:  someID,
		},
		{
			Description: "Repo error",
			Create:      someAddressCreate,
			MockReturn:  MockCreateReturn{Error: errors.New("some repo error")},
			ExpectedErr: service.ErrCreateAddress,
		},
	}

	for _, tCase := range tCases {
		t.Run(tCase.Description, func(t *testing.T) {
			mock.EXPECT().CreateAddress(ctx, tCase.Create).Return(tCase.MockReturn.ID, tCase.MockReturn.Error)
			id, err := svc.CreateAddress(ctx, tCase.Create)
			assertError(t, err, tCase.ExpectedErr)
			if id != tCase.ExpectedID {
				t.Fatalf("invalid id. expected: %v, got: %v", tCase.ExpectedID, id)
			}
		})
	}
}

func TestUpdateAddress(t *testing.T) {
	mock := mock_service.NewMockAddressRepo(gomock.NewController(t))
	svc := service.NewAddressService(mock)
	ctx := context.Background()

	newCity := "Saint Petersburg"
	addressUpdate := m.AddressUpdate{
		City: &newCity,
	}

	updatedAddress := m.Address{
		ID:        someAddress.ID,
		UserID:    someAddress.UserID,
		City:      newCity,
		Street:    someAddress.Street,
		ZipCode:   someAddress.ZipCode,
		IsDefault: someAddress.IsDefault,
		CreatedAt: someAddress.CreatedAt,
	}

	type testCase struct {
		Description     string
		UserID          int64
		AddressID       int64
		Update          m.AddressUpdate
		MockGetByID     MockAddressReturn
		MockUpdate      *MockAddressReturn
		ExpectedAddress m.Address
		ExpectedErr     error
	}

	tCases := []testCase{
		// positives
		{
			Description:     "Success",
			UserID:          someAddressUserID,
			AddressID:       someID,
			Update:          addressUpdate,
			MockGetByID:     MockAddressReturn{Address: someAddress},
			MockUpdate:      &MockAddressReturn{Address: updatedAddress},
			ExpectedAddress: updatedAddress,
		},
		// negatives
		{
			Description: "Address not found",
			UserID:      someAddressUserID,
			AddressID:   123,
			Update:      addressUpdate,
			MockGetByID: MockAddressReturn{Error: service.ErrNotFound},
			ExpectedErr: service.ErrAddressNotFound,
		},
		{
			Description: "GetByID repo error",
			UserID:      someAddressUserID,
			AddressID:   someID,
			Update:      addressUpdate,
			MockGetByID: MockAddressReturn{Error: errors.New("some repo error")},
			ExpectedErr: service.ErrGetAddressByID,
		},
		{
			Description: "Not your address",
			UserID:      otherUserID,
			AddressID:   someID,
			Update:      addressUpdate,
			MockGetByID: MockAddressReturn{Address: someAddress},
			ExpectedErr: service.ErrNotYourAddress,
		},
		{
			Description: "UpdateAddress repo error",
			UserID:      someAddressUserID,
			AddressID:   someID,
			Update:      addressUpdate,
			MockGetByID: MockAddressReturn{Address: someAddress},
			MockUpdate:  &MockAddressReturn{Error: errors.New("some repo error")},
			ExpectedErr: service.ErrUpdateAddress,
		},
	}

	for _, tCase := range tCases {
		t.Run(tCase.Description, func(t *testing.T) {
			mock.EXPECT().GetAddressByID(ctx, tCase.AddressID).Return(tCase.MockGetByID.Address, tCase.MockGetByID.Error)
			if tCase.MockUpdate != nil {
				mock.EXPECT().UpdateAddress(ctx, tCase.AddressID, tCase.Update).Return(tCase.MockUpdate.Address, tCase.MockUpdate.Error)
			}
			addr, err := svc.UpdateAddress(ctx, tCase.UserID, tCase.AddressID, tCase.Update)
			assertError(t, err, tCase.ExpectedErr)
			assertAddress(t, addr, tCase.ExpectedAddress)
		})
	}
}

func TestDeleteAddressByID(t *testing.T) {
	mock := mock_service.NewMockAddressRepo(gomock.NewController(t))
	svc := service.NewAddressService(mock)
	ctx := context.Background()

	type testCase struct {
		Description string
		UserID      int64
		AddressID   int64
		MockGetByID MockAddressReturn
		MockDelete  *error
		ExpectedErr error
	}

	tCases := []testCase{
		{
			Description: "Success",
			UserID:      someAddressUserID,
			AddressID:   someID,
			MockGetByID: MockAddressReturn{Address: someAddress},
			MockDelete:  ptrErr(nil),
		},
		{
			Description: "Address not found",
			UserID:      someAddressUserID,
			AddressID:   123,
			MockGetByID: MockAddressReturn{Error: service.ErrNotFound},
			ExpectedErr: service.ErrAddressNotFound,
		},
		{
			Description: "GetByID repo error",
			UserID:      someAddressUserID,
			AddressID:   someID,
			MockGetByID: MockAddressReturn{Error: errors.New("some repo error")},
			ExpectedErr: service.ErrGetAddressByID,
		},
		{
			Description: "Not your address",
			UserID:      otherUserID,
			AddressID:   someID,
			MockGetByID: MockAddressReturn{Address: someAddress},
			ExpectedErr: service.ErrNotYourAddress,
		},
		{
			Description: "DeleteAddress repo error",
			UserID:      someAddressUserID,
			AddressID:   someID,
			MockGetByID: MockAddressReturn{Address: someAddress},
			MockDelete:  ptrErr(errors.New("some repo error")),
			ExpectedErr: service.ErrDeleteAddressByID,
		},
	}

	for _, tCase := range tCases {
		t.Run(tCase.Description, func(t *testing.T) {
			mock.EXPECT().GetAddressByID(ctx, tCase.AddressID).Return(tCase.MockGetByID.Address, tCase.MockGetByID.Error)
			if tCase.MockDelete != nil {
				mock.EXPECT().DeleteAddressByID(ctx, tCase.AddressID).Return(*tCase.MockDelete)
			}
			err := svc.DeleteAddressByID(ctx, tCase.UserID, tCase.AddressID)
			assertError(t, err, tCase.ExpectedErr)
		})
	}
}
