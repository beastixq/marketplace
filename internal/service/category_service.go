package service

import (
	"context"
	"errors"
	"fmt"

	m "github.com/beastixq/marketplace/internal/model"
)

//go:generate mockgen -package mock_service -destination ../mocks/service/mock_category_repo.go github.com/beastixq/marketplace/internal/service CategoryRepo
type CategoryRepo interface {
	GetCategories(ctx context.Context, opts m.PaginationOpts) (cs []m.Category, err error)
	GetCategoryByID(ctx context.Context, id int64) (c m.Category, err error)
	CreateCategory(ctx context.Context, cc m.CategoryCreate) (id int64, err error)
	UpdateCategory(ctx context.Context, id int64, cu m.CategoryUpdate) (c m.Category, err error)
	DeleteCategoryByID(ctx context.Context, id int64) (err error)
}

type CategoryService struct {
	categoryRepo CategoryRepo
}

func NewCategoryService(categoryRepo CategoryRepo) CategoryService {
	return CategoryService{categoryRepo: categoryRepo}
}

func (cs CategoryService) GetCategories(ctx context.Context, opts m.PaginationOpts) (categories []m.Category, err error) {
	categories, err = cs.categoryRepo.GetCategories(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrGetCategories, err)
	}
	return categories, nil
}

func (cs CategoryService) GetCategoryByID(ctx context.Context, id int64) (c m.Category, err error) {
	c, err = cs.categoryRepo.GetCategoryByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return m.Category{}, ErrCategoryNotFound
		}
		return m.Category{}, fmt.Errorf("%w: %v", ErrGetCategoryByID, err)
	}
	return c, nil
}

func (cs CategoryService) CreateCategory(ctx context.Context, cc m.CategoryCreate) (id int64, err error) {
	id, err = cs.categoryRepo.CreateCategory(ctx, cc)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrCreateCategory, err)
	}
	return id, nil
}

func (cs CategoryService) UpdateCategory(ctx context.Context, id int64, cu m.CategoryUpdate) (c m.Category, err error) {
	c, err = cs.categoryRepo.UpdateCategory(ctx, id, cu)
	if err != nil {
		return m.Category{}, fmt.Errorf("%w: %v", ErrUpdateCategory, err)
	}
	return c, nil
}

func (cs CategoryService) DeleteCategoryByID(ctx context.Context, id int64) (err error) {
	err = cs.categoryRepo.DeleteCategoryByID(ctx, id)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDeleteCategory, err)
	}
	return nil
}
