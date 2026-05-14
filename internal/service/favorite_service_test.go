package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	mock_service "github.com/beastixq/marketplace/internal/mocks/service"
	m "github.com/beastixq/marketplace/internal/model"
	"github.com/beastixq/marketplace/internal/service"
	"go.uber.org/mock/gomock"
)

// shared test fixtures specific to favorites
const (
	favoriteUserID    int64 = 7
	favoriteProductID int64 = 99
)

var favoriteVisibleProduct = m.Product{
	ID:            favoriteProductID,
	SellerID:      someID,
	Name:          "Test Product",
	Price:         someProductPrice,
	StockQuantity: 5,
}

var favoriteSoftDeletedProduct = func() m.Product {
	t := someTime
	p := favoriteVisibleProduct
	p.DeletedAt = &t
	return p
}()

// ----- US1: Add -----

func TestFavoriteService_Add_Success_Created(t *testing.T) {
	ctrl := gomock.NewController(t)
	repoMock := mock_service.NewMockFavoriteRepo(ctrl)
	prodMock := mock_service.NewMockFavoriteProductGetter(ctrl)
	svc := service.NewFavoriteService(repoMock, prodMock)
	ctx := context.Background()

	prodMock.EXPECT().GetProductByID(ctx, favoriteProductID).Return(favoriteVisibleProduct, nil)
	repoMock.EXPECT().Add(ctx, favoriteUserID, favoriteProductID).Return(true, nil)

	created, err := svc.Add(ctx, favoriteUserID, favoriteProductID)
	assertError(t, err, nil)
	if !created {
		t.Fatalf("expected created=true, got false")
	}
}

func TestFavoriteService_Add_Idempotent_AlreadyExists(t *testing.T) {
	ctrl := gomock.NewController(t)
	repoMock := mock_service.NewMockFavoriteRepo(ctrl)
	prodMock := mock_service.NewMockFavoriteProductGetter(ctrl)
	svc := service.NewFavoriteService(repoMock, prodMock)
	ctx := context.Background()

	prodMock.EXPECT().GetProductByID(ctx, favoriteProductID).Return(favoriteVisibleProduct, nil)
	repoMock.EXPECT().Add(ctx, favoriteUserID, favoriteProductID).Return(false, nil)

	created, err := svc.Add(ctx, favoriteUserID, favoriteProductID)
	assertError(t, err, nil)
	if created {
		t.Fatalf("expected created=false on duplicate, got true")
	}
}

func TestFavoriteService_Add_ProductNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	repoMock := mock_service.NewMockFavoriteRepo(ctrl)
	prodMock := mock_service.NewMockFavoriteProductGetter(ctrl)
	svc := service.NewFavoriteService(repoMock, prodMock)
	ctx := context.Background()

	prodMock.EXPECT().GetProductByID(ctx, favoriteProductID).Return(m.Product{}, service.ErrProductNotFound)
	// repoMock.Add must NOT be called when product is not visible.

	_, err := svc.Add(ctx, favoriteUserID, favoriteProductID)
	assertError(t, err, service.ErrProductNotFound)
}

func TestFavoriteService_Add_ProductSoftDeleted(t *testing.T) {
	ctrl := gomock.NewController(t)
	repoMock := mock_service.NewMockFavoriteRepo(ctrl)
	prodMock := mock_service.NewMockFavoriteProductGetter(ctrl)
	svc := service.NewFavoriteService(repoMock, prodMock)
	ctx := context.Background()

	prodMock.EXPECT().GetProductByID(ctx, favoriteProductID).Return(favoriteSoftDeletedProduct, nil)

	_, err := svc.Add(ctx, favoriteUserID, favoriteProductID)
	assertError(t, err, service.ErrProductNotFound)
}

func TestFavoriteService_Add_RaceProductDeleted(t *testing.T) {
	ctrl := gomock.NewController(t)
	repoMock := mock_service.NewMockFavoriteRepo(ctrl)
	prodMock := mock_service.NewMockFavoriteProductGetter(ctrl)
	svc := service.NewFavoriteService(repoMock, prodMock)
	ctx := context.Background()

	// Visibility check passes...
	prodMock.EXPECT().GetProductByID(ctx, favoriteProductID).Return(favoriteVisibleProduct, nil)
	// ... but product is deleted before insert; repo surfaces ErrProductNotFound.
	repoMock.EXPECT().Add(ctx, favoriteUserID, favoriteProductID).Return(false, service.ErrProductNotFound)

	_, err := svc.Add(ctx, favoriteUserID, favoriteProductID)
	assertError(t, err, service.ErrProductNotFound)
}

func TestFavoriteService_Add_RepoError(t *testing.T) {
	ctrl := gomock.NewController(t)
	repoMock := mock_service.NewMockFavoriteRepo(ctrl)
	prodMock := mock_service.NewMockFavoriteProductGetter(ctrl)
	svc := service.NewFavoriteService(repoMock, prodMock)
	ctx := context.Background()

	prodMock.EXPECT().GetProductByID(ctx, favoriteProductID).Return(favoriteVisibleProduct, nil)
	repoMock.EXPECT().Add(ctx, favoriteUserID, favoriteProductID).Return(false, errors.New("db boom"))

	_, err := svc.Add(ctx, favoriteUserID, favoriteProductID)
	assertError(t, err, service.ErrCreateFavorite)
}

// ----- US2: List -----

func TestFavoriteService_List_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	repoMock := mock_service.NewMockFavoriteRepo(ctrl)
	prodMock := mock_service.NewMockFavoriteProductGetter(ctrl)
	svc := service.NewFavoriteService(repoMock, prodMock)
	ctx := context.Background()

	wantItems := []m.Favorite{{Product: favoriteVisibleProduct, AddedAt: someTime}}
	repoMock.EXPECT().List(ctx, favoriteUserID, 0, 20).Return(wantItems, 1, nil)

	items, total, err := svc.List(ctx, favoriteUserID, 1, 20)
	assertError(t, err, nil)
	if total != 1 || len(items) != 1 {
		t.Fatalf("expected 1 item / total=1, got %d / %d", len(items), total)
	}
}

func TestFavoriteService_List_ClampPageBelow1(t *testing.T) {
	ctrl := gomock.NewController(t)
	repoMock := mock_service.NewMockFavoriteRepo(ctrl)
	prodMock := mock_service.NewMockFavoriteProductGetter(ctrl)
	svc := service.NewFavoriteService(repoMock, prodMock)
	ctx := context.Background()

	repoMock.EXPECT().List(ctx, favoriteUserID, 0, 20).Return(nil, 0, nil)

	_, _, err := svc.List(ctx, favoriteUserID, 0, 20)
	assertError(t, err, nil)
}

func TestFavoriteService_List_ClampPageSizeOver100(t *testing.T) {
	ctrl := gomock.NewController(t)
	repoMock := mock_service.NewMockFavoriteRepo(ctrl)
	prodMock := mock_service.NewMockFavoriteProductGetter(ctrl)
	svc := service.NewFavoriteService(repoMock, prodMock)
	ctx := context.Background()

	repoMock.EXPECT().List(ctx, favoriteUserID, 100, 100).Return(nil, 0, nil)

	_, _, err := svc.List(ctx, favoriteUserID, 2, 500)
	assertError(t, err, nil)
}

func TestFavoriteService_List_DefaultPageSizeWhenZero(t *testing.T) {
	ctrl := gomock.NewController(t)
	repoMock := mock_service.NewMockFavoriteRepo(ctrl)
	prodMock := mock_service.NewMockFavoriteProductGetter(ctrl)
	svc := service.NewFavoriteService(repoMock, prodMock)
	ctx := context.Background()

	repoMock.EXPECT().List(ctx, favoriteUserID, 0, 20).Return(nil, 0, nil)

	_, _, err := svc.List(ctx, favoriteUserID, 1, 0)
	assertError(t, err, nil)
}

func TestFavoriteService_List_RepoError(t *testing.T) {
	ctrl := gomock.NewController(t)
	repoMock := mock_service.NewMockFavoriteRepo(ctrl)
	prodMock := mock_service.NewMockFavoriteProductGetter(ctrl)
	svc := service.NewFavoriteService(repoMock, prodMock)
	ctx := context.Background()

	repoMock.EXPECT().List(ctx, favoriteUserID, 0, 20).Return(nil, 0, errors.New("db boom"))

	_, _, err := svc.List(ctx, favoriteUserID, 1, 20)
	assertError(t, err, service.ErrListFavorites)
}

// ----- US3: Remove -----

func TestFavoriteService_Remove_DelegatesToRepo(t *testing.T) {
	ctrl := gomock.NewController(t)
	repoMock := mock_service.NewMockFavoriteRepo(ctrl)
	prodMock := mock_service.NewMockFavoriteProductGetter(ctrl)
	svc := service.NewFavoriteService(repoMock, prodMock)
	ctx := context.Background()

	repoMock.EXPECT().Remove(ctx, favoriteUserID, favoriteProductID).Return(nil)
	// productGetter must NOT be called for remove (idempotent, no visibility check).

	if err := svc.Remove(ctx, favoriteUserID, favoriteProductID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFavoriteService_Remove_RepoError(t *testing.T) {
	ctrl := gomock.NewController(t)
	repoMock := mock_service.NewMockFavoriteRepo(ctrl)
	prodMock := mock_service.NewMockFavoriteProductGetter(ctrl)
	svc := service.NewFavoriteService(repoMock, prodMock)
	ctx := context.Background()

	repoMock.EXPECT().Remove(ctx, favoriteUserID, favoriteProductID).Return(errors.New("db boom"))

	err := svc.Remove(ctx, favoriteUserID, favoriteProductID)
	assertError(t, err, service.ErrDeleteFavorite)
}

// ----- US4: IsFavorited -----

func TestFavoriteService_IsFavorited_True(t *testing.T) {
	ctrl := gomock.NewController(t)
	repoMock := mock_service.NewMockFavoriteRepo(ctrl)
	prodMock := mock_service.NewMockFavoriteProductGetter(ctrl)
	svc := service.NewFavoriteService(repoMock, prodMock)
	ctx := context.Background()

	addedAt := someTime
	prodMock.EXPECT().GetProductByID(ctx, favoriteProductID).Return(favoriteVisibleProduct, nil)
	repoMock.EXPECT().Exists(ctx, favoriteUserID, favoriteProductID).Return(true, &addedAt, nil)

	got, gotAddedAt, err := svc.IsFavorited(ctx, favoriteUserID, favoriteProductID)
	assertError(t, err, nil)
	if !got {
		t.Fatalf("expected favorited=true")
	}
	if gotAddedAt == nil || !gotAddedAt.Equal(addedAt) {
		t.Fatalf("expected addedAt=%v, got %v", addedAt, gotAddedAt)
	}
}

func TestFavoriteService_IsFavorited_False(t *testing.T) {
	ctrl := gomock.NewController(t)
	repoMock := mock_service.NewMockFavoriteRepo(ctrl)
	prodMock := mock_service.NewMockFavoriteProductGetter(ctrl)
	svc := service.NewFavoriteService(repoMock, prodMock)
	ctx := context.Background()

	prodMock.EXPECT().GetProductByID(ctx, favoriteProductID).Return(favoriteVisibleProduct, nil)
	repoMock.EXPECT().Exists(ctx, favoriteUserID, favoriteProductID).Return(false, nil, nil)

	got, gotAddedAt, err := svc.IsFavorited(ctx, favoriteUserID, favoriteProductID)
	assertError(t, err, nil)
	if got {
		t.Fatalf("expected favorited=false")
	}
	if gotAddedAt != nil {
		t.Fatalf("expected nil addedAt when not favorited, got %v", gotAddedAt)
	}
}

func TestFavoriteService_IsFavorited_ProductNotVisible(t *testing.T) {
	ctrl := gomock.NewController(t)
	repoMock := mock_service.NewMockFavoriteRepo(ctrl)
	prodMock := mock_service.NewMockFavoriteProductGetter(ctrl)
	svc := service.NewFavoriteService(repoMock, prodMock)
	ctx := context.Background()

	prodMock.EXPECT().GetProductByID(ctx, favoriteProductID).Return(m.Product{}, service.ErrProductNotFound)
	// repoMock.Exists must NOT be called.

	_, _, err := svc.IsFavorited(ctx, favoriteUserID, favoriteProductID)
	assertError(t, err, service.ErrProductNotFound)
}

func TestFavoriteService_IsFavorited_RepoError(t *testing.T) {
	ctrl := gomock.NewController(t)
	repoMock := mock_service.NewMockFavoriteRepo(ctrl)
	prodMock := mock_service.NewMockFavoriteProductGetter(ctrl)
	svc := service.NewFavoriteService(repoMock, prodMock)
	ctx := context.Background()

	prodMock.EXPECT().GetProductByID(ctx, favoriteProductID).Return(favoriteVisibleProduct, nil)
	repoMock.EXPECT().Exists(ctx, favoriteUserID, favoriteProductID).Return(false, (*time.Time)(nil), errors.New("db boom"))

	_, _, err := svc.IsFavorited(ctx, favoriteUserID, favoriteProductID)
	assertError(t, err, service.ErrGetFavorite)
}
