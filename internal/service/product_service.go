package service

import (
	"context"
	"time"

	m "github.com/beastixq/marketplace/internal/model"
)

type ProductRepo interface {
	GetProducts(ctx context.Context, options m.CatalogOptions) (p []m.Product, err error)
	GetProductByID(ctx context.Context, id int64) (p m.Product, err error)
	GetProductPriceHistory(ctx context.Context, pid int64, dateFrom time.Time, dateTo time.Time) (ph []m.ProductPriceHistory, err error)
	CreateProduct(ctx context.Context, pc m.ProductCreate) (id int64, err error)
	UpdateProduct(ctx context.Context, id int64, pu m.ProductUpdate) (p m.Product, err error)
	DeleteProductByID(ctx context.Context, id int64) (err error)
}

type ReviewRepo interface {
	GetReviewByID(ctx context.Context, id int64) (r m.Review, err error)
	GetReviewsByProductID(ctx context.Context, pid int64, opts m.PaginationOpts) (rs []m.Review, err error)
	CreateReview(ctx context.Context, rc m.ReviewCreate) (id int64, err error)
	UpdateReview(ctx context.Context, id int64, ru m.ReviewUpdate) (r m.Review, err error)
	DeleteReviewByID(ctx context.Context, id int64) (err error)
}
