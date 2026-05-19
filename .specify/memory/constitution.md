<!--
Sync Impact Report
Version change: template -> 1.0.0
Modified principles:
- PRINCIPLE_1_NAME placeholder -> I. Clean Architecture Boundaries
- PRINCIPLE_2_NAME placeholder -> II. Backend-Owned Scope Discipline
- PRINCIPLE_3_NAME placeholder -> III. PostgreSQL And Goose Schema Discipline
- PRINCIPLE_4_NAME placeholder -> IV. API And Error Contract Consistency
- PRINCIPLE_5_NAME placeholder -> V. Cost-Controlled Verification
Added sections:
- Technology And Scope Constraints
- Development Workflow And Review Gates
Removed sections:
- None
Templates requiring updates:
- ✅ .specify/templates/plan-template.md
- ✅ .specify/templates/spec-template.md
- ✅ .specify/templates/tasks-template.md
- ✅ .specify/templates/commands/*.md (no command templates present)
- ✅ README.md, AGENTS.md, docs/*.md reviewed; no runtime guidance changes required
Follow-up TODOs:
- None
-->
# Marketplace Constitution

## Core Principles

### I. Clean Architecture Boundaries
Runtime flow is `handler / web / techui -> service -> repository implementation`.
Source dependencies MUST remain `handler / web / techui -> service <- repository
implementation`. The service layer MUST own repository interfaces, accept
dependencies through constructors, and avoid imports of `internal/repository`,
`pgx`, `pgconn`, SQLSTATE values, constraint names, and repository-specific
libraries. Handlers and web controllers MUST NOT import repositories or own
business policy. Data mapping MUST stay explicit as `DB row -> model -> DTO /
view data`; SQL-specific types, HTTP DTOs, and web view data MUST NOT cross
layer boundaries.

Rationale: the project depends on testable business use cases and clear
ownership of persistence, transport, and domain behavior.

### II. Backend-Owned Scope Discipline
Feature work MUST stay inside the Go backend, server-rendered web MPA,
technology UI, configuration, migrations, and documentation unless the user
explicitly requests a scope expansion. The web surface MUST remain a
server-rendered MPA using `html/template`, `internal/web/templates/*.html`,
`internal/web/static/css/style.css`, `internal/web/web_handler.go`, and
`internal/web/router.go`. The project MUST NOT add npm, pnpm, yarn, Vite,
React, Vue, or a separate frontend/mobile application unless that expansion is
approved and recorded as a Constitution Check violation with a migration plan.
Mutations in the web UI MUST use POST forms and redirect after success.

Rationale: this coursework project demonstrates a Go marketplace backend with
an integrated server-rendered interface, not a separate SPA or mobile stack.

### III. PostgreSQL And Goose Schema Discipline
PostgreSQL is the source of truth. Schema changes MUST be implemented as goose
migrations in `migrations/`, and SQL MUST be parameterized. Repository code MUST
own SQL, transactions, pgx/pgxpool usage, DB-to-domain mapping, and database
error translation. Service code MUST explicitly enforce business rules even
when the database also has constraints, triggers, roles, or functions. Changes
to schema, roles, triggers, or functions MUST update `docs/database.md` and
`docs/db-schema.md` when those documents are affected.

Rationale: database integrity is central to the marketplace domain, but
persistence details must remain behind repository boundaries.

### IV. API And Error Contract Consistency
JSON API routes MUST live under `/api/v1/`. Request DTOs and response DTOs MUST
live at the handler boundary, request validation MUST happen in handlers, and
business invariants MUST remain in services. Error responses MUST use
`{"error":"message"}`. HTTP status mapping for service/application errors MUST
stay centralized in `internal/handler/service_error.go`. Code MUST wrap errors
with `%w`, check them with `errors.Is` or `errors.As`, use `pgerrcode.*` instead
of raw SQLSTATE strings in repository code, and use `net/http` constants instead
of raw status codes.

Rationale: stable public API behavior and database-agnostic application errors
keep clients, tests, and layer boundaries predictable.

### V. Cost-Controlled Verification
Agents and contributors MUST NOT run heavy verification, broad inspection, full
test suites, broad build/syntax checks, linters, formatters, code generation,
dependency downloads, broad diffs, or history inspection without explicit user
permission. Work MUST prefer lightweight targeted checks such as `rg`, focused
file reads, narrow `git status --short`, and package-specific tests when tests
are needed. Verification scope MUST scale with risk: service business rules use
unit tests with doubles or gomock, repository SQL/constraint behavior uses real
PostgreSQL integration tests, and handler/web behavior uses transport/template
tests. Generated mocks in `internal/mocks/` MUST NOT be edited manually.

Rationale: the project balances correctness with coursework iteration speed and
bounded agent cost.

## Technology And Scope Constraints

The default working scope for code changes is `cmd/`, `internal/`, `migrations/`,
`docs/`, `config/`, `docker-compose.yml`, `go.mod`, and `go.sum`. The application
entry points are `cmd/api`, `cmd/seed`, and `cmd/techui`. Domain models live in
`internal/model`; services, repository interfaces, and application errors live
in `internal/service`; repository implementations live in `internal/repository`;
JSON handlers live in `internal/handler`; server-rendered web controllers and
templates live in `internal/web`; cache mechanics live in `internal/cache`.

Cache key mechanics and invalidation MUST NOT be performed by handlers. Cache
behavior belongs behind cache abstractions or service decorators, and
`docs/cache.md` MUST stay synchronized with implemented keys. Business rules
such as cart lifecycle, checkout ownership, stock reservation, order status
transitions, review eligibility, and `price_at_purchase` fixing belong in the
service layer.

Do not inspect excluded coursework artifacts, generated PDFs/images, old
diagrams, reviews, prompts, or unrelated report artifacts unless the user
explicitly asks.

## Development Workflow And Review Gates

Project docs are the first stop for task-specific context. If docs and current
code conflict, contributors MUST verify the current code directly, trust the
code for implementation decisions, and update the affected docs as part of the
change. Plans MUST complete the Constitution Check before Phase 0 research and
re-check it after Phase 1 design. Specs MUST state scope boundaries, affected
layers, API/schema impact, and verification expectations. Tasks MUST include
exact file paths, preserve story-level independence where applicable, and add
documentation or focused tests when contracts, architecture boundaries, setup,
migrations, cache behavior, or test workflow change.

When using project subagents, work MUST be delegated only after the user
explicitly asks for agents or parallel agent work. Delegated tasks MUST be
scoped, must follow this constitution and `AGENTS.md`, and must respect the same
cost-controlled verification policy.

## Governance

This constitution is the Spec Kit planning gate for the marketplace project and
derives from `AGENTS.md`, the canonical runtime agent guidance. Generated specs,
plans, and tasks MUST comply with this constitution. If this constitution and
`AGENTS.md` diverge, the change MUST update whichever artifact is stale before
new implementation work proceeds.

Amendments MUST:

- Update this file with a Sync Impact Report.
- Propagate changes to affected Spec Kit templates and runtime guidance docs.
- Record the semantic version bump rationale.
- Use targeted validation unless the user explicitly approves heavier checks.

Versioning follows semantic versioning:

- MAJOR for incompatible principle removals or governance redefinitions.
- MINOR for new principles, new sections, or materially expanded guidance.
- PATCH for clarifications, wording fixes, and non-semantic refinements.

Compliance review is required during planning, task generation, code review, and
release checks. Any intentional violation MUST be listed in the plan's
Complexity Tracking table with the reason, rejected simpler alternative, and
follow-up mitigation.

**Version**: 1.0.0 | **Ratified**: 2026-05-14 | **Last Amended**: 2026-05-14
