# CLAUDE.md

**Docs:** [Readme.md](Readme.md) | [docs/db-schema.md](docs/db-schema.md) | [docs/tz.md](docs/tz.md)

## Architecture

Layer direction:

```text
handler → service → repository
```

Dependency direction goes inward. Dependency inversion is done through service-owned interfaces.
handler → service ← repository implementation

| Directory              | Purpose                                                          |
| ---------------------- | ---------------------------------------------------------------- |
| `internal/model/`      | Domain models, not DB rows and not HTTP DTOs                     |
| `internal/service/`    | Business logic, use cases, service-owned repo interfaces, errors |
| `internal/repository/` | Repository implementations, SQL/pgx code, DB mapping             |
| `internal/handler/`    | HTTP handlers, request/response DTOs, response helpers           |
| `internal/cache/`      | Redis/cache mechanics and cache-aside abstractions               |
| `internal/middleware/` | Auth and RBAC middleware                                         |
| `internal/mocks/`      | Generated mocks; do not edit manually                            |


## Dependency rules

* `service` must not import `internal/repository` or any repo specific library. minimal number of deps.
* `handler` must not import `internal/repository`.
* Repository interfaces are owned by the consuming service layer.
* In this project, repo interfaces should live in `internal/service` unless a narrower consumer package already exists.
* Repository implementations live in `internal/repository` and implement service-defined interfaces.
* Data mapping must be explicit:

```text
DB row → model → DTO
```

* DB rows, SQL-specific types, and HTTP DTOs must not leak across layers.
* Dependencies should be passed through constructors, not hidden globals.

Compile-time repository interface checks should usually use pointer form:

```go
var _ service.ProductRepo = (*ProductRepository)(nil)
```

## Errors

* Shared repository-facing application errors may be defined in `internal/service`, but they must remain database-agnostic.
* Repository code should translate known `pgx`/`pgconn`/database conditions into application/domain errors at the repository boundary.
* Service and handler code must not depend on DB-specific error types, SQLSTATE codes, constraint names, or `pgx`/`pgconn` details.
* Service may treat unknown repository errors as repository/internal failures without inspecting database-specific details.
* Handlers map service/application errors to HTTP status codes.
* Use `%w` when wrapping errors.
* Use `errors.Is` / `errors.As` when checking errors.
* Use `pgerrcode.*` instead of raw SQLSTATE strings.
* Use `net/http` constants instead of raw HTTP status codes.

## Service layer

* Business rules, ownership checks, authorization decisions, order lifecycle transitions, and orchestration belong in service code.
* Repositories may enforce data-integrity constraints and transactions, but should not decide business policy unless it is purely persistence-level integrity.
* Public service methods should take `context.Context` as the first argument.
* Service tests should be unit tests with test doubles.
* Gomock is preferred for generated mocks; hand-written fakes are acceptable when simpler.
* Service tests should generally use package `service_test`.
* Use package `service` only when intentionally testing unexported helpers.
* `go:generate mockgen` directives should live near service-defined interfaces or in a dedicated generation file.
* Generated mocks go to `internal/mocks/service/`.

## Repository layer

* Repository code should contain SQL, transactions, DB error translation, and DB-to-domain mapping.
* Repository methods should use domain/value types from `internal/model` or small parameter structs.
* Do not return DB rows or HTTP DTOs from repositories.
* SQL, transactions, constraints, triggers, and PostgreSQL roles should be tested with integration tests against a real database.
* Pure mapper/helper logic may be unit-tested separately.
* Repository integration tests should generally use package `repository_test`.
* Centralize integration test setup in `repository/testmain_test.go`.

`repository/testmain_test.go` should contain shared test infrastructure, not business test cases:

* `TestMain`
* test DB connection setup
* migrations or test schema preparation
* cleanup/truncation helpers
* shared repository test configuration

## Handler layer

* HTTP DTOs live in `internal/handler`.
* Request validation happens at the handler boundary.
* Request DTOs may expose `Validate() error`, or validation may use a dedicated validator.
* Business invariants stay in the service layer.
* Handlers must not contain business logic.
* Handlers must not access repositories directly.
* Handlers must not perform cache invalidation.
* Handlers must not know database details.

## Cache / Redis

Handlers must not perform cache invalidation.

Redis/key mechanics belong in `internal/cache`. Cache-aside behavior should be implemented behind cache abstractions or service decorators. Service/usecase code or a cache-aware decorator should trigger invalidation as part of mutations.

| Key                         | TTL              | Invalidation                               |
| --------------------------- | ---------------- | ------------------------------------------ |
| `products:catalog:page:{n}` | `5m`             | Any product change                         |
| `products:{id}`             | `5m`             | PATCH/DELETE product; create/update review |
| `categories:tree`           | `1h`             | Category CRUD                              |
| `sessions:{token}`          | Session lifetime | Logout                                     |

Cache failures should not break core business behavior unless the operation explicitly requires cache consistency.

## Business rules

* Cart is a draft order; there is no separate cart table.
* `address_id` is nullable for draft orders.
* One order belongs to one seller.
* Draft cart is split by seller during checkout.
* Order lifecycle:
draft → pending → paid → shipped → delivered
* Cancellation is allowed only before `shipped`.
* Seller can access only own products and own relevant orders.
* Buyer can access only own addresses and reviews.
* A user can leave only one review per product: `UNIQUE(user_id, product_id)`.
* `price_at_purchase` is fixed when draft becomes `pending`.
* Product price history may be enforced by a database trigger as an audit/data-integrity guarantee.
* Service logic must still explicitly enforce business rules such as fixing `price_at_purchase`.
* `get_seller_statistics(seller_id, date_from, date_to)` returns:
total_orders
total_revenue
avg_order_value
top_product_name

## PostgreSQL roles

| Role                  | Rights                                                           |
| --------------------- | ---------------------------------------------------------------- |
| `marketplace_buyer`   | SELECT products, categories; CRUD own orders, reviews, addresses |
| `marketplace_seller`  | CRUD own products; SELECT order_items for own products           |
| `marketplace_admin`   | ALL PRIVILEGES                                                   |
| `marketplace_analyst` | SELECT ALL, no writes                                            |

## Go and clean code

* Prefer small interfaces owned by consumers.
* Accept interfaces only when behavior is needed, not merely for abstraction.
* Return concrete/domain structs where appropriate.
* Avoid hidden global dependencies.
* Keep function boundaries clear.
* A function should do one thing.
* Comments should explain why, not restate what the code already says.
* Avoid unrelated refactors and formatting-only churn.
* Do not manually edit generated files.
