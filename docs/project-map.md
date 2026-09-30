# Project Map

This map describes the current PPO repository shape and runtime wiring used as
the WebLabs 2026 starting point. When this document conflicts with code, trust
code after checking it directly. The assignment is
[`WEB_labs/WebLabs_2026_AI_transformation.docx.pdf`](../WEB_labs/WebLabs_2026_AI_transformation.docx.pdf);
[`AGENTS.md`](../AGENTS.md) tracks the current/planned boundary.

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
| `WEB_labs/` | WebLabs 2026 assignment; requirements, not implemented artifacts. |
| `diagrams/src/` | Editable PPO diagrams and DBML; verify and extend for WebLab#1. |
| `scripts/` | Local helper scripts and SQL snippets. Not runtime application code. |

## Agent Tooling

`AGENTS.md` is the canonical project guidance for Codex in the AI track.
[`AI_REVIEW.md`](../AI_REVIEW.md) records factual work and checks per lab.

| Path | Purpose |
| --- | --- |
| `.codex/config.toml` | Project Codex CLI defaults. |
| `.codex/agents/` | Codex project agents for backend, frontend, and architecture work. |

The existing `/api/v1/` route list is in [`docs/api-contracts.md`](api-contracts.md).
The two student-selected WebLab#1 decisions are recorded in
[`docs/ADR.md`](ADR.md): stock reservation at checkout and the draft order as
the buyer's cart. There is no approved WebLab#2 OpenAPI YAML yet. The current
web surface is an MPA; the WebLab#4 `/legacy` route and WebLab#8 SPA are future
work, so their paths are not part of the runtime map above.

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

## Confirmed Constraints And Requirement Gaps

WebLab#1 AI-track review, 2026-09-30. These facts were checked in source code;
they are not results from running the application or measuring the NF targets.
The evidence is the current working tree. Pre-existing uncommitted review
changes restrict eligibility to `delivered`; baseline commit `7ad0f94` still
accepts `paid`/`shipped` as well. Those code changes are preserved separately
and are not part of the proposed WebLab#1 documentation commit. F-6 and the
accepted scenario criteria will define the expected WebLab#2/#3 behavior.

| Confirmed Fact | Evidence | Consequence For WebLab#1/#2 |
| --- | --- | --- |
| Checkout reserves stock, fixes prices and splits the draft cart by seller in one transaction. Ship consumes both stock and reservation; delivery changes status only. | [`order_service.go`](../internal/service/order_service.go), [`order-lifecycle.md`](order-lifecycle.md) | F-3/F-4 and scenario acceptance must preserve these invariants; neither a separate cart store nor physical stock decrement on payment exists. |
| The seller marks `shipped` and `delivered`; there is no carrier integration or buyer delivery-confirmation action. | [`order_state.go`](../internal/service/order_state.go), [`web/router.go`](../internal/web/router.go) | Fulfillment BPMN and screens must identify the seller as the actor for both mutations. |
| A new review requires a delivered purchase; one review per buyer/product is enforced. | [`review_service.go`](../internal/service/review_service.go), [`review_repo.go`](../internal/repository/review_repo.go), [`001_create_tables.sql`](../migrations/001_create_tables.sql) | Scenario 3 must include before-delivery and duplicate-review errors. |
| Mock Bank shares the HTTP process; no persisted payment records or real provider exist. Payment TTL uses `orders.created_at`. | [`payments.md`](payments.md), [`payment_service.go`](../internal/service/payment_service.go) | Logical bank participation is not a separate deployment. An old single-seller cart can expire immediately; [BUG-001](known-issues.md#bug-001-payment-ttl-starts-from-draft-cart-creation) remains open. |
| Product categories are supported by service/MPA but omitted from JSON create/update DTOs. Seller MPA statistics use the last year; API accepts a date range. | [`product.go`](../internal/handler/product.go), [`web/seller.go`](../internal/web/seller.go), [`seller.go`](../internal/handler/seller.go) | Scenario 2 works through MPA for category selection. The future API contract must resolve category writes; the period picker is a future screen control. |
| Seller statistics are publicly readable by seller ID; the service has no actor parameter for this read. | [`handler/router.go`](../internal/handler/router.go), [`seller_service.go`](../internal/service/seller_service.go) | The scenario tests the calculation, not report confidentiality. Define the read-access policy explicitly in WebLab#2; NF-S concerns protected mutations only. |
| Redis caches only product lookup by ID; stock and review mutations do not invalidate that cache. | [`product_repo_cache.go`](../internal/cache/product_repo_cache.go), [`cache.md`](cache.md) | Product availability/rating shown on a page can lag behind PostgreSQL. NF-R concerns failure after startup; startup with enabled but unavailable Redis currently fails. |

The student's scope answers are recorded: Backend solo, WebLab#1–#6; two
accepted ADRs; an instructor-approved exception allowing two BPMNs, Checkout
and Fulfillment, instead of the written minimum of three. The student accepted
Fulfillment's notation and pool/lane structure, and its
[`yEd PDF`](../diagrams/out/BPMN_Fulfillment.pdf) exists and was rendered and
visually checked in the final pass. ER Chen and checkout BPMN corrections
remain explicitly deferred in [`known-issues.md`](known-issues.md).
NF-P/NF-R/NF-S and the proposed i5-13500H / 16 GB / Ubuntu 24.04 stand are
accepted by the student as targets for future checks, not passed checks. Draft screens do not
imply an implemented SPA or approved REST contract.

The student agreed three additional Use Cases for F-4: buyer payment and
cancellation, seller management of own orders; their actor links are direct.
The student accepted success/error criteria for all four scenarios, including
the stated TTL/cache/API limitations. The student rejected a historical note
for the outdated checkout sequence diagram and requested an update from code.
The [PlantUML](../diagrams/src/SeqCheckoutComponent.puml), PNG and
[SVG](../diagrams/out/SeqCheckout.svg) now describe transactional checkout,
reservation/prices and single/multi-seller branches. After review, the student
limited the diagram to checkout only: no cart mutations or payment. Codex
checked the correction; another student viewing of the rendering is not claimed.
The student accepted the remaining material and the eight-commit handoff plan.
The student approved 46 documentation/diagram/agent-guidance files for thematic
commits; the local Fulfillment generator and its tests are excluded, not deleted.
Any additional diagram files need agreement before committing.
Commit messages are Russian and describe completed changes, not plans;
see the root AGENTS and the style guide.
Product-category writes are a confirmed gap in the current JSON
API; their contract is future WebLab#2 work, not an implementation task or a
required architecture choice in WebLab#1. BPMN count and ADR selection are
closed. The detailed contradiction review and
scenario → requirement/entity/screen links are in the
[`README`](../README.md); [`AI_REVIEW.md`](../AI_REVIEW.md) records actual checks.

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
