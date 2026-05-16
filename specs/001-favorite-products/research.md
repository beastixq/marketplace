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
soft-deleted products. Favorite listing joins active products only, so old
favorite rows for soft-deleted products are not returned. Favorite removal may
clear a relationship even if the product has since been soft-deleted.

**Rationale**: The current product model has `deleted_at` but no separate
visibility, archived, or published state. Catalog reads already use
`deleted_at IS NULL` as the active product boundary.

**Alternatives considered**:

- Add a new visibility/status column: rejected because the spec does not request
  product visibility management and that would expand scope.
- Hard-delete favorite rows when a product is soft-deleted: rejected because the
  existing product delete path is a soft delete and this feature does not need
  a cross-feature cleanup policy.

## Decision: Add authenticated JSON routes without role-specific middleware

Expose favorite behavior inside the authenticated route group, not buyer-only,
seller-only, or admin-only groups. Use product-scoped routes for add/remove/check
and a current-user route for listing.

**Rationale**: The clarified requirement says all authenticated users may
favorite products. Product-scoped add/remove/check routes match the product-level
feature, while `users/me` matches existing current-actor API style.

**Alternatives considered**:

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

## Decision: Use focused tests and hand-written fakes for service tests

Plan for service unit tests using small hand-written fakes, handler tests around
transport behavior, and repository integration tests for SQL constraints and
joins.

**Rationale**: This avoids code generation during implementation planning and
keeps verification cost controlled. Repository behavior must still be tested
against PostgreSQL because uniqueness, foreign keys, and active-product joins
are database behavior.

**Alternatives considered**:

- Generate gomock mocks immediately: deferred because code generation requires
  explicit permission under the project verification policy.
- Full `go test ./...`: deferred because full suites require explicit
  permission; focused packages are enough for this feature's risk.
