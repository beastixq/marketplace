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

Service tests are unit tests around business rules. They should use test doubles or generated gomock mocks and generally live in package `service_test`.

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
  Repository integration suites add `_Integration` to avoid collisions when
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
