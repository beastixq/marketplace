# AGENTS.md

This file is the single canonical source for agent guidance.
OpenCode and Codex CLI read `AGENTS.md` directly. `CLAUDE.md` is a legacy
symlink to this file.

Active project agent surfaces:
- OpenCode: `opencode.json`, `.opencode/agents/*.md`, `.opencode/commands/*.md`
- Codex: `.codex/config.toml`, `.codex/agents/*.toml`
- Legacy Claude Code: `.claude/agents/*.md`, `.claude/settings.json`

When changing agent guidance, keep the active OpenCode and Codex surfaces
aligned manually. Update the legacy Claude Code surface only if Claude Code is
used again.

## Documentation Use

Use the project docs as the first stop for task-specific context.

| Task                                                   | Read first                                                                                                                 |
| ------------------------------------------------------ | -------------------------------------------------------------------------------------------------------------------------- |
| Project layout, entry points, wiring map               | [docs/project-map.md](docs/project-map.md)                                                                                 |
| Architecture, dependencies, layer ownership            | [docs/architecture.md](docs/architecture.md)                                                                               |
| Order lifecycle, checkout, stock reservation           | [docs/order-lifecycle.md](docs/order-lifecycle.md)                                                                         |
| Payment flow, mock bank, payment TTL                   | [docs/payments.md](docs/payments.md)                                                                                       |
| Redis/cache behavior and implemented keys              | [docs/cache.md](docs/cache.md)                                                                                             |
| Local startup, config, migrations, app commands        | [docs/setup.md](docs/setup.md)                                                                                             |
| Unit/integration/web test workflow                     | [docs/testing.md](docs/testing.md)                                                                                         |
| JSON API routes, status codes, DTO/error conventions   | [docs/api-contracts.md](docs/api-contracts.md)                                                                             |
| Schema, migrations, triggers, DB roles                 | [docs/database.md](docs/database.md), then [docs/db-schema.md](docs/db-schema.md)                                          |
| Go style, web UI style, SQL style, documentation style | [docs/style-guide.md](docs/style-guide.md)                                                                                 |
| Common local failures                                  | [docs/troubleshooting.md](docs/troubleshooting.md)                                                                         |
| Pre-release or submission checklist                    | [docs/release-process.md](docs/release-process.md)                                                                         |
| Open bugs, product ideas, resolved bug history         | [docs/known-issues.md](docs/known-issues.md), [docs/ideas.md](docs/ideas.md), [docs/bugs-history.md](docs/bugs-history.md) |
| Coursework requirements and report context             | [docs/tz.md](docs/tz.md), [docs/RPZ.md](docs/RPZ.md)                                                                       |

If docs and current code conflict, trust the current code after verifying it directly, then update the relevant doc as part of the change. Keep docs concise and maintenance-oriented.

## Cost-Controlled Reads

Commands are cheap; context reads are expensive.

OK without asking when relevant:
- tests, builds, linters, formatters;
- code generation;
- make targets;
- dependency downloads;
- targeted git commands.

Avoid without a specific reason:
- reading generated mocks end-to-end;
- reading vendored/third-party code;
- reading large fixtures, PDFs, or reports;
- cat'ing 1k+ line files;
- re-reading files just edited.

Prefer targeted reads: `rg`, `Read` with offset/limit, `git status --short`, narrow `git log`.

## Custom Subagents

Project-scoped subagents:
- OpenCode: `.opencode/agents/*.md`
- Codex: `.codex/agents/*.toml`
- Legacy Claude Code: `.claude/agents/*.md`

Keep active surfaces aligned manually.

OpenCode project commands:
- `/review`: review the current diff without editing files.
- `/verify`: run focused verification for the current change without editing files.

Use subagents only when the user explicitly asks to delegate/split work or when a clearly scoped specialist review is useful.

| Agent            | Use for                                                                                                       |
| ---------------- | ------------------------------------------------------------------------------------------------------------- |
| `go-backend-dev` | Go backend implementation: services, repositories, API handlers, migrations, payment, caching, backend tests. |
| `frontend-dev`   | Server-rendered web UI: `html/template`, `internal/web`, forms, CSS, role dashboards, web routes.             |
| `architecture`   | Clean Architecture review/design, dependency-boundary analysis, cross-layer refactors, ownership decisions.   |

Do not delegate work only because a matching agent exists. When using agents, keep tasks scoped and make each agent follow this `AGENTS.md` plus the relevant docs above.

## Architecture

Runtime call flow:

```text
handler / web / techui -> service -> repository implementation
```

Source dependency direction goes inward. Dependency inversion is done through service-owned interfaces.

```text
handler / web / techui -> service <- repository implementation
```

The service layer does not depend on repository implementations. Service exports the repository interfaces it needs, accepts those interfaces through constructors, and calls through them. Repository implementations depend on service-defined interfaces and satisfy them.

| Directory              | Purpose                                                          |
| ---------------------- | ---------------------------------------------------------------- |
| `internal/model/`      | Domain models, not DB rows and not HTTP DTOs                     |
| `internal/service/`    | Business logic, use cases, service-owned repo interfaces, errors |
| `internal/repository/` | Repository implementations, SQL/pgx code, DB mapping             |
| `internal/handler/`    | HTTP handlers, request/response DTOs, response helpers           |
| `internal/web/`        | Server-rendered MPA controllers, templates, static CSS           |
| `internal/cache/`      | Redis/cache mechanics and cache-aside abstractions               |
| `internal/middleware/` | Auth, RBAC, logging, recovery, actor propagation                 |
| `internal/port/`       | External service contracts (e.g. payment gateway)                |
| `internal/adapter/`    | External service implementations satisfying ports                |
| `internal/component/`  | Wiring helpers for repository/service/tech UI composition        |
| `internal/mocks/`      | Generated mocks; do not edit manually                            |
| `cmd/api`              | API + web server entry point                                     |
| `cmd/seed`             | Seed data command                                                |
| `cmd/techui`           | Console UI driving service-layer use cases without HTTP          |

## Dependency Rules

- `service` must not import `internal/repository`, `pgx`, `pgconn`, SQLSTATE values, constraint names, or repository-specific libraries. Keep dependencies minimal.
- `handler` and `web` must not import `internal/repository`.
- Repository interfaces are owned by the consuming service layer.
- In this project, repo interfaces should live in `internal/service` unless a narrower consumer package already exists.
- Repository implementations live in `internal/repository` and implement service-defined interfaces.
- Data mapping must be explicit:

```text
DB row -> model -> DTO / view data
```

- DB rows, SQL-specific types, HTTP DTOs, and web view data must not leak across layers.
- Dependencies should be passed through constructors, not hidden globals.
- Compile-time repository interface checks should usually use pointer form:

```go
var _ service.ProductRepo = (*ProductRepository)(nil)
```

## Errors

- Shared repository-facing application errors may be defined in `internal/service`, but they must remain database-agnostic.
- Repository code should translate known `pgx`/`pgconn`/database conditions into application/domain errors at the repository boundary.
- Service and handler code must not depend on DB-specific error types, SQLSTATE codes, constraint names, or `pgx`/`pgconn` details.
- Service may treat unknown repository errors as repository/internal failures without inspecting database-specific details.
- Handlers map service/application errors to HTTP status codes. Centralized mapping lives in `internal/handler/service_error.go`.
- Use `%w` when wrapping errors.
- Use `errors.Is` / `errors.As` when checking errors.
- Use `pgerrcode.*` instead of raw SQLSTATE strings.
- Use `net/http` constants instead of raw HTTP status codes.

## Service Layer

- Business rules, ownership checks, authorization decisions, order lifecycle transitions, and orchestration belong in service code.
- Repositories may enforce data-integrity constraints and transactions, but should not decide business policy unless it is purely persistence-level integrity.
- Service owns both inbound usecase APIs and outbound ports. Outer layers call service APIs; repositories/adapters satisfy service-defined interfaces.
- The service is the business core and owns the contracts at both its boundaries. Inbound (handler/web/techui → service): the service publishes its public methods and the outer callers accept those signatures — they do not redefine them. Outbound (service → repository/port): the service declares the interface in `internal/service` and the repository implementation in `internal/repository` (or adapter in `internal/adapter`) satisfies it — the implementation does not get to define what the contract means. This is hexagonal/ports-and-adapters: the core declares ports on both sides, adapters plug in.
- Interface method comments in `internal/service` define semantic contracts: return meaning, error sentinels, idempotency, and forbidden caller assumptions. Implementation comments may mention only the mechanism.
- For atomic check-then-mutate flows, choose the weakest safe mechanism: plain pre-check if stale rows cannot leak into user-visible incorrect behavior; transaction/locking when money, inventory, or lifecycle correctness depends on it. State the choice in review, not as noisy implementation comments.
- Public service methods should take `context.Context` as the first argument.
- Service tests should be unit tests with test doubles, generally in package `service_test`.
- Use package `service` only when intentionally testing unexported helpers.
- Gomock is preferred for service-owned interfaces. Hand-written fakes are acceptable only when the fake is substantially clearer than a generated mock or when mock generation is unavailable.
- `go:generate mockgen` directives should live near service-defined interfaces or in a dedicated generation file.
- Generated mocks go to `internal/mocks/service/`. Do not inspect or edit generated mocks during normal work; regenerate them from the owning interface instead.

## Repository Layer

- Repository code should contain SQL, transactions, DB error translation, and DB-to-domain mapping.
- Repository methods should use domain/value types from `internal/model` or small parameter structs.
- Do not return DB rows or HTTP DTOs from repositories.
- Build queries with squirrel (`sq.StatementBuilder.PlaceholderFormat(sq.Dollar)`). Use `Suffix(...)` for `RETURNING`, `ON CONFLICT`, and `FOR UPDATE`. Raw SQL is acceptable only when squirrel cannot express the query; new raw SQL blocks are a style smell and need justification.
- Existence checks use `SELECT 1 ... LIMIT 1` + `pgx.ErrNoRows`, not `SELECT EXISTS (...)`.
- When porting code from another branch, scaffold, or AI-generated output, normalize it to these conventions *before* committing. Idiomatic style in the source repository is not a license to deviate here.
- SQL, transactions, constraints, triggers, and PostgreSQL roles should be tested with integration tests against a real database.
- Pure mapper/helper logic may be unit-tested separately.
- Repository integration tests should generally use package `repository_test`.
- Centralize integration test setup in `repository/testmain_test.go`.

`repository/testmain_test.go` should contain shared test infrastructure, not business test cases:

- `TestMain`
- test DB connection setup
- migrations or test schema preparation
- cleanup/truncation helpers
- shared repository test configuration

## Handler Layer

- HTTP DTOs live in `internal/handler`.
- Request validation happens at the handler boundary.
- Request DTOs may expose `Validate() error`, or validation may use a dedicated validator.
- Business invariants stay in the service layer.
- Handlers must not contain business logic.
- Handlers must not access repositories directly.
- Handlers must not perform cache invalidation.
- Handlers must not know database details.
- API routes are prefixed with `/api/v1/`. Error responses use `{"error":"message"}`.

## Web UI Layer

- The web UI is a server-rendered MPA, not a SPA. Do not introduce npm/pnpm/yarn/Vite/React/Vue unless the user explicitly asks.
- Templates live in `internal/web/templates/*.html`. Each page defines `title` and `content`, then renders through `templates/layout.html`.
- Shared CSS lives in `internal/web/static/css/style.css`.
- Web handlers live in `internal/web/web_handler.go`; routes live in `internal/web/router.go`.
- Web handlers may parse request/form data, call services, shape view data, set cookies, redirect, and render templates.
- Web handlers must not import `internal/repository`, must not contain business invariants, and must not perform cache invalidation.
- Use POST forms for mutations and redirect after success.

## Cache / Redis

Handlers must not perform cache invalidation. Redis/key mechanics belong in `internal/cache`. Cache-aside behavior should be implemented behind cache abstractions or service decorators. Service/usecase code or a cache-aware decorator should trigger invalidation as part of mutations. Keep [docs/cache.md](docs/cache.md) synchronized with the implemented keys.

| Key | TTL | Invalidation |
| --- | --- | --- |
| `products:{id}` | `redis.product_ttl` | Product/review/category relation changes |
| `products:catalog:{canonical-query}` | `redis.catalog_ttl` | Product/review/category changes |
| `categories:list:{canonical-query}` | `redis.category_ttl` | Category CRUD |
| `products:{id}:reviews:page={page}&limit={limit}` | `redis.review_ttl` | Review changes |
| `sessions:{jti}` | Remaining JWT lifetime | Logout/token expiration |

Cache failures should not break core business behavior unless the operation explicitly requires cache consistency.

## Business Rules

- Cart is a draft order; there is no separate cart table.
- Each buyer can have only one draft cart.
- `address_id` is nullable for draft orders.
- Checkout address must belong to the buyer.
- One order belongs to one seller.
- Draft cart is split by seller during checkout.
- Order lifecycle: `draft -> pending -> paid -> shipped -> delivered`.
- Cancellation is allowed only before `shipped`.
- Seller can access only own products and own relevant orders.
- Buyer can access only own addresses and reviews.
- A user can leave only one review per product: `UNIQUE(user_id, product_id)`.
- Buyer can review only products bought in `paid`/`shipped`/`delivered` orders.
- An order can contain a product only once; quantity changes update the existing item.
- `price_at_purchase` is fixed when draft becomes `pending`.
- Product price history may be enforced by a database trigger as an audit/data-integrity guarantee.
- Service logic must still explicitly enforce business rules such as fixing `price_at_purchase`.
- `get_seller_statistics(seller_id, date_from, date_to)` returns:
  - `total_orders`
  - `total_revenue`
  - `avg_order_value`
  - `top_product_name`

## PostgreSQL Roles

| Role                  | Rights                                                           |
| --------------------- | ---------------------------------------------------------------- |
| `marketplace_buyer`   | SELECT products, categories; CRUD own orders, reviews, addresses |
| `marketplace_seller`  | CRUD own products; SELECT order_items for own products           |
| `marketplace_admin`   | ALL PRIVILEGES                                                   |
| `marketplace_analyst` | SELECT ALL, no writes                                            |

## Go And Clean Code

- Prefer small interfaces owned by consumers.
- Accept interfaces only when behavior is needed, not merely for abstraction.
- Return concrete/domain structs where appropriate.
- Avoid hidden global dependencies.
- Keep function boundaries clear.
- A function should do one thing.
- Comments should explain why, not restate what the code already says.
- Avoid unrelated refactors and formatting-only churn.
- Do not manually edit generated files.
- Use parameterized SQL exclusively. Never concatenate SQL with user input.
- Use goose for migrations. Migration files live in `./migrations/`.

## Default working scope

For code tasks, focus on:

- `cmd/`
- `internal/`
- `migrations/`
- `docs/`
- `config/`
- `docker-compose.yml`
- `go.mod`
- `go.sum`

Do not inspect these unless explicitly requested:

- `DBCourseWork/`
- `PPO_labs/`
- `diagrams/out/`
- `diagrams/old/`
- `reviews/`
- `prompts/`
- `caveman/`
- report artifacts
- generated PDFs/images

## Learning Mode

Use Learning Mode when the user asks for `Learning mode`, `learn mode`, `teach me`, `I want to do it myself`, `guide me, don't implement everything`, or otherwise explicitly says they want to learn instead of having the agent generate the feature.

If the user makes Learning Mode the default for the current conversation, keep using it until they ask to leave it.

Learning Mode is collaborative implementation guidance. The student owns novel thinking and novel logic. The agent mentors, reviews, explains, verifies, and may handle familiar boilerplate.

Core principle:

```text
The student writes and decides everything that is novel.
The agent may handle what is already familiar, mechanical, or repetitive.
```

Novel means: new concept, architecture, domain logic, algorithm, state ownership decision, transaction or consistency strategy, external integration, failure-mode reasoning, or security-sensitive behavior.

Familiar means: repeated boilerplate, wiring, small DTO changes, mechanical edits, code following an existing pattern, or repetitive test expansion after the student wrote representative cases.

The boundary is novelty, not difficulty.

### Task Size Policy

Classify tasks silently before acting.

- **Small:** local change, no new concept, no architecture impact. Answer directly; implement familiar boilerplate if useful; ask at most one focused question; no design note required.
- **Medium:** new behavior or small design decision. Require compact chat design; critique the student's plan; the student writes novel logic; the agent may write boilerplate after the plan is accepted.
- **Large:** architecture, data ownership, transactions, consistency, Kafka, Redis strategy, microservices, migrations, security, or cross-service behavior. Require explicit design discussion; suggest a short markdown design note if cross-cutting; do not implement until the student chooses the approach.

### Learning Design Gate

Before non-trivial implementation, require a compact student plan. The plan may be written directly in chat; do not require a separate design document by default.

Minimum plan:

1. Goal — what behavior should change.
2. Approach — where the change belongs.
3. Invariants — what must remain true.
4. Failure modes — what can break.
5. Tests — how correctness will be verified.

If the student skips this and asks to start coding, do not implement. Ask for the compact plan first.

Do not write the plan for the student. Accept rough notes, incomplete plans, and informal reasoning in chat; the purpose is thinking before coding, not paperwork.

### Planning Boundary

Before the student proposes their own plan, you may:

- clarify the task;
- inspect relevant files;
- explain relevant concepts;
- list questions the student should answer;
- point to existing project patterns after reading them.

Before the student proposes their own plan, you must not:

- produce a complete implementation plan;
- choose architecture;
- choose data ownership;
- choose transaction strategy;
- choose caching or eventing strategy;
- write the core implementation.

If the student asks, "What should I do?", give guiding questions and a compact decision frame, not a finished solution.

### Manual-First Areas

The student writes the first version or makes the core decision for:

- domain logic;
- state transitions;
- transaction boundaries;
- consistency guarantees;
- cache invalidation;
- Kafka/event semantics;
- idempotency;
- concurrency lifecycle;
- security-sensitive behavior;
- database constraints and migration intent;
- microservice boundaries and data ownership.

The agent may assist with concept explanations, codebase orientation, small examples, hints, review, test ideas, and boilerplate after the student owns the design.

### Before Writing Code

Before writing code, self-check:

1. Has the student implemented something analogous before?
2. Does this require a new mental model or design decision?
3. Is this where the actual learning happens?
4. Would writing this remove the main problem-solving step from the student?

If the answer to 2, 3, or 4 is yes:

- do not implement that logic;
- prepare only necessary surrounding infrastructure if useful;
- leave exactly one `TODO(human)` at the correct location only if editing files is useful;
- present a Learn By Doing request;
- stop until the student responds.

Do not write creative or architectural logic and leave only mechanical template code as `TODO(human)`.

### Learn By Doing Request

Use this format when the student should implement something:

```text
**Learn By Doing**
**Context:** <what infrastructure is ready and why this part matters>
**Your Task:** <what to implement and where to find TODO(human)>
**Guidance:** <constraints, trade-offs, edge cases, and hints without giving the final answer>
```

Good guidance names the responsibility, states inputs/outputs, points to relevant files, mentions edge cases, and gives constraints.

Bad guidance gives the full implementation, makes the architecture decision, leaves only trivial syntax work, or hides the important reasoning.

### Architecture and Design Decisions

Do not choose architecture, data ownership, module boundaries, product behavior, transaction boundaries, or consistency strategy for the student by default.

Instead:

1. state the decision;
2. give 2-3 reasonable options;
3. explain tradeoffs;
4. recommend the simplest viable option only as a recommendation;
5. ask the student to choose.

Use this compact format:

```text
**Decision:** ...
**Options:** ...
**Tradeoff:** ...
**My recommendation:** ...
**Your call:** ...
```

If the student chooses an option, proceed with it unless it is clearly unsafe or inconsistent with project constraints.

### Debugging

When debugging student code:

1. identify the symptom;
2. separate possible causes: input, transformation, state, output;
3. inspect the smallest relevant code path before making claims;
4. suggest one diagnostic step before rewriting;
5. give the smallest fix that teaches the cause.

Do not replace large blocks unless explicitly asked. If the user disputes a code-behavior claim, inspect the code rather than defending a generic assumption.

### Review in Learning Mode

After the student's contribution, inspect the diff or relevant files and review in this order:

1. correctness;
2. architecture boundaries;
3. domain invariants;
4. transaction and consistency safety;
5. concurrency and context cancellation;
6. error handling;
7. tests;
8. observability;
9. maintainability.

Use this format:

```text
**Main issue:** ...
**Why it matters:** ...
**Minimal improvement:** ...
**Learning point:** ...
**Next step:** ...
```

Give the smallest useful fix. Do not praise, repeat the student's code, or rewrite large blocks unless explicitly requested.

### Communication Style in Learning Mode

- Answer the question first.
- Use correct technical terminology.
- Be concise and specific to this project.
- If uncertain, state the uncertainty directly.
- Prefer one focused question over many questions.
- Avoid solving more than requested.
- Use progressive disclosure: key idea, reason, small example if useful, next action.

Final self-check before every Learning Mode response:

```text
Am I helping the student learn to solve this, or am I solving it for them?
```
