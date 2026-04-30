# API Contracts

## Base Contract

The JSON API is mounted under:

```text
/api/v1
```

Requests and responses use JSON. Error responses use:

```json
{"error":"message"}
```

Authenticated endpoints expect a bearer token:

```http
Authorization: Bearer <jwt>
```

Public web pages are separate from the JSON API and are routed from `internal/web/router.go`.

## Status Code Conventions

Use `net/http` constants in code.

| Status | Use |
| --- | --- |
| `200 OK` | Successful read or mutation with a body. |
| `201 Created` | Resource created. |
| `204 No Content` | Successful mutation with no response body. |
| `400 Bad Request` | Bad input, invalid query params, empty cart, invalid quantity. |
| `401 Unauthorized` | Missing/invalid auth or wrong password. |
| `403 Forbidden` | Role/ownership/permission failure. |
| `404 Not Found` | Missing resource. |
| `409 Conflict` | Domain conflict such as duplicate email/review, review before purchase, invalid order state, stock conflict. |
| `410 Gone` | Expired payment. |
| `422 Unprocessable Entity` | Payment declined or amount mismatch. |
| `500 Internal Server Error` | Unexpected internal failure. |

Service error mapping lives in `internal/handler/service_error.go`.

## Public Routes

| Method | Path | Purpose |
| --- | --- | --- |
| `POST` | `/api/v1/auth/register` | Register buyer or seller user. |
| `POST` | `/api/v1/auth/login` | Login and receive token. |
| `POST` | `/api/v1/payments/callback/mock-bank` | Mock bank callback. |
| `GET` | `/api/v1/products` | Catalog listing. |
| `GET` | `/api/v1/products/{id}` | Product details. |
| `GET` | `/api/v1/products/{id}/price-history` | Product price history. |
| `GET` | `/api/v1/products/{id}/reviews` | Product reviews. |
| `GET` | `/api/v1/categories` | Category list/tree. |
| `GET` | `/api/v1/sellers/{id}` | Seller profile. |
| `GET` | `/api/v1/sellers/{id}/stats` | Seller statistics. |

## Authenticated User Routes

| Method | Path | Purpose |
| --- | --- | --- |
| `POST` | `/api/v1/auth/logout` | Revoke current session/token when blocklist is available. |
| `GET` | `/api/v1/users/me` | Current user profile. |
| `PATCH` | `/api/v1/users/me` | Update current user profile. |
| `DELETE` | `/api/v1/users/me` | Delete current account. |
| `PATCH` | `/api/v1/users/me/password` | Change password. |

## Buyer Routes

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/api/v1/addresses` | List own addresses. |
| `POST` | `/api/v1/addresses` | Create address. |
| `PATCH` | `/api/v1/addresses/{id}` | Update own address. |
| `DELETE` | `/api/v1/addresses/{id}` | Delete own address. |
| `GET` | `/api/v1/cart` | Get draft cart. |
| `POST` | `/api/v1/cart/items` | Add cart item. |
| `PATCH` | `/api/v1/cart/items/{id}` | Update cart item quantity. |
| `DELETE` | `/api/v1/cart/items/{id}` | Remove cart item. |
| `GET` | `/api/v1/orders` | List own orders. |
| `GET` | `/api/v1/orders/{id}` | Get own order. |
| `GET` | `/api/v1/orders/{id}/items` | Get order items. |
| `POST` | `/api/v1/orders` | Checkout cart. |
| `POST` | `/api/v1/orders/{id}/payment-link` | Create payment link. |
| `POST` | `/api/v1/orders/{id}/cancel` | Cancel order before shipped. |
| `POST` | `/api/v1/reviews` | Create review for product bought by current buyer. |
| `PATCH` | `/api/v1/reviews/{id}` | Update own review. |

## Seller Routes

| Method | Path | Purpose |
| --- | --- | --- |
| `POST` | `/api/v1/sellers` | Create own seller profile. |
| `PATCH` | `/api/v1/sellers/{id}` | Update own seller profile. |
| `DELETE` | `/api/v1/sellers/{id}` | Delete own seller profile. |
| `GET` | `/api/v1/sellers/me/orders` | List orders relevant to seller. |
| `POST` | `/api/v1/orders/{id}/ship` | Mark own paid order as shipped. |
| `POST` | `/api/v1/orders/{id}/deliver` | Mark own shipped order as delivered. |

## Seller Or Admin Routes

| Method | Path | Purpose |
| --- | --- | --- |
| `POST` | `/api/v1/products` | Create product. |
| `PATCH` | `/api/v1/products/{id}` | Update product. |
| `DELETE` | `/api/v1/products/{id}` | Delete product. |

## Buyer Or Admin Routes

| Method | Path | Purpose |
| --- | --- | --- |
| `DELETE` | `/api/v1/reviews/{id}` | Delete review. Buyer deletes own review; admin moderates. |

## Admin Routes

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/api/v1/admin/users` | List users. |
| `GET` | `/api/v1/admin/users/{id}` | Get user by ID. |
| `PATCH` | `/api/v1/admin/users/{id}` | Update user. |
| `DELETE` | `/api/v1/admin/users/{id}` | Delete user. |
| `PATCH` | `/api/v1/admin/sellers/{id}` | Update seller profile. |
| `DELETE` | `/api/v1/admin/sellers/{id}` | Delete seller profile. |
| `POST` | `/api/v1/categories` | Create category. |
| `PATCH` | `/api/v1/categories/{id}` | Update category. |
| `DELETE` | `/api/v1/categories/{id}` | Delete category. |

## Contract Maintenance

When adding or changing an endpoint:

- Update route registration in `internal/handler/router.go`.
- Add request validation at the handler boundary.
- Keep business rules in service.
- Add/update DTOs in `internal/handler/dto.go` when response shape changes.
- Update service error mapping when introducing a new public error.
- Update this document and any relevant `.http` request examples.
