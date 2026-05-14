# Implementation Plan: Favorite Products

**Branch**: `002-favorite-products` | **Date**: 2026-05-14 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/002-favorite-products/spec.md`

## Summary

Add a per-user favorites primitive on top of the existing catalog: any
authenticated user (buyer, seller, admin) can mark products as favorites,
list their own favorites paginated newest-first, remove favorites
idempotently, and check whether a specific product is currently favorited.
Backend-only — JSON API under `/api/v1/favorites/`, service + repository +
goose migration + integration tests + docs updates. No web UI, no terminal
UI, no notifications, no derived feeds.

Persistence is a `favorites (user_id, product_id, created_at)` join table
with a compound primary key and `ON DELETE CASCADE` to both `users` and
`products`. The schema models the resolved Q1/Q2/Q3 decisions exactly:
roles are not restricted, cascade-delete keeps the list coherent, and no
derived behaviors ride along.

## Technical Context

**Language/Version**: Go (project go.mod toolchain — current repo default).

**Primary Dependencies**: `pgx/v5` (PostgreSQL driver, already wired),
`chi` router (existing), `gomock` for service-side mocks, `goose` for
migrations. No new third-party dependencies.

**Storage**: PostgreSQL. New table `favorites`. No Redis cache for this
feature in v1 — see research.md "no-cache" decision.

**Testing**: Unit tests for the service layer with `gomock` (package
`service_test`); integration tests for the repository with a real
PostgreSQL instance under `repository/testmain_test.go` shared setup.
Handler tests follow the existing pattern (e.g. `internal/handler/review.go`
neighborhood — handler-level test coverage if and where the rest of the
package has it; otherwise the contract is validated by service + repo
tests plus quickstart.md curls).

**Target Platform**: Linux server (the existing `cmd/api` binary).

**Project Type**: Web service (Go MPA + JSON API). This feature touches the
JSON API only.

**Performance Goals**: A favorites list of up to 100 entries returns within
1 second under normal load (matches SC-003). Index `(user_id, created_at
desc)` makes pagination on the natural ordering an index scan.

**Constraints**:
- Backend-only scope: NO changes under `internal/web/`, NO templates,
  NO CSS, NO terminal UI (`cmd/techui`).
- Must respect Clean Architecture (Constitution I, II) — service-owned
  repo interface, no `pgx` or SQLSTATE leaking out of `internal/repository`.
- Must use goose migrations and parameterized SQL only (Constitution III).
- Error envelope and `/api/v1/` prefix are fixed (Constitution IV).
- Code generation (mockgen) runs directly, no permission prompt
  (Constitution V).

**Scale/Scope**:
- Expected typical favorite count per user: dozens to low hundreds.
- Edge case: low thousands. Index design must remain valid at that scale.
- Total feature surface: ~1 migration, 1 service file + test, 1 repo file +
  test, 1 handler file, ~4 routes, ~3 doc updates.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
| --- | --- | --- |
| I. Clean Architecture & Dependency Inversion | PASS | New `service.FavoriteRepo` interface owned in `internal/service`; impl in `internal/repository`. Handlers go through service. No cross-imports. |
| II. Service-Owned Business Logic | PASS | Auth-scope check ("caller may only operate on their own favorites"), idempotency on add/remove, and pagination clamping all live in `favorite_service.go`. Handler stays mechanical; repo stays mechanical SQL. |
| III. PostgreSQL Persistence & Goose Migrations | PASS | Goose migration `013_create_favorites.sql` with parameterized SQL, compound PK, two `ON DELETE CASCADE` FKs, and explicit role grants for `marketplace_buyer`, `marketplace_seller`, `marketplace_admin`. Integration tests cover the SQL, cascades, and grants. |
| IV. API & Error Conventions | PASS | All routes under `/api/v1/favorites`. Status codes follow [docs/api-contracts.md](../../docs/api-contracts.md): 200/201/204/401/404/409. Service errors mapped through `internal/handler/service_error.go`. |
| V. Cost-Controlled Verification | PASS | Plan adds targeted tests, no broad refactors. Generation (`mockgen` for `FavoriteRepo`) runs directly. |
| VI. Generated Code Discipline | PASS | Mock for `service.FavoriteRepo` generated to `internal/mocks/service/`. Agents do not read generated file; regenerate when interface changes. |
| VII. Backend-Only Working Scope | PASS | Scope is confined to `migrations/`, `internal/service/`, `internal/repository/`, `internal/handler/`, `internal/model/`, `internal/mocks/service/`, `cmd/api/router wiring`, and `docs/`. No `internal/web/` or `cmd/techui/` changes. |

**Gate verdict (pre-design)**: PASS — no Constitution Check violations.

**Post-design re-check (after Phase 1 artifacts written)**: PASS,
unchanged. Specifically:

- `research.md` introduced no new external dependencies, no new packages,
  and no new layers — every decision lands inside existing boundaries.
- `data-model.md` keeps DB rows behind the repository; the `model.Favorite`
  shape exposes a domain `Product` (already a domain type) and a
  `time.Time`, not a pgx row.
- `contracts/favorites-api.md` keeps all status codes within the project's
  established envelope and routes them under `/api/v1/`.
- `quickstart.md` is documentation only.

Complexity Tracking table below is intentionally empty.

## Project Structure

### Documentation (this feature)

```text
specs/002-favorite-products/
├── plan.md                # This file (/speckit-plan output)
├── spec.md                # Feature spec (input)
├── research.md            # Phase 0 output (decisions + rationale)
├── data-model.md          # Phase 1 output (Favorite entity + relations)
├── quickstart.md          # Phase 1 output (manual curl smoke test)
├── contracts/
│   └── favorites-api.md   # Phase 1 output (route-by-route contract)
└── checklists/
    └── requirements.md    # Created by /speckit-specify
```

### Source Code (repository root)

This feature adds files alongside the existing layered layout. No directory
moves, no renames.

```text
migrations/
└── 013_create_favorites.sql           # NEW — favorites table + indexes + role grants

internal/model/
└── favorite.go                        # NEW — domain Favorite struct (Product summary + AddedAt)

internal/service/
├── favorite_service.go                # NEW — service + service.FavoriteRepo interface + mockgen directive
└── favorite_service_test.go           # NEW — unit tests with gomock

internal/repository/
├── favorite_repo.go                   # NEW — pgx implementation of FavoriteRepo
└── favorite_repo_test.go              # NEW — integration tests (cascades, grants, uniqueness)

internal/handler/
└── favorite.go                        # NEW — HTTP handlers, request/response DTOs, route wiring

internal/handler/router.go             # MODIFIED — mount /api/v1/favorites/* routes (auth-required)

internal/mocks/service/
├── mock_favorite_repo.go              # GENERATED via go:generate mockgen — DO NOT EDIT, DO NOT READ
└── mock_favorite_product_getter.go    # GENERATED via go:generate mockgen — DO NOT EDIT, DO NOT READ

cmd/api/main.go                        # MODIFIED — wire FavoriteService into the component graph

docs/
├── api-contracts.md                   # MODIFIED — add favorites routes
├── database.md                        # MODIFIED — add favorites table to schema overview
├── db-schema.md                       # MODIFIED — append favorites table description
└── project-map.md                     # MODIFIED — note the new files
```

**Structure Decision**: Reuse the existing `internal/{model,service,
repository,handler}` slice — favorites is structurally identical to
reviews and addresses (one buyer-owned resource with create/delete and a
list scoped to the current user), so it slots into the established layout
without new packages. The join-table pattern mirrors `product_categories`
in migration 001 (compound PK + double cascade), so the migration shape is
also already in the codebase.

## Complexity Tracking

> No Constitution Check violations to justify.

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| _none_    | —          | —                                    |
