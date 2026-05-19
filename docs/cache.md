# Cache

PostgreSQL is the source of truth. Redis is a performance cache only; cache
state must never be required for core business correctness.

## Current Runtime Wiring

`cmd/api/main.go` connects Redis only when `redis.enabled` is true. If Redis is
enabled and the initial ping fails, the API process exits. To run without Redis,
set `redis.enabled: false`.

When Redis is connected, `cmd/api` wraps `ProductRepo` with
`cache.NewProductRepoCache(productRepo, rdb, 5*time.Minute)`.

`cmd/techui` does not enable Redis caching by default because it builds
repositories through `internal/component/repository.New`. The component package
can wrap `ProductRepo` when `NewFromPoolWithCache` receives a Redis client.

## Implemented Keys

| Key | TTL | Read Path | Invalidation |
| --- | --- | --- | --- |
| `products:{id}` | `5m` in `cmd/api` | `ProductRepoCache.GetProductByID` | After `UpdateProduct` and `DeleteProductByID`. |

No other Redis keys are currently used by the runtime application.

## Cache-Aside Contract

`internal/cache/cacheaside.go` implements generic cache-aside reads:

- On cache hit, unmarshal JSON and return the cached value.
- On cache miss, call the loader, marshal JSON, write with TTL, and return the
  loaded value.
- Loader errors are returned unchanged and are not cached.
- Redis read errors fall back to the loader and log a warning.
- Redis write, marshal, and unmarshal errors log warnings but do not fail the
  business read path.
- Negative caching is intentionally not implemented.

## Product Cache Behavior

`ProductRepoCache` satisfies both `service.ProductRepo` and
`service.ProductCategoryRepo`, so category-related product service behavior
continues to work after wrapping.

Cached:

- `GetProductByID`

Passed through without caching:

- catalog/list reads through `GetProducts`;
- `GetProductByIDForUpdate`, because row locking must hit PostgreSQL inside a
  transaction;
- price history reads;
- product category reads and replacements;
- product creation;
- stock/reservation changes.

Current staleness caveats:

- `ChangeStockAndReserved` does not invalidate `products:{id}`. Product stock,
  reserved quantity, available quantity, and derived display state may be stale
  until the TTL expires.
- Review create/update/delete does not invalidate `products:{id}`. Product and
  seller ratings are recalculated by a database trigger, but cached product
  rating can be stale until the TTL expires.
- Product category replacement does not invalidate `products:{id}`. The cached
  product entity does not include category rows, but pages that combine product
  and category data should be checked carefully before caching broader views.

## Documented But Not Implemented

These keys are architectural conventions, not current runtime behavior:

| Key | Intended TTL | Intended Invalidation |
| --- | --- | --- |
| `products:catalog:page:{n}` | `5m` | Any product change. |
| `categories:tree` | `1h` | Category CRUD. |
| `sessions:{token}` | Session lifetime | Logout/token revocation. |

`AuthService` has a `TokenBlocklist` interface, but `cmd/api` currently passes
`nil`, so logout does not store revoked JWT IDs in Redis.

## Rules For Extending Cache

- Keep Redis mechanics in `internal/cache` or a cache-aware decorator.
- Do not invalidate cache from handlers or web controllers.
- Do not cache `FOR UPDATE` reads or anything that must participate in database
  locking.
- Use explicit TTLs and document each key in this file.
- Do not cache auth-sensitive data unless the key format, TTL, and invalidation
  are documented first.
- Cache failures should not break reads after startup. Mutations may log failed
  invalidation but should not hide the database result unless a use case
  explicitly requires cache consistency.
