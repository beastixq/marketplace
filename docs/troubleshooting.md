# Troubleshooting

## API Cannot Connect To PostgreSQL

Symptoms:

- `connect database` or `ping database` in logs.
- Server exits during startup.

Checks:

```bash
docker compose ps
docker compose logs db
echo "$DATABASE_URL"
```

Use the local DSN:

```bash
export DATABASE_URL='postgres://postgres:postgres@localhost:5432/marketplace?sslmode=disable'
```

Then rerun migrations and the API.

## Migrations Fail

Check goose status:

```bash
goose -dir migrations postgres "$DATABASE_URL" status
```

Common causes:

- `DATABASE_URL` points at the wrong database.
- A previous migration partially applied during manual testing.
- Local database volume has stale schema from an older branch.

For coursework/dev databases, recreating the Docker volume may be acceptable, but do not do this against shared or production data.

## Repository Tests Are Skipped

`internal/repository/testmain_test.go` skips repository integration tests when `DATABASE_URL` is unset.

Set it before running repository tests:

```bash
export DATABASE_URL='postgres://postgres:postgres@localhost:5432/marketplace?sslmode=disable'
go test ./internal/repository
```

## Go Build Cache Permission Errors

In restricted environments, the default Go build cache may not be writable.

Use:

```bash
GOCACHE=/tmp/marketplace-go-build-cache go test ./...
```

## Port 8080 Is Already In Use

Change `server.addr` in `config/config.yaml`, for example:

```yaml
server:
  addr: ":8081"
```

Then restart:

```bash
go run ./cmd/api -config config/config.yaml
```

## Logs Are Missing

Default config writes to `logs/marketplace.log` and stdout. If no logs appear, check:

- `logging.console`
- `logging.file`
- directory permissions for `logs/`
- startup errors before logger initialization

## Login Works But Protected Routes Fail

Check:

- The `Authorization: Bearer <token>` header for API calls.
- Token expiration from `auth.jwt_ttl`.
- `JWT_SECRET` consistency between login and later requests.
- User role matches the route group in `internal/handler/router.go`.

## Permission Denied

The service layer is the source of truth for ownership and role policy. Check:

- Actor role in the token/session.
- Ownership checks in the relevant service.
- Route role middleware.
- Whether admin is intentionally allowed for that use case.

Do not fix permission failures by bypassing service checks in handlers.

## Payment Link Expires Quickly

Local `payment.ttl` defaults to `2m`. Change it in `config/config.yaml` when manual testing needs more time.

## Stock Or Reservation Errors

Check:

- `products.stock_quantity`
- `products.reserved_quantity`
- pending/paid orders reserving stock
- service transitions that reserve or release stock

The DB constraint `reserved_quantity <= stock_quantity` is intentional.

## Template Rendering Fails

Templates are embedded from `internal/web/templates/*.html`. Check:

- The page is listed in the template parsing list in `internal/web/web_handler.go`.
- The template defines both `title` and `content`.
- The handler passes all fields the template expects.

Run:

```bash
go test ./internal/web
```

## Codex Agent Files Do Not Show In Git Status

The repository currently ignores `.codex/`. Files under `.codex/agents/` can still be used locally by Codex, but Git will not track them unless `.gitignore` is changed.
