package service

import (
	"context"

	m "github.com/beastixq/marketplace/internal/model"
)

type CategoryRepo interface {
	GetCategories(ctx context.Context, opts m.PaginationOpts) (cs []m.Category, err error)
	GetCategoryByID(ctx context.Context, id int64) (c m.Category, err error)
	CreateCategory(ctx context.Context, cc m.CategoryCreate) (id int64, err error)
	UpdateCategory(ctx context.Context, id int64, cu m.CategoryUpdate) (c m.Category, err error)
	DeleteCategoryByID(ctx context.Context, id int64) (err error)
}

type CategoryService
