package cache

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

// GetOrLoad implements the read-side of the cache-aside pattern.
//
// Behavior contract (what your implementation must provide):
//
//   - On cache HIT: unmarshal cached bytes into T and return it.
//     The loader MUST NOT be called.
//
//   - On cache MISS (redis.Nil): call loader to fetch from source of truth.
//     If loader succeeds, write the result to cache with ttl, then return it.
//     A failure to write the cache must NOT fail the read — log and continue.
//
//   - On any other Redis error: degrade gracefully — call loader and return its
//     result. A broken cache must never break the read path. Log the cache
//     error so the operator notices.
//
//   - On loader error: return the error unchanged. Do not write cache.
//     (Negative caching is intentionally NOT done here per design decision.)
//
// Marshal format: encoding/json. Keep it simple; revisit later if hot.
func GetOrLoad[T any](
	ctx context.Context,
	rdb *redis.Client,
	key string,
	ttl time.Duration,
	loader func(ctx context.Context) (T, error),
) (T, error) {
	val, err := rdb.Get(ctx, key).Bytes()
	var entity T
	if errors.Is(err, redis.Nil) {
		// miss
		if entity, err = loader(ctx); err != nil {
			return entity, err
		}
		// miss succeeded -> write to cache
		value, err := json.Marshal(entity)
		if err != nil {
			slog.Default().Warn("Cache marshal failed", "err", err, "key", key)
			return entity, nil
		}
		err = rdb.Set(ctx, key, value, ttl).Err()
		if err != nil {
			slog.Default().Warn("Failed to write to cache", "err", err, "key", key)
		}
		return entity, nil
	} else if err != nil {
		slog.Default().Warn("Redis error", "err", err, "key", key)
		return loader(ctx)
	}
	// hit
	if err = json.Unmarshal(val, &entity); err != nil {
		slog.Default().Warn("Cache unmarshal failed", "err", err, "key", key)
		// call loader to get from source of truth
		entity, err = loader(ctx)
		return entity, err
	}
	return entity, nil
}
