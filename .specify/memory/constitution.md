<!--
SYNC IMPACT REPORT
==================
Version change: 0.0.0 (uninitialized template) → 1.0.0
Bump rationale: MAJOR — first ratified constitution; replaces all placeholder
tokens with concrete project principles derived from AGENTS.md.

Modified principles:
  - [PRINCIPLE_1_NAME]        → I. Clean Architecture & Dependency Inversion
  - [PRINCIPLE_2_NAME]        → II. Service-Owned Business Logic
  - [PRINCIPLE_3_NAME]        → III. PostgreSQL Persistence & Goose Migrations
  - [PRINCIPLE_4_NAME]        → IV. API & Error Conventions
  - [PRINCIPLE_5_NAME]        → V. Cost-Controlled Verification
  (added) → VI. Generated Code Discipline
  (added) → VII. Backend-Only Working Scope

Added sections:
  - Technology & Scope Constraints (replaces [SECTION_2_NAME])
  - Development Workflow & Quality Gates (replaces [SECTION_3_NAME])

Removed sections:
  - None

Templates requiring updates:
  - ✅ .specify/templates/plan-template.md — Constitution Check gates referenced;
       no edits needed (gates are derived at /speckit-plan time from this file).
  - ✅ .specify/templates/spec-template.md — no constitution-specific tokens.
  - ✅ .specify/templates/tasks-template.md — no constitution-specific tokens.
  - ✅ AGENTS.md (= CLAUDE.md) — amended in same change set to lift code
       generation out of cost-gated verification and forbid reading generated files.

Follow-up TODOs:
  - None.
-->

# Marketplace Constitution

## Core Principles

### I. Clean Architecture & Dependency Inversion (NON-NEGOTIABLE)

Source dependencies MUST flow inward toward `internal/service`. The runtime call
flow is `handler / web / techui -> service -> repository implementation`, but the
source dependency direction is inverted via service-owned interfaces:
`handler / web / techui -> service <- repository implementation`.

Hard rules:

- `internal/service` MUST NOT import `internal/repository`, `pgx`, `pgconn`,
  SQLSTATE values, constraint names, or any repository-specific library.
- `internal/handler` and `internal/web` MUST NOT import `internal/repository`.
- Repository interfaces are owned by the consuming service layer and live in
  `internal/service` unless a narrower consumer package already exists.
- Repository implementations live in `internal/repository` and satisfy
  service-defined interfaces. Compile-time conformance SHOULD be asserted in
  pointer form: `var _ service.ProductRepo = (*ProductRepository)(nil)`.
- Data mapping is explicit: `DB row -> model -> DTO / view data`. DB rows,
  SQL-specific types, HTTP DTOs, and web view data MUST NOT leak across layers.
- Dependencies are passed through constructors. Hidden globals are forbidden.

**Rationale**: This is the project's foundational discipline. It keeps SQL,
HTTP, and template concerns from contaminating business logic, makes the service
layer trivially mockable, and is the only reason the codebase can be tested at
multiple layers without a database round-trip.

### II. Service-Owned Business Logic

Business rules, ownership checks, authorization decisions, order lifecycle
transitions, and orchestration MUST live in `internal/service`. Handlers and web
controllers translate transport; repositories translate storage. Neither decides
policy.

- Handlers MUST NOT contain business logic, access repositories directly,
  perform cache invalidation, or know database details.
- Web handlers MUST NOT import `internal/repository`, contain business
  invariants, or perform cache invalidation.
- Repositories MAY enforce data-integrity constraints and transactions, but MUST
  NOT decide business policy beyond purely persistence-level integrity.
- DB triggers (e.g. price history) are audit/integrity guarantees; the service
  layer MUST still explicitly enforce the corresponding business rule
  (e.g. fixing `price_at_purchase` when a draft becomes `pending`).
- Cache invalidation belongs to service/usecase code or a cache-aware decorator,
  never to handlers. Redis key mechanics live in `internal/cache`.
- Public service methods MUST take `context.Context` as the first argument.

**Rationale**: A single layer owning policy is what makes ownership checks,
order state transitions, and cache invalidation testable and auditable. Letting
handlers or triggers "also" enforce rules duplicates logic and produces drift.

### III. PostgreSQL Persistence & Goose Migrations

PostgreSQL is the system of record. All schema evolution flows through goose.

- Migrations MUST live in `./migrations/` and be authored as goose migrations.
- SQL MUST be parameterized exclusively; SQL string concatenation with user
  input is forbidden.
- Repository code translates known `pgx`/`pgconn` conditions into
  application/domain errors at the repository boundary. Use `pgerrcode.*`
  constants, never raw SQLSTATE strings.
- The four PostgreSQL roles (`marketplace_buyer`, `marketplace_seller`,
  `marketplace_admin`, `marketplace_analyst`) define DB-level least privilege
  and MUST be respected by any new table or operation.
- Repository SQL, transactions, constraints, triggers, and role behavior MUST
  be covered by integration tests against a real database. Shared test
  infrastructure (`TestMain`, connection setup, migrations, truncation helpers)
  lives in `repository/testmain_test.go` and contains no business test cases.
- Repository methods accept/return domain types from `internal/model` or small
  parameter structs — never DB rows or HTTP DTOs.

**Rationale**: Goose gives deterministic, ordered, reversible schema change.
Parameterized SQL is the floor for safety. DB roles enforce a second line of
defense behind service-layer authorization.

### IV. API & Error Conventions

The JSON API and error model are part of the public contract.

- JSON API routes are prefixed with `/api/v1/`.
- Error responses use the exact shape `{"error":"message"}`.
- Use `net/http` status constants; do not write raw HTTP status numbers.
- Centralized service-error → HTTP mapping lives in
  `internal/handler/service_error.go`. Handlers MUST go through it.
- Shared application errors MAY be declared in `internal/service` but MUST be
  database-agnostic. Service and handler code MUST NOT branch on DB-specific
  error types, SQLSTATE codes, or `pgx`/`pgconn` details.
- Wrap errors with `%w`; check with `errors.Is` / `errors.As`. Service code MAY
  treat unknown repository errors as repository/internal failures.
- HTTP DTOs and request validation live in `internal/handler`. Business
  invariants MUST NOT be re-implemented at the DTO layer.

**Rationale**: A single error envelope, a single mapping function, and a single
versioned prefix make the API predictable for clients and keep handler code
mechanical.

### V. Cost-Controlled Verification

Heavy verification is gated on explicit user permission. Lightweight, targeted
checks are the default.

- Ask first before running: full test suites, broad build/syntax checks,
  linters, formatters, dependency downloads, broad diffs or history inspection
  (`git diff`, `git show`, `git log -p`, large patch views), or any command
  expected to produce large output or scan most of the repository when a
  narrower check would do.
- Code generation (e.g. `go generate`, `mockgen`) is NOT cost-gated — run it
  directly when a task needs it.
- Prefer `rg`, short `sed`/`nl` ranges, focused file reads, and
  `git status --short` by default.
- When asking permission, state the exact command, why it is useful, and
  whether it may produce large output or take noticeable time. If permission is
  not granted, continue with targeted checks and clearly flag any remaining
  verification gap.

**Rationale**: Token and time budget is finite. Running the full suite or
piping `git log -p` through the model is rarely the cheapest way to answer the
actual question; targeted checks usually are.

### VI. Generated Code Discipline

Generated artifacts are derived, not authored.

- Generated mocks live in `internal/mocks/service/`. Gomock is the preferred
  generator; hand-written fakes are acceptable when simpler.
- `go:generate mockgen` directives SHOULD live near the service-defined
  interface they describe, or in a dedicated generation file.
- Generated files MUST NOT be edited manually. When the interface changes,
  regenerate.
- Agents MUST NOT read generated files — their contents are mechanically
  derivable from the source interface and reading them wastes context. Inspect
  the interface instead, then regenerate.

**Rationale**: Treating generated code as derived (not source) keeps mocks in
sync with interfaces, prevents drift, and removes a class of pointless reads
from the agent loop.

### VII. Backend-Only Working Scope

This is a Go backend with a server-rendered MPA. Tasks operate within the
backend scope unless the user explicitly broadens it.

- The default working set is `cmd/`, `internal/`, `migrations/`, `docs/`,
  `config/`, `docker-compose.yml`, `go.mod`, `go.sum`.
- Do not inspect `DBCourseWork/`, `PPO_labs/`, `diagrams/out/`, `diagrams/old/`,
  `reviews/`, `prompts/`, `caveman/`, report artifacts, or generated
  PDFs/images unless explicitly requested.
- The web UI is server-rendered through `html/template`. Do NOT introduce
  npm/pnpm/yarn/Vite/React/Vue unless the user explicitly asks. Templates live
  in `internal/web/templates/*.html`; shared CSS in
  `internal/web/static/css/style.css`. Mutations use POST forms with redirect
  after success.

**Rationale**: The repo contains coursework, archived diagrams, and reports
that are not the codebase. Constraining scope keeps work fast and prevents
accidentally treating archival material as live source.

## Technology & Scope Constraints

- Language: Go. Persistence: PostgreSQL via `pgx`. Cache: Redis via
  `internal/cache`. Migrations: goose. Mocks: gomock (preferred).
- External service contracts (e.g. payment gateway) live in `internal/port`;
  implementations in `internal/adapter`. Cross the boundary only through ports.
- Cache keys, TTLs, and invalidation triggers are enumerated in
  [docs/cache.md](../../docs/cache.md) and MUST be kept synchronized with
  `internal/cache` and the invalidating service code. Cache failures MUST NOT
  break core business behavior unless the operation explicitly requires cache
  consistency.
- Authoritative documentation lives under `docs/`. When code and a doc
  conflict, the code wins after direct verification, and the doc MUST be
  updated as part of the same change.

## Development Workflow & Quality Gates

- Service tests are unit tests with test doubles, typically in package
  `service_test`. Use package `service` only when intentionally testing
  unexported helpers.
- Repository tests are integration tests in package `repository_test` against a
  real database, sharing setup from `repository/testmain_test.go`.
- A change that touches a service-defined interface MUST also regenerate the
  corresponding mock; reviewers should treat a stale mock as a defect.
- A change to cache key shape, TTL, or invalidation MUST update both
  `internal/cache` and [docs/cache.md](../../docs/cache.md) in the same commit.
- A change to API routes, status codes, DTO shape, or error envelope MUST also
  update [docs/api-contracts.md](../../docs/api-contracts.md).
- Pre-commit: prefer `git status --short` and targeted `rg` over broad diffs.
  Do not run the full test suite or formatters without permission (see
  Principle V).
- Custom subagent guidance is duplicated in `.claude/agents/*.md` and
  `.codex/agents/*.toml`; both surfaces MUST be kept aligned when agent
  guidance changes.

## Governance

This constitution supersedes ad-hoc conventions and any conflicting guidance
elsewhere in the repository. `AGENTS.md` (symlinked from `CLAUDE.md`) remains
the operational runtime guide for agents; when a conflict arises, the
constitution prevails and `AGENTS.md` MUST be amended to match.

Amendment procedure:

1. Propose the change as an edit to `.specify/memory/constitution.md` together
   with the propagated edits to dependent templates and `AGENTS.md`.
2. Determine the version bump:
   - MAJOR: removed principle, redefined principle in a backward-incompatible
     way, or governance change that invalidates prior compliance reviews.
   - MINOR: new principle or materially expanded principle/section.
   - PATCH: wording, clarification, typo, non-semantic refinement.
3. Update the Sync Impact Report at the top of this file.
4. Bump `Version` and `Last Amended` on the footer line. `Ratified` does not
   change after the initial ratification.

Compliance review: every plan generated by `/speckit-plan` MUST pass a
Constitution Check gate derived from these principles before Phase 0 research,
and re-pass after Phase 1 design. Justified deviations are recorded in the plan
under Complexity Tracking with the simpler alternative explicitly rejected.

**Version**: 1.0.0 | **Ratified**: 2026-05-14 | **Last Amended**: 2026-05-14
