package service

import (
	"context"
	"errors"
	"fmt"

	m "github.com/beastixq/marketplace/internal/model"
)

//go:generate mockgen -package mock_service -destination ../mocks/service/mock_favorite_repo.go github.com/beastixq/marketplace/internal/service FavoriteRepo
type FavoriteRepo interface {
	AddFavorite(ctx context.Context, userID int64, productID int64) (created bool, err error)
	DeleteFavorite(ctx context.Context, userID int64, productID int64) error
	ListFavoriteProductsByUserID(ctx context.Context, userID int64, opts m.PaginationOpts) ([]m.Product, error)
	IsFavorite(ctx context.Context, userID int64, productID int64) (bool, error)
}

// FavoriteProductGetter is the narrow product lookup FavoriteService needs to
// validate that favorites only target active, visible products.
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

func (fs FavoriteService) AddFavorite(ctx context.Context, actor Actor, productID int64) (bool, error) {
	if _, err := fs.activeProduct(ctx, productID); err != nil {
		return false, err
	}
	created, err := fs.favoriteRepo.AddFavorite(ctx, actor.UserID, productID)
	if err != nil {
		if errors.Is(err, ErrProductNotFound) || errors.Is(err, ErrNotFound) {
			return false, ErrProductNotFound
		}
		return false, fmt.Errorf("%w: %v", ErrCreateFavorite, err)
	}
	// Repo's atomic INSERT ... SELECT ... WHERE deleted_at IS NULL returned
	// nothing: either the row already existed (re-add), or the product was
	// soft-deleted between our visibility check and the insert. Re-check to
	// disambiguate and surface ErrProductDeleted in the race case.
	if !created {
		if _, err := fs.activeProduct(ctx, productID); err != nil {
			return false, err
		}
	}
	return created, nil
}

func (fs FavoriteService) RemoveFavorite(ctx context.Context, actor Actor, productID int64) error {
	if err := fs.favoriteRepo.DeleteFavorite(ctx, actor.UserID, productID); err != nil {
		return fmt.Errorf("%w: %v", ErrDeleteFavorite, err)
	}
	return nil
}

func (fs FavoriteService) GetFavoriteProducts(ctx context.Context, actor Actor, opts m.PaginationOpts) ([]m.Product, error) {
	products, err := fs.favoriteRepo.ListFavoriteProductsByUserID(ctx, actor.UserID, opts)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrGetFavorites, err)
	}
	return products, nil
}

func (fs FavoriteService) IsProductFavorite(ctx context.Context, actor Actor, productID int64) (m.FavoriteState, error) {
	if _, err := fs.activeProduct(ctx, productID); err != nil {
		return m.FavoriteState{}, err
	}
	isFavorite, err := fs.favoriteRepo.IsFavorite(ctx, actor.UserID, productID)
	if err != nil {
		return m.FavoriteState{}, fmt.Errorf("%w: %v", ErrCheckFavorite, err)
	}
	return m.FavoriteState{ProductID: productID, IsFavorite: isFavorite}, nil
}

func (fs FavoriteService) activeProduct(ctx context.Context, productID int64) (m.Product, error) {
	product, err := fs.productGetter.GetProductByID(ctx, productID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return m.Product{}, ErrProductNotFound
		}
		return m.Product{}, fmt.Errorf("%w: %v", ErrGetProductByID, err)
	}
	if product.DeletedAt != nil {
		return m.Product{}, ErrProductDeleted
	}
	return product, nil
}
