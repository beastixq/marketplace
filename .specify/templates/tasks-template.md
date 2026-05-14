---

description: "Task list template for feature implementation"
---

# Tasks: [FEATURE NAME]

**Input**: Design documents from `/specs/[###-feature-name]/`

**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**Tests**: Include focused tests when required by the feature specification, risk,
or Constitution Check. Heavy verification commands still require explicit user
permission.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **Commands**: `cmd/api`, `cmd/seed`, `cmd/techui`
- **Domain/service/repository**: `internal/model`, `internal/service`, `internal/repository`
- **JSON API**: `internal/handler`
- **Server-rendered web MPA**: `internal/web`
- **Cache/middleware/ports/adapters/components**: `internal/cache`, `internal/middleware`, `internal/port`, `internal/adapter`, `internal/component`
- **Database/docs/config**: `migrations`, `docs`, `config`, `docker-compose.yml`
- Do not create `frontend/`, `backend/src/`, mobile app, or npm/pnpm/yarn/Vite/React/Vue structure unless the user explicitly approved a scope expansion in plan.md.

<!--
  ============================================================================
  IMPORTANT: The tasks below are SAMPLE TASKS for illustration purposes only.

  The /speckit-tasks command MUST replace these with actual tasks based on:
  - User stories from spec.md (with their priorities P1, P2, P3...)
  - Feature requirements from plan.md
  - Entities from data-model.md
  - Endpoints from contracts/

  Tasks MUST be organized by user story so each story can be:
  - Implemented independently
  - Tested independently
  - Delivered as an MVP increment

  DO NOT keep these sample tasks in the generated tasks.md file.
  ============================================================================
-->

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic structure

- [ ] T001 Create project structure per implementation plan
- [ ] T002 Confirm affected Go packages, migrations, docs, and config paths
- [ ] T003 [P] Add or update focused test scaffolding only where the plan requires it

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

Examples of foundational tasks (adjust based on your project):

- [ ] T004 Add goose migration in `migrations/` if schema changes are required
- [ ] T005 [P] Define or update service-owned interfaces in `internal/service`
- [ ] T006 [P] Create or update domain models in `internal/model`
- [ ] T007 Implement repository mapping/transactions in `internal/repository`
- [ ] T008 Configure API/web routing and middleware in `internal/handler` or `internal/web`
- [ ] T009 Update centralized service error mapping in `internal/handler/service_error.go`

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - [Title] (Priority: P1) 🎯 MVP

**Goal**: [Brief description of what this story delivers]

**Independent Test**: [How to verify this story works on its own]

### Tests for User Story 1 (include when required by spec, risk, or plan) ⚠️

> **NOTE: When tests are included, write them FIRST and ensure they FAIL before implementation**

- [ ] T010 [P] [US1] Service test for [business rule] in `internal/service/[name]_test.go`
- [ ] T011 [P] [US1] Handler/web/repository test for [boundary behavior] in `internal/[handler|web|repository]/[name]_test.go`

### Implementation for User Story 1

- [ ] T012 [P] [US1] Create or update [Entity1] domain model in `internal/model/[file].go`
- [ ] T013 [P] [US1] Create or update [Entity2] domain model in `internal/model/[file].go`
- [ ] T014 [US1] Implement [Service] use case in `internal/service/[file].go` (depends on T012, T013)
- [ ] T015 [US1] Implement repository behavior in `internal/repository/[file].go`
- [ ] T016 [US1] Implement API handler or web controller in `internal/[handler|web]/[file].go`
- [ ] T017 [US1] Add validation, centralized error mapping, and docs updates

**Checkpoint**: At this point, User Story 1 should be fully functional and testable independently

---

## Phase 4: User Story 2 - [Title] (Priority: P2)

**Goal**: [Brief description of what this story delivers]

**Independent Test**: [How to verify this story works on its own]

### Tests for User Story 2 (include when required by spec, risk, or plan) ⚠️

- [ ] T018 [P] [US2] Service test for [business rule] in `internal/service/[name]_test.go`
- [ ] T019 [P] [US2] Handler/web/repository test for [boundary behavior] in `internal/[handler|web|repository]/[name]_test.go`

### Implementation for User Story 2

- [ ] T020 [P] [US2] Create or update [Entity] model in `internal/model/[file].go`
- [ ] T021 [US2] Implement [Service] in `internal/service/[file].go`
- [ ] T022 [US2] Implement API/web/repository behavior in `internal/[handler|web|repository]/[file].go`
- [ ] T023 [US2] Integrate with User Story 1 components (if needed)

**Checkpoint**: At this point, User Stories 1 AND 2 should both work independently

---

## Phase 5: User Story 3 - [Title] (Priority: P3)

**Goal**: [Brief description of what this story delivers]

**Independent Test**: [How to verify this story works on its own]

### Tests for User Story 3 (include when required by spec, risk, or plan) ⚠️

- [ ] T024 [P] [US3] Service test for [business rule] in `internal/service/[name]_test.go`
- [ ] T025 [P] [US3] Handler/web/repository test for [boundary behavior] in `internal/[handler|web|repository]/[name]_test.go`

### Implementation for User Story 3

- [ ] T026 [P] [US3] Create or update [Entity] model in `internal/model/[file].go`
- [ ] T027 [US3] Implement [Service] in `internal/service/[file].go`
- [ ] T028 [US3] Implement API/web/repository behavior in `internal/[handler|web|repository]/[file].go`

**Checkpoint**: All user stories should now be independently functional

---

[Add more user story phases as needed, following the same pattern]

---

## Phase N: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [ ] TXXX [P] Documentation updates in docs/
- [ ] TXXX Code cleanup and refactoring
- [ ] TXXX Performance optimization across all stories
- [ ] TXXX [P] Additional focused tests in affected `internal/...` packages
- [ ] TXXX Security hardening
- [ ] TXXX Run quickstart.md validation

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3+)**: All depend on Foundational phase completion
  - User stories can then proceed in parallel (if staffed)
  - Or sequentially in priority order (P1 → P2 → P3)
- **Polish (Final Phase)**: Depends on all desired user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 2 (P2)**: Can start after Foundational (Phase 2) - May integrate with US1 but should be independently testable
- **User Story 3 (P3)**: Can start after Foundational (Phase 2) - May integrate with US1/US2 but should be independently testable

### Within Each User Story

- Tests (if included) MUST be written and FAIL before implementation
- Models before services
- Services before endpoints
- Core implementation before integration
- Story complete before moving to next priority

### Parallel Opportunities

- All Setup tasks marked [P] can run in parallel
- All Foundational tasks marked [P] can run in parallel (within Phase 2)
- Once Foundational phase completes, all user stories can start in parallel (if team capacity allows)
- All tests for a user story marked [P] can run in parallel
- Models within a story marked [P] can run in parallel
- Different user stories can be worked on in parallel by different team members

---

## Parallel Example: User Story 1

```bash
# Launch all focused tests for User Story 1 together (if included):
Task: "Service test for [business rule] in internal/service/[name]_test.go"
Task: "Handler/web/repository test for [boundary behavior] in internal/[handler|web|repository]/[name]_test.go"

# Launch independent model updates for User Story 1 together:
Task: "Create or update [Entity1] model in internal/model/[file].go"
Task: "Create or update [Entity2] model in internal/model/[file].go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL - blocks all stories)
3. Complete Phase 3: User Story 1
4. **STOP and VALIDATE**: Test User Story 1 independently
5. Deploy/demo if ready

### Incremental Delivery

1. Complete Setup + Foundational → Foundation ready
2. Add User Story 1 → Test independently → Deploy/Demo (MVP!)
3. Add User Story 2 → Test independently → Deploy/Demo
4. Add User Story 3 → Test independently → Deploy/Demo
5. Each story adds value without breaking previous stories

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup + Foundational together
2. Once Foundational is done:
   - Developer A: User Story 1
   - Developer B: User Story 2
   - Developer C: User Story 3
3. Stories complete and integrate independently

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Verify tests fail before implementing
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Avoid: vague tasks, same file conflicts, cross-story dependencies that break independence
