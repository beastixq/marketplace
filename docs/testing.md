# Testing

## Main Commands

Run all tests:

```bash
go test ./...
```

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

Repository tests use a real PostgreSQL database when `DATABASE_URL` is set. If `DATABASE_URL` is not set, `internal/repository/testmain_test.go` skips repository integration tests.

Local integration test flow:

```bash
docker compose up -d
export DATABASE_URL='postgres://postgres:postgres@localhost:5432/marketplace?sslmode=disable'
goose -dir migrations postgres "$DATABASE_URL" up
go test ./internal/repository
```

Repository tests should cover SQL behavior, transactions, constraints, triggers, and PostgreSQL roles. Do not replace these with mocks when database behavior is the point of the test.

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

- Put broad business behavior tests at the service layer.
- Put SQL/constraint behavior tests at the repository layer with a real database.
- Put transport behavior tests at handler/web layer.
- Keep tests table-driven when it improves clarity.
- Do not assert implementation details when a public behavior assertion is enough.
