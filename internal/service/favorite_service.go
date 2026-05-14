package service

import (
	"context"
	"errors"
	"fmt"

	m "github.com/beastixq/marketplace/internal/model"
)

//go:generate mockgen -package mock_service -destination ../mocks/service/mock_favorite_repo.go github.com/beastixq/marketplace/internal/service FavoriteRepo
type FavoriteRepo interface {
	AddFavorite(ctx context.Context, userID int64, productID int64) error
	RemoveFavorite(ctx context.Context, userID int64, productID int64) error
	GetFavoritesByUserID(ctx context.Context, userID int64, opts m.PaginationOpts) ([]m.Favorite, error)
}

// FavoriteProductGetter is the narrow product lookup FavoriteService needs
// to reject favoriting soft-deleted or missing products.
type FavoriteProductGetter interface {
	GetProductByID(ctx context.Context, id int64) (m.Product, error)
}

type FavoriteService struct {
	favoriteRepo  FavoriteRepo
	productGetter FavoriteProductGetter
}

func NewFavoriteService(favoriteRepo FavoriteRepo, productGetter FavoriteProductGetter) FavoriteService {
	return FavoriteService{favoriteRepo: favoriteRepo, productGetter: productGetter}
}

func (fs FavoriteService) AddFavorite(ctx context.Context, actor Actor, productID int64) error {
	if !actor.HasRole(m.RoleBuyer) {
		return ErrPermissionDenied
	}

	product, err := fs.productGetter.GetProductByID(ctx, productID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return ErrProductNotFound
		}
		return fmt.Errorf("%w: %v", ErrGetProductByID, err)
	}
	if product.DeletedAt != nil {
		return ErrProductDeleted
	}

	if err := fs.favoriteRepo.AddFavorite(ctx, actor.UserID, productID); err != nil {
		if errors.Is(err, ErrFavoriteAlreadyExists) {
			return ErrFavoriteAlreadyExists
		}
		return fmt.Errorf("%w: %v", ErrAddFavorite, err)
	}
	return nil
}

func (fs FavoriteService) RemoveFavorite(ctx context.Context, actor Actor, productID int64) error {
	if !actor.HasRole(m.RoleBuyer) {
		return ErrPermissionDenied
	}

	if err := fs.favoriteRepo.RemoveFavorite(ctx, actor.UserID, productID); err != nil {
		if errors.Is(err, ErrNotFound) {
			return ErrFavoriteNotFound
		}
		return fmt.Errorf("%w: %v", ErrRemoveFavorite, err)
	}
	return nil
}

func (fs FavoriteService) GetMyFavorites(ctx context.Context, actor Actor, opts m.PaginationOpts) ([]m.Favorite, error) {
	if !actor.HasRole(m.RoleBuyer) {
		return nil, ErrPermissionDenied
	}

	favorites, err := fs.favoriteRepo.GetFavoritesByUserID(ctx, actor.UserID, opts)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrGetFavoritesByUserID, err)
	}
	return favorites, nil
}
