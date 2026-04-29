# AGENTS.md

**Core docs:** [Readme.md](Readme.md) | [docs/architecture.md](docs/architecture.md) | [docs/setup.md](docs/setup.md) | [docs/testing.md](docs/testing.md) | [docs/api-contracts.md](docs/api-contracts.md) | [docs/database.md](docs/database.md) | [docs/style-guide.md](docs/style-guide.md) | [docs/troubleshooting.md](docs/troubleshooting.md) | [docs/release-process.md](docs/release-process.md) | [docs/db-schema.md](docs/db-schema.md) | [docs/tz.md](docs/tz.md)

## Documentation Use

Use the project docs as the first stop for task-specific context:

| Task | Read first |
| --- | --- |
| Architecture, dependencies, layer ownership | [docs/architecture.md](docs/architecture.md) |
| Local startup, config, migrations, app commands | [docs/setup.md](docs/setup.md) |
| Unit/integration/web test workflow | [docs/testing.md](docs/testing.md) |
| JSON API routes, status codes, DTO/error conventions | [docs/api-contracts.md](docs/api-contracts.md) |
| Schema, migrations, triggers, DB roles | [docs/database.md](docs/database.md), then [docs/db-schema.md](docs/db-schema.md) |
| Go style, web UI style, SQL style, documentation style | [docs/style-guide.md](docs/style-guide.md) |
| Common local failures | [docs/troubleshooting.md](docs/troubleshooting.md) |
| Pre-release or submission checklist | [docs/release-process.md](docs/release-process.md) |
| Coursework requirements and report context | [docs/tz.md](docs/tz.md), [docs/RPZ.md](docs/RPZ.md) |

If docs and current code conflict, trust the current code after verifying it directly, then update the relevant doc as part of the change. Keep docs concise and maintenance-oriented.

## Custom Codex Agents

Project-scoped Codex agents live in `.codex/agents/`.

Use the matching custom agent when the user explicitly asks to use agents, delegate work, or split work by specialty:

| Agent | Use for |
| --- | --- |
| `go-backend-dev` | Go backend implementation: services, repositories, API handlers, migrations, payment, caching, backend tests. |
| `frontend-dev` | Server-rendered web UI: `html/template`, `internal/web`, forms, CSS, role dashboards, web routes. |
| `architecture` | Clean Architecture review/design, dependency-boundary analysis, cross-layer refactors, ownership decisions. |

Do not delegate work only because a matching agent exists. When using agents, keep tasks scoped and make each agent follow this `AGENTS.md` plus the relevant docs above.

## Architecture

Runtime call flow:

```text
handler -> service -> repository
```

Source dependency direction goes inward. Dependency inversion is done through service-owned interfaces.

```text
handler -> service <- repository implementation
```

The service layer does not depend on repository implementations. Service exports the repository interfaces it needs, accepts those interfaces through constructors, and calls through them. Repository implementations depend on service-defined interfaces and satisfy them.

| Directory              | Purpose                                                          |
| ---------------------- | ---------------------------------------------------------------- |
| `internal/model/`      | Domain models, not DB rows and not HTTP DTOs                     |
| `internal/service/`    | Business logic, use cases, service-owned repo interfaces, errors |
| `internal/repository/` | Repository implementations, SQL/pgx code, DB mapping             |
| `internal/handler/`    | HTTP handlers, request/response DTOs, response helpers           |
| `internal/cache/`      | Redis/cache mechanics and cache-aside abstractions               |
| `internal/middleware/` | Auth and RBAC middleware                                         |
| `internal/mocks/`      | Generated mocks; do not edit manually                            |

## Dependency Rules

* `service` must not import `internal/repository` or any repository-specific library. Keep dependencies minimal.
* `handler` must not import `internal/repository`.
* Repository interfaces are owned by the consuming service layer.
* In this project, repo interfaces should live in `internal/service` unless a narrower consumer package already exists.
* Repository implementations live in `internal/repository` and implement service-defined interfaces.
* Data mapping must be explicit:

```text
DB row -> model -> DTO
```

* DB rows, SQL-specific types, and HTTP DTOs must not leak across layers.
* Dependencies should be passed through constructors, not hidden globals.

## Errors

* Shared repository-facing application errors may be defined in `internal/service`, but they must remain database-agnostic.
* Repository code should translate known `pgx`/`pgconn`/database conditions into application/domain errors at the repository boundary.
* Service and handler code must not depend on DB-specific error types, SQLSTATE codes, constraint names, or `pgx`/`pgconn` details.
* Service may treat unknown repository errors as repository/internal failures without inspecting database-specific details.
* Handlers map service/application errors to HTTP status codes.

## Service Layer

* Business rules, ownership checks, authorization decisions, order lifecycle transitions, and orchestration belong in service code.
* Repositories may enforce data-integrity constraints and transactions, but should not decide business policy unless it is purely persistence-level integrity.
* Gomock is preferred for generated mocks; hand-written fakes are acceptable when simpler.
* Use package `service` only when intentionally testing unexported helpers.
* `go:generate mockgen` directives should live near service-defined interfaces or in a dedicated generation file.

## Repository Layer

* Repository code should contain SQL, transactions, DB error translation, and DB-to-domain mapping.
* Repository methods should use domain/value types from `internal/model` or small parameter structs.
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

## Handler Layer

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

## Business Rules

* Cart is a draft order; there is no separate cart table.
* `address_id` is nullable for draft orders.
* One order belongs to one seller.
* Draft cart is split by seller during checkout.
* Order lifecycle: draft -> pending -> paid -> shipped -> delivered.
* Cancellation is allowed only before `shipped`.
* Seller can access only own products and own relevant orders.
* Buyer can access only own addresses and reviews.
* A user can leave only one review per product: `UNIQUE(user_id, product_id)`.
* `price_at_purchase` is fixed when draft becomes `pending`.
* Product price history may be enforced by a database trigger as an audit/data-integrity guarantee.
* Service logic must still explicitly enforce business rules such as fixing `price_at_purchase`.
* `get_seller_statistics(seller_id, date_from, date_to)` returns:
  * `total_orders`
  * `total_revenue`
  * `avg_order_value`
  * `top_product_name`

## PostgreSQL Roles

| Role                  | Rights                                                           |
| --------------------- | ---------------------------------------------------------------- |
| `marketplace_buyer`   | SELECT products, categories; CRUD own orders, reviews, addresses |
| `marketplace_seller`  | CRUD own products; SELECT order_items for own products           |
| `marketplace_admin`   | ALL PRIVILEGES                                                   |
| `marketplace_analyst` | SELECT ALL, no writes                                            |

## Go And Clean Code

* Prefer small interfaces owned by consumers.
* Accept interfaces only when behavior is needed, not merely for abstraction.
* Return concrete/domain structs where appropriate.
* Avoid hidden global dependencies.
* Keep function boundaries clear.
* A function should do one thing.
* Comments should explain why, not restate what the code already says.
* Avoid unrelated refactors and formatting-only churn.
* Do not manually edit generated files.
