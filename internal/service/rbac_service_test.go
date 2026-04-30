package service_test

import (
	"context"
	"testing"

	mock_service "github.com/beastixq/marketplace/internal/mocks/service"
	m "github.com/beastixq/marketplace/internal/model"
	"github.com/beastixq/marketplace/internal/service"
	"go.uber.org/mock/gomock"
	"golang.org/x/crypto/bcrypt"
)

func TestRBACBuyerOnlyUseCases(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)

	addressSvc := service.NewAddressService(mock_service.NewMockAddressRepo(ctrl))
	_, err := addressSvc.GetAddressesByUserID(ctx, testActor(someID, m.RoleSeller))
	assertError(t, err, service.ErrPermissionDenied)

	reviewSvc := service.NewReviewService(mock_service.NewMockReviewRepo(ctrl), fakeReviewPurchaseChecker{})
	_, err = reviewSvc.CreateReview(ctx, testActor(someID, m.RoleSeller), m.ReviewCreate{ProductID: someProductID, Rating: 5})
	assertError(t, err, service.ErrPermissionDenied)

	orderSvc, _, _, _, _, _ := newOrderService(ctrl)
	err = orderSvc.AddItemToCart(ctx, testActor(someID, m.RoleSeller), someProductID, 1)
	assertError(t, err, service.ErrPermissionDenied)
	err = orderSvc.PayOrder(ctx, testActor(someID, m.RoleAdmin), someOrderID)
	assertError(t, err, service.ErrPermissionDenied)

	paymentSvc, _, _ := newPaymentService(ctrl, fakeClock{now: someTime})
	_, _, err = paymentSvc.GetOrderPaymentURL(ctx, testActor(someID, m.RoleAdmin), someOrderID)
	assertError(t, err, service.ErrPermissionDenied)
}

func TestRBACSellerOnlyUseCases(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)

	sellerSvc := service.NewSellerService(mock_service.NewMockSellerRepo(ctrl))
	_, err := sellerSvc.CreateSeller(ctx, testActor(someID, m.RoleAdmin), m.SellerCreate{CompanyName: "Acme"})
	assertError(t, err, service.ErrPermissionDenied)

	productSvc, _, _, _ := newProductService(ctrl)
	_, err = productSvc.CreateProduct(ctx, testActor(someID, m.RoleBuyer), m.ProductCreate{SellerID: someSellerID})
	assertError(t, err, service.ErrPermissionDenied)

	orderSvc, _, _, _, _, _ := newOrderService(ctrl)
	err = orderSvc.ShipOrder(ctx, testActor(someID, m.RoleBuyer), someOrderID)
	assertError(t, err, service.ErrPermissionDenied)
	err = orderSvc.CancelOrder(ctx, testActor(someID, m.RoleSeller), someOrderID)
	assertError(t, err, service.ErrPermissionDenied)
}

func TestRBACAdminOnlyUseCases(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)

	categorySvc := service.NewCategoryService(mock_service.NewMockCategoryRepo(ctrl))
	_, err := categorySvc.CreateCategory(ctx, testActor(someID, m.RoleSeller), m.CategoryCreate{Name: "Electronics"})
	assertError(t, err, service.ErrPermissionDenied)

	userSvc := service.NewUserService(mock_service.NewMockUserRepo(ctrl), bcrypt.MinCost)
	_, err = userSvc.GetUsers(ctx, testActor(someID, m.RoleAnalyst), m.UserListOptions{})
	assertError(t, err, service.ErrPermissionDenied)

	_, err = userSvc.GetUserByEmail(ctx, testActor(someID, m.RoleAnalyst), someEmail)
	assertError(t, err, service.ErrPermissionDenied)

	_, err = userSvc.CreateUser(ctx, m.UserCreate{
		Email:    someEmail,
		Password: someStrongPassword,
		FullName: someFullName,
		Role:     m.RoleAdmin,
	})
	assertError(t, err, service.ErrPermissionDenied)
}

func TestRBACAdminCannotUpdateReviews(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)

	reviewSvc := service.NewReviewService(mock_service.NewMockReviewRepo(ctrl), fakeReviewPurchaseChecker{})
	_, err := reviewSvc.UpdateReview(ctx, testActor(someID, m.RoleAdmin), someID, m.ReviewUpdate{})
	assertError(t, err, service.ErrPermissionDenied)
}
