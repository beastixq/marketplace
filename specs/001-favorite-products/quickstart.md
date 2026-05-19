# Quickstart: Favorite Products

This quickstart describes the expected backend behavior after implementation.
It does not require web GUI, templates, CSS, or terminal UI.

## Prerequisites

```bash
docker compose up -d
export DATABASE_URL='postgres://postgres:postgres@localhost:5432/marketplace?sslmode=disable'
goose -dir migrations postgres "$DATABASE_URL" up
go run ./cmd/api -config config/config.yaml
```

Use any authenticated user token. All authenticated roles are allowed to use
favorite endpoints.

```bash
TOKEN='<jwt>'
PRODUCT_ID='<active-visible-product-id>'
```

## Add Favorite

```bash
curl -i -X PUT "http://localhost:8080/api/v1/favorites/${PRODUCT_ID}" \
  -H "Authorization: Bearer ${TOKEN}"
```

Expected result:

- `201 Created` when the favorite relationship is created.
- `200 OK` when the product was already favorited (idempotent re-add).
- `404 Not Found` when the product does not exist.
- `409 Conflict` with `{"error":"Product is deleted"}` when the product is not
  active and visible.

## Check Favorite State

```bash
curl -s "http://localhost:8080/api/v1/favorites/${PRODUCT_ID}" \
  -H "Authorization: Bearer ${TOKEN}"
```

Expected body:

```json
{"product_id":1,"is_favorite":true}
```

## List Current Actor Favorites

```bash
curl -s "http://localhost:8080/api/v1/favorites?page=1&limit=20" \
  -H "Authorization: Bearer ${TOKEN}"
```

Expected result: a JSON array using the existing product DTO shape, containing
only active visible products favorited by the current actor.

## Remove Favorite

```bash
curl -i -X DELETE "http://localhost:8080/api/v1/favorites/${PRODUCT_ID}" \
  -H "Authorization: Bearer ${TOKEN}"
```

Expected result: `204 No Content` whether the favorite was present or already
absent.

## Focused Verification After Implementation

Targeted commands to consider after implementation:

```bash
go test ./internal/service
go test ./internal/handler
DATABASE_URL="$DATABASE_URL" go test ./internal/repository
```

Full `go test ./...`, broad diffs, formatters, linters, code generation, and
dependency downloads require explicit permission under the project verification
policy.
