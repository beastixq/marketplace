package service

import (
	"context"
	"errors"
	"fmt"

	m "github.com/beastixq/marketplace/internal/model"
)

// FavoriteRepo is the persistence contract FavoriteService depends on.
// Implementations live in internal/repository and must honor the semantics
// documented per method; product-visibility policy is not part of this
// contract and is enforced by the service before calling here.
//
//go:generate mockgen -package mock_service -destination ../mocks/service/mock_favorite_repo.go github.com/beastixq/marketplace/internal/service FavoriteRepo
type FavoriteRepo interface {
	// AddFavorite inserts the (userID, productID) pair if it is absent.
	//   - created == true  → a new row was inserted.
	//   - created == false → the row already existed; this is not an error.
	//   - ErrNotFound      → the referenced user or product does not exist.
	// Implementations must not infer or enforce product-visibility policy
	// (deleted_at, archived, etc.); that belongs to the caller.
	AddFavorite(ctx context.Context, userID int64, productID int64) (created bool, err error)

	// DeleteFavorite removes the (userID, productID) pair if present.
	// Removing an absent row is not an error; the method is idempotent.
	DeleteFavorite(ctx context.Context, userID int64, productID int64) error

	// ListFavoriteProductsByUserID returns products favorited by userID,
	// newest favorite first, filtered to currently active visible products.
	// opts controls pagination; an unset/zero opts means no pagination.
	ListFavoriteProductsByUserID(ctx context.Context, userID int64, opts m.PaginationOpts) ([]m.Product, error)

	// IsFavorite reports whether a favorite row exists for the pair.
	// It does not consult product visibility; the caller decides whether
	// a stale row against a soft-deleted product should be surfaced.
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
