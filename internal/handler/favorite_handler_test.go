package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	mock_service "github.com/beastixq/marketplace/internal/mocks/service"
	"github.com/beastixq/marketplace/internal/middleware"
	m "github.com/beastixq/marketplace/internal/model"
	"github.com/beastixq/marketplace/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/shopspring/decimal"
	"go.uber.org/mock/gomock"
)

const favoriteHandlerPassword = "xy1szo87vcnawkz&zj1SZ1NZC"
const favoriteHandlerPasswordHash = "$2a$04$6TTOHTypNV5UN4zlH94M1Oc20yvkekVJPJ2fYBGnQuiAcCWcHSga6"

func newFavoriteHandlerTestRouter(t *testing.T, ctrl *gomock.Controller, favoriteRepo service.FavoriteRepo, productGetter service.FavoriteProductGetter) (http.Handler, string) {
	t.Helper()
	user := m.User{
		ID:           7,
		Email:        "favorite_actor@example.com",
		PasswordHash: favoriteHandlerPasswordHash,
		FullName:     "Favorite Actor",
		Role:         m.RoleAnalyst,
		CreatedAt:    time.Now(),
	}
	userProvider := mock_service.NewMockAuthUserProvider(ctrl)
	userProvider.EXPECT().GetAuthUserByEmail(gomock.Any(), user.Email).Return(user, nil)
	userProvider.EXPECT().GetAuthUserByID(gomock.Any(), user.ID).Return(user, nil).AnyTimes()

	authService := service.NewAuthService(userProvider, nil, "favorite-test-secret", time.Hour)
	token, err := authService.Login(context.Background(), user.Email, favoriteHandlerPassword)
	if err != nil {
		t.Fatalf("login test actor: %v", err)
	}

	favoriteService := service.NewFavoriteService(favoriteRepo, productGetter)
	favoriteHandler := NewFavoriteHandler(favoriteService)

	r := chi.NewRouter()
	r.Use(middleware.ActorHolder())
	r.Group(func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(authService))
		r.Get("/api/v1/users/me/favorites", favoriteHandler.GetFavoriteProducts)
		r.Post("/api/v1/products/{id}/favorite", favoriteHandler.AddFavorite)
		r.Delete("/api/v1/products/{id}/favorite", favoriteHandler.RemoveFavorite)
		r.Get("/api/v1/products/{id}/favorite", favoriteHandler.GetFavoriteState)
	})
	return r, token
}

func TestFavoriteHandler_AddFavorite(t *testing.T) {
	activeProduct := m.Product{
		ID:            42,
		SellerID:      3,
		Name:          "Keyboard",
		Price:         decimal.NewFromInt(100),
		StockQuantity: 5,
		CreatedAt:     time.Now(),
	}
	deletedAt := time.Now()
	deletedProduct := activeProduct
	deletedProduct.ID = 99
	deletedProduct.DeletedAt = &deletedAt

	t.Run("Created and repeated favorite", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		favoriteRepo := mock_service.NewMockFavoriteRepo(ctrl)
		productRepo := mock_service.NewMockProductRepo(ctrl)
		router, token := newFavoriteHandlerTestRouter(t, ctrl, favoriteRepo, productRepo)

		gomock.InOrder(
			productRepo.EXPECT().GetProductByID(gomock.Any(), int64(42)).Return(activeProduct, nil),
			favoriteRepo.EXPECT().AddFavorite(gomock.Any(), int64(7), int64(42)).Return(true, nil),
			productRepo.EXPECT().GetProductByID(gomock.Any(), int64(42)).Return(activeProduct, nil),
			favoriteRepo.EXPECT().AddFavorite(gomock.Any(), int64(7), int64(42)).Return(false, nil),
			productRepo.EXPECT().GetProductByID(gomock.Any(), int64(42)).Return(activeProduct, nil),
		)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/products/42/favorite", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if rr.Code != http.StatusCreated {
			t.Fatalf("status: got %d, want %d", rr.Code, http.StatusCreated)
		}

		req = httptest.NewRequest(http.MethodPost, "/api/v1/products/42/favorite", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rr = httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if rr.Code != http.StatusNoContent {
			t.Fatalf("status: got %d, want %d", rr.Code, http.StatusNoContent)
		}
	})

	t.Run("Missing auth", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		favoriteRepo := mock_service.NewMockFavoriteRepo(ctrl)
		productRepo := mock_service.NewMockProductRepo(ctrl)
		router, _ := newFavoriteHandlerTestRouter(t, ctrl, favoriteRepo, productRepo)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/products/42/favorite", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("status: got %d, want %d", rr.Code, http.StatusUnauthorized)
		}
	})

	t.Run("Invalid product id", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		favoriteRepo := mock_service.NewMockFavoriteRepo(ctrl)
		productRepo := mock_service.NewMockProductRepo(ctrl)
		router, token := newFavoriteHandlerTestRouter(t, ctrl, favoriteRepo, productRepo)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/products/bad/favorite", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status: got %d, want %d", rr.Code, http.StatusBadRequest)
		}
	})

	t.Run("Missing product", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		favoriteRepo := mock_service.NewMockFavoriteRepo(ctrl)
		productRepo := mock_service.NewMockProductRepo(ctrl)
		router, token := newFavoriteHandlerTestRouter(t, ctrl, favoriteRepo, productRepo)

		productRepo.EXPECT().GetProductByID(gomock.Any(), int64(42)).Return(m.Product{}, service.ErrNotFound)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/products/42/favorite", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if rr.Code != http.StatusNotFound {
			t.Fatalf("status: got %d, want %d", rr.Code, http.StatusNotFound)
		}
	})

	t.Run("Inactive product", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		favoriteRepo := mock_service.NewMockFavoriteRepo(ctrl)
		productRepo := mock_service.NewMockProductRepo(ctrl)
		router, token := newFavoriteHandlerTestRouter(t, ctrl, favoriteRepo, productRepo)

		productRepo.EXPECT().GetProductByID(gomock.Any(), int64(99)).Return(deletedProduct, nil)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/products/99/favorite", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if rr.Code != http.StatusConflict {
			t.Fatalf("status: got %d, want %d", rr.Code, http.StatusConflict)
		}
	})
}

func TestFavoriteHandler_RemoveListAndCheck(t *testing.T) {
	product := m.Product{
		ID:            42,
		SellerID:      3,
		Name:          "Keyboard",
		Price:         decimal.NewFromInt(100),
		StockQuantity: 5,
		CreatedAt:     time.Now(),
	}

	t.Run("Check favorite state", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		favoriteRepo := mock_service.NewMockFavoriteRepo(ctrl)
		productRepo := mock_service.NewMockProductRepo(ctrl)
		router, token := newFavoriteHandlerTestRouter(t, ctrl, favoriteRepo, productRepo)

		productRepo.EXPECT().GetProductByID(gomock.Any(), int64(42)).Return(product, nil)
		favoriteRepo.EXPECT().IsFavorite(gomock.Any(), int64(7), int64(42)).Return(true, nil)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/products/42/favorite", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status: got %d, want %d", rr.Code, http.StatusOK)
		}
		var got FavoriteStateDTO
		if err := json.NewDecoder(rr.Body).Decode(&got); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if got.ProductID != 42 || !got.IsFavorite {
			t.Fatalf("state: got %+v, want favorite product 42", got)
		}
	})

	t.Run("List favorite products", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		favoriteRepo := mock_service.NewMockFavoriteRepo(ctrl)
		productRepo := mock_service.NewMockProductRepo(ctrl)
		router, token := newFavoriteHandlerTestRouter(t, ctrl, favoriteRepo, productRepo)

		favoriteRepo.EXPECT().
			ListFavoriteProductsByUserID(gomock.Any(), int64(7), m.PaginationOpts{Page: 1, Limit: 10}).
			Return([]m.Product{product}, nil)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me/favorites?page=1&limit=10", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status: got %d, want %d", rr.Code, http.StatusOK)
		}
		var got []ProductDTO
		if err := json.NewDecoder(rr.Body).Decode(&got); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if len(got) != 1 || got[0].ID != 42 {
			t.Fatalf("favorites: got %+v, want one product 42", got)
		}
	})

	t.Run("Remove favorite", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		favoriteRepo := mock_service.NewMockFavoriteRepo(ctrl)
		productRepo := mock_service.NewMockProductRepo(ctrl)
		router, token := newFavoriteHandlerTestRouter(t, ctrl, favoriteRepo, productRepo)

		productRepo.EXPECT().GetProductByID(gomock.Any(), int64(42)).Return(product, nil)
		favoriteRepo.EXPECT().DeleteFavorite(gomock.Any(), int64(7), int64(42)).Return(nil)

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/products/42/favorite", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if rr.Code != http.StatusNoContent {
			t.Fatalf("status: got %d, want %d", rr.Code, http.StatusNoContent)
		}
	})
}
