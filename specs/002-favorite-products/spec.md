# Feature Specification: Favorite Products

**Feature Branch**: `002-favorite-products`

**Created**: 2026-05-14

**Status**: Draft

**Input**: User description: "Favorite products feature, backend only. Authenticated users can mark products as favorites. One user can favorite multiple products. One product can be favorited by multiple users. Backend only: JSON API, service/repository/database/docs. Do not touch web GUI, templates, CSS, or terminal UI."

## User Scenarios & Testing *(mandatory)*

### User Story 1 — Save a product to my favorites (Priority: P1)

As an authenticated marketplace user, I want to mark a product as a favorite so
that I can find it again later without searching the catalog. This is the core
"save for later" interaction that motivates the whole feature.

**Why this priority**: Without the ability to add, the feature delivers no
value at all — every other story depends on a saved set existing.

**Independent Test**: An authenticated user can call the "add to favorites"
action for a valid product and then see that product appear in their own
favorites list on a subsequent fetch.

**Acceptance Scenarios**:

1. **Given** an authenticated user and an existing product that they have
   never favorited, **When** the user marks the product as a favorite,
   **Then** the product appears in that user's favorites list and is reported
   as favorited by that user.
2. **Given** an authenticated user who has already favorited a product,
   **When** the user attempts to favorite the same product again, **Then** the
   action succeeds without creating a duplicate entry, and the favorites list
   still contains exactly one occurrence of that product.
3. **Given** an unauthenticated client, **When** the client attempts to
   favorite any product, **Then** the request is rejected as unauthorized and
   no favorites state changes.
4. **Given** an authenticated user, **When** the user attempts to favorite a
   product ID that does not exist, **Then** the request is rejected with a
   not-found outcome and the user's favorites list is unchanged.

---

### User Story 2 — View my favorites (Priority: P1)

As an authenticated user, I want to retrieve the list of products I have
favorited so that I can browse and act on them later (revisit, share, or buy).

**Why this priority**: A saved set with no read path is invisible to the user;
P1 because the feature is incomplete without it.

**Independent Test**: After at least one product has been added via Story 1,
the same user can request their favorites list and receive at least the
products they added, scoped strictly to themselves.

**Acceptance Scenarios**:

1. **Given** an authenticated user with N favorited products, **When** the
   user fetches their favorites, **Then** the response contains exactly those
   N products with enough information to identify and display each one
   (at minimum: product id, name, price, and a way to detect availability).
2. **Given** two different authenticated users A and B with overlapping but
   non-identical favorites, **When** each fetches their own favorites,
   **Then** each sees only their own selections; B never sees A's list and
   vice versa.
3. **Given** an authenticated user with zero favorites, **When** the user
   fetches their favorites, **Then** the response is a successful empty list,
   not an error.
4. **Given** an authenticated user whose favorites list is large, **When** the
   user fetches their favorites, **Then** the result is returned in a paginated
   or otherwise size-bounded form so that the response remains tractable.

---

### User Story 3 — Remove a product from my favorites (Priority: P1)

As an authenticated user, I want to remove a product from my favorites so
that my saved list reflects my current interest.

**Why this priority**: Users who cannot prune their list will stop trusting it;
removal is part of the minimum viable feature.

**Independent Test**: After Story 1 adds a product, the same user can call
the "remove from favorites" action for that product and verify it no longer
appears in their favorites list.

**Acceptance Scenarios**:

1. **Given** an authenticated user with a product in their favorites,
   **When** the user removes that product, **Then** the product no longer
   appears in their favorites list, and the action is idempotent: a second
   removal call for the same product succeeds without error and produces no
   further change.
2. **Given** an authenticated user, **When** the user removes a product that
   they had never favorited, **Then** the request is treated as a no-op
   success (the post-condition "this product is not in the user's favorites"
   already holds).
3. **Given** an unauthenticated client, **When** the client attempts to remove
   any favorite, **Then** the request is rejected as unauthorized.
4. **Given** users A and B where both have favorited the same product,
   **When** user A removes it, **Then** user B's favorites are unaffected.

---

### User Story 4 — Know whether a specific product is favorited (Priority: P2)

As an authenticated user, I want to know whether a specific product is
currently in my favorites so that a client surface (mobile app, future web UI)
can render the correct state (e.g., filled vs. empty bookmark) on a product
detail view without fetching the whole list.

**Why this priority**: Strictly speaking, this state can be derived by
scanning the full list returned by Story 2, but for large lists or one-off
product views, a direct check is the natural primitive. P2 because clients
can ship without it by reusing Story 2.

**Independent Test**: After Story 1 favorites product P, asking "is P
favorited by me?" returns true; after Story 3 removes P, the same query
returns false.

**Acceptance Scenarios**:

1. **Given** an authenticated user who has favorited product P, **When** the
   client queries the favorite status of P for that user, **Then** the
   response indicates favorited=true.
2. **Given** an authenticated user who has not favorited product P, **When**
   the client queries the favorite status of P for that user, **Then** the
   response indicates favorited=false.
3. **Given** an unauthenticated client, **When** the client queries any
   favorite status, **Then** the request is rejected as unauthorized.

---

### User Story 5 — Favorites remain coherent as products evolve (Priority: P3)

As an authenticated user, I want my favorites list to remain coherent when
products I favorited change state in the catalog (price changes, stock runs
out, the seller unlists or deletes the product), so that I am not surprised
by ghost entries or silent data loss.

**Why this priority**: Reasonable defaults exist for each case, so this is
not strictly required for the MVP. But it is a real cross-cutting concern that
must be answered before launch.

**Independent Test**: Stand up a product, favorite it, then mutate the
product's state (price change, stock=0, soft-delete by seller) and confirm
that the user's favorites list reflects the chosen policy without crashing
or leaking unauthorized data.

**Acceptance Scenarios**:

1. **Given** a user has favorited product P, **When** P's price changes,
   **Then** the favorites list for that user continues to show P, and the
   price returned reflects the current product price (not a snapshot from
   the time of favoriting).
2. **Given** a user has favorited product P, **When** P goes out of stock,
   **Then** P still appears in the user's favorites list, and the entry
   conveys enough information for the client to render an "unavailable" or
   "out of stock" affordance.
3. **Given** a user has favorited product P, **When** the seller deletes P
   from the catalog, **Then** the corresponding favorite is cleaned up
   automatically along with the product, so the user's favorites list no
   longer references P at all (no ghost entry, no error).

---

### Edge Cases

- An authenticated user attempts to favorite a product that exists but is not
  visible to them (e.g. a hypothetical "soft-deleted" or "draft" product).
  The system MUST treat this the same as favoriting a non-existent product
  (not-found), to avoid leaking the existence of hidden products.
- An authenticated user tries to favorite their own product (if such a user
  is also a seller of the product). The system MUST allow this; favoriting
  is open to any authenticated user.
- An authenticated user's account is deleted. Their favorites entries MUST
  be cleaned up so they cannot be enumerated and do not retain a dangling
  reference to that user.
- A product is hard-deleted from the catalog. The corresponding favorite
  rows MUST be removed together with the product, so no read path ever
  returns a row pointing at a non-existent product. (Cascade cleanup; see
  FR-011.)
- Two near-simultaneous "add" requests for the same (user, product) pair
  MUST not produce duplicate entries — the saved set is logically a set,
  not a sequence.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST allow an authenticated user to add a specific
  existing product to their own favorites.
- **FR-002**: The system MUST treat the favorites collection as a set per
  user: a single (user, product) pair MUST appear at most once.
- **FR-003**: The system MUST allow an authenticated user to remove a specific
  product from their own favorites, and the operation MUST be idempotent
  (removing a non-favorite is a success with no further side effects).
- **FR-004**: The system MUST allow an authenticated user to retrieve the
  list of products they have favorited, scoped strictly to that user. No
  user may read another user's favorites list.
- **FR-005**: The system MUST allow an authenticated user (or a client acting
  on their behalf) to determine whether a specific product is currently
  favorited by that user, without requiring the full list to be fetched.
- **FR-006**: The system MUST reject any favorite-related operation from an
  unauthenticated caller as unauthorized, with no state change.
- **FR-007**: The system MUST reject "add to favorites" requests that target
  a product the caller has no permission to see or that does not exist, with
  a not-found outcome (rather than revealing whether the product exists).
- **FR-008**: The system MUST return favorites in a stable, predictable order
  on read. The default order MUST be most-recently-favorited first, so that
  a user's latest interest is surfaced first.
- **FR-009**: The system MUST return enough per-entry information for a client
  to render each favorited product: at minimum product identifier, current
  name, current price, and an availability indicator (e.g. in-stock vs. not).
  Prices and availability MUST reflect the current product state, not a
  snapshot from the moment of favoriting — favoriting is not a price lock.
- **FR-010**: The system MUST clean up a user's favorites when that user's
  account is deleted, so that orphan rows do not survive account deletion.
- **FR-011**: When a product is deleted from the catalog, every favorite
  row referencing that product MUST be removed as part of the same deletion
  (cascade). No read path may return a favorite that points at a deleted
  product, and the deletion MUST NOT fail merely because favorites exist.
- **FR-012**: The favorites list MUST be returned in a size-bounded form
  (pagination or equivalent) so that a user with many favorites does not
  produce arbitrarily large responses. A reasonable default page size is
  20 items; the upper bound MUST be capped.
- **FR-013**: Every authenticated user role MAY use the favorites feature —
  buyers, sellers, and admins. The feature is gated on authentication, not on
  role. Sellers MAY favorite any product, including their own. All three
  end-user PostgreSQL roles (`marketplace_buyer`, `marketplace_seller`,
  `marketplace_admin`) therefore need the privileges required to manage
  their own favorite rows.
- **FR-014**: This iteration ships the pure saved-list MVP only. The system
  MUST NOT expose favorite counts, favorite users, "users who favorited this
  also liked" derivations, or any notification on changes to favorited
  products. Favorites are strictly private to the favoriting user, and no
  derived feature rides along with this one.

### Key Entities *(include if feature involves data)*

- **User**: An authenticated marketplace participant. Already exists in the
  system; this feature does not introduce or modify users. Relevant attribute
  for this feature: identity (so the favorites set can be scoped).
- **Product**: A marketplace product that can be favorited. Already exists.
  Relevant attributes for this feature: identity, visibility/availability
  state (so a favorite can be presented with current information).
- **Favorite**: The association between a User and a Product expressing
  "this user has marked this product as a favorite". Logically a set element
  with a creation timestamp used for ordering ("most recent first").
  Uniqueness invariant: at most one Favorite per (User, Product) pair.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: From an authenticated session, a user can save a product and
  retrieve it from their favorites within the same session (round-trip
  observable to the user) without errors, on the first attempt, 100% of the
  time under normal load.
- **SC-002**: An authenticated user's favorites list is never observable to
  another user under any read path exposed by this feature. Verified by
  scenario tests covering at least two distinct users with overlapping
  favorites.
- **SC-003**: A user can fetch a favorites list of up to 100 entries and
  receive a response within 1 second under normal load.
- **SC-004**: Repeating "add" for an already-favorited product, and "remove"
  for a not-favorited product, both succeed without creating duplicates or
  errors — measured by an integration test exercising both idempotency
  cases.
- **SC-005**: After a user's account is deleted, no favorite rows referencing
  that user remain reachable from any read path — measured by an integration
  test that deletes a user and then attempts to enumerate favorites for them.

## Assumptions

- Authentication and session/identity propagation already exist in the
  project and will be reused as-is. This feature does not introduce a new
  auth method.
- Product visibility/permission rules are already enforced by the existing
  product read path. The favorites feature relies on those rules and does
  not duplicate them.
- Pagination conventions used elsewhere in the project (page-based,
  default page size, max page size) are reusable here. The exact values
  (default 20, hard cap to be confirmed at plan time) are conventions, not
  product decisions.
- Favorites are private. There is no public "popular products" surface in
  scope; if one is later requested, it is a separate feature.
- Favoriting does not lock any price, reserve any stock, or notify the
  seller. It is purely a user-side bookmark.
- All operations are scoped to backend (JSON API + service + repository +
  database + docs). Web GUI, templates, CSS, and the terminal UI are
  explicitly out of scope for this feature.

## Resolved Decisions

The following product-level decisions were surfaced during specification
drafting and resolved before planning:

- **Q1 — Eligible actor roles** (resolved 2026-05-14):
  **Any authenticated user may favorite products** — buyers, sellers, and
  admins. The feature is gated on authentication, not on role. See FR-013
  and the edge case "user tries to favorite their own product".

- **Q2 — Lifecycle when a favorited product is removed from the catalog**
  (resolved 2026-05-14):
  **Cascade-delete favorites with the product.** Favorites are derived
  state with no archaeological value; when the underlying product is
  deleted, the corresponding favorite rows are removed in the same
  operation. See FR-011, US5 acceptance scenario 3, and the
  "product is hard-deleted" edge case.

- **Q3 — Out-of-scope derived behaviors** (resolved 2026-05-14):
  **None.** This iteration ships the pure saved-list MVP only — no
  per-product favorite counts surfaced to sellers, no notifications, no
  recommendations. Any such behavior is a separate future feature. See
  FR-014.
