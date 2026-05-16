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
	"go.uber.org/mock/gomock"
)

func TestFavoriteService_AddFavorite(t *testing.T) {
	ctx := context.Background()
	actor := testActor(someID, m.RoleAnalyst)
	deletedAt := time.Now()

	tCases := []struct {
		Description string
		Product     m.Product
		ProductErr  error
		RepoCreated bool
		Expected    bool
		ExpectedErr error
		RepoCalls   int
		ProductCalls int
	}{
		{
			Description: "Success created",
			Product:     someProduct,
			RepoCreated: true,
			Expected:    true,
			RepoCalls:   1,
			ProductCalls: 1,
		},
		{
			Description:  "Product not found",
			ProductErr:   service.ErrNotFound,
			ExpectedErr:  service.ErrProductNotFound,
			ProductCalls: 1,
		},
		{
			Description:  "Inactive product",
			Product:      m.Product{ID: someProduct.ID, DeletedAt: &deletedAt},
			ExpectedErr:  service.ErrProductDeleted,
			ProductCalls: 1,
		},
		{
			Description: "Already favorite",
			Product:     someProduct,
			RepoCreated: false,
			Expected:    false,
			RepoCalls:   1,
			ProductCalls: 2,
		},
	}

	for _, tCase := range tCases {
		t.Run(tCase.Description, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			favoriteRepo := mock_service.NewMockFavoriteRepo(ctrl)
			productRepo := mock_service.NewMockProductRepo(ctrl)
			svc := service.NewFavoriteService(favoriteRepo, productRepo)

			productRepo.EXPECT().GetProductByID(ctx, someProduct.ID).Return(tCase.Product, tCase.ProductErr).Times(tCase.ProductCalls)
			if tCase.RepoCalls > 0 {
				favoriteRepo.EXPECT().AddFavorite(ctx, actor.UserID, someProduct.ID).Return(tCase.RepoCreated, nil).Times(tCase.RepoCalls)
			}

			created, err := svc.AddFavorite(ctx, actor, someProduct.ID)
			assertError(t, err, tCase.ExpectedErr)
			if created != tCase.Expected {
				t.Fatalf("created: got %t, want %t", created, tCase.Expected)
			}
		})
	}
}

func TestFavoriteService_RemoveListAndCheck(t *testing.T) {
	ctx := context.Background()
	actor := testActor(someID, m.RoleSeller)
	deletedAt := time.Now()

	t.Run("Remove favorite allows deleted product", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		favoriteRepo := mock_service.NewMockFavoriteRepo(ctrl)
		productRepo := mock_service.NewMockProductRepo(ctrl)
		svc := service.NewFavoriteService(favoriteRepo, productRepo)

		// Service no longer pre-checks product on Remove; the FK CASCADE handles
		// orphans and a missing favorite row is a no-op DELETE.
		_ = deletedAt
		favoriteRepo.EXPECT().DeleteFavorite(ctx, actor.UserID, someProduct.ID).Return(nil)

		err := svc.RemoveFavorite(ctx, actor, someProduct.ID)
		assertError(t, err, nil)
	})

	t.Run("List favorites returns products", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		favoriteRepo := mock_service.NewMockFavoriteRepo(ctrl)
		productRepo := mock_service.NewMockProductRepo(ctrl)
		svc := service.NewFavoriteService(favoriteRepo, productRepo)

		pg := m.PaginationOpts{Page: 1, Limit: 10}
		favoriteRepo.EXPECT().ListFavoriteProductsByUserID(ctx, actor.UserID, pg).Return([]m.Product{someProduct}, nil)

		got, err := svc.GetFavoriteProducts(ctx, actor, pg)
		assertError(t, err, nil)
		if !reflect.DeepEqual(got, []m.Product{someProduct}) {
			t.Fatalf("products: got %v, want %v", got, []m.Product{someProduct})
		}
	})

	t.Run("Check favorite state", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		favoriteRepo := mock_service.NewMockFavoriteRepo(ctrl)
		productRepo := mock_service.NewMockProductRepo(ctrl)
		svc := service.NewFavoriteService(favoriteRepo, productRepo)

		productRepo.EXPECT().GetProductByID(ctx, someProduct.ID).Return(someProduct, nil)
		favoriteRepo.EXPECT().IsFavorite(ctx, actor.UserID, someProduct.ID).Return(true, nil)

		got, err := svc.IsProductFavorite(ctx, actor, someProduct.ID)
		assertError(t, err, nil)
		want := m.FavoriteState{ProductID: someProduct.ID, IsFavorite: true}
		if got != want {
			t.Fatalf("state: got %v, want %v", got, want)
		}
	})

	t.Run("Repository errors are wrapped", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		favoriteRepo := mock_service.NewMockFavoriteRepo(ctrl)
		productRepo := mock_service.NewMockProductRepo(ctrl)
		svc := service.NewFavoriteService(favoriteRepo, productRepo)

		productRepo.EXPECT().GetProductByID(ctx, someProduct.ID).Return(someProduct, nil)
		favoriteRepo.EXPECT().IsFavorite(ctx, actor.UserID, someProduct.ID).Return(false, errors.New("db down"))

		_, err := svc.IsProductFavorite(ctx, actor, someProduct.ID)
		assertError(t, err, service.ErrCheckFavorite)
	})
}
