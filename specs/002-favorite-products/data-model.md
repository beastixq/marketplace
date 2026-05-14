# Phase 1 Data Model: Favorite Products

**Feature**: 002-favorite-products

## Entities

### Favorite (new)

The association between a User and a Product expressing "this user has
marked this product as a favorite".

**Logical fields**

| Field        | Type           | Notes                                                            |
| ------------ | -------------- | ---------------------------------------------------------------- |
| `user_id`    | bigint         | FK → `users.id`. Identifies the favoriting user.                 |
| `product_id` | bigint         | FK → `products.id`. Identifies the favorited product.            |
| `created_at` | timestamptz    | Set at insert time. Drives the "most recent first" list order.   |

**Invariants**

- Uniqueness: at most one row per `(user_id, product_id)` pair
  (FR-002 — set semantics). Enforced by a compound primary key.
- Existential integrity: both `user_id` and `product_id` MUST reference
  live rows; orphaned rows are forbidden.
- Cascade on user delete (FR-010): when a user is deleted, all of their
  favorites are deleted with them — no orphan rows survive.
- Cascade on product delete (FR-011 / Q2): when a product is deleted, all
  favorites referencing it are deleted with it.

**Lifecycle**

- Created by the service layer when an authenticated caller invokes the
  "add" use case for a product they can see.
- Deleted explicitly by the service layer when the caller invokes the
  "remove" use case, OR implicitly by PostgreSQL when the referenced
  `users.id` or `products.id` row is deleted.
- Never updated. There is no `updated_at`.

### User (existing — not modified)

Referenced as the favoriting actor. The favorites feature reuses the
existing identity propagated by the auth/middleware layer. No new fields.

### Product (existing — not modified)

Referenced as the favorited target. The favorites list response surfaces
selected current fields from `products` (id, name, current price,
availability indicator) — not snapshots. Favoriting does not pin price.

## Relationships

```text
users    (1)──┐
              │  cascade-delete
              ├── favorites (N) ───┤  cascade-delete  ┌── products (1)
              │                    │                   │
              └────────────────────┴───────────────────┘
                  (compound PK = user_id, product_id)
```

- One user can have many favorites.
- One product can appear in many users' favorites.
- The pair `(user_id, product_id)` is unique — a user's relationship to a
  given product is a set membership, not a sequence.

## Domain model (Go-level)

The service layer expresses two thin domain shapes — neither leaks DB
rows nor HTTP DTOs:

```go
// internal/model/favorite.go
package model

import "time"

// Favorite is a user's bookmark of a specific product, returned in a list view.
type Favorite struct {
    Product Product   // current product fields, NOT a snapshot
    AddedAt time.Time // = favorites.created_at
}
```

Note: the repository may also use a minimal pair value (`{UserID, ProductID}`)
internally for "add" / "remove" / "exists?" calls. That is an internal
helper, not a domain export.

## SQL shape

The migration introduces the following structure. Final wording is
captured in `migrations/013_create_favorites.sql` during implementation.

```sql
create table favorites (
    user_id    bigint      not null,
    product_id bigint      not null,
    created_at timestamptz not null default now(),

    constraint pk_favorites primary key (user_id, product_id),

    constraint fk_favorites_user_id
        foreign key (user_id)
        references users (id) on delete cascade,

    constraint fk_favorites_product_id
        foreign key (product_id)
        references products (id) on delete cascade
);

-- Drives the per-user newest-first list query (FR-008) as an index scan.
create index idx_favorites_user_created_at_desc
    on favorites (user_id, created_at desc);

-- Role grants (per Constitution III + Decision 8 in research.md)
grant select, insert, delete on favorites to marketplace_buyer;
grant select, insert, delete on favorites to marketplace_seller;
grant select, insert, delete on favorites to marketplace_admin;
grant select                   on favorites to marketplace_analyst;
```

## State transitions

The Favorite entity has only two states with respect to a given
`(user, product)` pair:

```text
            add (PUT)                remove (DELETE)
not-favorited ───────────► favorited ─────────────► not-favorited
        ▲                                                 │
        └──── product or user is deleted (cascade) ───────┘
```

- `add` on already-favorited: no-op success (200 OK, not 201) — idempotent.
- `remove` on not-favorited: no-op success (204) — idempotent.
- Cascade transitions are invisible to the user beyond the entry
  disappearing from their list on the next read.

## Validation rules

| Source        | Rule                                                      | Enforcement layer       |
| ------------- | --------------------------------------------------------- | ----------------------- |
| FR-001        | `product_id` must reference an existing, visible product. | Service (lookup) + repo (FK)  |
| FR-002        | At most one row per `(user_id, product_id)`.              | DB compound PK          |
| FR-004        | List is scoped to the caller's `user_id`.                 | Service (filter)        |
| FR-006        | Unauthenticated callers are rejected.                     | Middleware              |
| FR-007        | Invisible / non-existent product → not-found, not leak.   | Service                 |
| FR-008        | List ordered by `created_at` desc, then `product_id`.     | Repo SQL (`ORDER BY`)   |
| FR-010        | User deletion cleans up favorites.                        | DB `ON DELETE CASCADE`  |
| FR-011 / Q2   | Product deletion cleans up favorites.                     | DB `ON DELETE CASCADE`  |
| FR-012        | List page_size capped to 100, default 20.                 | Service (clamp)         |
| FR-013 / Q1   | Any authenticated user role may use the feature.          | Middleware role grants  |
