# Tasks: Favorite Products

**Input**: Design documents from `/specs/001-favorite-products/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md

**Tests**: Focused tests are included because this feature changes schema, service behavior, JSON API contracts, and repository constraints. Heavy verification commands still require explicit user permission.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **Commands**: `cmd/api`
- **Domain/service/repository**: `internal/model`, `internal/service`, `internal/repository`
- **JSON API**: `internal/handler`
- **Database/docs/config**: `migrations`, `docs`
- Do not modify `internal/web`, `cmd/techui`, templates, CSS, terminal UI, frontend, mobile, or npm/pnpm/yarn/Vite/React/Vue files.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Create shared backend files for the favorite feature without implementing story behavior yet.

- [X] T001 [P] Create `ProductFavorite` and `FavoriteState` domain types in `internal/model/product_favorite.go`
- [X] T002 [P] Create `FavoriteService` scaffold and service-owned `FavoriteRepo` interface scaffold in `internal/service/favorite_service.go`
- [X] T003 [P] Create `FavoriteRepoImpl` scaffold with constructor and compile-time interface assertion in `internal/repository/favorite_repo.go`
- [X] T004 [P] Create `FavoriteHandler` scaffold with constructor in `internal/handler/favorite.go`
- [X] T005 [P] Create goose migration file with up/down sections in `migrations/013_product_favorites.sql`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Establish schema, shared boundaries, and wiring that all user stories depend on.

**CRITICAL**: No user story work can begin until this phase is complete.

- [X] T006 Complete `product_favorites` table, `(user_id, product_id)` uniqueness, indexes, foreign keys, grants, and rollback in `migrations/013_product_favorites.sql`
- [X] T007 [P] Add favorite operation errors such as create/get/delete/list failures in `internal/service/errors.go`
- [X] T008 [P] Document migration order, favorite table purpose, and PostgreSQL role rights in `docs/database.md`
- [X] T009 [P] Document `product_favorites` fields, constraints, and indexes in `docs/db-schema.md`
- [X] T010 Define `FavoriteRepo` methods for add, remove, list, and actor-state checks in `internal/service/favorite_service.go`
- [X] T011 Implement shared active-product validation helper using `products.deleted_at IS NULL` semantics in `internal/service/favorite_service.go`
- [X] T012 Implement shared row mapping and active-product join helpers in `internal/repository/favorite_repo.go`
- [X] T013 Wire favorite repository, service, and handler construction in `cmd/api/main.go`
- [X] T014 Add `FavoriteHandler` to `NewRouter` dependencies without adding web or tech UI wiring in `internal/handler/router.go`

**Checkpoint**: Schema, domain/service/repository/handler boundaries, and API server wiring are ready for story implementation.

---

## Phase 3: User Story 1 - Save A Product (Priority: P1)

**Goal**: Authenticated users can mark an eligible active product as favorite.

**Independent Test**: With an authenticated user and an active product, perform the favorite action and verify exactly one favorite relationship exists for that user-product pair.

### Tests for User Story 1

> Write these tests before implementation and verify they fail for the missing behavior.

- [X] T015 [P] [US1] Add service tests for add favorite success, product not found, and inactive product rejection in `internal/service/favorite_service_test.go`
- [X] T016 [P] [US1] Add repository integration tests for inserting a favorite for an active product and rejecting missing/deleted products in `internal/repository/favorite_repo_test.go`
- [X] T017 [P] [US1] Add handler tests for authenticated `PUT /api/v1/favorites/{productID}`, missing auth, invalid id, missing product, and inactive product in `internal/handler/favorite_handler_test.go`

### Implementation for User Story 1

- [X] T018 [US1] Implement `FavoriteService.AddFavorite` with actor identity, active-product validation, and repository call in `internal/service/favorite_service.go`
- [X] T019 [US1] Implement favorite insert SQL for active products in `internal/repository/favorite_repo.go`
- [X] T020 [US1] Implement `FavoriteHandler.AddFavorite` response mapping for created favorites and service errors in `internal/handler/favorite.go`
- [X] T021 [US1] Register authenticated `PUT /api/v1/favorites/{productID}` outside role-specific groups in `internal/handler/router.go`
- [X] T022 [US1] Add any new public favorite error status mappings to `internal/handler/service_error.go`

**Checkpoint**: User Story 1 is independently testable with one authenticated user favoriting one active product.

---

## Phase 4: User Story 2 - Avoid Duplicate Favorites (Priority: P1)

**Goal**: Repeated favorite actions on the same product keep one favorite relationship and return a stable result.

**Independent Test**: With an authenticated user and an already favorited product, repeat the favorite action and verify there is still exactly one favorite relationship.

### Tests for User Story 2

> Write these tests before implementation and verify they fail for duplicate handling.

- [X] T023 [P] [US2] Add service tests for repeated `AddFavorite` returning an already-present result without duplicate state in `internal/service/favorite_service_test.go`
- [X] T024 [P] [US2] Add repository integration tests proving duplicate inserts keep one `(user_id, product_id)` row in `internal/repository/favorite_repo_test.go`
- [X] T025 [P] [US2] Add handler tests for repeated `PUT /api/v1/favorites/{productID}` returning `200 OK` after the first create in `internal/handler/favorite_handler_test.go`

### Implementation for User Story 2

- [X] T026 [US2] Implement conflict-tolerant insert behavior with created/already-present result in `internal/repository/favorite_repo.go`
- [X] T027 [US2] Propagate created/already-present result from repository through `FavoriteService.AddFavorite` in `internal/service/favorite_service.go`
- [X] T028 [US2] Return `201 Created` for new favorites and `200 OK` for already-present favorites in `internal/handler/favorite.go`

**Checkpoint**: User Story 1 and User Story 2 both work, and duplicate favorite attempts are idempotent.

---

## Phase 5: User Story 3 - Manage Saved Products (Priority: P2)

**Goal**: Authenticated users can remove favorites, list their favorite products, and check whether a product is favorited by the current actor.

**Independent Test**: After a user has favorites, remove one, list favorites, and check favorite state for favorited and non-favorited products without using web GUI, templates, CSS, or terminal UI.

### Tests for User Story 3

> Write these tests before implementation and verify they fail for missing management behavior.

- [X] T029 [P] [US3] Add service tests for remove favorite, list current actor favorites, and check favorite state in `internal/service/favorite_service_test.go`
- [X] T030 [P] [US3] Add repository integration tests for idempotent delete, newest-first active-only list, and favorite-state checks in `internal/repository/favorite_repo_test.go`
- [X] T031 [P] [US3] Add handler tests for `DELETE /api/v1/favorites/{productID}`, `GET /api/v1/favorites/{productID}`, and `GET /api/v1/favorites` in `internal/handler/favorite_handler_test.go`

### Implementation for User Story 3

- [X] T032 [US3] Implement `DeleteFavorite`, `ListFavoriteProductsByUserID`, and `IsFavorite` SQL behavior in `internal/repository/favorite_repo.go`
- [X] T033 [US3] Implement `RemoveFavorite`, `GetFavoriteProducts`, and `IsProductFavorite` service methods in `internal/service/favorite_service.go`
- [X] T034 [US3] Add `FavoriteStateDTO` and favorite list response mapping in `internal/handler/dto.go`
- [X] T035 [US3] Implement remove, list, and favorite-state handler methods in `internal/handler/favorite.go`
- [X] T036 [US3] Register authenticated `DELETE /api/v1/favorites/{productID}`, `GET /api/v1/favorites/{productID}`, and `GET /api/v1/favorites` routes in `internal/handler/router.go`
- [X] T037 [US3] Document favorite endpoints, auth requirements, status codes, and error responses in `docs/api-contracts.md`

**Checkpoint**: All favorite lifecycle behavior is independently functional through the JSON API.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Final documentation and focused verification after all selected stories are implemented.

- [X] T038 [P] Update `specs/001-favorite-products/quickstart.md` if implemented favorite status codes or response bodies differ from the planned contract
- [X] T039 [P] Confirm no favorite Redis keys were introduced, or document any intentional cache behavior in `docs/cache.md`
- [X] T040 Run focused service tests for `internal/service/favorite_service_test.go` with `go test ./internal/service`
- [X] T041 Run focused handler tests for `internal/handler/favorite_handler_test.go` with `go test ./internal/handler`
- [ ] T042 Run repository integration tests for `internal/repository/favorite_repo_test.go` with `DATABASE_URL="$DATABASE_URL" go test ./internal/repository`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies.
- **Foundational (Phase 2)**: Depends on Setup completion and blocks all user stories.
- **User Story 1 (Phase 3)**: Depends on Foundational; provides core add favorite behavior.
- **User Story 2 (Phase 4)**: Depends on User Story 1; hardens duplicate add behavior required for P1 completeness.
- **User Story 3 (Phase 5)**: Depends on User Stories 1 and 2; adds remove/list/check lifecycle behavior.
- **Polish (Phase 6)**: Depends on all implemented user stories.

### User Story Dependencies

- **US1 - Save A Product**: Can start after Phase 2.
- **US2 - Avoid Duplicate Favorites**: Depends on US1 add-favorite flow.
- **US3 - Manage Saved Products**: Depends on US1 relationship creation and US2 idempotent add semantics.

### Within Each User Story

- Tests first, then implementation.
- Repository/service/handler tests can be drafted in parallel because they target different files.
- Repository behavior before service integration.
- Service behavior before handler response mapping.
- Route registration after handler methods exist.

---

## Parallel Opportunities

- Setup scaffolding tasks T001-T005 can run in parallel.
- Foundational documentation tasks T008-T009 can run in parallel with service/repository scaffolding tasks T007, T010, and T012.
- US1 test tasks T015-T017 can run in parallel.
- US2 test tasks T023-T025 can run in parallel.
- US3 test tasks T029-T031 can run in parallel.
- Final docs checks T038-T039 can run in parallel after implementation.

## Parallel Example: User Story 1

```bash
Task: "Add service tests for add favorite success, product not found, and inactive product rejection in internal/service/favorite_service_test.go"
Task: "Add repository integration tests for inserting a favorite for an active product and rejecting missing/deleted products in internal/repository/favorite_repo_test.go"
Task: "Add handler tests for authenticated PUT /api/v1/favorites/{productID}, missing auth, invalid id, missing product, and inactive product in internal/handler/favorite_handler_test.go"
```

## Parallel Example: User Story 3

```bash
Task: "Implement DeleteFavorite, ListFavoriteProductsByUserID, and IsFavorite SQL behavior in internal/repository/favorite_repo.go"
Task: "Add FavoriteStateDTO and favorite list response mapping in internal/handler/dto.go"
Task: "Document favorite endpoints, auth requirements, status codes, and error responses in docs/api-contracts.md"
```

---

## Implementation Strategy

### MVP First

1. Complete Phase 1 and Phase 2.
2. Complete US1 and US2 because both are P1 and together provide a reliable add-favorite MVP.
3. Stop and validate with focused service, handler, and repository tests for add and duplicate behavior.

### Incremental Delivery

1. Add US1 to create favorite relationships.
2. Add US2 to make repeated favorite actions idempotent.
3. Add US3 to support remove/list/check lifecycle behavior.
4. Update API/database docs and run focused verification.

### Verification Boundaries

- Focused package tests are planned: `go test ./internal/service`, `go test ./internal/handler`, and `DATABASE_URL="$DATABASE_URL" go test ./internal/repository`.
- Full `go test ./...`, broad diffs/history inspection, formatters, linters, code generation, or dependency downloads require explicit user permission.
- Do not edit generated mocks in `internal/mocks/`; use hand-written fakes unless mock generation is explicitly approved.

---

## Notes

- [P] tasks use different files and can run in parallel after their dependencies are met.
- Each user story has independent test criteria and a checkpoint.
- The task list intentionally excludes `internal/web`, `cmd/techui`, templates, CSS, terminal UI, frontend, mobile, and JavaScript tooling.
