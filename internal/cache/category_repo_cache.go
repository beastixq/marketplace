package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"

	m "github.com/beastixq/marketplace/internal/model"
	svc "github.com/beastixq/marketplace/internal/service"
)

type CategoryRepoCache struct {
	inner svc.CategoryRepo
	rdb   *redis.Client
	ttl   time.Duration
}

var _ svc.CategoryRepo = (*CategoryRepoCache)(nil)

func NewCategoryRepoCache(inner svc.CategoryRepo, rdb *redis.Client, ttl time.Duration) *CategoryRepoCache {
	return &CategoryRepoCache{inner: inner, rdb: rdb, ttl: ttl}
}

func (c *CategoryRepoCache) GetCategories(ctx context.Context, opts m.CategoryListOptions) ([]m.Category, error) {
	return GetOrLoad(ctx, c.rdb, CategoryListKey(opts), c.ttl, func(ctx context.Context) ([]m.Category, error) {
		return c.inner.GetCategories(ctx, opts)
	})
}

func (c *CategoryRepoCache) GetCategoryByID(ctx context.Context, id int64) (m.Category, error) {
	return c.inner.GetCategoryByID(ctx, id)
}

func (c *CategoryRepoCache) CreateCategory(ctx context.Context, cc m.CategoryCreate) (int64, error) {
	id, err := c.inner.CreateCategory(ctx, cc)
	if err != nil {
		return 0, err
	}
	c.invalidateCategoryReadModels(ctx)
	return id, nil
}

func (c *CategoryRepoCache) UpdateCategory(ctx context.Context, id int64, cu m.CategoryUpdate) (m.Category, error) {
	category, err := c.inner.UpdateCategory(ctx, id, cu)
	if err != nil {
		return category, err
	}
	c.invalidateCategoryReadModels(ctx)
	return category, nil
}

func (c *CategoryRepoCache) DeleteCategoryByID(ctx context.Context, id int64) error {
	if err := c.inner.DeleteCategoryByID(ctx, id); err != nil {
		return err
	}
	c.invalidateCategoryReadModels(ctx)
	return nil
}

func (c *CategoryRepoCache) invalidateCategoryReadModels(ctx context.Context) {
	invalidateAfterCommit(ctx, func(ctx context.Context) {
		deleteByPrefix(ctx, c.rdb, CategoryListPrefix())
		deleteByPrefix(ctx, c.rdb, ProductCatalogPrefix())
	})
}
