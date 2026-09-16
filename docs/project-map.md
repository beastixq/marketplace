# Project Map

This map describes the current repository shape and runtime wiring. When this
document conflicts with code, trust code after checking it directly.

## Entry Points

| Path | Purpose | Notes |
| --- | --- | --- |
| `cmd/api/main.go` | Main HTTP server. | Loads config, connects PostgreSQL, optionally connects Redis, wires repositories/services/handlers/web, starts the order expiration worker, and serves API plus web UI. |
| `cmd/techui/main.go` | Console technical UI. | Uses the same service layer through component packages. It bypasses HTTP handlers and starts the order expiration worker. |
| `cmd/seed/main.go` | Local data generator. | Inserts synthetic users, sellers, products, orders, reviews, and price history in one transaction. |

The API server mounts both surfaces on one `http.ServeMux`:

```text
/api/* -> internal/handler router
/*     -> internal/web router
```

## Runtime And Source Flow

Runtime request flow:

```text
handler / web / techui -> service -> repository implementation -> PostgreSQL
                                  -> port -> adapter
```

Source dependency direction:

```text
handler / web / techui -> service <- repository implementation
service -> port <- adapter implementation
```

The service layer owns repository interfaces. Repository implementations import
`internal/service` and satisfy those interfaces. Handlers and web controllers
must not import repositories.

## Directory Map

| Path | What Lives There |
| --- | --- |
| `internal/model/` | Domain models and value structs used across service/repository boundaries. |
| `internal/service/` | Business rules, ownership checks, lifecycle transitions, service-owned repository interfaces, and application errors. |
| `internal/repository/` | PostgreSQL SQL, pgx/pgxpool access, transactions, row mapping, and DB error translation. |
| `internal/handler/` | JSON API handlers, DTOs, request validation, router groups, and service error mapping. |
| `internal/web/` | Server-rendered MPA controllers, templates, static CSS, and web form flows. |
| `internal/middleware/` | Auth, role checks, actor propagation, request logging, and panic recovery. |
| `internal/cache/` | Redis client setup and cache-aside decorators. Current runtime cache covers product lookup by ID. |
| `internal/port/` | External-facing service contracts, currently the payment gateway interface. |
| `internal/adapter/` | Implementations of ports, currently the mock bank payment gateway. |
| `internal/component/` | Wiring helpers used by `cmd/techui` and reusable composition paths. `cmd/api` wires manually. |
| `internal/config/` | YAML config loader, environment overrides, and config validation. |
| `internal/logging/` | slog setup for stdout/file logging. |
| `internal/testsupport/` | Shared test helpers, currently transaction support. |
| `internal/mocks/` | Generated gomock mocks. Do not edit manually. |
| `migrations/` | Goose migrations for schema, indexes, roles, triggers, functions, and constraints. |
| `config/` | Default local YAML config. |
| `docs/` | Maintenance documentation. Keep it synchronized with current code. |
| `scripts/` | Local helper scripts and SQL snippets. Not runtime application code. |

## Agent Tooling

`AGENTS.md` is the canonical project guidance for AI coding agents.

| Path | Purpose |
| --- | --- |
| `.codex/config.toml` | Project Codex CLI defaults. |
| `.codex/agents/` | Codex project agents for backend, frontend, and architecture work. |

## Main Use-Case Map

| Area | Primary Service | Repository/Port | API/Web Surface |
| --- | --- | --- | --- |
| Auth and users | `AuthService`, `UserService` | `UserRepo`, optional `TokenBlocklist` | `AuthHandler`, `UserHandler`, web login/register/profile |
| Catalog and products | `ProductService` | `ProductRepo`, `ProductCategoryRepo`, `ReviewRepo` | product/category handlers, catalog and product pages |
| Cart and orders | `OrderService` | `OrderRepo`, `OrderItemRepo`, `ProductRepo`, `AddressRepo`, `SellerRepo` | cart/order handlers and web pages |
| Payments | `PaymentService` | `OrderRepo`, `PaymentGateway` port | payment handler, mock bank web page, tech UI |
| Reviews and ratings | `ReviewService` | `ReviewRepo`, product lookup, purchase checker | review handlers and product page forms |
| Seller workflow | `SellerService`, `OrderService`, `ProductService` | seller/product/order repos | seller API routes and seller web dashboard |
| Admin/backoffice | `UserService`, `SellerService`, `BackofficeService` | user/seller/backoffice repos | admin API routes and admin web pages |
| Analyst reports | `BackofficeService` | `BackofficeRepo` | analyst web dashboard |

## Important Wiring Details

- `cmd/api` wraps `ProductRepo` with `cache.ProductRepoCache` only when
  `redis.enabled` is true and Redis ping succeeds.
- `cmd/api` currently passes `nil` as the auth token blocklist. Logout is a
  no-op in that runtime path until a blocklist implementation is wired.
- `cmd/techui` uses `internal/component/repository.New`, which does not enable
  Redis caching by default.
- `OrderExpirationWorker` runs in both API and tech UI commands. It expires
  pending orders using `payment.ttl` and `orders.expiration_check_interval`.
- PostgreSQL remains the source of truth. Redis is a cache layer only.

## Read Next

| Topic | Document |
| --- | --- |
| Layer boundaries and dependency rules | `docs/architecture.md` |
| Order status transitions, stock reservation, and checkout splitting | `docs/order-lifecycle.md` |
| Mock bank flow, payment TTL, and payment errors | `docs/payments.md` |
| Redis cache keys, cache-aside behavior, and current gaps | `docs/cache.md` |
| JSON API routes and status codes | `docs/api-contracts.md` |
| Migrations and database constraints | `docs/database.md`, then `docs/db-schema.md` |
| Local run/test commands | `docs/setup.md`, `docs/testing.md` |
