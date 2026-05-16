# Implementation Plan: [FEATURE]

**Branch**: `[###-feature-name]` | **Date**: [DATE] | **Spec**: [link]

**Input**: Feature specification from `/specs/[###-feature-name]/spec.md`

**Note**: This template is filled in by the `/speckit-plan` command. See `.specify/templates/plan-template.md` for the execution workflow.

## Summary

[Extract from feature spec: primary requirement + technical approach from research]

## Technical Context

<!--
  ACTION REQUIRED: Replace the content in this section with the technical details
  for the project. The structure here is presented in advisory capacity to guide
  the iteration process.
-->

**Language/Version**: Go 1.24+ or [NEEDS CLARIFICATION if changing runtime]

**Primary Dependencies**: chi, pgx/v5 + pgxpool, squirrel, html/template, go-redis/v9

**Storage**: PostgreSQL 16 via goose migrations; Redis 7 for cache/session behavior

**Testing**: `go test` focused packages; repository integration tests require PostgreSQL

**Target Platform**: Linux server / Docker Compose local environment

**Project Type**: Go backend with JSON API, server-rendered web MPA, and console tech UI

**Performance Goals**: [domain-specific, e.g., 1000 req/s, 10k lines/sec, 60 fps or NEEDS CLARIFICATION]

**Constraints**: Clean Architecture boundaries; backend-owned scope; no SPA/mobile toolchain unless explicitly approved; cost-controlled verification

**Scale/Scope**: [domain-specific, e.g., 10k users, 1M LOC, 50 screens or NEEDS CLARIFICATION]

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- [ ] Clean Architecture: planned code preserves `handler/web/techui -> service <- repository implementation`; service owns repository interfaces; DTOs, DB rows, SQL-specific types, and view data stay in their layers.
- [ ] Backend scope: work stays in `cmd/`, `internal/`, `migrations/`, `docs/`, `config/`, `docker-compose.yml`, `go.mod`, and `go.sum`; no npm/pnpm/yarn/Vite/React/Vue/mobile stack unless explicitly approved and tracked as a violation.
- [ ] PostgreSQL/goose: schema changes use goose migrations in `migrations/`; SQL is parameterized; repository code owns pgx/transactions/error translation; affected DB docs are updated.
- [ ] API/errors: JSON routes stay under `/api/v1/`; error responses use `{"error":"message"}`; HTTP status mapping remains centralized in `internal/handler/service_error.go`.
- [ ] Verification: plan names focused checks or tests and flags any full suite, broad diff/history inspection, formatter, linter, codegen, dependency download, or other heavy command that needs explicit permission.

## Project Structure

### Documentation (this feature)

```text
specs/[###-feature]/
├── plan.md              # This file (/speckit-plan command output)
├── research.md          # Phase 0 output (/speckit-plan command)
├── data-model.md        # Phase 1 output (/speckit-plan command)
├── quickstart.md        # Phase 1 output (/speckit-plan command)
├── contracts/           # Phase 1 output (/speckit-plan command)
└── tasks.md             # Phase 2 output (/speckit-tasks command - NOT created by /speckit-plan)
```

### Source Code (repository root)
<!--
  ACTION REQUIRED: Replace the placeholder tree below with the concrete layout
  for this feature. Delete unused options and expand the chosen structure with
  real paths. The delivered plan must stay within the backend/server-rendered
  MPA scope unless the user explicitly approved a scope expansion.
-->

```text
cmd/
├── api/
├── seed/
└── techui/

internal/
├── model/
├── service/
├── repository/
├── handler/
├── web/
├── cache/
├── middleware/
├── port/
├── adapter/
└── component/

migrations/
docs/
config/
docker-compose.yml
go.mod
go.sum
```

**Structure Decision**: [Document the selected structure and reference the real
directories captured above]

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| [e.g., separate SPA/mobile client] | [current need] | [why server-rendered MPA/API change is insufficient] |
| [e.g., cross-layer dependency] | [specific problem] | [why service-owned interface or DTO mapping is insufficient] |
