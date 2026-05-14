# Quickstart: Favorite Products

**Feature**: 002-favorite-products

This document is a manual smoke test for the feature. It assumes the
project is running locally per [docs/setup.md](../../docs/setup.md) — a
PostgreSQL container, migrations applied through `013_create_favorites.sql`,
and the API listening on the default port.

Throughout this guide we use `:8080` as the API host; substitute your
local port if different. Replace `TOKEN`, `BUYER_TOKEN`, `OTHER_TOKEN`,
and `PRODUCT_ID` with values obtained from the prerequisites step.

## Prerequisites

1. Apply migrations (so `favorites` exists and role grants are in place):

   ```bash
   goose -dir migrations postgres "$DATABASE_URL" up
   ```

2. Seed at least two test users and one visible product. Either run the
   seed command (`cmd/seed`) or create them via the auth/products
   endpoints. Record:

   - `BUYER_TOKEN` — bearer token for user A (buyer role).
   - `OTHER_TOKEN` — bearer token for user B (any role).
   - `PRODUCT_ID`  — an existing, visible product id.

## Happy path

### 1. Add to favorites (first time → 201)

```bash
curl -i -X PUT \
  -H "Authorization: Bearer $BUYER_TOKEN" \
  "http://localhost:8080/api/v1/favorites/$PRODUCT_ID"
```

Expected: `HTTP/1.1 201 Created`, no body.

### 2. Add again (idempotent → 200)

```bash
curl -i -X PUT \
  -H "Authorization: Bearer $BUYER_TOKEN" \
  "http://localhost:8080/api/v1/favorites/$PRODUCT_ID"
```

Expected: `HTTP/1.1 200 OK`, no body. No duplicate row should exist
(verify in the DB if needed: `SELECT count(*) FROM favorites
WHERE user_id = … AND product_id = …;` returns `1`).

### 3. List my favorites

```bash
curl -s \
  -H "Authorization: Bearer $BUYER_TOKEN" \
  "http://localhost:8080/api/v1/favorites?page=1&page_size=20" | jq
```

Expected (truncated):

```json
{
  "items": [
    { "product": { "id": <PRODUCT_ID>, "name": "...", "price": "...", "in_stock": true },
      "added_at": "2026-05-14T..." }
  ],
  "page": 1,
  "page_size": 20,
  "total": 1
}
```

### 4. Is this product favorited by me?

```bash
curl -s \
  -H "Authorization: Bearer $BUYER_TOKEN" \
  "http://localhost:8080/api/v1/favorites/$PRODUCT_ID" | jq
```

Expected: `{"favorited": true, "added_at": "..."}`.

### 5. Remove from favorites

```bash
curl -i -X DELETE \
  -H "Authorization: Bearer $BUYER_TOKEN" \
  "http://localhost:8080/api/v1/favorites/$PRODUCT_ID"
```

Expected: `HTTP/1.1 204 No Content`.

### 6. Confirm gone

```bash
curl -s \
  -H "Authorization: Bearer $BUYER_TOKEN" \
  "http://localhost:8080/api/v1/favorites/$PRODUCT_ID" | jq
```

Expected: `{"favorited": false}` (no `added_at`).

```bash
curl -s \
  -H "Authorization: Bearer $BUYER_TOKEN" \
  "http://localhost:8080/api/v1/favorites" | jq '.total'
```

Expected: `0`.

## Edge cases (also worth running manually)

### A. Unauthenticated → 401 everywhere

```bash
curl -i -X PUT     "http://localhost:8080/api/v1/favorites/$PRODUCT_ID"
curl -i            "http://localhost:8080/api/v1/favorites"
curl -i -X DELETE  "http://localhost:8080/api/v1/favorites/$PRODUCT_ID"
curl -i            "http://localhost:8080/api/v1/favorites/$PRODUCT_ID"
```

All four MUST return `401 Unauthorized` with the `{"error":"..."}` envelope.

### B. Favorite a non-existent product → 404

```bash
curl -i -X PUT \
  -H "Authorization: Bearer $BUYER_TOKEN" \
  "http://localhost:8080/api/v1/favorites/9999999"
```

Expected: `HTTP/1.1 404 Not Found`, `{"error":"product not found"}`.

### C. Delete a never-favorited product → still 204

```bash
curl -i -X DELETE \
  -H "Authorization: Bearer $BUYER_TOKEN" \
  "http://localhost:8080/api/v1/favorites/9999999"
```

Expected: `HTTP/1.1 204 No Content` (delete is idempotent and does not
check the product's existence).

### D. Cross-user isolation

1. User A adds `PRODUCT_ID` to favorites.
2. User B's `GET /api/v1/favorites` returns 0 items.
3. User B's `GET /api/v1/favorites/$PRODUCT_ID` returns `{"favorited": false}`.
4. User B removes `$PRODUCT_ID` (it's not even theirs) → 204; User A's
   list still contains `PRODUCT_ID`.

### E. Cascade — product hard-delete

1. User A favorites `PRODUCT_ID`.
2. As an admin, delete `PRODUCT_ID` from the catalog.
3. User A's `GET /api/v1/favorites` no longer contains it (`total = 0`).
   No 500s, no ghost row.

### F. Cascade — user delete

1. User A favorites two products.
2. Delete user A's account (`DELETE /api/v1/users/me`).
3. Confirm in the DB: `SELECT count(*) FROM favorites WHERE user_id =
   <A's id>;` returns `0`.

### G. Pagination clamp

```bash
curl -s \
  -H "Authorization: Bearer $BUYER_TOKEN" \
  "http://localhost:8080/api/v1/favorites?page=0&page_size=500" | jq '.page, .page_size'
```

Expected: `1` and `100` respectively (silently clamped per Decision 6).

## What the smoke test does NOT cover

- Concurrent double-`PUT` of the same `(user, product)` pair — covered by
  the repository integration test using two simultaneous transactions.
- Role grant matrix — covered by the repository integration test using
  the four `marketplace_*` roles.
- Index plan verification (`EXPLAIN`) — out of scope for the smoke test;
  verified at PR time if list latency is suspect.
