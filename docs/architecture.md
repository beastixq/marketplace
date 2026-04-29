# Architecture

This project is an online marketplace backend and server-rendered web app written in Go. PostgreSQL is the source of truth; Redis is part of the target cache/session architecture.

## Runtime Flow Vs Source Dependencies

Runtime call flow:

```text
handler / web / techui -> service -> repository implementation
```

Source dependency direction:

```text
handler / web / techui -> service <- repository implementation
```

The distinction is mandatory. The service layer uses repository behavior through interfaces, but it does not depend on repository implementations. Service packages export the interfaces they need. Repository packages import service interfaces and implement them.

## Layers

| Layer | Path | Responsibility |
| --- | --- | --- |
| Domain model | `internal/model/` | Business/domain structs and value types. Not DB rows and not HTTP DTOs. |
| Service | `internal/service/` | Use cases, business rules, authorization decisions, order lifecycle, service-owned interfaces, application errors. |
| Repository | `internal/repository/` | SQL, pgx/pgxpool, transactions, DB row mapping, database error translation. |
| API handler | `internal/handler/` | JSON HTTP handlers, request/response DTOs, request validation, service error mapping. |
| Web UI | `internal/web/` | Server-rendered MPA controllers, `html/template` templates, static CSS. |
| Middleware | `internal/middleware/` | Auth, RBAC, request logging, recovery, actor propagation. |
| Ports/adapters | `internal/port/`, `internal/adapter/` | External service contracts and implementations, currently payment gateway code. |
| Components | `internal/component/` | Wiring helpers for repository/service/tech UI composition. |
| Commands | `cmd/api`, `cmd/seed`, `cmd/techui` | Application entry points. |

## Dependency Rules

- `internal/service` must not import `internal/repository`, `pgx`, `pgconn`, SQLSTATE values, or constraint names.
- `internal/handler` and `internal/web` must not import `internal/repository`.
- Repository interfaces live where consumed, normally `internal/service`.
- Repository implementations live in `internal/repository`.
- Dependencies are passed through constructors, not hidden globals.
- Compile-time repository checks should usually use pointer form:

```go
var _ service.ProductRepo = (*ProductRepository)(nil)
```

## Data Mapping

Mapping must remain explicit:

```text
DB row -> model -> DTO / view data
```

DB structs and SQL-specific types do not cross into service. HTTP DTOs and web view data do not cross into service or repository. The service layer should work in domain models and small parameter structs.

## Business Rules

Business rules belong in service code:

- Cart is an order with status `draft`; there is no separate cart table.
- `address_id` is nullable for draft orders.
- One order belongs to one seller.
- Checkout splits draft carts by seller.
- Order lifecycle is `draft -> pending -> paid -> shipped -> delivered`.
- Cancellation is allowed only before `shipped`.
- Seller access is limited to own products and own relevant orders.
- Buyer access is limited to own addresses, orders, and reviews.
- A buyer can leave one review per product.
- `price_at_purchase` is fixed when draft becomes pending.
- Product price history can be backed by a DB trigger, but service code still enforces business rules explicitly.

## Error Boundaries

Repository code translates database conditions into database-agnostic service errors at the repository boundary. Service and handler code must not inspect `pgx`, `pgconn`, SQLSTATE codes, or constraint names.

Handlers map service/application errors to HTTP status codes in `internal/handler/service_error.go`. Use `errors.Is` / `errors.As`, wrap with `%w`, use `pgerrcode.*` in repository code, and use `net/http` constants in HTTP code.

## Web And API Surfaces

The JSON API is routed in `internal/handler/router.go` under `/api/v1`.

The web MPA is routed in `internal/web/router.go`. Templates live in `internal/web/templates`; shared CSS lives in `internal/web/static/css/style.css`.

Both surfaces call service methods. Neither surface owns business policy, persistence, or cache invalidation.

## Cache Boundary

Cache key mechanics belong behind cache abstractions, not handlers. Cache invalidation should be triggered from service/usecase code or service decorators as part of mutations.

Documented key conventions:

| Key | TTL | Invalidation |
| --- | --- | --- |
| `products:catalog:page:{n}` | `5m` | Any product change |
| `products:{id}` | `5m` | Product PATCH/DELETE; review create/update |
| `categories:tree` | `1h` | Category CRUD |
| `sessions:{token}` | Session lifetime | Logout |

Cache failures should not break core business behavior unless the operation explicitly requires cache consistency.
