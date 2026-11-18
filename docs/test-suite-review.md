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

## Stand Orchestration Responsibilities

The public mode is explicit: commands without `-isolated` attach to the common
laptop stand; suffixed commands own automatically named disposable resources.
Only `test-stand-up` explicitly provisions the persistent stand. CI calls the
single-stage isolated commands in three jobs connected by GitHub `needs`;
local `test-all-isolated` remains available. Both paths reuse the same tests.

| Owner | Current boundary |
| --- | --- |
| Compose/configuration and Dockerfiles | Infrastructure topology, tool images, ports and service configuration; primarily platform/DevOps maintenance. |
| GitHub workflow / Composite Actions | CI job dependencies, shared stage setup and artifact delivery. |
| `controller.run` / `pipeline` | Local full-run order or one isolated stage, result publication and recovery attempts. |
| `Stand` | Concrete Docker processes, resource ownership and PostgreSQL snapshot/recovery; currently in the controller module. |
| `runner.py` / `results.py` | Tool execution, result conversion/redaction and Allure generation; no purchase business assertions. |
| Go tests, pytest scenarios and their fixtures | Developer/QA expectations, arrangement, requests/assertions and owned-data cleanup. |

These are code boundaries, not exclusive job titles. DevOps need not edit the
purchase scenario to change topology, and QA need not implement Docker lifecycle
to add a business assertion. One shared entrypoint keeps local and CI recovery
semantics consistent; replacing it with a long Make recipe would not remove that
responsibility.

The controller is nevertheless partially overloaded. `Stand.source` couples
execution metadata to source delivery/checkout. Local tests now mount the working
tree directly; persistent deployment no longer uses this method. `Stand.reports` also owns branch-history storage and
now imports history before a single generation. Full and sanitized public reports
are still separate outputs of that one stage.
Those are independent reasons to change this module, not PostgreSQL lifecycle.
The recommended later boundary is a thin orchestrator with separate image/source
delivery and report-history owners; `Stand` should own only the concrete stand
lifecycle. File size alone is not evidence for a new framework or more services.
This review records the remaining coupling; responsibility extraction has not
been presented as implemented. Duplicate history generation is fixed.
The refactoring scope, deployment handoff and acceptance criteria are tracked
as TD-003 in [known issues](known-issues.md), including the current inability to
attach the container command to an arbitrary existing stand by DSN/URL alone.
Independent temporary stands do not prove that the entire integration suite is
safe for concurrent developers on one shared database (Lab2 requirement 15).

The approved API-packaging change separates the root server Dockerfile from the
runner Dockerfile. API builds from the working tree or CI checkout used by tests,
contains the executable/embedded UI, runs without root and mounts only config.
Revision/content tags and metadata distinguish API from test tooling. This closes
the external-binary packaging issue, not the remaining deployment/source/history
responsibility split above. Explicit stand setup is the operator's upgrade action;
attached tests keep the running API unchanged.

Migrations now run through the host's Goose CLI and the root `make migrate`
target, after PostgreSQL readiness. SQL comes from the working tree or CI checkout;
the CLI connects through the published localhost port. CI installs the same
verified 3.26.0 release separately. The test image no longer contains Goose,
and its `prepare` stage does not apply SQL. This removes that deployment task
from the test container; source/config preparation and API deployment remain
used only by isolated provisioning. The common stand is now deployed by root
Compose, with versioned configuration and a named database volume; it has no
dependency on runner preparation or ownership metadata. Source archives are removed.
