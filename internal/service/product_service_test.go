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

var someProductDescription = "A great product"
var someProductPrice = decimal.NewFromFloat(99.99)
var someProductStockQuantity = 50

var someProduct = m.Product{
	ID:            someID,
	SellerID:      someID,
	Name:          "Test Product",
	Description:   &someProductDescription,
	Price:         someProductPrice,
	StockQuantity: someProductStockQuantity,
	CreatedAt:     someTime,
}

type MockProductReturn struct {
	Product m.Product
	Error   error
}

type MockProductListReturn struct {
	Products []m.Product
	Error    error
}

type MockPriceHistoryReturn struct {
	History []m.ProductPriceHistory
	Error   error
}

type MockReviewListReturn struct {
	Reviews []m.Review
	Error   error
}

func assertProduct(t *testing.T, got, want m.Product) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("invalid product. expected: %v, got: %v", want, got)
	}
}

func newProductService(ctrl *gomock.Controller) (service.ProductService, *mock_service.MockProductRepo, *mock_service.MockReviewRepo, *mock_service.MockSellerRepo) {
	productMock := mock_service.NewMockProductRepo(ctrl)
	reviewMock := mock_service.NewMockReviewRepo(ctrl)
	sellerMock := mock_service.NewMockSellerRepo(ctrl)
	svc := service.NewProductService(productMock, reviewMock, sellerMock, passThroughTxManager{})
	return svc, productMock, reviewMock, sellerMock
}

func TestGetProducts(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc, productMock, _, _ := newProductService(ctrl)
	ctx := context.Background()

	catalogOpts := m.CatalogOptions{}

	type testCase struct {
		Description string
		MockReturn  MockProductListReturn
		ExpectedErr error
		ExpectEmpty bool
	}

	tCases := []testCase{
		{
			Description: "Success",
			MockReturn:  MockProductListReturn{Products: []m.Product{someProduct}},
		},
		{
			Description: "Empty result",
			MockReturn:  MockProductListReturn{Products: []m.Product{}},
			ExpectedErr: service.ErrProductNotFound,
		},
		{
			Description: "Repo error",
			MockReturn:  MockProductListReturn{Error: errors.New("db error")},
			ExpectedErr: service.ErrGetProducts,
		},
	}

	for _, tCase := range tCases {
		t.Run(tCase.Description, func(t *testing.T) {
			productMock.EXPECT().GetProducts(ctx, catalogOpts).Return(tCase.MockReturn.Products, tCase.MockReturn.Error)

			products, err := svc.GetProducts(ctx, catalogOpts)
			assertError(t, err, tCase.ExpectedErr)
			if tCase.ExpectedErr == nil && len(products) != len(tCase.MockReturn.Products) {
				t.Fatalf("expected %d products, got %d", len(tCase.MockReturn.Products), len(products))
			}
		})
	}
}

func TestGetProductByID(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc, productMock, _, _ := newProductService(ctrl)
	ctx := context.Background()

	type testCase struct {
		Description string
		MockReturn  MockProductReturn
		ExpectedErr error
	}

	tCases := []testCase{
		{
			Description: "Success",
			MockReturn:  MockProductReturn{Product: someProduct},
		},
		{
			Description: "Not found",
			MockReturn:  MockProductReturn{Error: service.ErrNotFound},
			ExpectedErr: service.ErrProductNotFound,
		},
		{
			Description: "Repo error",
			MockReturn:  MockProductReturn{Error: errors.New("db error")},
			ExpectedErr: service.ErrGetProductByID,
		},
	}

	for _, tCase := range tCases {
		t.Run(tCase.Description, func(t *testing.T) {
			productMock.EXPECT().GetProductByID(ctx, someID).Return(tCase.MockReturn.Product, tCase.MockReturn.Error)

			product, err := svc.GetProductByID(ctx, someID)
			assertError(t, err, tCase.ExpectedErr)
			if tCase.ExpectedErr == nil {
				assertProduct(t, product, tCase.MockReturn.Product)
			}
		})
	}
}

func TestGetProductPriceHistory(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc, productMock, _, _ := newProductService(ctrl)
	ctx := context.Background()

	dateFrom := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	dateTo := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)

	someHistory := []m.ProductPriceHistory{
		{
			ID:        1,
			ProductID: someID,
			OldPrice:  decimal.NewFromFloat(79.99),
			NewPrice:  someProductPrice,
			ChangedAt: someTime,
			ChangedBy: "trigger",
		},
	}

	type testCase struct {
		Description      string
		MockGetByID      MockProductReturn
		MockPriceHistory *MockPriceHistoryReturn
		ExpectedErr      error
	}

	tCases := []testCase{
		{
			Description:      "Success",
			MockGetByID:      MockProductReturn{Product: someProduct},
			MockPriceHistory: &MockPriceHistoryReturn{History: someHistory},
		},
		{
			Description: "Product not found",
			MockGetByID: MockProductReturn{Error: service.ErrNotFound},
			ExpectedErr: service.ErrProductNotFound,
		},
		{
			Description: "GetProductByID error",
			MockGetByID: MockProductReturn{Error: errors.New("db error")},
			ExpectedErr: service.ErrGetProductByID,
		},
		{
			Description:      "Price history repo error",
			MockGetByID:      MockProductReturn{Product: someProduct},
			MockPriceHistory: &MockPriceHistoryReturn{Error: errors.New("db error")},
			ExpectedErr:      service.ErrGetProductPriceHistory,
		},
	}

	for _, tCase := range tCases {
		t.Run(tCase.Description, func(t *testing.T) {
			productMock.EXPECT().GetProductByID(ctx, someID).Return(tCase.MockGetByID.Product, tCase.MockGetByID.Error)
			if tCase.MockPriceHistory != nil {
				productMock.EXPECT().GetProductPriceHistory(ctx, someID, dateFrom, dateTo).Return(tCase.MockPriceHistory.History, tCase.MockPriceHistory.Error)
			}

			history, err := svc.GetProductPriceHistory(ctx, someID, dateFrom, dateTo)
			assertError(t, err, tCase.ExpectedErr)
			if tCase.ExpectedErr == nil && !reflect.DeepEqual(history, tCase.MockPriceHistory.History) {
				t.Fatalf("invalid price history. expected: %v, got: %v", tCase.MockPriceHistory.History, history)
			}
		})
	}
}

func TestGetReviewsByProductID(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc, productMock, reviewMock, _ := newProductService(ctrl)
	ctx := context.Background()

	opts := m.PaginationOpts{Page: 1, Limit: 10}
	someReviews := []m.Review{someReview}

	type testCase struct {
		Description string
		MockGetByID MockProductReturn
		MockReviews *MockReviewListReturn
		ExpectedErr error
	}

	tCases := []testCase{
		{
			Description: "Success",
			MockGetByID: MockProductReturn{Product: someProduct},
			MockReviews: &MockReviewListReturn{Reviews: someReviews},
		},
		{
			Description: "Product not found",
			MockGetByID: MockProductReturn{Error: service.ErrNotFound},
			ExpectedErr: service.ErrProductNotFound,
		},
		{
			Description: "GetProductByID error",
			MockGetByID: MockProductReturn{Error: errors.New("db error")},
			ExpectedErr: service.ErrGetProductByID,
		},
		{
			Description: "Reviews repo error",
			MockGetByID: MockProductReturn{Product: someProduct},
			MockReviews: &MockReviewListReturn{Error: errors.New("db error")},
			ExpectedErr: service.ErrGetReviewsByProductID,
		},
	}

	for _, tCase := range tCases {
		t.Run(tCase.Description, func(t *testing.T) {
			productMock.EXPECT().GetProductByID(ctx, someID).Return(tCase.MockGetByID.Product, tCase.MockGetByID.Error)
			if tCase.MockReviews != nil {
				reviewMock.EXPECT().GetReviewsByProductID(ctx, someID, opts).Return(tCase.MockReviews.Reviews, tCase.MockReviews.Error)
			}

			reviews, err := svc.GetReviewsByProductID(ctx, someID, opts)
			assertError(t, err, tCase.ExpectedErr)
			if tCase.ExpectedErr == nil && !reflect.DeepEqual(reviews, tCase.MockReviews.Reviews) {
				t.Fatalf("invalid reviews. expected: %v, got: %v", tCase.MockReviews.Reviews, reviews)
			}
		})
	}
}

func TestCreateProduct(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc, productMock, _, sellerMock := newProductService(ctrl)
	ctx := context.Background()

	productCreate := m.ProductCreate{
		SellerID:      someID,
		Name:          "Test Product",
		Description:   &someProductDescription,
		Price:         someProductPrice,
		StockQuantity: someProductStockQuantity,
	}

	type testCase struct {
		Description  string
		UserID       int64
		MockSeller   MockSellerReturn
		MockCreate   *MockCreateReturn
		ExpectedErr  error
	}

	tCases := []testCase{
		{
			Description: "Success",
			UserID:      someSellerUserID,
			MockSeller:  MockSellerReturn{Seller: someSeller},
			MockCreate:  &MockCreateReturn{ID: someID},
		},
		{
			Description: "Seller not found",
			UserID:      someSellerUserID,
			MockSeller:  MockSellerReturn{Error: service.ErrNotFound},
			ExpectedErr: service.ErrSellerNotFound,
		},
		{
			Description: "GetSellerByID error",
			UserID:      someSellerUserID,
			MockSeller:  MockSellerReturn{Error: errors.New("db error")},
			ExpectedErr: service.ErrGetSellerByID,
		},
		{
			Description: "Not your seller",
			UserID:      otherUserID,
			MockSeller:  MockSellerReturn{Seller: someSeller},
			ExpectedErr: service.ErrNotYourSeller,
		},
		{
			Description: "CreateProduct repo error",
			UserID:      someSellerUserID,
			MockSeller:  MockSellerReturn{Seller: someSeller},
			MockCreate:  &MockCreateReturn{Error: errors.New("db error")},
			ExpectedErr: service.ErrCreateProduct,
		},
	}

	for _, tCase := range tCases {
		t.Run(tCase.Description, func(t *testing.T) {
			sellerMock.EXPECT().GetSellerByID(ctx, productCreate.SellerID).Return(tCase.MockSeller.Seller, tCase.MockSeller.Error)
			if tCase.MockCreate != nil {
				productMock.EXPECT().CreateProduct(ctx, productCreate).Return(tCase.MockCreate.ID, tCase.MockCreate.Error)
			}

			id, err := svc.CreateProduct(ctx, testActor(tCase.UserID, m.RoleSeller), productCreate)
			assertError(t, err, tCase.ExpectedErr)
			if tCase.ExpectedErr == nil && id != tCase.MockCreate.ID {
				t.Fatalf("expected id %d, got %d", tCase.MockCreate.ID, id)
			}
		})
	}
}

func TestUpdateProduct(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc, productMock, _, sellerMock := newProductService(ctrl)
	ctx := context.Background()

	newName := "Updated Product"
	productUpdate := m.ProductUpdate{Name: &newName}
	updatedProduct := someProduct
	updatedProduct.Name = newName

	type testCase struct {
		Description string
		UserID      int64
		MockGetByID MockProductReturn
		MockSeller  *MockSellerReturn
		MockUpdate  *MockProductReturn
		ExpectedErr error
	}

	tCases := []testCase{
		{
			Description: "Success",
			UserID:      someSellerUserID,
			MockGetByID: MockProductReturn{Product: someProduct},
			MockSeller:  &MockSellerReturn{Seller: someSeller},
			MockUpdate:  &MockProductReturn{Product: updatedProduct},
		},
		{
			Description: "Product not found",
			UserID:      someSellerUserID,
			MockGetByID: MockProductReturn{Error: service.ErrNotFound},
			ExpectedErr: service.ErrProductNotFound,
		},
		{
			Description: "GetProductByID error",
			UserID:      someSellerUserID,
			MockGetByID: MockProductReturn{Error: errors.New("db error")},
			ExpectedErr: service.ErrGetProductByID,
		},
		{
			Description: "Seller not found",
			UserID:      someSellerUserID,
			MockGetByID: MockProductReturn{Product: someProduct},
			MockSeller:  &MockSellerReturn{Error: service.ErrNotFound},
			ExpectedErr: service.ErrSellerNotFound,
		},
		{
			Description: "GetSellerByID error",
			UserID:      someSellerUserID,
			MockGetByID: MockProductReturn{Product: someProduct},
			MockSeller:  &MockSellerReturn{Error: errors.New("db error")},
			ExpectedErr: service.ErrGetSellerByID,
		},
		{
			Description: "Not your seller",
			UserID:      otherUserID,
			MockGetByID: MockProductReturn{Product: someProduct},
			MockSeller:  &MockSellerReturn{Seller: someSeller},
			ExpectedErr: service.ErrNotYourSeller,
		},
		{
			Description: "UpdateProduct repo error",
			UserID:      someSellerUserID,
			MockGetByID: MockProductReturn{Product: someProduct},
			MockSeller:  &MockSellerReturn{Seller: someSeller},
			MockUpdate:  &MockProductReturn{Error: errors.New("db error")},
			ExpectedErr: service.ErrUpdateProduct,
		},
	}

	for _, tCase := range tCases {
		t.Run(tCase.Description, func(t *testing.T) {
			productMock.EXPECT().GetProductByID(ctx, someID).Return(tCase.MockGetByID.Product, tCase.MockGetByID.Error)
			if tCase.MockSeller != nil {
				sellerMock.EXPECT().GetSellerByID(ctx, someProduct.SellerID).Return(tCase.MockSeller.Seller, tCase.MockSeller.Error)
			}
			// GetProductByIDForUpdate is called inside the tx for any case that reaches the update
			if tCase.MockUpdate != nil {
				productMock.EXPECT().GetProductByIDForUpdate(ctx, someID).Return(tCase.MockGetByID.Product, nil)
				productMock.EXPECT().UpdateProduct(ctx, someID, productUpdate).Return(tCase.MockUpdate.Product, tCase.MockUpdate.Error)
			}

			product, err := svc.UpdateProduct(ctx, testActor(tCase.UserID, m.RoleSeller), someID, productUpdate)
			assertError(t, err, tCase.ExpectedErr)
			if tCase.ExpectedErr == nil {
				assertProduct(t, product, tCase.MockUpdate.Product)
			}
		})
	}
}

func TestUpdateProductLockFires(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc, productMock, _, sellerMock := newProductService(ctrl)
	ctx := context.Background()

	newName := "Lock test product"
	productUpdate := m.ProductUpdate{Name: &newName}
	updatedProduct := someProduct
	updatedProduct.Name = newName

	// happy path: verify GetProductByIDForUpdate is called exactly once before UpdateProduct
	productMock.EXPECT().GetProductByID(ctx, someID).Return(someProduct, nil)
	sellerMock.EXPECT().GetSellerByID(ctx, someProduct.SellerID).Return(someSeller, nil)
	productMock.EXPECT().GetProductByIDForUpdate(ctx, someID).Return(someProduct, nil)
	productMock.EXPECT().UpdateProduct(ctx, someID, productUpdate).Return(updatedProduct, nil)

	got, err := svc.UpdateProduct(ctx, testActor(someSellerUserID, m.RoleSeller), someID, productUpdate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertProduct(t, got, updatedProduct)
}

func TestDeleteProduct(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc, productMock, _, sellerMock := newProductService(ctrl)
	ctx := context.Background()

	type testCase struct {
		Description string
		UserID      int64
		MockGetByID MockProductReturn
		MockSeller  *MockSellerReturn
		MockDelete  *error
		ExpectedErr error
	}

	tCases := []testCase{
		{
			Description: "Success",
			UserID:      someSellerUserID,
			MockGetByID: MockProductReturn{Product: someProduct},
			MockSeller:  &MockSellerReturn{Seller: someSeller},
			MockDelete:  ptrErr(nil),
		},
		{
			Description: "Product not found",
			UserID:      someSellerUserID,
			MockGetByID: MockProductReturn{Error: service.ErrNotFound},
			ExpectedErr: service.ErrProductNotFound,
		},
		{
			Description: "GetProductByID error",
			UserID:      someSellerUserID,
			MockGetByID: MockProductReturn{Error: errors.New("db error")},
			ExpectedErr: service.ErrGetProductByID,
		},
		{
			Description: "Seller not found",
			UserID:      someSellerUserID,
			MockGetByID: MockProductReturn{Product: someProduct},
			MockSeller:  &MockSellerReturn{Error: service.ErrNotFound},
			ExpectedErr: service.ErrSellerNotFound,
		},
		{
			Description: "GetSellerByID error",
			UserID:      someSellerUserID,
			MockGetByID: MockProductReturn{Product: someProduct},
			MockSeller:  &MockSellerReturn{Error: errors.New("db error")},
			ExpectedErr: service.ErrGetSellerByID,
		},
		{
			Description: "Not your seller",
			UserID:      otherUserID,
			MockGetByID: MockProductReturn{Product: someProduct},
			MockSeller:  &MockSellerReturn{Seller: someSeller},
			ExpectedErr: service.ErrNotYourSeller,
		},
		{
			Description: "DeleteProduct repo error",
			UserID:      someSellerUserID,
			MockGetByID: MockProductReturn{Product: someProduct},
			MockSeller:  &MockSellerReturn{Seller: someSeller},
			MockDelete:  ptrErr(errors.New("db error")),
			ExpectedErr: service.ErrDeleteProduct,
		},
	}

	for _, tCase := range tCases {
		t.Run(tCase.Description, func(t *testing.T) {
			productMock.EXPECT().GetProductByID(ctx, someID).Return(tCase.MockGetByID.Product, tCase.MockGetByID.Error)
			if tCase.MockSeller != nil {
				sellerMock.EXPECT().GetSellerByID(ctx, someProduct.SellerID).Return(tCase.MockSeller.Seller, tCase.MockSeller.Error)
			}
			if tCase.MockDelete != nil {
				productMock.EXPECT().DeleteProductByID(ctx, someID).Return(*tCase.MockDelete)
			}

			err := svc.DeleteProductByID(ctx, testActor(tCase.UserID, m.RoleSeller), someID)
			assertError(t, err, tCase.ExpectedErr)
		})
	}
}
