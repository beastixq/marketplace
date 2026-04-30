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

type ProductService struct {
	productRepo ProductRepo
	reviewRepo  ReviewRepo
	sellerRepo  SellerRepo
	txManager   TxManager
}

func NewProductService(productRepo ProductRepo, reviewRepo ReviewRepo, sellerRepo SellerRepo, txManager TxManager) ProductService {
	return ProductService{
		productRepo: productRepo,
		reviewRepo:  reviewRepo,
		sellerRepo:  sellerRepo,
		txManager:   txManager,
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

	id, err = ps.productRepo.CreateProduct(ctx, pc)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrCreateProduct, err)
	}
	return id, nil
}

func (ps ProductService) UpdateProduct(ctx context.Context, actor Actor, id int64, pu m.ProductUpdate) (p m.Product, err error) {
	if !actor.IsAdmin() && !actor.HasRole(m.RoleSeller) {
		return m.Product{}, ErrPermissionDenied
	}

	p, err = ps.productRepo.GetProductByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return m.Product{}, ErrProductNotFound
		}
		return m.Product{}, fmt.Errorf("%w: %v", ErrGetProductByID, err)
	}

	s, err := ps.sellerRepo.GetSellerByID(ctx, p.SellerID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return m.Product{}, ErrSellerNotFound
		}
		return m.Product{}, fmt.Errorf("%w: %v", ErrGetSellerByID, err)
	}
	if !actor.IsAdmin() {
		if s.UserID != actor.UserID {
			return m.Product{}, ErrNotYourSeller
		}
	}

	// Lock and validate inside tx so DeletedAt and stock-vs-reserved checks
	// run against fresh data; concurrent checkout/cancel may have changed
	// reserved_quantity since the outer read.
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
		if pu.StockQuantity != nil && *pu.StockQuantity < locked.ReservedQuantity {
			return ErrStockBelowReserved
		}
		updated, err = ps.productRepo.UpdateProduct(ctx, id, pu)
		if err != nil {
			if errors.Is(err, ErrStockBelowReserved) {
				return ErrStockBelowReserved
			}
			return fmt.Errorf("%w: %v", ErrUpdateProduct, err)
		}
		return nil
	}); err != nil {
		return m.Product{}, err
	}
	return updated, nil
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
