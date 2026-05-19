# Research: Favorite Products

## Decision: Model favorites as a separate product-level relationship

Use a new `product_favorites` relationship between `users` and `products` with
one row per user-product pair and `created_at` for ordering.

**Rationale**: The spec requires many-to-many behavior and uniqueness per
user-product pair. A separate relationship keeps product data, user data, and
favorite state independent.

**Alternatives considered**:

- Array or JSON field on `users`: rejected because it weakens constraints,
  indexing, joins, and per-product cleanup.
- Counter or denormalized field on `products`: rejected because counts are not
  in scope and it cannot represent each user's favorite state.

## Decision: Enforce uniqueness with database and idempotent service behavior

Use a composite key or unique constraint on `(user_id, product_id)`. Add favorite
with conflict-tolerant behavior and expose duplicate add attempts as a stable
success outcome with no duplicate row.

**Rationale**: The spec requires duplicate prevention and stable repeated
favorite actions. The database should enforce the invariant, and service logic
should make retries safe.

**Alternatives considered**:

- Return conflict for duplicate favorite attempts: rejected because the spec
  asks for a clear stable result and no duplicate, not an error.
- Check-then-insert only in service: rejected because concurrent requests can
  still race without a database constraint.

## Decision: Treat "active and visible" as `products.deleted_at IS NULL`

For the current schema, active visible products are products whose
`deleted_at` is null. Favorite creation and favorite-state checks reject
soft-deleted products at the service layer. Favorite listing joins active
products only, so old favorite rows for soft-deleted products are not
returned. Favorite removal may clear a relationship even if the product
has since been soft-deleted.

**Rationale**: The current product model has `deleted_at` but no separate
visibility, archived, or published state. Catalog reads already use
`deleted_at IS NULL` as the active product boundary.

**Alternatives considered**:

- Add a new visibility/status column: rejected because the spec does not request
  product visibility management and that would expand scope.
- Hard-delete favorite rows when a product is soft-deleted: rejected because the
  existing product delete path is a soft delete and this feature does not need
  a cross-feature cleanup policy.

## Decision: Pre-check visibility in service, plain INSERT in repository, accept a tiny race window

`FavoriteService.AddFavorite` calls `activeProduct` once, then the repository
does a plain `INSERT ... ON CONFLICT DO NOTHING`. The repository contract
exposes only set-membership semantics: `created == true` means a row was
inserted, `created == false` means the pair already existed, and
`service.ErrNotFound` means the user or product FK target is missing.
Product-visibility policy lives entirely in the service.

In the microseconds between the visibility pre-check and the insert, a
concurrent soft-delete could land. The favorite row is then created against
a now-soft-deleted product. That stale row is invisible to every user-facing
read path (`ListFavoriteProductsByUserID` joins `WHERE deleted_at IS NULL`
and `IsProductFavorite` returns `ErrProductDeleted` before consulting the
favorite row), and the FK `ON DELETE CASCADE` cleans it up when the product
is hard-deleted.

**Rationale**: Reserve `TxManager.WithTransaction` plus `SELECT ... FOR UPDATE`
for invariants where a stale row would corrupt user-visible state or money/
inventory accounting. Favorites do not have that failure mode; the stronger
mechanism would write-lock a hot `products` row for every favorite add for
no observable benefit. The chosen approach also keeps the repository contract
minimal: the service never has to reason about how the repo handles conflicts
or visibility, so the layer boundary stays clean.

**Alternatives considered**:

- Repository-side atomic `INSERT ... SELECT FROM products WHERE deleted_at IS NULL ... ON CONFLICT DO NOTHING RETURNING true`: rejected because it forces the service to know which `created == false` cases mean "already favorited" and which mean "product soft-deleted concurrently". That is a layer-boundary leak — service comments would have to describe repository SQL.
- `TxManager.WithTransaction` + `SELECT ... FOR UPDATE` on `products`: rejected for favorites specifically. Acceptable for invariants that protect inventory, money, or order state transitions; out of proportion for set-membership.

## Decision: Add authenticated JSON routes under `/api/v1/favorites`

Expose favorite behavior inside the authenticated route group, not buyer-only,
seller-only, or admin-only groups. Mount the feature under a dedicated
`/api/v1/favorites` namespace: `GET /favorites` for the current actor's list,
and `GET`/`PUT`/`DELETE /favorites/{productID}` for state probe, idempotent add,
and idempotent remove on a single product.

**Rationale**: The clarified requirement says all authenticated users may
favorite products. A dedicated `/favorites` resource keeps the surface uniform
and avoids nesting a literal segment under `/products/{id}` (which would rely on
chi route-precedence to disambiguate `favorites` from a product id).

**Alternatives considered**:

- Mount under `/products/{id}/favorite` + `/users/me/favorites`: rejected
  because it splits the feature across two prefixes and relies on the router's
  static-over-wildcard precedence for safety.
- Buyer-only routes: rejected by clarification.
- Admin or seller special cases: rejected because the spec says all
  authenticated users use the same favorite behavior.
- Public favorite-state route: rejected because favorite state is actor-specific.

## Decision: Do not add Redis keys or cache invalidation for favorites

Favorite state remains PostgreSQL-backed. Do not cache authenticated favorite
state in Redis, and do not invalidate `products:{id}` when favorites change.

**Rationale**: Favorite state is user-specific. The existing product cache stores
product lookup data only; this feature does not mutate the product entity or
introduce favorite counts.

**Alternatives considered**:

- Cache per-user favorite lists: rejected because it adds auth-sensitive cache
  mechanics and invalidation that are not required by the feature.
- Invalidate product cache on favorite mutation: rejected because product DTO
  content is unchanged and counts are out of scope.

## Decision: Use focused tests with gomock for the service-owned interface

Service unit tests use gomock-generated `MockFavoriteRepo` and `MockProductRepo`
(the latter already exists). Handler tests cover transport behavior with the
same mocks behind a real `FavoriteService`. Repository integration tests cover
SQL constraints and the `products`-join behavior against PostgreSQL.

**Rationale**: `FavoriteRepo` is service-owned; gomock matches the rest of the
codebase. The AGENTS.md gomock-exception clause explicitly allows targeted
`mockgen` for new service-owned interfaces during the implementing task, so the
generated mock stays in `internal/mocks/service/` (gitignored, regenerated from
the `go:generate` directive next to the interface).

**Alternatives considered**:

- Hand-written fakes: rejected because the rest of the project standardizes on
  gomock for service-owned interfaces, and a fake would only have to be rewritten
  once the rest of the codebase touches `FavoriteRepo`.
- Full `go test ./...`: deferred because full suites require explicit
  permission; focused packages are enough for this feature's risk.
