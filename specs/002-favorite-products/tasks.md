---

description: "Task list for feature 002-favorite-products"
---

# Tasks: Favorite Products

**Input**: Design documents from `/specs/002-favorite-products/`

**Prerequisites**: `plan.md`, `spec.md` (required); `research.md`, `data-model.md`, `contracts/favorites-api.md`, `quickstart.md` (all present)

**Tests**: Required. The spec's measurable outcomes (SC-002, SC-004, SC-005) explicitly demand integration tests; Constitution III requires integration tests for repository SQL/cascades/role grants; Constitution Service Layer rule requires gomock-based unit tests for the service.

**Organization**: Tasks are grouped by user story. US1, US2, US3 (all P1) together form the MVP. US4 (P2) and US5 (P3) add the "is favorited?" probe and explicit cascade coverage.

## Format

`[ ] [TaskID] [P?] [Story?] Description with file path`

- **[P]**: Parallelizable — different files, no dependency on incomplete tasks in the same phase.
- **[USx]**: Required for user-story phase tasks only; setup, foundational, and polish phases carry no story label.

## Path Conventions (repo-rooted)

- Migrations: `migrations/`
- Models: `internal/model/`
- Service: `internal/service/`
- Repository: `internal/repository/`
- Handler / router: `internal/handler/`
- Generated mocks: `internal/mocks/service/` (NEVER edited or read; regenerated)
- API wiring: `cmd/api/main.go`
- Docs: `docs/`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Bring the new files into existence so subsequent phases can target real paths.

- [X] T001 [P] Create empty file shells with package declarations for the new files: `migrations/013_create_favorites.sql` (empty), `internal/model/favorite.go` (package model), `internal/service/favorite_service.go` (package service), `internal/service/favorite_service_test.go` (package service_test), `internal/repository/favorite_repo.go` (package repository), `internal/repository/favorite_repo_test.go` (package repository_test), `internal/handler/favorite.go` (package handler).

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Land schema, domain model, repo interface, mocks, and repo scaffold so every user story can implement its slice. **No user-story work may begin until this phase is complete.**

- [X] T002 Write goose migration `migrations/013_create_favorites.sql`: table `favorites(user_id bigint NOT NULL, product_id bigint NOT NULL, created_at timestamptz NOT NULL DEFAULT now())` with compound PK `(user_id, product_id)`, two `ON DELETE CASCADE` FKs to `users(id)` and `products(id)`, index `idx_favorites_user_created_at_desc ON favorites(user_id, created_at DESC)`, and role grants per `data-model.md` (`SELECT, INSERT, DELETE` to `marketplace_buyer`, `marketplace_seller`, `marketplace_admin`; `SELECT` to `marketplace_analyst`). Include a `-- +goose Down` section that drops the index and the table.
- [X] T003 [P] Implement domain model `internal/model/favorite.go`: `Favorite struct { Product model.Product; AddedAt time.Time }`. No DB-row leakage, no DTO fields.
- [X] T004 Define service-owned interfaces and constructor in `internal/service/favorite_service.go`. `FavoriteRepo` interface with method signatures `Add(ctx, userID, productID) (created bool, err error)`, `Remove(ctx, userID, productID) error`, `List(ctx, userID, offset, limit int) (items []model.Favorite, total int, err error)`, `Exists(ctx, userID, productID) (favorited bool, addedAt *time.Time, err error)`. `FavoriteProductGetter` interface with `GetProductByID(ctx, id int64) (model.Product, error)` — same narrow shape that `ReviewProductGetter` already uses in `internal/service/review_service.go`. `FavoriteService` struct with `NewFavoriteService(repo FavoriteRepo, productGetter FavoriteProductGetter) FavoriteService`. Add two `go:generate` directives matching the existing project convention (one per interface, in this exact form): `//go:generate mockgen -package mock_service -destination ../mocks/service/mock_favorite_repo.go github.com/beastixq/marketplace/internal/service FavoriteRepo` and `//go:generate mockgen -package mock_service -destination ../mocks/service/mock_favorite_product_getter.go github.com/beastixq/marketplace/internal/service FavoriteProductGetter`. Append service-level error `ErrFavoriteProductNotFound` to `internal/service/errors.go`. Methods remain stubs returning `errors.New("not implemented")` for now.
- [X] T005 Run `go generate ./internal/service/...` to (re)produce `internal/mocks/service/mock_favorite_repo.go` and `internal/mocks/service/mock_favorite_product_getter.go`. Do not hand-edit or open these files afterwards. (Note: a stale `mock_favorite_repo.go` may already exist from a prior attempt; regeneration overwrites it.)
- [X] T006 Scaffold repository implementation in `internal/repository/favorite_repo.go`: define `FavoritePgxRepo` struct with a `*pgxpool.Pool` (matching the existing convention in this package), a `New...` constructor, and a compile-time assertion `var _ service.FavoriteRepo = (*FavoritePgxRepo)(nil)`. Method bodies remain stubs that return `errors.New("not implemented")` and will be filled in per story.
- [X] T007 Add a service-error → HTTP-status mapping for `ErrFavoriteProductNotFound` → `404` in `internal/handler/service_error.go`. (Centralized mapping rule from Constitution IV.)

**Checkpoint**: Migration applied, mocks exist, interface fixed. User stories can now proceed.

---

## Phase 3: User Story 1 — Add a product to my favorites (Priority: P1) 🎯 MVP

**Goal**: Authenticated callers can `PUT /api/v1/favorites/{productId}` to mark a visible product as favorited. Add is idempotent (FR-002): repeat calls return 200 instead of 201 and never produce duplicate rows.

**Independent Test**: After this phase, the quickstart sections "1. Add to favorites (first time → 201)" and "2. Add again (idempotent → 200)" pass end-to-end, and `SELECT count(*) FROM favorites WHERE (user_id,product_id)=(…,…)` returns `1` regardless of how many times the call is retried.

### Implementation for User Story 1

- [X] T008 [US1] Implement `Add(ctx, userID, productID)` in `internal/repository/favorite_repo.go`: `INSERT INTO favorites(user_id, product_id) VALUES($1, $2) ON CONFLICT (user_id, product_id) DO NOTHING`; return `created=true` iff `RowsAffected()==1`, `created=false` on conflict. Translate `pgerrcode.ForeignKeyViolation` on `fk_favorites_product_id` into a sentinel that the service can recognize (e.g., a package-internal error wrapped to a service-level `service.ErrFavoriteProductNotFound` at the boundary).
- [X] T009 [US1] [P] Repository integration tests in `internal/repository/favorite_repo_test.go`: `TestFavoriteRepo_Add_Inserts`, `TestFavoriteRepo_Add_IdempotentOnConflict`, `TestFavoriteRepo_Add_ProductFKViolation`, `TestFavoriteRepo_Add_RoleGrant_Buyer/Seller/Admin` (run as each `marketplace_*` role and assert INSERT is permitted). Reuse the shared `repository/testmain_test.go` setup.
- [X] T010 [US1] Implement `Add(ctx, userID, productID) (created bool, err error)` in `internal/service/favorite_service.go`: call `productGetter.GetProductByID(ctx, productID)` first — on not-found return `service.ErrFavoriteProductNotFound`. Then call `repo.Add(...)` and surface its `created` flag. Map repo FK signal → `service.ErrFavoriteProductNotFound` as a safety net for race conditions.
- [X] T011 [US1] [P] Service unit tests in `internal/service/favorite_service_test.go` (package `service_test`) using `internal/mocks/service`: `TestFavoriteService_Add_Success_Created`, `TestFavoriteService_Add_Idempotent_AlreadyExists`, `TestFavoriteService_Add_ProductNotVisible`. Assert that on visibility failure the repo is **not** called.
- [X] T012 [US1] Implement HTTP handler in `internal/handler/favorite.go`: `func (h *FavoriteHandler) Put(w, r)` reads `productId` from the URL, reads caller `user_id` from the actor context (see `internal/handler/actor.go`), calls `service.Add`, returns `201 Created` on `created=true`, `200 OK` on `created=false`, no body. Map errors via `service_error.go`.
- [X] T013 [US1] Wire feature into the composition root: in `cmd/api/main.go` construct `FavoriteService` (with the existing product service / lookup and the new repo) and `FavoriteHandler`; in `internal/handler/router.go` mount `PUT /api/v1/favorites/{productId}` behind the existing auth middleware (no role gate — any authenticated user, per Q1).

**Checkpoint**: US1 is independently shippable. A bearer-token caller can favorite a product, retries are safe, and FK + visibility errors surface as 404.

---

## Phase 4: User Story 2 — View my favorites (Priority: P1)

**Goal**: Authenticated callers can `GET /api/v1/favorites?page=&page_size=` and receive their own favorites ordered newest-first, paginated, with current product fields (FR-009).

**Independent Test**: Quickstart section "3. List my favorites" returns the previously-added product with current price/availability and the cross-user isolation case (edge case D) holds: user B's list never reflects user A's adds.

### Implementation for User Story 2

- [X] T014 [US2] Implement `List(ctx, userID, offset, limit) (items []model.Favorite, total int, err error)` in `internal/repository/favorite_repo.go`: one query joining `favorites` with `products` to assemble each `Favorite{Product, AddedAt}` with current product fields, `WHERE user_id=$1 ORDER BY created_at DESC, product_id ASC LIMIT $2 OFFSET $3`; a second query (or windowed count) for `total`. Map DB rows → `model.Product` → `model.Favorite` explicitly; do not return DB row types.
- [X] T015 [US2] [P] Repository integration tests in `internal/repository/favorite_repo_test.go`: `TestFavoriteRepo_List_OrderedNewestFirst`, `TestFavoriteRepo_List_PaginationOffsetLimit`, `TestFavoriteRepo_List_CrossUserIsolation`, `TestFavoriteRepo_List_EmptyForUnusedUser`.
- [X] T016 [US2] Implement `List(ctx, userID, page, pageSize int) (items, total, err)` in `internal/service/favorite_service.go`: clamp `page` to `>=1`, clamp `pageSize` to `[1,100]`, default `pageSize=20`; convert to repo `offset = (page-1)*pageSize, limit = pageSize`.
- [X] T017 [US2] [P] Service unit tests in `internal/service/favorite_service_test.go`: `TestFavoriteService_List_ClampsPageSizeOver100`, `TestFavoriteService_List_ClampsPageBelow1`, `TestFavoriteService_List_DefaultPageSize`, `TestFavoriteService_List_PassesUserScoping`.
- [X] T018 [US2] Implement `func (h *FavoriteHandler) List(w, r)` in `internal/handler/favorite.go`: parse `page`/`page_size` query params with `400` on non-numeric, call `service.List`, shape DTO per `contracts/favorites-api.md` (`{items:[{product:{...},added_at}], page, page_size, total}`).
- [X] T019 [US2] Mount `GET /api/v1/favorites` in `internal/handler/router.go` behind auth.

**Checkpoint**: US1 + US2 together = a working MVP minus removal.

---

## Phase 5: User Story 3 — Remove a product from my favorites (Priority: P1)

**Goal**: Authenticated callers can `DELETE /api/v1/favorites/{productId}` and the action is idempotent (FR-003): always 204, regardless of whether the row existed.

**Independent Test**: Quickstart sections "5. Remove from favorites" → 204 and "C. Delete a never-favorited product → still 204" both pass.

### Implementation for User Story 3

- [X] T020 [US3] Implement `Remove(ctx, userID, productID)` in `internal/repository/favorite_repo.go`: `DELETE FROM favorites WHERE user_id=$1 AND product_id=$2`; return `nil` regardless of `RowsAffected` (treat both 0 and 1 as success).
- [X] T021 [US3] [P] Repository integration tests in `internal/repository/favorite_repo_test.go`: `TestFavoriteRepo_Remove_ExistingRow`, `TestFavoriteRepo_Remove_NonExistingRow_NoError`, `TestFavoriteRepo_Remove_DoesNotAffectOtherUsers`.
- [X] T022 [US3] Implement `Remove(ctx, userID, productID) error` in `internal/service/favorite_service.go`: no product-visibility precondition (delete is purely a post-condition assertion); call `repo.Remove`.
- [X] T023 [US3] [P] Service unit tests in `internal/service/favorite_service_test.go`: `TestFavoriteService_Remove_DelegatesToRepo`, `TestFavoriteService_Remove_NoProductLookup` (asserts the product service mock is never called).
- [X] T024 [US3] Implement `func (h *FavoriteHandler) Delete(w, r)` in `internal/handler/favorite.go`: parse `productId`, call `service.Remove`, return `204 No Content` on success.
- [X] T025 [US3] Mount `DELETE /api/v1/favorites/{productId}` in `internal/handler/router.go` behind auth.

**Checkpoint**: P1 MVP is complete — add, list, remove all functional and independently testable.

---

## Phase 6: User Story 4 — Is a specific product favorited by me? (Priority: P2)

**Goal**: Authenticated callers can `GET /api/v1/favorites/{productId}` and receive `{"favorited": bool, "added_at"?}` without needing to scan the full list. `404` is reserved for "product itself not reachable", per Decision 5 in `research.md`.

**Independent Test**: Quickstart section "4. Is this product favorited by me?" returns `true` after US1 and `false` after US3; edge case B's `404` path still applies for non-existent products.

### Implementation for User Story 4

- [X] T026 [US4] Implement `Exists(ctx, userID, productID) (bool, *time.Time, error)` in `internal/repository/favorite_repo.go`: `SELECT created_at FROM favorites WHERE user_id=$1 AND product_id=$2`; on `pgx.ErrNoRows` return `(false, nil, nil)`; otherwise return `(true, &t, nil)`.
- [X] T027 [US4] [P] Repository integration test in `internal/repository/favorite_repo_test.go`: `TestFavoriteRepo_Exists_TrueAndFalse`.
- [X] T028 [US4] Implement `IsFavorited(ctx, userID, productID) (bool, *time.Time, error)` in `internal/service/favorite_service.go`: call `productGetter.GetProductByID(ctx, productID)` first — if not-found return `service.ErrFavoriteProductNotFound`; then call `repo.Exists`.
- [X] T029 [US4] [P] Service unit tests in `internal/service/favorite_service_test.go`: `TestFavoriteService_IsFavorited_True`, `TestFavoriteService_IsFavorited_False`, `TestFavoriteService_IsFavorited_ProductNotVisible`.
- [X] T030 [US4] Implement `func (h *FavoriteHandler) Get(w, r)` in `internal/handler/favorite.go`: response DTO `{favorited bool, added_at *time.Time omitempty}` — omit `added_at` entirely when `favorited=false` (per contract).
- [X] T031 [US4] Mount `GET /api/v1/favorites/{productId}` in `internal/handler/router.go` behind auth.

**Checkpoint**: All four endpoints live. The full contract from `contracts/favorites-api.md` is exercisable.

---

## Phase 7: User Story 5 — Cascade coherence (Priority: P3)

**Goal**: The cascade behavior built into the migration (Q2 resolution) is locked in by tests so it can't regress: deleting a product or a user cleans up that party's favorite rows automatically, satisfying FR-010, FR-011, and SC-005.

**Independent Test**: Quickstart sections E (product hard-delete) and F (user delete) both pass; both also have automated integration coverage.

### Implementation for User Story 5

- [X] T032 [US5] Add cascade integration tests in `internal/repository/favorite_repo_test.go`: `TestFavoriteRepo_Cascade_OnProductDelete` (delete a product via the existing product-repo path, assert dependent favorite rows for multiple users are gone) and `TestFavoriteRepo_Cascade_OnUserDelete` (delete a user, assert their favorites are gone and no other user's favorites are touched).

**Checkpoint**: All five user stories independently functional and tested.

---

## Phase 8: Polish & Cross-Cutting Concerns

- [X] T033 [P] Update `docs/api-contracts.md`: add the four `/api/v1/favorites` routes to the authenticated routes section (any-role table), with status-code matrix per `contracts/favorites-api.md`.
- [X] T034 [P] Update `docs/database.md`: add `favorites` to the schema overview and reference the migration number.
- [X] T035 [P] Update `docs/db-schema.md`: append a `favorites` table description (columns, PK, FKs, index, role grants).
- [X] T036 [P] Update `docs/project-map.md`: list the new files under their layers (model/service/repository/handler/migration/mock).
- [ ] T037 Run targeted Go tests (no broad suite): `go test ./internal/service/... -run Favorite` and `go test ./internal/repository/... -run Favorite` against the integration database; resolve any flakiness. Per Constitution V, do not run the full test suite without permission.
- [ ] T038 Walk through `specs/002-favorite-products/quickstart.md` against a freshly migrated local instance: confirm all six happy-path steps and the seven edge-case checks (A through G) produce the documented outputs.

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies.
- **Foundational (Phase 2)**: Depends on Setup. **Blocks** every user story.
- **User Stories (Phase 3–7)**: Each depends only on Foundational being complete.
  - US1, US2, US3 are all P1 and form the MVP. Within each, the order is repo → repo tests → service → service tests → handler → wiring.
  - US4 (P2) and US5 (P3) depend only on Foundational; US5 depends specifically on the migration (T002) being in place (it tests cascade behavior, not new code).
- **Polish (Phase 8)**: Depends on all desired user stories being complete.

### User Story Dependencies

- **US1**: Foundational only.
- **US2**: Foundational only. May begin in parallel with US1 (different service/repo methods, different handler routes).
- **US3**: Foundational only. May begin in parallel with US1/US2.
- **US4**: Foundational only. Most natural after US1 (the "exists" check uses the same `(user, product)` pair shape that US1 inserts), but technically independent.
- **US5**: Foundational only (specifically T002). Pure test coverage — no production code.

### Parallel Opportunities

Within a story phase, `[P]`-marked test tasks run in parallel with their non-`[P]` sibling implementation tasks — they live in different files (`*_test.go` vs. `*.go` or `*_test.go` vs. a handler/router file). Across stories, all of US1/US2/US3 can be developed in parallel by different contributors once Phase 2 is complete, because each story touches its own service method, its own repo method, its own handler method, and adds its own route line.

```bash
# Example: launch US1 implementation and its tests in parallel
Task: "T008 Implement Add in internal/repository/favorite_repo.go"
Task: "T009 [P] Repo integration tests for Add in internal/repository/favorite_repo_test.go"
Task: "T010 Implement service.Add in internal/service/favorite_service.go"
Task: "T011 [P] Service unit tests for Add in internal/service/favorite_service_test.go"
```

---

## Implementation Strategy

### MVP First (US1 + US2 + US3)

1. Complete Phase 1 (Setup) and Phase 2 (Foundational).
2. Land US1, US2, US3 in any order (they're independent). All three are P1; the feature is not user-visible without all three.
3. **STOP and validate**: walk through quickstart sections 1–6 plus edge cases A–D. This is the demoable slice.

### Incremental Delivery

1. MVP shipped.
2. Add US4 (P2): one more endpoint; clients gain a constant-time "is this favorited?" probe for product detail views.
3. Add US5 (P3): lock cascade behavior in tests.
4. Polish (Phase 8): docs and final verification.

---

## Notes

- `[P]` tasks operate on different files.
- `[USx]` labels map tasks to spec user stories for traceability.
- Each user story is independently completable and testable.
- The generated mock at `internal/mocks/service/favorite_repo_mock.go` is touched by `go generate` only; never edited or read by hand (Constitution VI).
- No web UI, no terminal UI, no cache key family is added — the spec and plan are backend-only.
