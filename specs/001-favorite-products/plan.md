# Implementation Plan: Favorite Products

**Branch**: `001-favorite-products` | **Date**: 2026-05-14 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/001-favorite-products/spec.md`

**Note**: This template is filled in by the `/speckit-plan` command. See `.specify/templates/plan-template.md` for the execution workflow.

## Summary

Add backend-only product favorites for authenticated users. The feature will
let any authenticated actor add a product favorite, remove it, list their own
favorite products, and check whether a product is favorited by the current
actor. The design stays in the JSON API, service/repository/database/docs
surface and explicitly excludes web GUI, templates, CSS, and terminal UI.

The implementation should introduce a product favorite relationship persisted
in PostgreSQL, enforce "active and visible" eligibility using the current
product soft-delete boundary (`products.deleted_at IS NULL`), and expose
authenticated `/api/v1` routes that do not require role-specific middleware.

## Technical Context

**Language/Version**: Go 1.24.x

**Primary Dependencies**: chi, pgx/v5 + pgxpool, squirrel, html/template, go-redis/v9

**Storage**: PostgreSQL 16 via goose migrations; Redis 7 remains only the existing product lookup cache

**Testing**: Focused `go test` packages: `./internal/service`, `./internal/handler`, and `./internal/repository` when `DATABASE_URL` is set

**Target Platform**: Linux server / Docker Compose local environment

**Project Type**: Go backend with JSON API, server-rendered web MPA, and console tech UI

**Performance Goals**: Favorite add/remove/check complete in one backend interaction; list favorites supports existing pagination conventions for bounded result sizes

**Constraints**: Clean Architecture boundaries; backend-only scope; no web GUI, templates, CSS, terminal UI, SPA, mobile, or npm/pnpm/yarn/Vite/React/Vue work; cost-controlled verification

**Scale/Scope**: One authenticated user can favorite many products, one product can be favorited by many users, and each user-product pair is unique

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- [x] Clean Architecture: planned code preserves `handler -> service <- repository implementation`; a new service-owned favorite repository interface will live in `internal/service`, and repository code will map SQL rows to domain models.
- [x] Backend scope: work is limited to `cmd/`, `internal/`, `migrations/`, `docs/`, `go.mod`/`go.sum` only if needed; this feature will not touch web GUI, templates, CSS, or terminal UI.
- [x] PostgreSQL/goose: schema changes will use one new goose migration in `migrations/`; SQL will be parameterized; repository code will own pgx/squirrel SQL and database error translation; `docs/database.md` and `docs/db-schema.md` must be updated during implementation.
- [x] API/errors: JSON routes stay under `/api/v1/`; error responses stay `{"error":"message"}`; any new public service error mapping must be added in `internal/handler/service_error.go`.
- [x] Verification: plan names focused checks only. Full test suites, broad diffs/history inspection, formatters, linters, code generation, and dependency downloads require explicit permission.

Post-design re-check: PASS. The generated data model, API contract, and quickstart stay within the same boundaries and introduce no justified violations.

## Project Structure

### Documentation (this feature)

```text
specs/001-favorite-products/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── favorites.openapi.yaml
└── checklists/
    └── requirements.md
```

### Source Code (repository root)

```text
cmd/
└── api/                         # wire repository, service, and handler

internal/
├── model/                       # FavoriteState domain model
├── service/                     # FavoriteService and service-owned FavoriteRepo interface
├── repository/                  # PostgreSQL favorite repository implementation
└── handler/                     # JSON favorite handler, DTOs, route registration, error mapping

migrations/
└── 013_product_favorites.sql     # product_favorites relationship table

docs/
├── api-contracts.md             # favorite routes, status codes, DTO/error behavior
├── database.md                  # migration order and constraints
└── db-schema.md                 # product_favorites table and indexes
```

**Structure Decision**: Implement favorites as a backend-only product feature
with a new service and repository boundary. Do not modify `internal/web`,
`cmd/techui`, templates, CSS, or terminal UI.

## Complexity Tracking

No Constitution Check violations.
