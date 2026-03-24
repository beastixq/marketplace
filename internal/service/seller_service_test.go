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
	"github.com/shopspring/decimal"
	"go.uber.org/mock/gomock"
)

const someCompanyName = "Some Company"

var someDescription = "Some description"
var someRating = float32(4.5)
var someSellerUserID int64 = 100
var otherUserID int64 = 999

var someSeller = m.Seller{
	ID:          someID,
	UserID:      someSellerUserID,
	CompanyName: someCompanyName,
	Description: &someDescription,
	Rating:      &someRating,
	CreatedAt:   someTime,
}

type MockSellerReturn struct {
	Seller m.Seller
	Error  error
}

type MockSellerStatsReturn struct {
	Stats m.SellerStats
	Error error
}

func assertSeller(t *testing.T, got, want m.Seller) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("invalid seller. expected: %v, got: %v", want, got)
	}
}

func assertSellerStats(t *testing.T, got, want m.SellerStats) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("invalid seller stats. expected: %v, got: %v", want, got)
	}
}

func TestGetSellerByID(t *testing.T) {
	mock := mock_service.NewMockSellerRepo(gomock.NewController(t))
	svc := service.NewSellerService(mock)
	ctx := context.Background()

	type testCase struct {
		Description    string
		SellerID       int64
		MockReturn     MockSellerReturn
		ExpectedSeller m.Seller
		ExpectedErr    error
	}

	tCases := []testCase{
		{
			Description:    "Success",
			SellerID:       someID,
			MockReturn:     MockSellerReturn{Seller: someSeller},
			ExpectedSeller: someSeller,
		},
		{
			Description: "Seller not found",
			SellerID:    123,
			MockReturn:  MockSellerReturn{Error: service.ErrNotFound},
			ExpectedErr: service.ErrSellerNotFound,
		},
		{
			Description: "Repo error",
			SellerID:    someID,
			MockReturn:  MockSellerReturn{Error: errors.New("some repo error")},
			ExpectedErr: service.ErrGetSellerByID,
		},
	}

	for _, tCase := range tCases {
		t.Run(tCase.Description, func(t *testing.T) {
			mock.EXPECT().GetSellerByID(ctx, tCase.SellerID).Return(tCase.MockReturn.Seller, tCase.MockReturn.Error)
			seller, err := svc.GetSellerByID(ctx, tCase.SellerID)
			assertError(t, err, tCase.ExpectedErr)
			assertSeller(t, seller, tCase.ExpectedSeller)
		})
	}
}

func TestGetSellerByUserID(t *testing.T) {
	mock := mock_service.NewMockSellerRepo(gomock.NewController(t))
	svc := service.NewSellerService(mock)
	ctx := context.Background()

	type testCase struct {
		Description    string
		UserID         int64
		MockReturn     MockSellerReturn
		ExpectedSeller m.Seller
		ExpectedErr    error
	}

	tCases := []testCase{
		{
			Description:    "Success",
			UserID:         someSellerUserID,
			MockReturn:     MockSellerReturn{Seller: someSeller},
			ExpectedSeller: someSeller,
		},
		{
			Description: "Seller not found",
			UserID:      123,
			MockReturn:  MockSellerReturn{Error: service.ErrNotFound},
			ExpectedErr: service.ErrSellerNotFound,
		},
		{
			Description: "Repo error",
			UserID:      someSellerUserID,
			MockReturn:  MockSellerReturn{Error: errors.New("some repo error")},
			ExpectedErr: service.ErrGetSellerByID,
		},
	}

	for _, tCase := range tCases {
		t.Run(tCase.Description, func(t *testing.T) {
			mock.EXPECT().GetSellerByUserID(ctx, tCase.UserID).Return(tCase.MockReturn.Seller, tCase.MockReturn.Error)
			seller, err := svc.GetSellerByUserID(ctx, tCase.UserID)
			assertError(t, err, tCase.ExpectedErr)
			assertSeller(t, seller, tCase.ExpectedSeller)
		})
	}
}

func TestCreateSeller(t *testing.T) {
	mock := mock_service.NewMockSellerRepo(gomock.NewController(t))
	svc := service.NewSellerService(mock)
	ctx := context.Background()

	someSellerCreate := m.SellerCreate{
		UserID:      someSellerUserID,
		CompanyName: someCompanyName,
		Description: &someDescription,
	}

	type testCase struct {
		Description string
		Create      m.SellerCreate
		MockReturn  MockCreateReturn
		ExpectedID  int64
		ExpectedErr error
	}

	tCases := []testCase{
		{
			Description: "Success",
			Create:      someSellerCreate,
			MockReturn:  MockCreateReturn{ID: someID},
			ExpectedID:  someID,
		},
		{
			Description: "Repo error",
			Create:      someSellerCreate,
			MockReturn:  MockCreateReturn{Error: errors.New("some repo error")},
			ExpectedErr: service.ErrCreateSeller,
		},
	}

	for _, tCase := range tCases {
		t.Run(tCase.Description, func(t *testing.T) {
			mock.EXPECT().CreateSeller(ctx, tCase.Create).Return(tCase.MockReturn.ID, tCase.MockReturn.Error)
			id, err := svc.CreateSeller(ctx, tCase.Create)
			assertError(t, err, tCase.ExpectedErr)
			if id != tCase.ExpectedID {
				t.Fatalf("invalid id. expected: %v, got: %v", tCase.ExpectedID, id)
			}
		})
	}
}

func TestUpdateSeller(t *testing.T) {
	mock := mock_service.NewMockSellerRepo(gomock.NewController(t))
	svc := service.NewSellerService(mock)
	ctx := context.Background()

	newCompanyName := "New Company"
	sellerUpdate := m.SellerUpdate{
		CompanyName: &newCompanyName,
	}

	updatedSeller := m.Seller{
		ID:          someSeller.ID,
		UserID:      someSeller.UserID,
		CompanyName: newCompanyName,
		Description: someSeller.Description,
		Rating:      someSeller.Rating,
		CreatedAt:   someSeller.CreatedAt,
	}

	type testCase struct {
		Description    string
		UserID         int64
		SellerID       int64
		Update         m.SellerUpdate
		MockGetByID    MockSellerReturn
		MockUpdate     *MockSellerReturn
		ExpectedSeller m.Seller
		ExpectedErr    error
	}

	tCases := []testCase{
		// positives
		{
			Description:    "Success",
			UserID:         someSellerUserID,
			SellerID:       someID,
			Update:         sellerUpdate,
			MockGetByID:    MockSellerReturn{Seller: someSeller},
			MockUpdate:     &MockSellerReturn{Seller: updatedSeller},
			ExpectedSeller: updatedSeller,
		},
		// negatives
		{
			Description: "Seller not found",
			UserID:      someSellerUserID,
			SellerID:    123,
			Update:      sellerUpdate,
			MockGetByID: MockSellerReturn{Error: service.ErrNotFound},
			ExpectedErr: service.ErrSellerNotFound,
		},
		{
			Description: "GetByID repo error",
			UserID:      someSellerUserID,
			SellerID:    someID,
			Update:      sellerUpdate,
			MockGetByID: MockSellerReturn{Error: errors.New("some repo error")},
			ExpectedErr: service.ErrGetSellerByID,
		},
		{
			Description: "Not your seller",
			UserID:      otherUserID,
			SellerID:    someID,
			Update:      sellerUpdate,
			MockGetByID: MockSellerReturn{Seller: someSeller},
			ExpectedErr: service.ErrNotYourSeller,
		},
		{
			Description: "UpdateSeller repo error",
			UserID:      someSellerUserID,
			SellerID:    someID,
			Update:      sellerUpdate,
			MockGetByID: MockSellerReturn{Seller: someSeller},
			MockUpdate:  &MockSellerReturn{Error: errors.New("some repo error")},
			ExpectedErr: service.ErrUpdateSeller,
		},
	}

	for _, tCase := range tCases {
		t.Run(tCase.Description, func(t *testing.T) {
			mock.EXPECT().GetSellerByID(ctx, tCase.SellerID).Return(tCase.MockGetByID.Seller, tCase.MockGetByID.Error)
			if tCase.MockUpdate != nil {
				mock.EXPECT().UpdateSeller(ctx, tCase.SellerID, tCase.Update).Return(tCase.MockUpdate.Seller, tCase.MockUpdate.Error)
			}
			seller, err := svc.UpdateSeller(ctx, tCase.UserID, tCase.SellerID, tCase.Update)
			assertError(t, err, tCase.ExpectedErr)
			assertSeller(t, seller, tCase.ExpectedSeller)
		})
	}
}

func TestDeleteSellerByID(t *testing.T) {
	mock := mock_service.NewMockSellerRepo(gomock.NewController(t))
	svc := service.NewSellerService(mock)
	ctx := context.Background()

	type testCase struct {
		Description string
		UserID      int64
		SellerID    int64
		MockGetByID MockSellerReturn
		MockDelete  *error
		ExpectedErr error
	}

	tCases := []testCase{
		{
			Description: "Success",
			UserID:      someSellerUserID,
			SellerID:    someID,
			MockGetByID: MockSellerReturn{Seller: someSeller},
			MockDelete:  ptrErr(nil),
		},
		{
			Description: "Seller not found",
			UserID:      someSellerUserID,
			SellerID:    123,
			MockGetByID: MockSellerReturn{Error: service.ErrNotFound},
			ExpectedErr: service.ErrSellerNotFound,
		},
		{
			Description: "GetByID repo error",
			UserID:      someSellerUserID,
			SellerID:    someID,
			MockGetByID: MockSellerReturn{Error: errors.New("some repo error")},
			ExpectedErr: service.ErrGetSellerByID,
		},
		{
			Description: "Not your seller",
			UserID:      otherUserID,
			SellerID:    someID,
			MockGetByID: MockSellerReturn{Seller: someSeller},
			ExpectedErr: service.ErrNotYourSeller,
		},
		{
			Description: "DeleteSeller repo error",
			UserID:      someSellerUserID,
			SellerID:    someID,
			MockGetByID: MockSellerReturn{Seller: someSeller},
			MockDelete:  ptrErr(errors.New("some repo error")),
			ExpectedErr: service.ErrDeleteSellerByID,
		},
	}

	for _, tCase := range tCases {
		t.Run(tCase.Description, func(t *testing.T) {
			mock.EXPECT().GetSellerByID(ctx, tCase.SellerID).Return(tCase.MockGetByID.Seller, tCase.MockGetByID.Error)
			if tCase.MockDelete != nil {
				mock.EXPECT().DeleteSellerByID(ctx, tCase.SellerID).Return(*tCase.MockDelete)
			}
			err := svc.DeleteSellerByID(ctx, tCase.UserID, tCase.SellerID)
			assertError(t, err, tCase.ExpectedErr)
		})
	}
}

func TestGetSellerStats(t *testing.T) {
	mock := mock_service.NewMockSellerRepo(gomock.NewController(t))
	svc := service.NewSellerService(mock)
	ctx := context.Background()

	dateFrom := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	dateTo := time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC)

	someStats := m.SellerStats{
		TotalOrders:    150,
		TotalRevenue:   decimal.NewFromFloat(50000.00),
		AvgOrderValue:  decimal.NewFromFloat(333.33),
		TopProductName: "Best Product",
	}

	type testCase struct {
		Description   string
		UserID        int64
		SellerID      int64
		DateFrom      time.Time
		DateTo        time.Time
		MockGetByID   MockSellerReturn
		MockStats     *MockSellerStatsReturn
		ExpectedStats m.SellerStats
		ExpectedErr   error
	}

	tCases := []testCase{
		// positives
		{
			Description:   "Success",
			UserID:        someSellerUserID,
			SellerID:      someID,
			DateFrom:      dateFrom,
			DateTo:        dateTo,
			MockGetByID:   MockSellerReturn{Seller: someSeller},
			MockStats:     &MockSellerStatsReturn{Stats: someStats},
			ExpectedStats: someStats,
		},
		// negatives
		{
			Description: "Seller not found",
			UserID:      someSellerUserID,
			SellerID:    123,
			DateFrom:    dateFrom,
			DateTo:      dateTo,
			MockGetByID: MockSellerReturn{Error: service.ErrNotFound},
			ExpectedErr: service.ErrSellerNotFound,
		},
		{
			Description: "GetByID repo error",
			UserID:      someSellerUserID,
			SellerID:    someID,
			DateFrom:    dateFrom,
			DateTo:      dateTo,
			MockGetByID: MockSellerReturn{Error: errors.New("some repo error")},
			ExpectedErr: service.ErrGetSellerByID,
		},
		{
			Description: "Not your seller",
			UserID:      otherUserID,
			SellerID:    someID,
			DateFrom:    dateFrom,
			DateTo:      dateTo,
			MockGetByID: MockSellerReturn{Seller: someSeller},
			ExpectedErr: service.ErrNotYourSeller,
		},
		{
			Description: "GetSellerStats repo error",
			UserID:      someSellerUserID,
			SellerID:    someID,
			DateFrom:    dateFrom,
			DateTo:      dateTo,
			MockGetByID: MockSellerReturn{Seller: someSeller},
			MockStats:   &MockSellerStatsReturn{Error: errors.New("some repo error")},
			ExpectedErr: service.ErrGetSellerStats,
		},
	}

	for _, tCase := range tCases {
		t.Run(tCase.Description, func(t *testing.T) {
			mock.EXPECT().GetSellerByID(ctx, tCase.SellerID).Return(tCase.MockGetByID.Seller, tCase.MockGetByID.Error)
			if tCase.MockStats != nil {
				mock.EXPECT().GetSellerStats(ctx, tCase.SellerID, tCase.DateFrom, tCase.DateTo).Return(tCase.MockStats.Stats, tCase.MockStats.Error)
			}
			stats, err := svc.GetSellerStats(ctx, tCase.UserID, tCase.SellerID, tCase.DateFrom, tCase.DateTo)
			assertError(t, err, tCase.ExpectedErr)
			assertSellerStats(t, stats, tCase.ExpectedStats)
		})
	}
}
