package cache

import (
	"context"
	"log/slog"

	svc "github.com/beastixq/marketplace/internal/service"
	"github.com/redis/go-redis/v9"
)

func invalidateAfterCommit(ctx context.Context, fn func(context.Context)) {
	if svc.AddAfterCommit(ctx, fn) {
		return
	}
	fn(ctx)
}

func deleteKeys(ctx context.Context, rdb *redis.Client, keys ...string) {
	if len(keys) == 0 {
		return
	}
	if err := rdb.Del(ctx, keys...).Err(); err != nil {
		slog.Default().Warn("cache invalidation failed", "keys", keys, "error", err)
	}
}

func deleteByPrefix(ctx context.Context, rdb *redis.Client, prefix string) {
	if prefix == "" {
		return
	}
	var cursor uint64
	for {
		keys, next, err := rdb.Scan(ctx, cursor, prefix+"*", 100).Result()
		if err != nil {
			slog.Default().Warn("cache prefix scan failed", "prefix", prefix, "error", err)
			return
		}
		deleteKeys(ctx, rdb, keys...)
		cursor = next
		if cursor == 0 {
			return
		}
	}
}

func invalidateProductReadModels(ctx context.Context, rdb *redis.Client, productID int64) {
	deleteKeys(ctx, rdb, ProductByIDKey(productID))
	deleteByPrefix(ctx, rdb, ProductReviewsPrefix(productID))
	deleteByPrefix(ctx, rdb, ProductCatalogPrefix())
}
