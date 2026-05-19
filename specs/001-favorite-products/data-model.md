# Data Model: Favorite Products

## Entity: Product Favorite

Represents one authenticated user's saved relationship to one product.

### Fields

| Field | Type | Required | Notes |
| --- | --- | --- | --- |
| `user_id` | integer | yes | References the authenticated user who owns the favorite. |
| `product_id` | integer | yes | References the favorited product. |
| `created_at` | timestamp | yes | Used for stable favorite-list ordering. |

### Relationships

- Many favorites belong to one user.
- Many favorites belong to one product.
- One user-product pair has at most one favorite.

### Validation Rules

- `user_id` must reference an existing user.
- `product_id` must reference an existing product.
- `(user_id, product_id)` must be unique.
- New favorites are allowed only when the product is active and visible.
- In the current schema, active and visible means `products.deleted_at IS NULL`.

### Lifecycle

```text
absent --add favorite--> present
present --add favorite again--> present
present --remove favorite--> absent
absent --remove favorite again--> absent
```

### Product State Effects

- If a product is soft-deleted after being favorited, the raw favorite
  relationship may remain.
- Favorite list behavior must return only active visible products.
- Favorite-state check must reject soft-deleted products rather than report them
  as favorited.
- Favorite removal may clear a relationship for a product that has since become
  soft-deleted.

## Entity: Favorite State

Read-only actor-specific result for a single product.

### Fields

| Field | Type | Required | Notes |
| --- | --- | --- | --- |
| `product_id` | integer | yes | Product being checked. |
| `is_favorite` | boolean | yes | Whether the current actor has a favorite relationship for the product. |

## Entity: Favorite Product List

Read-only list of products favorited by the current actor.

### Fields

| Field | Type | Required | Notes |
| --- | --- | --- | --- |
| `products` | list of product summaries | yes | Existing product DTO shape is reused. |

### Ordering And Filtering

- Default ordering should be newest favorite first by `product_favorites.created_at`.
- Optional pagination should follow existing handler pagination conventions.
- Products that are no longer active and visible must be omitted.
