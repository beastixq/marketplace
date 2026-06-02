package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"

	m "github.com/beastixq/marketplace/internal/model"
	svc "github.com/beastixq/marketplace/internal/service"
)

type ReviewRepoCache struct {
	inner svc.ReviewRepo
	rdb   *redis.Client
	ttl   time.Duration
}

var (
	_ svc.ReviewRepo            = (*ReviewRepoCache)(nil)
	_ svc.ReviewPurchaseChecker = (*ReviewRepoCache)(nil)
)

func NewReviewRepoCache(inner svc.ReviewRepo, rdb *redis.Client, ttl time.Duration) *ReviewRepoCache {
	return &ReviewRepoCache{inner: inner, rdb: rdb, ttl: ttl}
}

func (c *ReviewRepoCache) GetReviewByID(ctx context.Context, id int64) (m.Review, error) {
	return c.inner.GetReviewByID(ctx, id)
}

func (c *ReviewRepoCache) GetReviewsByProductID(ctx context.Context, pid int64, opts m.PaginationOpts) ([]m.Review, error) {
	return GetOrLoad(ctx, c.rdb, ProductReviewsKey(pid, opts), c.ttl, func(ctx context.Context) ([]m.Review, error) {
		return c.inner.GetReviewsByProductID(ctx, pid, opts)
	})
}

func (c *ReviewRepoCache) CreateReview(ctx context.Context, rc m.ReviewCreate) (int64, error) {
	id, err := c.inner.CreateReview(ctx, rc)
	if err != nil {
		return 0, err
	}
	c.invalidateProductReviews(ctx, rc.ProductID)
	return id, nil
}

func (c *ReviewRepoCache) UpdateReview(ctx context.Context, id int64, ru m.ReviewUpdate) (m.Review, error) {
	review, err := c.inner.UpdateReview(ctx, id, ru)
	if err != nil {
		return review, err
	}
	c.invalidateProductReviews(ctx, review.ProductID)
	return review, nil
}

func (c *ReviewRepoCache) DeleteReviewByID(ctx context.Context, id int64) error {
	review, getErr := c.inner.GetReviewByID(ctx, id)
	if err := c.inner.DeleteReviewByID(ctx, id); err != nil {
		return err
	}
	invalidateAfterCommit(ctx, func(ctx context.Context) {
		if getErr == nil {
			invalidateProductReadModels(ctx, c.rdb, review.ProductID)
			return
		}
		deleteByPrefix(ctx, c.rdb, ProductCatalogPrefix())
	})
	return nil
}

func (c *ReviewRepoCache) UserPurchasedProduct(ctx context.Context, userID int64, productID int64) (bool, error) {
	checker, ok := c.inner.(svc.ReviewPurchaseChecker)
	if !ok {
		return false, svc.ErrCheckReviewPurchase
	}
	return checker.UserPurchasedProduct(ctx, userID, productID)
}

func (c *ReviewRepoCache) invalidateProductReviews(ctx context.Context, productID int64) {
	invalidateAfterCommit(ctx, func(ctx context.Context) {
		invalidateProductReadModels(ctx, c.rdb, productID)
	})
}
