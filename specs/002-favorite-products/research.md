# Phase 0 Research: Favorite Products

**Feature**: 002-favorite-products
**Status**: Complete — all spec-level `NEEDS CLARIFICATION` markers were
resolved during `/speckit-specify` (see `spec.md` § "Resolved Decisions").
This document captures the remaining implementation-level decisions that
the plan rests on.

---

## Decision 1 — Persistence shape: join table, not array column

**Decision**: Model favorites as a join table
`favorites (user_id, product_id, created_at)` with a compound primary key
on `(user_id, product_id)` and `ON DELETE CASCADE` to both `users` and
`products`.

**Rationale**:
- Matches the existing pattern used by `product_categories`
  (migration 001), so it is idiomatic for this codebase and consistent with
  what reviewers expect.
- Supports the per-user, newest-first list with a single covering index
  `(user_id, created_at desc)` — a B-tree index range scan, no sort.
- Cascade resolves Q2 ("delete favorites with product") at the storage
  boundary, which means service code does not need to chase orphans.
- Compound PK is the uniqueness invariant from FR-002 (set semantics).
  No application-level deduplication is needed.

**Alternatives considered**:
- *Array column on `users`* (e.g. `favorite_product_ids bigint[]`).
  Rejected: contradicts the project's relational-first style, breaks
  referential integrity (no FK), forbids partial-index pagination, and
  makes concurrent add/remove race-prone.
- *Separate `favorites_log` event table without a uniqueness constraint*.
  Rejected: turns "is favorited?" into an aggregation query and admits the
  duplicate-row class of bugs the spec explicitly forbids (FR-002).
- *FK with `ON DELETE SET NULL`*. Rejected: requires a nullable
  `product_id` which contradicts FR-009 ("each entry surfaces product
  identifier and current state") and the Q2 decision.

---

## Decision 2 — No cache layer in v1

**Decision**: Do not introduce a Redis key family for favorites in this
iteration. Reads go straight to PostgreSQL.

**Rationale**:
- Favorites traffic is per-user, write-mixed, and bounded by user activity.
  Unlike the public catalog (which has high read amplification), favorite
  lists are small (dozens to low hundreds typical) and only the owning
  user reads them.
- Adding a cache here would multiply invalidation paths (every add /
  remove / product-delete / user-delete invalidates), and constitution
  Principle II forbids handlers from doing cache invalidation — meaning
  invalidation would have to ride along the service. That cost is not
  justified by current load.
- If favorite-fetch latency ever becomes the bottleneck, a per-user list
  cache (`favorites:user:{id}` invalidated on add/remove) is a clean
  follow-up; the read path stays a single repository method either way.

**Alternatives considered**:
- *Cache the full per-user list under `favorites:user:{id}`*. Rejected
  for v1 (premature; see above). Documented as a future extension only.
- *Cache the "is favorited?" check on product detail*. Rejected: would
  require cross-resource invalidation when a user toggles favorites, and
  the underlying query is a single PK lookup.

**Implication**: [docs/cache.md](../../docs/cache.md) does not need to be
updated for this feature.

---

## Decision 3 — HTTP verb for add: `PUT /favorites/{productId}`

**Decision**: Use `PUT /api/v1/favorites/{productId}` for the "add to
favorites" action.

**Rationale**:
- Add is idempotent by spec (FR-002 / US1 scenario 2). `PUT` is the HTTP
  verb that encodes idempotency at the protocol level; clients and
  intermediaries get to retry safely without producing duplicates.
- The natural state of a favorite is a set membership keyed by
  `(caller, productId)`. `PUT` matches the set-membership semantics:
  "this product is favorited by me" is either true after the call or it
  was already true.
- Path identifies the resource (the favorite indexed by product id from
  the caller's perspective), which is more REST-idiomatic than POST with a
  body for a single-field create.

**Alternatives considered**:
- *POST `/api/v1/favorites` with `{product_id: N}` body*. Matches
  conventions used by reviews and addresses in this codebase. Rejected
  here because both reviews and addresses are NOT inherently idempotent
  (a review has free-text content; an address has an id distinct from the
  owning user), whereas a favorite is structurally idempotent.
- *POST `/api/v1/products/{id}/favorite` action endpoint*. Rejected:
  treats favorites as a verb on products rather than as a first-class
  resource; harder to list / delete uniformly.

**Status codes**:
- `201 Created` on first favorite of that product by the caller.
- `200 OK` on a repeat call (already favorited; no state change). Body
  unchanged.
- `404 Not Found` if the product does not exist or is not visible.
- `401 Unauthorized` if the caller is unauthenticated.

---

## Decision 4 — Delete is idempotent and silent

**Decision**: `DELETE /api/v1/favorites/{productId}` returns `204 No
Content` whether or not the (caller, productId) pair existed before the
call.

**Rationale**:
- Spec FR-003 mandates idempotency.
- Returning `404` for "you never favorited that product" leaks no useful
  signal and complicates clients that just want to ensure the bookmark is
  off (e.g. heart-toggle in a future UI).
- Matches the standard "DELETE is idempotent" guarantee in RFC 9110.

**Alternatives considered**:
- *Return `404` if the row was missing*. Rejected; non-idempotent in
  outcome, fights the spec.
- *Return `200` with `{"removed": bool}`*. Rejected; gratuitous body for
  a delete.

---

## Decision 5 — "Is favorited?" returns a JSON boolean, not a status code

**Decision**: `GET /api/v1/favorites/{productId}` returns
`200 {"favorited": true|false}`. Returns `404` only if the product itself
does not exist or is not visible.

**Rationale**:
- The natural answer is a boolean, and using HTTP status code as the
  payload (204 vs 404) overloads "the product does not exist" with "the
  product is not favorited", which the spec explicitly distinguishes
  (FR-007 vs FR-005).
- A JSON boolean keeps the response shape regular with the rest of the
  API and is easy to extend later (e.g. include `favorited_at`) without
  a breaking change.

**Alternatives considered**:
- *`HEAD /api/v1/favorites/{productId}` → 204 vs 404*. Rejected: see
  payload overloading above; also less self-describing.

---

## Decision 6 — Pagination conventions reused

**Decision**: List endpoint reads `?page=N&page_size=M`, page indexing is
1-based, default `page_size=20`, hard cap `page_size=100`. Out-of-range
inputs (page<1, page_size<1) are clamped to the defaults; page_size>100 is
clamped down to 100.

**Rationale**:
- Reuses existing project pagination convention (catalog listing uses the
  same shape, see `internal/handler/product.go` neighborhood).
- Cap of 100 matches SC-003 ("up to 100 entries in under 1 second").
- Clamping in service (not handler) keeps the policy testable without
  HTTP.

**Alternatives considered**:
- *Cursor-based pagination keyed on `created_at, product_id`*. Rejected
  for v1: the per-user dataset is small; page-based is simpler and matches
  what the rest of the API does. Cursor is a clean follow-up if a user
  ever hits the cap repeatedly.

---

## Decision 7 — Authorization scope: caller can only act on own favorites

**Decision**: Every favorite endpoint reads the caller's user id from the
authenticated actor context and uses it as the `user_id` filter. There is
no path parameter that lets one user reference another user's favorites.

**Rationale**:
- FR-004 and the privacy posture (FR-014) require this.
- Q1 made every authenticated role eligible to USE favorites, but
  eligibility is orthogonal to scope — admins do not gain a "see anyone's
  favorites" power from this feature. If admin tooling ever needs that,
  it is a separate feature with its own justification.

**Alternatives considered**:
- *Admin override (`GET /api/v1/users/{id}/favorites` for admins)*.
  Rejected; not in spec, expands the security surface for no current
  use case.

---

## Decision 8 — Database role grants

**Decision**: Grant `SELECT, INSERT, DELETE` on `favorites` to
`marketplace_buyer`, `marketplace_seller`, and `marketplace_admin`. Grant
`SELECT` to `marketplace_analyst`. No `UPDATE` grant (the only mutable
field, `created_at`, is set at insert and never updated).

**Rationale**:
- Q1 resolution: every authenticated user role can favorite.
- Constitution III requires explicit role grants at the database layer.
- Absence of `UPDATE` is intentional — favoriting is a set operation, not
  an editable record.
- Analyst gets read access in line with their existing
  "SELECT ALL, no writes" charter.

**Alternatives considered**:
- *Grant `ALL` to admin*. Rejected: explicit grants make the audit
  story tighter and match how the existing tables grant privileges
  (see migration 003).

---

## Open follow-ups (out of scope for v1, recorded for later)

- Per-user list cache under `favorites:user:{id}` if read latency becomes
  a hotspot (see Decision 2).
- Cursor pagination if any user hits page_size=100 repeatedly
  (see Decision 6).
- Seller-facing "demand signal" view (favorite count per own product) —
  Q3 explicitly deferred (see spec FR-014).
- User-facing notifications on price drop / restock — Q3 explicitly
  deferred.
