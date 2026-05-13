package cache

import (
	"context"
	"log/slog"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	m "github.com/beastixq/marketplace/internal/model"
	svc "github.com/beastixq/marketplace/internal/service"
)

// ProductRepoCache decorates svc.ProductRepo with Redis cache-aside reads
// and write-through invalidation. It also satisfies svc.ProductCategoryRepo
// (pass-through) so the service constructor's type assertion still succeeds.
type ProductRepoCache struct {
	inner svc.ProductRepo
	rdb   *redis.Client
	ttl   time.Duration
}

// Compile-time interface checks. Decorator must satisfy BOTH interfaces
// the underlying repo satisfies, otherwise the type assertion in
// NewProductService will silently lose ProductCategoryRepo features.
// var (
// 	_ svc.ProductRepo         = (*ProductRepoCache)(nil)
// 	_ svc.ProductCategoryRepo = (*ProductRepoCache)(nil)
// )

func NewProductRepoCache(inner svc.ProductRepo, rdb *redis.Client, ttl time.Duration) *ProductRepoCache {
	return &ProductRepoCache{inner: inner, rdb: rdb, ttl: ttl}
}

// ProductByIDKey is the currently implemented Redis product entity key:
//
//	products:{id}    TTL configured by wrapper, 5m in cmd/api
func ProductByIDKey(id int64) string {
	return "products:" + strconv.FormatInt(id, 10)
}

// invalidateProduct deletes products:{id} from cache. Fire-and-forget: logs
// on failure but never fails the mutation.
func (c *ProductRepoCache) invalidateProduct(ctx context.Context, id int64) {
	if err := c.rdb.Del(ctx, ProductByIDKey(id)).Err(); err != nil {
		slog.Default().Warn("cache invalidation failed", "key", ProductByIDKey(id), "error", err)
	}
}

// ---------- Cached reads ----------

func (c *ProductRepoCache) GetProductByID(ctx context.Context, id int64) (m.Product, error) {
	return GetOrLoad(ctx, c.rdb, ProductByIDKey(id), c.ttl, func(ctx context.Context) (m.Product, error) {
		return c.inner.GetProductByID(ctx, id)
	})
}

// ---------- Pass-through reads (not cached this round) ----------

func (c *ProductRepoCache) GetProducts(ctx context.Context, options m.CatalogOptions) ([]m.Product, error) {
	return c.inner.GetProducts(ctx, options)
}

func (c *ProductRepoCache) GetProductByIDForUpdate(ctx context.Context, id int64) (m.Product, error) {
	// FOR UPDATE locks the row in a transaction. Caching this would defeat
	// the lock semantics. Always pass through.
	return c.inner.GetProductByIDForUpdate(ctx, id)
}

func (c *ProductRepoCache) GetProductPriceHistory(ctx context.Context, pid int64, dateFrom, dateTo time.Time) ([]m.ProductPriceHistory, error) {
	return c.inner.GetProductPriceHistory(ctx, pid, dateFrom, dateTo)
}

// ---------- Mutations and invalidation ----------

func (c *ProductRepoCache) CreateProduct(ctx context.Context, pc m.ProductCreate) (int64, error) {
	return c.inner.CreateProduct(ctx, pc)
}

func (c *ProductRepoCache) UpdateProduct(ctx context.Context, id int64, pu m.ProductUpdate) (m.Product, error) {
	p, err := c.inner.UpdateProduct(ctx, id, pu)
	if err != nil {
		return p, err
	}
	c.invalidateProduct(ctx, id)
	return p, nil
}

func (c *ProductRepoCache) ChangeStockAndReserved(ctx context.Context, productID int64, stockDelta, reservedDelta int) error {
	// Stock changes happen on every order. Caching products:{id} means stock
	// can be stale until TTL expiry. See docs/cache.md for the current
	// cache contract and known staleness caveats.
	return c.inner.ChangeStockAndReserved(ctx, productID, stockDelta, reservedDelta)
}

func (c *ProductRepoCache) DeleteProductByID(ctx context.Context, id int64) error {
	err := c.inner.DeleteProductByID(ctx, id)
	if err != nil {
		return err
	}
	c.invalidateProduct(ctx, id)
	return nil
}

// ---------- ProductCategoryRepo pass-through ----------
//
// Required so the type assertion in service.NewProductService still detects
// category support after wrapping. Implementing only ProductRepo would
// silently disable category features — see CLAUDE.md note about decorator
// + secondary-interface assertions.

func (c *ProductRepoCache) GetProductCategories(ctx context.Context, productID int64) ([]m.Category, error) {
	pcr, ok := c.inner.(svc.ProductCategoryRepo)
	if !ok {
		return nil, nil
	}
	return pcr.GetProductCategories(ctx, productID)
}

func (c *ProductRepoCache) ReplaceProductCategories(ctx context.Context, productID int64, categoryIDs []int64) error {
	pcr, ok := c.inner.(svc.ProductCategoryRepo)
	if !ok {
		return nil
	}
	return pcr.ReplaceProductCategories(ctx, productID, categoryIDs)
}
