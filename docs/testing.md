# Testing

## Main Commands

Run all tests:

```bash
go test ./...
```

Run the unit suite, including randomized and dependency-offline modes:

```bash
make test-unit
make test-unit-shuffle
make test-unit-offline
```

`make tools` downloads dependencies once. The offline command sets `GOPROXY=off`
and requires dependencies and the Go toolchain to be cached. This disables
module downloads; it is not an operating-system network restriction. Service
and repository unit tests replace external dependencies with doubles.

Generate test results and coverage reports:

```bash
make test-artifacts
```

`make lab1` remains an alias. The output directory is `artifacts/lab1/`:

| File | Use |
| --- | --- |
| `test-results.html` | Browser-readable results, search, failure output and shuffle seeds. Counts leaf cases rather than parent suites. |
| `unit-tests.xml` | JUnit-compatible input for CI/reporting tools. Not a browser report. |
| `unit-tests.jsonl` | Go test events used to generate the HTML results. |
| `coverage.html` | Coverage of source code; separate from test results. |
| `coverage.out`, `coverage.txt` | Standard Go statement coverage profile and function summary. |
| `branch-coverage.txt` | Boolean branch outcomes from gobco. |

`make test-report` runs tests and produces XML, JSONL and HTML, including HTML
when tests fail. `make test-report-html` regenerates HTML from an existing JSONL
file without running tests. `cmd/testreport` is the local HTML renderer; it uses
the standard library and needs no browser-side services or external assets.

In sandboxed environments where the default Go build cache is not writable, use:

```bash
GOCACHE=/tmp/marketplace-go-build-cache go test ./...
```

Run focused packages:

```bash
go test ./internal/service
go test ./internal/handler
go test ./internal/web
go test ./internal/repository
```

Check formatting whitespace before committing:

```bash
git diff --check
```

## Repository Integration Tests

Repository unit tests live in `*_repo_unit_test.go`, use `pgxmock`, and run
without a database or `DATABASE_URL`. Tests in `*_repo_integration_test.go`
exercise real SQL, constraints, triggers, and transactions. They carry the
`integration` build tag and require PostgreSQL. `testmain_test.go` owns the
entrypoint with a no-op default setup; the tagged `testmain_integration_test.go`
registers database initialization and shared integration fixtures. The setup
hook avoids duplicate declarations in gobco's test-file analysis.

Local integration test flow:

```bash
docker compose up -d
export DATABASE_URL='postgres://postgres:postgres@localhost:5432/marketplace?sslmode=disable'
goose -dir migrations postgres "$DATABASE_URL" up
make test-repository-integration
```

Do not replace the integration suite with mocks when database behavior is the
point of the test. The two suites are complementary: unit tests cover public
repository behavior and error translation; integration tests prove that the
queries and database guarantees work against PostgreSQL.

## Service Tests

Service unit tests cover business rules with doubles or generated gomock mocks,
generally in package `service_test`. Tagged `*_service_integration_test.go` suites
compose real services, repositories, `PgxTxManager` and PostgreSQL. Their fixtures
live with the service tests; only connection setup is shared with the repository
suite through `internal/testsupport`. Production service imports remain unchanged.

Use package `service` only when intentionally testing unexported helpers.

Examples of service behavior to test:

- Ownership checks.
- Role and permission rules.
- Order lifecycle transitions.
- Cart and checkout invariants.
- Payment validation.
- Error wrapping and `errors.Is` behavior.

## Handler And Web Tests

API handler tests should focus on:

- Request parsing and validation.
- DTO shape and status code mapping.
- Service error mapping.
- Auth/role behavior where the handler owns transport behavior.

Web tests should focus on:

- Template parsing and rendering.
- View data shape.
- Redirects after POST.
- Form validation and error display.

Existing web tests live in `internal/web/web_handler_test.go`.

## Generated Mocks

Generated mocks live under `internal/mocks/service/` and should not be edited manually.

Mock generation directives live near service-owned interfaces and ports:

```bash
go generate ./...
```

Regenerate mocks after changing a service interface or port interface, then run the affected tests.

## Test Design Rules

- Test files belong to a component, not a coursework assignment or report.
- Use `Test<Component>_<Method>` for a method suite and `t.Run` for its scenarios.
  Service/repository integration suites add `_Integration` to avoid collisions when
  both test types are compiled together.
- Each scenario prepares data/doubles, calls one method, then checks the result.
  Blank lines express Arrange–Act–Assert; repetitive stage comments are not needed.
- Create gomock controllers and mocks inside each scenario so expectations and
  cleanup belong to that scenario.
- Keep different public entrypoint contracts when useful, but share lookup
  fixtures/case tables rather than duplicating the same error-handling matrix.
- Keep service-only builders/mothers in `internal/service/fixtures_test.go`.
  `internal/testsupport` is for helpers actually shared between packages.
- Put broad business behavior tests at the service layer.
- Put SQL/constraint behavior tests at the repository layer with a real database.
- Put transport behavior tests at handler/web layer.
- Keep tests table-driven when it improves clarity.
- Do not assert implementation details when a public behavior assertion is enough.

Integration's shared fixture owns the PostgreSQL pool, exact account cleanup and
typed service composition. Component files own their data builders and persistence
assertions. Register cleanup ownership before a creating call; a failed subsequent
lookup must not leak the account. Negative role checks use otherwise valid input.

Test-data techniques are described here rather than encoded in test names:

| Scenario group | Technique and example |
| --- | --- |
| Normal valid results and distinct invalid input classes | Equivalence partitioning: successful lookup, absent resource, wrong payment amount. |
| Empty updates/lists, pagination limits, quantity and time limits | Boundary values: no changed fields; stock exactly sufficient/insufficient; expired payment. |
| Actor role, ownership, purchase prerequisites | Decision table: own product vs another seller's product; buyer vs seller/admin. |
| Order lifecycle and transaction outcomes | State transitions: pending to cancelled; already-paid callback; commit vs rollback. |
| Dependency query/scan/iteration/exec faults | Error guessing: a list query succeeds but scanning or iteration fails. |
| Several sellers or filters in one valid fixture | Ordinary equivalence cases unless the test explicitly varies independent factors; multiple fields alone do not make a combinatorial test. |

London-style service tests use gomock expectations on outbound ports. Pure
order-transition tests exercise actual functions without mocks or stubs.
Unit and integration checks of the same repository method are complementary,
not duplicate coverage of the same boundary.

## Go Test Processes And Ordering

`go test` builds one test binary and starts one process per package. Package
build/test concurrency is controlled by `-p`. Within one test binary,
`-parallel` limits only tests that call `t.Parallel`; the current suites do not,
so tests and subtests within a package are sequential. `GOMAXPROCS` controls the
Go scheduler, not the number of test binaries.

`-shuffle=on` randomizes top-level test and benchmark order and prints the seed.
It does not shuffle a table's loop or its `t.Run` calls: scenarios run in the
order the test calls them. Re-run with `-shuffle=<seed>` to reproduce the same
top-level order. Fixtures and mocks must still be independent per scenario.

Show just the executed test binaries and their count:

```bash
make test-processes
```

The target runs tests with `-count=1 -x`, saves the otherwise noisy trace to
`artifacts/lab1/processes.log`, and prints only binary executions. The current
`UNIT_PACKAGES` set starts six test processes: service, repository, config,
logging, middleware and web. This count excludes the Go driver and build tools.
`go test ./...` additionally runs the report-renderer tests, so its count can
differ. Use `make test-processes UNIT_PACKAGES='./...'` to inspect that command.

## Coverage

The standard Go profile reports covered statements/basic blocks and provides a
line-oriented HTML view. `gobco -branch` is run separately because the standard
tool does not report true/false branch outcomes. The native percentage is
statement coverage, not an exact physical-line percentage. No percentage gate
is applied; add missing meaningful behaviors and reachable failure cases,
rather than mocking query builders solely to execute defensive branches.

## Container Test Stand

Testing Lab2 is separate from WebLab#2. The stand reuses the current PostgreSQL
16.11/Redis 8.6 services through `docker-compose.test.yml`, adds one API/MPA
process and a separate test runner. The normal `docker-compose.yml` is unchanged.
Go 1.25, pytest/requests, Allure 2.34.1, Goose and capture tools run in the image;
Python packages are pinned in `tests/requirements.txt`. Host prerequisites are
Linux, Docker/Compose 2.24.4+, Python 3 and make. First use downloads the image
and dependencies; subsequent runs reuse image/module/build caches.

```bash
make test-stand-up                         # default STAND_ID=local; prints UI URL
STAND_ID=local make test-e2e                # shared stand, only own API fixtures
STAND_ID=local make test-integration        # exclusive; PostgreSQL repository + service
STAND_ID=local make test-all                # unit → integration → E2E, with Lab1 coverage
make test-infra-check                      # isolated acceptance stands, deliberate failures
STAND_ID=local make test-traffic            # curl purchase + PCAP + tshark verification
make test-restore RUN_ID=<interrupted-run>  # exclusive recovery, never use during manual writes
```

Every command gets a unique `RUN_ID`; an explicit ID must not already exist.
`STAND_ID` defaults to `local` for all commands. The same pair server/DB serves
manual and automated work, not two permanent deployments. Without an existing
stand, a full/integration invocation creates its own and removes its containers
and network after verified recovery. `test-stand-up` keeps the stand available.
Shared E2E/traffic require an existing ready stand and never recreate the server.
Local developers using the same Docker daemon choose different IDs, for example
`STAND_ID=alice` and `STAND_ID=bob`; UI ports are allocated independently. The
default UI binding is localhost; `UI_PORT` can select a port at first startup.
`STAND_ID=verification` is only a chosen name, not a special mode. Each stand has
its own PostgreSQL process/data directory, Redis, API and network; it does not add
a database to the normal application's PostgreSQL. Runner image tags depend on
Dockerfile/requirements/bootstrap contents, so different checkouts do not replace
each other's differently configured runner. Images and Go caches remain reusable.

For a persistent manual/automatic stand, start it with `test-stand-up` first.
`test-e2e` then creates only temporary runner containers and keeps API/DB/Redis
running. `test-all` on that stand preserves it afterwards but stops the API for
snapshot/restore; it is not a zero-downtime operation. A full run that creates its
own stand removes only that stand's containers after verified recovery.

Local runs archive tracked and non-ignored working-copy files, excluding `.serena`
and reports. The runner extracts that snapshot. CI fetches the exact commit
specified by `TEST_COMMIT_SHA` from `TEST_REPOSITORY_URL`. Uncommitted local runs
are recorded as working-copy snapshots, not claimed to be checks of HEAD alone.
Runtime DB directories and compilation caches use `/tmp/marketplace-test-runtime`
(`TEST_RUNTIME_ROOT` can override it); reports/recovery snapshots are copied to
`artifacts/lab2/<RUN_ID>/`. Do not remove runtime directories while a stand is in
use. Bind-mounted runtime data is retained for diagnosis after container removal;
there is no global Docker prune or automatic deletion of unrelated resources.
Directories are execution records, not a list of live stands: `check-*` includes
deliberate failures/repeats/parallel checks; `ci-*` identifies cloud runs; `history`
holds report history. Start with the relevant run's `stages.json`, `recovery.json`
and `allure-report/index.html`. Local retention is manual: do not remove recovery
snapshots of unfinished runs. Existing diagnostic directories are not auto-deleted.

### Data Ownership And Recovery

`shared`: API fixtures register unique buyers/sellers, create seller profiles,
products and addresses. The scenario itself creates cart, checkout and review.
The fixture manifest is saved before any creation and updated after each response.
Current `/users/me` omits the ID, so arrangement reads the server-issued JWT's
UserID and checks the profile email/role; no API change is introduced.

Pytest registers finalization before arrangement. A host-owned cleanup stage also
handles a killed test process. PostgreSQL cleanup locks confirmed fixture rows,
refuses foreign order/review/product links before deleting, and hard-deletes only
owned rows. It never resets global sequences or Redis. Only confirmed own product
cache keys are removed. Temporary synthetic tokens are kept in private runtime
credential files for post-cleanup HTTP 401 verification and then deleted; they
are excluded from published artifacts. This is test tooling, not an application
session store. On a foreign-link refusal, retain diagnostics and resolve the
foreign activity before retrying cleanup; do not change the query to delete it.
Confirmed cache IDs survive a transient Redis failure, so cleanup can retry after
PostgreSQL deletion. JWTs are stateless: the HTTP 401 check confirms absence of
the synthetic account now, not a permanent token blacklist. Exclusive sequence
restoration can later reuse an ID; raw captures are sensitive even after cleanup.
Shared isolation protects ownership, not global catalog/analytics counts: manual
testers may temporarily see synthetic products/orders. Use an exclusive window
for exact whole-platform aggregates. Never buy a synthetic test product from a
manual account; if that happens, cleanup refuses foreign links and keeps evidence.

`exclusive`: agree a window with no manual writes. `STAND_ID` locking serializes
automation but cannot prevent a human from submitting a form. The host controller
stops the API/expiration worker, snapshots PostgreSQL after migrations and before
tests, then runs integration packages sequentially (`-p=1`). The API starts for
E2E. Regardless of success/failure/interruption, it stops writers, restores through
`pg_restore` with ownership and grants preserved, flushes only this stand's Redis,
and compares schema ACL/owners and complete dumped data including
sequences and Goose migration rows with baseline. Recovery is completed only after
HTTP readiness and synthetic-token checks pass. Recovery failure
makes the command fail and keeps the stand for diagnosis. Verified recovery returns
the API to readiness, checks synthetic tokens and preserves user-owned stands.

If the host controller is SIGKILLed, run `test-restore` with its ID. It removes
only known orphan runner names for that run, restores the saved snapshot and marks
an unfinished stage `broken` and unstarted stages `skipped`, then regenerates the
report. Power loss does not guarantee immediate restoration. Complete explicit
recovery before another full/shared run or resuming manual writes. New automated
runs refuse a stand with an unfinished recovery record, even after the dead host
process releases its lock. Repeating `test-restore` after successful recovery is a
non-mutating no-op: it must not overwrite later manual changes. Resource ownership
is rechecked after acquiring the stand lock before deciding whether teardown is safe.
Shared runs also persist recovery intent before fixtures: the same `test-restore`
removes their orphan runner and retries only their addressed cleanup/401 checks,
without stopping the common API, resetting PostgreSQL or flushing Redis. Unfinished
shared cleanup blocks new automation too. Parallel manual work remains allowed;
automated runs on one stand must finish cleanup before the next run starts.
Recovery requires
the stand/runtime storage to remain available; preserve the saved artifacts if
runtime storage is lost. Do not restore an old snapshot over new manual work.

### Scenarios And Reports

Service integration covers account hashing/authentication, ownership/roles,
product history/categories and rollback, single/multi-seller checkout, stock
reservation/cancellation/expiration/shipment/delivery, payment boundaries and
idempotency, delivered-only reviews, aggregates and competing buyers. Existing
SQL suites remain; backoffice and real transaction checks complement them.
Tagged suites missing either package or containing skips fail the runner.

`tests/e2e/test_purchase.py` drives login → cart → checkout → payment link →
mock-bank callback → ship → deliver → review and reads the saved review through
HTTP. Allure distinguishes fixture setup errors from assertion failures and
attaches redacted exchanges. Product-stock assertions stay in service integration
because the optional HTTP product cache may be stale. Review reading supplies
`page=1&limit=20`: the unpaginated route's confirmed defect is [BUG-004](known-issues.md).
Price-changing service fixtures provide `ChangedBy`, as the current HTTP handler
does; missing metadata is recorded as RISK-001. Seller-statistics windows use UTC
day boundaries because its PostgreSQL function accepts dates. Payment checks use
the existing controllable clock; the purchase creates a fresh cart due to BUG-001.
The scenario's HTTP actions and business assertions are visible in the test file;
`purchase.py --curl` invokes the same pytest scenario with another transport,
including its fixtures/finalizer. Crash-test checkpoints live in `faults.py`.

Each run retains Go JSONL/JUnit, Python infrastructure JUnit, native pytest Allure, stage statuses, server logs,
environment/source hash, fixture manifests, original Lab1 HTML/statement/branch
coverage and `allure-report/index.html`. Go JUnit parents are removed only when
leaf results capture their outcome; an independent suite cleanup failure remains
visible. A small JUnit importer supplies `Unit`/`Integration`/`Infrastructure`
hierarchy while preserving Allure's original JUnit history IDs. It does not invent
Go business steps; detailed steps/HTTP attachments belong to the native pytest E2E.
Allure generates the HTML and trends; `cmd/testreport` is not extended.
History is serialized and reused per branch, without overwriting run artifacts.
Pipeline markers live in a separate Infrastructure suite; they are not additional
product scenarios. Reports remain available after test/recovery failures.

Serve an existing Allure report without installing Java/Allure on the host:

```bash
python3 -m http.server 8090 --bind 127.0.0.1 --directory artifacts/lab2/<RUN_ID>/allure-report
```

Open `http://127.0.0.1:8090`; direct `file://` loading can block the report's JSON.

After successful deployment, open [Allure on GitHub Pages](https://beastixq.github.io/marketplace/).
`public-allure-report/` is generated from allowlisted result fields: statuses,
hierarchy, timings and step names, without HTTP attachments, parameters, failure
traces, fixture containers or environment files. History is sanitized too. The
full report/logs/coverage remain in the Actions artifact, not in the Pages upload.
Failed tests still appear as failed on the public site; details require the full
diagnostic artifact. This is static HTML hosting, not an installed Allure service.

`test-traffic` captures only the synthetic client's HTTP connection inside the
server's Docker network, repeats purchase with curl and checks endpoints/statuses
with tshark. Capture contains synthetic credentials/JWTs: confirm account cleanup
before sharing it. Requests/responses in Allure are redacted. Tcpdump uses native
timed rotation rather than disabling a host AppArmor profile.

### CI And Bug Handoff

`.github/workflows/testing.yml` calls the same `make test-all`, restores branch
history using the job's GITHUB_TOKEN and uploads reports/history with `always()`.
Coverage remains part of the unit stage. No PAT/MCP installation is necessary.
The `test` job uses a GitHub-hosted `ubuntu-24.04` runner; no self-hosted runner is
installed. One VM owns the Docker stand through unit → integration → E2E and
recovery. Three ordinary jobs would get three independent VMs and lose that
continuous stand. The job summary lists individual stage statuses; `publish-report`
is a separate dependent job that deploys only sanitized HTML, including reports
from failed test runs. Pull requests never publish the site.

Pages requires a one-time repository setting: Settings → Pages → Source: GitHub
Actions; permit TAD-Lab2 in the `github-pages` environment if restricted. Deployment
then uses only `pages: write`/OIDC and the job token, no PAT. Test artifacts/history
are uploaded with `always()` independently of publication.
See the [Lab2 requirement audit](testing-lab2.md) for scope, evidence and limitations.
The revised test job was verified on 2026-10-15 in
[Actions run 38775314127](https://github.com/beastixq/marketplace/actions/runs/38775314127)
at `d28f259`: unit → PostgreSQL integration → pytest E2E and recovery passed;
full/public Allure contain 744 passed and no failed/broken/skipped results, including
Python infrastructure tests. Lab1 reports/coverage and previous history were retained.
The first publication attempt was rejected by the `github-pages` environment's
branch policy (only main was allowed), before executing deployment steps. Test-job
success does not imply publication: check the separate `publish-report` job/site.
The earlier Git bootstrap ownership failure was fixed with an exact container
checkout `safe.directory` and a real-Git regression.
Local success alone is not a GitHub Actions result. Future commit/push requires
separate consent.

See [bug handoff](bug-handoff.md) for minimal reproduction, evidence, retest and
regression. Never create a remote issue just because a local test failed.

`test-infra-check` creates only dedicated stands and verifies three repeat runs,
two concurrent independent full runs, unit fail-fast, partial API arrangement,
guarded cleanup, container termination after committed integration/E2E data,
controller death with explicit recovery, successful next run, history and PCAP.
Its deliberate fault controls (`TEST_FAULT`, `TEST_FAULT_GATE`) are test-only and
unset during normal runs. Simultaneous browser/manual behavior is checked separately
on independent accounts; GUI automation is not the MVP E2E requirement.
The Git-checkout bootstrap unit check runs on the host/CI checkout and checks three
consecutive runner entries with a simulated foreign filesystem owner and an
isolated Git configuration. It is skipped only in a local snapshot container,
which intentionally has no `.git`; PostgreSQL integration skips are never accepted.
