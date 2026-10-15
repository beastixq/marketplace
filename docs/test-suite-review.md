# Test Suite Review

The test suites are maintained by component. Test code and fixtures do not
belong to individual coursework assignments. Generated reports may use a
submission-specific directory without changing the source organization.

## Findings And Corrections

| Finding | Correction |
| --- | --- |
| A separate `lab1_service_test.go` mixed unrelated services with a duplicate product lookup and shared user-lookup behavior. | Integrate useful cases into component files, remove the duplicate stub-based lookup, and share fixtures/cases across distinct public lookup entrypoints. |
| The added expiration success case only checked an empty list. | Exercise real cancellation, lock ordering, stock release, and failure/continuation behavior in the order-service suite. |
| Parent-level gomock controllers let one scenario's expectations survive until the entire suite finished. | Own controllers and mocks inside each scenario. |
| RBAC tests mixed several Act calls and assertions for different components. | Place permission checks under independent method scenarios. |
| Repository integration scenarios were scattered across top-level functions; some unit files mixed components. | One method suite with subtests; component-specific unit/integration filenames and explicit `_Integration` suite suffixes. |
| A unit case named `not found` supplied a generic database error. | Distinguish `pgx.ErrNoRows` mapping from ordinary scan errors. |
| Update fixtures returned unchanged rows, and assertions often checked only the ID. | Return realistic updated rows and assert changed values. |
| List and transaction tests omitted scan/iteration, begin/commit and reuse failures. | Cover meaningful reachable error paths without adding seams only for impossible query-builder errors. |
| Default TestMain inspected `DATABASE_URL`, tying unit initialization to integration configuration. | Select database initialization with build tags; unit tests ignore database configuration. |
| Reports counted parent suites as independent scenarios and offered no result HTML. | Render leaf cases from JSONL in `test-results.html`, preserve failures including package/parent failures, and retain XML for CI. |
| `GOPROXY=off` was described as complete network isolation; shuffle was described without distinguishing subtests. | Document module-download restrictions and top-level shuffle semantics precisely. |
| E2E was a one-line call hiding all actions/assertions and crash waits. | Keep the purchase flow and assertions in the pytest test; curl CLI reuses that same scenario, with fault checkpoints in a separate module. |
| A common integration fixture mixed all services, builders, assertions and crash waits; cleanup depended on a successful lookup. | Keep pool/ownership/typed composition shared, move helpers to components, register exact emails before creation, and gate only after a verified committed account. |
| Role checks used empty input, reserves were manufactured directly, and persistence was sometimes checked through another service. | Use valid negative-case input, real pending reservations, repository persistence assertions and stronger rollback/review checks. |
| Allure merged Go unit/integration by package and omitted infrastructure unit results. | Import original JUnit with explicit hierarchy, stable existing history IDs and Python infrastructure results; preserve raw XML and do not fabricate Go steps. |
| Report access required downloading HTML without a public entrypoint. | Deploy a separately sanitized static Allure site; retain full diagnostics only in Actions artifacts. |

## Preserved Boundaries

- Service mocks verify business behavior through service-owned interfaces.
- Repository mocks verify arguments, mapping and error handling without a DB.
- Repository integration tests still verify actual SQL, constraints, triggers
  and persisted state. They are not redundant with mock-based tests.
- Pure order-transition tests provide examples without mocks or stubs.
- Service-only builders/mothers stay in test fixtures. Transaction doubles used
  by more than one package remain in `internal/testsupport`.

Commands, naming rules, test-data techniques, report formats and process
configuration are documented in `docs/testing.md`. Current results and
coverage percentages are generated artifacts rather than fixed claims here.
