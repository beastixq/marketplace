package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	m "github.com/beastixq/marketplace/internal/model"
)

//go:generate mockgen -package mock_service -destination ../mocks/service/mock_product_repo.go github.com/beastixq/marketplace/internal/service ProductRepo
type ProductRepo interface {
	GetProducts(ctx context.Context, options m.CatalogOptions) (ps []m.Product, err error)
	GetProductByID(ctx context.Context, id int64) (p m.Product, err error)
	GetProductByIDForUpdate(ctx context.Context, id int64) (p m.Product, err error)
	GetProductPriceHistory(ctx context.Context, pid int64, dateFrom time.Time, dateTo time.Time) (ph []m.ProductPriceHistory, err error)
	CreateProduct(ctx context.Context, pc m.ProductCreate) (id int64, err error)
	UpdateProduct(ctx context.Context, id int64, pu m.ProductUpdate) (p m.Product, err error)
	ChangeStockAndReserved(ctx context.Context, productID int64, stockDelta int, reservedDelta int) (err error)
	DeleteProductByID(ctx context.Context, id int64) (err error)
}

type ProductCategoryRepo interface {
	GetProductCategories(ctx context.Context, productID int64) ([]m.Category, error)
	ReplaceProductCategories(ctx context.Context, productID int64, categoryIDs []int64) error
}

//go:generate mockgen -package mock_service -destination ../mocks/service/mock_product_favorite_repo.go github.com/beastixq/marketplace/internal/service ProductFavoriteRepo
type ProductFavoriteRepo interface {
	GetFavoriteProducts(ctx context.Context, userID int64, opts m.PaginationOpts) ([]m.Product, error)
	AddFavoriteProduct(ctx context.Context, userID int64, productID int64) error
	RemoveFavoriteProduct(ctx context.Context, userID int64, productID int64) error
	IsFavoriteProduct(ctx context.Context, userID int64, productID int64) (bool, error)
}

type ProductService struct {
	productRepo         ProductRepo
	productCategoryRepo ProductCategoryRepo
	productFavoriteRepo ProductFavoriteRepo
	reviewRepo          ReviewRepo
	sellerRepo          SellerRepo
	txManager           TxManager
}

func NewProductService(productRepo ProductRepo, reviewRepo ReviewRepo, sellerRepo SellerRepo, txManager TxManager) ProductService {
	productCategoryRepo, _ := productRepo.(ProductCategoryRepo)
	productFavoriteRepo, _ := productRepo.(ProductFavoriteRepo)
	return ProductService{
		productRepo:         productRepo,
		productCategoryRepo: productCategoryRepo,
		productFavoriteRepo: productFavoriteRepo,
		reviewRepo:          reviewRepo,
		sellerRepo:          sellerRepo,
		txManager:           txManager,
	}
}

func (psvc ProductService) GetProducts(ctx context.Context, options m.CatalogOptions) (ps []m.Product, err error) {
	ps, err = psvc.productRepo.GetProducts(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrGetProducts, err)
	}
	if len(ps) == 0 {
		return nil, ErrProductNotFound
	}
	return ps, nil
}

func (ps ProductService) GetProductByID(ctx context.Context, id int64) (p m.Product, err error) {
	p, err = ps.productRepo.GetProductByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return m.Product{}, ErrProductNotFound
		}
		return m.Product{}, fmt.Errorf("%w: %v", ErrGetProductByID, err)
	}
	return p, nil
}

func (ps ProductService) GetProductPriceHistory(ctx context.Context, pid int64, dateFrom time.Time, dateTo time.Time) (ph []m.ProductPriceHistory, err error) {
	_, err = ps.productRepo.GetProductByID(ctx, pid)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrProductNotFound
		}
		return nil, fmt.Errorf("%w: %v", ErrGetProductByID, err)
	}
	ph, err = ps.productRepo.GetProductPriceHistory(ctx, pid, dateFrom, dateTo)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrGetProductPriceHistory, err)
	}
	return ph, nil
}

func (ps ProductService) GetProductCategories(ctx context.Context, productID int64) ([]m.Category, error) {
	if ps.productCategoryRepo == nil {
		return nil, ErrGetProductCategories
	}
	categories, err := ps.productCategoryRepo.GetProductCategories(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrGetProductCategories, err)
	}
	return categories, nil
}

func (ps ProductService) GetReviewsByProductID(ctx context.Context, pid int64, opts m.PaginationOpts) (rs []m.Review, err error) {
	_, err = ps.productRepo.GetProductByID(ctx, pid)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrProductNotFound
		}
		return nil, fmt.Errorf("%w: %v", ErrGetProductByID, err)
	}

	rs, err = ps.reviewRepo.GetReviewsByProductID(ctx, pid, opts)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrGetReviewsByProductID, err)
	}
	return rs, nil
}

func (ps ProductService) GetFavoriteProducts(ctx context.Context, actor Actor, opts m.PaginationOpts) ([]m.Product, error) {
	if ps.productFavoriteRepo == nil {
		return nil, ErrGetFavoriteProducts
	}

	products, err := ps.productFavoriteRepo.GetFavoriteProducts(ctx, actor.UserID, opts)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrGetFavoriteProducts, err)
	}
	return products, nil
}

func (ps ProductService) AddFavoriteProduct(ctx context.Context, actor Actor, productID int64) error {
	if ps.productFavoriteRepo == nil {
		return ErrAddFavoriteProduct
	}
	if err := ps.ensureProductCanBeFavorited(ctx, productID); err != nil {
		return err
	}

	if err := ps.productFavoriteRepo.AddFavoriteProduct(ctx, actor.UserID, productID); err != nil {
		if errors.Is(err, ErrNotFound) {
			return ErrProductNotFound
		}
		return fmt.Errorf("%w: %v", ErrAddFavoriteProduct, err)
	}
	return nil
}

func (ps ProductService) RemoveFavoriteProduct(ctx context.Context, actor Actor, productID int64) error {
	if ps.productFavoriteRepo == nil {
		return ErrRemoveFavoriteProduct
	}

	if err := ps.productFavoriteRepo.RemoveFavoriteProduct(ctx, actor.UserID, productID); err != nil {
		return fmt.Errorf("%w: %v", ErrRemoveFavoriteProduct, err)
	}
	return nil
}

func (ps ProductService) IsFavoriteProduct(ctx context.Context, actor Actor, productID int64) (bool, error) {
	if ps.productFavoriteRepo == nil {
		return false, ErrCheckFavoriteProduct
	}
	if err := ps.ensureProductCanBeFavorited(ctx, productID); err != nil {
		return false, err
	}

	favorite, err := ps.productFavoriteRepo.IsFavoriteProduct(ctx, actor.UserID, productID)
	if err != nil {
		return false, fmt.Errorf("%w: %v", ErrCheckFavoriteProduct, err)
	}
	return favorite, nil
}

func (ps ProductService) ensureProductCanBeFavorited(ctx context.Context, productID int64) error {
	product, err := ps.productRepo.GetProductByID(ctx, productID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return ErrProductNotFound
		}
		return fmt.Errorf("%w: %v", ErrGetProductByID, err)
	}
	if product.DeletedAt != nil {
		return ErrProductDeleted
	}
	return nil
}

func (ps ProductService) CreateProduct(ctx context.Context, actor Actor, pc m.ProductCreate) (id int64, err error) {
	if !actor.IsAdmin() && !actor.HasRole(m.RoleSeller) {
		return 0, ErrPermissionDenied
	}

	s, err := ps.sellerRepo.GetSellerByID(ctx, pc.SellerID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return 0, ErrSellerNotFound
		}
		return 0, fmt.Errorf("%w: %v", ErrGetSellerByID, err)
	}
	if !actor.IsAdmin() {
		if s.UserID != actor.UserID {
			return 0, ErrNotYourSeller
		}
	}

	create := func(ctx context.Context) error {
		id, err = ps.productRepo.CreateProduct(ctx, pc)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrCreateProduct, err)
		}
		if len(pc.CategoryIDs) > 0 {
			if ps.productCategoryRepo == nil {
				return ErrUpdateProductCategories
			}
			if err = ps.productCategoryRepo.ReplaceProductCategories(ctx, id, pc.CategoryIDs); err != nil {
				return fmt.Errorf("%w: %v", ErrUpdateProductCategories, err)
			}
		}
		return nil
	}
	if len(pc.CategoryIDs) > 0 {
		if err = ps.txManager.WithTransaction(ctx, create); err != nil {
			return 0, err
		}
	} else if err = create(ctx); err != nil {
		return 0, err
	}
	return id, nil
}

func (ps ProductService) UpdateProduct(ctx context.Context, actor Actor, id int64, pu m.ProductUpdate) (p m.Product, err error) {
	if !actor.IsAdmin() && !actor.HasRole(m.RoleSeller) {
		return m.Product{}, ErrPermissionDenied
	}

	// Lock first, then authorize and validate against fresh product data.
	var updated m.Product
	if err = ps.txManager.WithTransaction(ctx, func(ctx context.Context) error {
		locked, err := ps.productRepo.GetProductByIDForUpdate(ctx, id)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				return ErrProductNotFound
			}
			return fmt.Errorf("%w: %v", ErrGetProductByID, err)
		}
		if locked.DeletedAt != nil {
			return ErrProductDeleted
		}
		s, err := ps.sellerRepo.GetSellerByID(ctx, locked.SellerID)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				return ErrSellerNotFound
			}
			return fmt.Errorf("%w: %v", ErrGetSellerByID, err)
		}
		if !actor.IsAdmin() && s.UserID != actor.UserID {
			return ErrNotYourSeller
		}
		if pu.StockQuantity != nil && *pu.StockQuantity < locked.ReservedQuantity {
			return ErrStockBelowReserved
		}
		if !productUpdateHasProductFields(pu) && pu.CategoryIDs == nil {
			return ErrNoChangesInUpdate
		}
		updated = locked
		if productUpdateHasProductFields(pu) {
			updated, err = ps.productRepo.UpdateProduct(ctx, id, pu)
			if err != nil {
				if errors.Is(err, ErrStockBelowReserved) {
					return ErrStockBelowReserved
				}
				return fmt.Errorf("%w: %v", ErrUpdateProduct, err)
			}
		}
		if pu.CategoryIDs != nil {
			if ps.productCategoryRepo == nil {
				return ErrUpdateProductCategories
			}
			if err = ps.productCategoryRepo.ReplaceProductCategories(ctx, id, *pu.CategoryIDs); err != nil {
				return fmt.Errorf("%w: %v", ErrUpdateProductCategories, err)
			}
		}
		return nil
	}); err != nil {
		return m.Product{}, err
	}
	return updated, nil
}

func (ps ProductService) ReplaceProductCategories(ctx context.Context, actor Actor, productID int64, categoryIDs []int64) error {
	if !actor.IsAdmin() && !actor.HasRole(m.RoleSeller) {
		return ErrPermissionDenied
	}
	if ps.productCategoryRepo == nil {
		return ErrUpdateProductCategories
	}

	return ps.txManager.WithTransaction(ctx, func(ctx context.Context) error {
		product, err := ps.productRepo.GetProductByIDForUpdate(ctx, productID)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				return ErrProductNotFound
			}
			return fmt.Errorf("%w: %v", ErrGetProductByID, err)
		}
		if product.DeletedAt != nil {
			return ErrProductDeleted
		}
		seller, err := ps.sellerRepo.GetSellerByID(ctx, product.SellerID)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				return ErrSellerNotFound
			}
			return fmt.Errorf("%w: %v", ErrGetSellerByID, err)
		}
		if !actor.IsAdmin() && seller.UserID != actor.UserID {
			return ErrNotYourSeller
		}
		if err = ps.productCategoryRepo.ReplaceProductCategories(ctx, productID, categoryIDs); err != nil {
			return fmt.Errorf("%w: %v", ErrUpdateProductCategories, err)
		}
		return nil
	})
}

func (ps ProductService) DeleteProductByID(ctx context.Context, actor Actor, id int64) (err error) {
	if !actor.IsAdmin() && !actor.HasRole(m.RoleSeller) {
		return ErrPermissionDenied
	}

	p, err := ps.productRepo.GetProductByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return ErrProductNotFound
		}
		return fmt.Errorf("%w: %v", ErrGetProductByID, err)
	}

	s, err := ps.sellerRepo.GetSellerByID(ctx, p.SellerID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return ErrSellerNotFound
		}
		return fmt.Errorf("%w: %v", ErrGetSellerByID, err)
	}
	if !actor.IsAdmin() {
		if s.UserID != actor.UserID {
			return ErrNotYourSeller
		}
	}

	err = ps.productRepo.DeleteProductByID(ctx, id)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDeleteProduct, err)
	}
	return nil
}

func productUpdateHasProductFields(pu m.ProductUpdate) bool {
	return pu.SellerID != nil ||
		pu.Name != nil ||
		pu.Description != nil ||
		pu.Price != nil ||
		pu.StockQuantity != nil
}
