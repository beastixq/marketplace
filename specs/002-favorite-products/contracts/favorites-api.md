# API Contract: Favorites

**Feature**: 002-favorite-products
**Base path**: `/api/v1/favorites`
**Auth**: Bearer token required on every route (middleware layer).
**Error envelope**: `{"error":"message"}` per
[docs/api-contracts.md](../../../docs/api-contracts.md).

All operations are scoped to the caller identified by the bearer token.
There is no path parameter that targets another user's favorites.

---

## `PUT /api/v1/favorites/{productId}` — Add a product to my favorites

**Path params**

| Name        | Type   | Description                                  |
| ----------- | ------ | -------------------------------------------- |
| `productId` | int64  | The product to mark as favorited.            |

**Request body**: none.

**Responses**

| Status            | When                                                                | Body                                          |
| ----------------- | ------------------------------------------------------------------- | --------------------------------------------- |
| `201 Created`     | Newly favorited by the caller.                                      | none                                          |
| `200 OK`          | Already favorited by the caller (no-op).                            | none                                          |
| `401 Unauthorized`| Missing/invalid bearer token.                                       | `{"error":"unauthorized"}`                    |
| `404 Not Found`   | Product does not exist or is not visible to the caller.             | `{"error":"product not found"}`               |
| `500 Internal …`  | Unexpected internal failure.                                        | `{"error":"internal error"}`                  |

**Semantics**: Idempotent (FR-002). A retried call after a network blip
MUST NOT produce a duplicate row.

---

## `DELETE /api/v1/favorites/{productId}` — Remove a product from my favorites

**Path params**

| Name        | Type   | Description                                  |
| ----------- | ------ | -------------------------------------------- |
| `productId` | int64  | The product to unmark.                       |

**Request body**: none.

**Responses**

| Status            | When                                                                | Body                                          |
| ----------------- | ------------------------------------------------------------------- | --------------------------------------------- |
| `204 No Content`  | The (caller, productId) pair is not favorited after this call.       | none                                          |
| `401 Unauthorized`| Missing/invalid bearer token.                                       | `{"error":"unauthorized"}`                    |
| `500 Internal …`  | Unexpected internal failure.                                        | `{"error":"internal error"}`                  |

**Semantics**: Idempotent (FR-003). Removing a non-favorite is a `204`,
not a `404`. The endpoint does not check whether the product exists in
the catalog — it only enforces the post-condition "this product is not in
the caller's favorites".

---

## `GET /api/v1/favorites` — List my favorites

**Query params**

| Name        | Type   | Default | Cap  | Description                                |
| ----------- | ------ | ------- | ---- | ------------------------------------------ |
| `page`      | int    | `1`     | —    | 1-based page index. Clamped to ≥1.         |
| `page_size` | int    | `20`    | `100`| Items per page. Clamped to `[1, 100]`.     |

**Request body**: none.

**Success response** — `200 OK`:

```json
{
  "items": [
    {
      "product": {
        "id": 1234,
        "name": "Bluetooth Speaker",
        "price": "1499.00",
        "in_stock": true
      },
      "added_at": "2026-05-14T10:23:00Z"
    }
  ],
  "page": 1,
  "page_size": 20,
  "total": 1
}
```

**Notes**
- `items[].product` fields reflect the **current** product state, not a
  snapshot at the moment of favoriting (FR-009). A separate "price at
  favoriting" is explicitly out of scope.
- `in_stock` is the availability indicator required by FR-009; final
  field name may align with however the existing product DTO names it.
- Items are ordered by `added_at` descending, then `product_id`
  ascending as a stable tiebreaker (FR-008).
- `total` is the caller's total favorite count (used for pagination UI).

**Error responses**

| Status            | When                                  | Body                          |
| ----------------- | ------------------------------------- | ----------------------------- |
| `401 Unauthorized`| Missing/invalid bearer token.         | `{"error":"unauthorized"}`    |
| `400 Bad Request` | Malformed pagination params.          | `{"error":"invalid page"}` (or similar) |
| `500 Internal …`  | Unexpected internal failure.          | `{"error":"internal error"}`  |

---

## `GET /api/v1/favorites/{productId}` — Is a specific product favorited by me?

**Path params**

| Name        | Type   | Description                                  |
| ----------- | ------ | -------------------------------------------- |
| `productId` | int64  | The product whose favorite-state to check.   |

**Request body**: none.

**Success response** — `200 OK`:

```json
{
  "favorited": true,
  "added_at": "2026-05-14T10:23:00Z"
}
```

- `favorited` is always present.
- `added_at` is present iff `favorited == true`. When `false`, this
  field is omitted (not `null`).

**Error responses**

| Status            | When                                                                | Body                                          |
| ----------------- | ------------------------------------------------------------------- | --------------------------------------------- |
| `401 Unauthorized`| Missing/invalid bearer token.                                       | `{"error":"unauthorized"}`                    |
| `404 Not Found`   | Product does not exist or is not visible to the caller.             | `{"error":"product not found"}`               |
| `500 Internal …`  | Unexpected internal failure.                                        | `{"error":"internal error"}`                  |

**Semantics**: 404 means "the product itself is not reachable", NOT
"you have not favorited it". The distinction matters for clients that
render product detail pages (a 404 here implies the product page itself
should also 404).

---

## Cross-cutting

- **Authentication**: All routes sit behind the existing auth middleware
  in `cmd/api/main.go`. Anonymous traffic gets `401` before reaching the
  handler.
- **Authorization**: No role gate. Every authenticated role (buyer,
  seller, admin) may use these endpoints. Sellers may favorite their own
  products. (Q1 resolution.)
- **Service errors → HTTP**: Mapped centrally in
  `internal/handler/service_error.go`. New domain errors (if any) added
  by this feature MUST be wired there.
- **Pagination**: see Decision 6 in research.md. Out-of-range values are
  clamped silently in the service; malformed values (non-numeric) return
  `400` from the handler.
- **Cache**: This feature does not add cache keys. Existing
  product/catalog caches are not invalidated by favorite operations
  (favorites do not change product fields).
