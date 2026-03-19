package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	m "github.com/beastixq/marketplace/internal/model"
)

type ProductRepo interface {
	GetProducts(ctx context.Context, options m.CatalogOptions) (ps []m.Product, err error)
	GetProductByID(ctx context.Context, id int64) (p m.Product, err error)
	GetProductPriceHistory(ctx context.Context, pid int64, dateFrom time.Time, dateTo time.Time) (ph []m.ProductPriceHistory, err error)
	CreateProduct(ctx context.Context, pc m.ProductCreate) (id int64, err error)
	UpdateProduct(ctx context.Context, id int64, pu m.ProductUpdate) (p m.Product, err error)
	DeleteProductByID(ctx context.Context, id int64) (err error)
}

type ProductService struct {
	productRepo ProductRepo
	reviewRepo  ReviewRepo
	sellerRepo  SellerRepo
}

func NewProductService(productRepo ProductRepo, reviewRepo ReviewRepo, sellerRepo SellerRepo) ProductService {
	return ProductService{
		productRepo: productRepo,
		reviewRepo:  reviewRepo,
		sellerRepo:  sellerRepo}
}

func (psvc ProductService) GetProducts(ctx context.Context, options m.CatalogOptions) (ps []m.Product, err error) {
	ps, err = psvc.productRepo.GetProducts(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrGetProducts, err)
	}
	if len(ps) == 0 {
		return nil, ErrNotFound
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

func (ps ProductService) CreateProduct(ctx context.Context, userID int64, pc m.ProductCreate) (id int64, err error) {
	s, err := ps.sellerRepo.GetSellerByID(ctx, pc.SellerID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return 0, ErrSellerNotFound
		}
		return 0, fmt.Errorf("%w: %v", ErrGetSellerByID, err)
	}
	if s.UserID != userID {
		return 0, ErrNotYourSeller
	}

	id, err = ps.productRepo.CreateProduct(ctx, pc)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrCreateProduct, err)
	}
	return id, nil
}

func (ps ProductService) UpdateProduct(ctx context.Context, userID, id int64, pu m.ProductUpdate) (p m.Product, err error) {
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
	if s.UserID != userID {
		return m.Product{}, ErrNotYourSeller
	}

	p, err = ps.productRepo.UpdateProduct(ctx, id, pu)
	if err != nil {
		return m.Product{}, fmt.Errorf("%w: %v", ErrUpdateProduct, err)
	}
	return p, nil
}

func (ps ProductService) DeleteProductByID(ctx context.Context, userID, id int64) (err error) {
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
	if s.UserID != userID {
		return ErrNotYourSeller
	}

	err = ps.productRepo.DeleteProductByID(ctx, id)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDeleteProduct, err)
	}
	return nil
}
