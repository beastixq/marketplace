# Setup

## Prerequisites

- Go `1.25.0` or newer compatible toolchain (the module declares `go 1.25.0`).
- Docker and Docker Compose for local PostgreSQL and Redis.
- `goose` for database migrations.
- `mockgen` only when regenerating mocks.

## Local Services

The persistent stand is defined by the root `docker-compose.yml`: API, PostgreSQL
and Redis. Its configuration is `config/docker.yaml`; PostgreSQL uses the named
volume `marketplace_postgres-data`. It needs no `TEST_RUNTIME_ROOT`, `stand.json`
or test image. The old WebLab volume `marketplace_db-data` is retained separately.

For the first deployment:

```bash
docker compose build api
docker compose up -d --wait db cache
DATABASE_URL='postgres://postgres:postgres@127.0.0.1:5432/marketplace?sslmode=disable' make migrate
docker compose up -d
```

Subsequent starts use `docker compose up -d`. After code changes, use
`docker compose up -d --build api`; after adding migrations, repeat `make migrate`.
`docker compose down` retains data; `down -v` deletes this stand's data volume.

Test commands also need host Python 3 and `make`, plus Goose CLI in `PATH` for
isolated provisioning. Docker/Compose 2.24.4+ is supported. Goose 3.26.0 is verified locally and pinned
in CI; binaries are available in the [official release](https://github.com/pressly/goose/releases/tag/v3.26.0).
Go, pytest, Allure and traffic tools live inside the runner. Migrations run on
the host, without a migration container. Use `make test-stand-up`; see
[testing stand](testing.md#container-test-stand).
Source-based application startup remains available below.
`make test-integration`, `make test-e2e` and `make test-all` use this existing common
stand without starting/restarting services, migrating or restoring a snapshot;
an unavailable database makes integration fail. Commands with `-isolated` create
and remove their own temporary environment with automatically unique Docker names;
no `STAND_ID` is needed. `test-stand-up` is the explicit persistent setup command.
Integration and E2E commands show test output and save raw results/logs without
generating Allure HTML. Add `-report` to generate it, e.g. `test-integration-report`
or `test-e2e-isolated-report`. Full `test-all` runs, including CI, retain reports.
The full integration suite can still modify shared orders, so prefer
`test-integration-isolated` for checks parallel to manual work. GitHub uses hosted
runners, not a runner installed on the student's computer. The public static
Allure and Actions artifact access are described in [testing](testing.md#ci-and-bug-handoff).

For a host-side Go process, start dependencies only (and stop any existing API
container before taking its port):

```bash
docker compose up -d db cache
```

Default services:

| Service | Address | Credentials |
| --- | --- | --- |
| PostgreSQL | `localhost:5432` | `postgres` / `postgres`, database `marketplace` |
| Redis | `localhost:6379` | no password in local compose |

The Compose file uses PostgreSQL `16.11-alpine`, Redis `8.6-alpine` and the root
API Dockerfile. UI/API is at `http://localhost:8080`. The host Go commands below
are an alternative to the API container, using the same port.

Check containers:

```bash
docker compose ps
```

## Configuration

For the container test stand, the root `Dockerfile` now builds a separate API
image with the server binary and embedded UI. The test-tool image stays in
`tests/Dockerfile`. API mounts only a read-only configuration file and runs as
UID/GID `10001:10001`; its logs are available through `docker logs`. Build tags,
the persistent runtime directory and the explicit upgrade command are described
in [testing](testing.md#container-test-stand). The source-based startup below
remains available independently of that stand.

The persistent testing containers are `marketplace-api` (UI/API on
`http://localhost:8080`), `marketplace-db` (PostgreSQL on `127.0.0.1:5432`) and
`marketplace-cache` (Redis on `127.0.0.1:6379`). The system `redis-server.service`
was stopped and disabled by the operator to free 6379. Existing API operations
are ordinary Docker commands: `docker start marketplace-api`,
`docker stop marketplace-api`, `docker logs -f marketplace-api`.

The API reads `config/config.yaml` by default:

```bash
go run ./cmd/api -config config/config.yaml
```

Important config values:

| Setting | Default | Override |
| --- | --- | --- |
| `server.addr` | `:8080` | edit config |
| `database.dsn` | local PostgreSQL DSN | `DATABASE_URL` |
| `auth.jwt_secret` | dev secret | `JWT_SECRET` |
| `auth.jwt_ttl` | `24h` | edit config |
| `payment.ttl` | `2m` | edit config |
| `payment.gateway_url` | `http://localhost:8080` | edit config |
| `logging.file` | `logs/marketplace.log` | edit config |

Do not commit real secrets. Use environment variables for local overrides:

```bash
export DATABASE_URL='postgres://postgres:postgres@localhost:5432/marketplace?sslmode=disable'
export JWT_SECRET='local-dev-secret'
```

## Database Migrations

Run migrations with the installed Goose CLI. `make migrate` requires an explicit
`DATABASE_URL`; it does not start PostgreSQL or run tests:

```bash
make migrate
```

For the testing stand prepared with `make test-stand-up`, use its forwarded port:

```bash
DATABASE_URL='postgres://postgres:postgres@127.0.0.1:5432/marketplace?sslmode=disable' make migrate
```

The old WebLab container `marketplace-db-1` is stopped; its named data volume is
retained. Port 5432 now belongs to the testing stand. Initial
deployment applies migrations explicitly through the same target; repeating
it applies only pending versions. `test-stand-up` itself only starts Compose services. The test server and runner still connect to
`db:5432` inside Docker. `MIGRATIONS_DIR` defaults to `migrations`, and `GOOSE`
can specify the CLI executable. These overrides do not change the database DSN.

Check migration status:

```bash
goose -dir migrations postgres "$DATABASE_URL" status
```

Rollback the last migration only when you intentionally need to test a down migration:

```bash
goose -dir migrations postgres "$DATABASE_URL" down
```

## Seed Data

The seed command uses the same config loader:

```bash
go run ./cmd/seed -config config/config.yaml
```

Set `DATABASE_URL` first if you want to override the YAML DSN.

## Running The App

Start the API and web MPA:

```bash
go run ./cmd/api -config config/config.yaml
```

The same server mounts:

- JSON API under `/api/v1/...`
- Web UI under `/`
- Static web assets under `/static/...`
- Mock bank web flow under `/mock_bank/payment`

The default URL is:

```text
http://localhost:8080
```

## Tech UI

The console tech UI drives service-layer use cases without HTTP handlers:

```bash
DATABASE_URL='postgres://postgres:postgres@localhost:5432/marketplace?sslmode=disable' go run ./cmd/techui -config config/config.yaml
```

## Logs

Default logging writes JSON logs to `logs/marketplace.log` and mirrors to stdout. Change `logging.format`, `logging.file`, and `logging.console` in `config/config.yaml` for local debugging.
