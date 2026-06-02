# Cache

PostgreSQL is the source of truth. Redis is a performance cache and token
revocation store; cached state must never be required for core business
correctness.

## Runtime Wiring

`cmd/api/main.go` connects Redis only when `redis.enabled` is true. If Redis is
enabled and the initial ping fails, the API process exits. To run without Redis,
set `redis.enabled: false`.

When Redis is connected, `cmd/api` wraps:

- `ProductRepo` with `cache.ProductRepoCache`;
- `CategoryRepo` with `cache.CategoryRepoCache`;
- `ReviewRepo` with `cache.ReviewRepoCache`;
- `AuthService` with `cache.TokenBlocklist`.

`cmd/techui` does not enable Redis by default because it builds repositories
through `internal/component/repository.New`. The component package can wrap the
same read repositories when `NewFromPoolWithCache` receives a Redis client.

## Implemented Keys

| Key | TTL | Read Path | Invalidation |
| --- | --- | --- | --- |
| `products:{id}` | `redis.product_ttl` | `ProductRepoCache.GetProductByID` | Product update/delete, stock or reserve change, product category replacement, review mutation. |
| `products:catalog:{canonical-query}` | `redis.catalog_ttl` | `ProductRepoCache.GetProducts` | Any product change, stock or reserve change, product category replacement, category CRUD, review mutation. |
| `categories:list:{canonical-query}` | `redis.category_ttl` | `CategoryRepoCache.GetCategories` | Category CRUD. |
| `products:{id}:reviews:page={page}&limit={limit}` | `redis.review_ttl` | `ReviewRepoCache.GetReviewsByProductID` | Review mutation for the product. |
| `sessions:{jti}` | Remaining JWT lifetime | `TokenBlocklist.Contains` | Key TTL expiration. Logout writes the key. |

Catalog and category keys use readable canonical query strings instead of hash
values. Equivalent options produce the same key: pagination is explicit,
category filters are sorted, and parameters are URL-encoded in stable order.

Examples:

```text
products:catalog:limit=12&page=1
products:catalog:category=Books&limit=12&page=1&sort=asc
categories:list:limit=1000&page=1&root=true
```

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

## Invalidation

Cache decorators invalidate Redis from `internal/cache`, not from handlers or
web controllers. Prefix invalidation uses Redis `SCAN`, never `KEYS`.

When a mutation runs inside the service transaction manager, invalidation is
registered as an after-commit hook. This avoids deleting a key before commit and
then letting a concurrent read repopulate Redis with stale PostgreSQL data.

## Non-Cached Areas

These areas intentionally stay uncached:

- `GetProductByIDForUpdate`, because row locking must hit PostgreSQL inside a
  transaction;
- product price history, because it is audit data and less frequently read;
- carts, orders, payments, profiles, and reports, because they are
  user-specific, rapidly changing, or authorization-sensitive.

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
