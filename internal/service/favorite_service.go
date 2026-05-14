package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	m "github.com/beastixq/marketplace/internal/model"
)

const (
	favoritesDefaultPageSize = 20
	favoritesMaxPageSize     = 100
)

//go:generate mockgen -package mock_service -destination ../mocks/service/mock_favorite_repo.go github.com/beastixq/marketplace/internal/service FavoriteRepo
type FavoriteRepo interface {
	Add(ctx context.Context, userID, productID int64) (created bool, err error)
	Remove(ctx context.Context, userID, productID int64) error
	List(ctx context.Context, userID int64, offset, limit int) (items []m.Favorite, total int, err error)
	Exists(ctx context.Context, userID, productID int64) (favorited bool, addedAt *time.Time, err error)
}

//go:generate mockgen -package mock_service -destination ../mocks/service/mock_favorite_product_getter.go github.com/beastixq/marketplace/internal/service FavoriteProductGetter
type FavoriteProductGetter interface {
	GetProductByID(ctx context.Context, id int64) (m.Product, error)
}

type FavoriteService struct {
	repo          FavoriteRepo
	productGetter FavoriteProductGetter
}

func NewFavoriteService(repo FavoriteRepo, productGetter FavoriteProductGetter) FavoriteService {
	return FavoriteService{repo: repo, productGetter: productGetter}
}

// Add favorites a product for the given user. The bool return is true when a
// new row was inserted, false when the (user, product) pair was already
// favorited. Idempotent.
func (fs FavoriteService) Add(ctx context.Context, userID, productID int64) (created bool, err error) {
	if err := fs.assertProductVisible(ctx, productID); err != nil {
		return false, err
	}

	created, err = fs.repo.Add(ctx, userID, productID)
	if err != nil {
		// Race condition: product was deleted between visibility check and
		// insert. Surface as not-found to avoid leaking existence.
		if errors.Is(err, ErrProductNotFound) {
			return false, ErrProductNotFound
		}
		return false, fmt.Errorf("%w: %v", ErrCreateFavorite, err)
	}
	return created, nil
}

// Remove unfavorites a product for the given user. Idempotent: removing a
// non-favorite is a no-op success.
func (fs FavoriteService) Remove(ctx context.Context, userID, productID int64) error {
	if err := fs.repo.Remove(ctx, userID, productID); err != nil {
		return fmt.Errorf("%w: %v", ErrDeleteFavorite, err)
	}
	return nil
}

// List returns the caller's favorites newest-first with pagination clamping.
// page is 1-based; values <1 are clamped to 1. pageSize is clamped to
// [1, favoritesMaxPageSize]; <1 falls back to favoritesDefaultPageSize.
func (fs FavoriteService) List(ctx context.Context, userID int64, page, pageSize int) (items []m.Favorite, total int, err error) {
	page, pageSize = clampPagination(page, pageSize)
	offset := (page - 1) * pageSize

	items, total, err = fs.repo.List(ctx, userID, offset, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("%w: %v", ErrListFavorites, err)
	}
	return items, total, nil
}

// IsFavorited reports whether the user has favorited the product, and when.
// addedAt is nil when favorited == false. Returns ErrProductNotFound if the
// product itself is not visible to the caller (matches the spec rule that
// 404 means "product gone", not "not favorited").
func (fs FavoriteService) IsFavorited(ctx context.Context, userID, productID int64) (favorited bool, addedAt *time.Time, err error) {
	if err := fs.assertProductVisible(ctx, productID); err != nil {
		return false, nil, err
	}

	favorited, addedAt, err = fs.repo.Exists(ctx, userID, productID)
	if err != nil {
		return false, nil, fmt.Errorf("%w: %v", ErrGetFavorite, err)
	}
	return favorited, addedAt, nil
}

func (fs FavoriteService) assertProductVisible(ctx context.Context, productID int64) error {
	product, err := fs.productGetter.GetProductByID(ctx, productID)
	if err != nil {
		if errors.Is(err, ErrNotFound) || errors.Is(err, ErrProductNotFound) {
			return ErrProductNotFound
		}
		return fmt.Errorf("%w: %v", ErrGetProductByID, err)
	}
	if product.DeletedAt != nil {
		// Soft-deleted products are treated as not-found from the favorites'
		// point of view (matches the spec edge case: never reveal hidden
		// products' existence).
		return ErrProductNotFound
	}
	return nil
}

// PageSize returns the effective default page size for callers that want to
// expose it in pagination metadata when the caller did not specify one.
func (fs FavoriteService) DefaultPageSize() int { return favoritesDefaultPageSize }

func clampPagination(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = favoritesDefaultPageSize
	}
	if pageSize > favoritesMaxPageSize {
		pageSize = favoritesMaxPageSize
	}
	return page, pageSize
}
