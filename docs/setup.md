# Setup

## Prerequisites

- Go `1.24.4` or compatible Go `1.24.x`.
- Docker and Docker Compose for local PostgreSQL and Redis.
- `goose` for database migrations.
- `mockgen` only when regenerating mocks.

## Local Services

Start PostgreSQL and Redis:

```bash
docker compose up -d
```

Default services:

| Service | Address | Credentials |
| --- | --- | --- |
| PostgreSQL | `localhost:5432` | `postgres` / `postgres`, database `marketplace` |
| Redis | `localhost:6379` | no password in local compose |

Check containers:

```bash
docker compose ps
```

## Configuration

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

Run migrations with goose:

```bash
goose -dir migrations postgres "$DATABASE_URL" up
```

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
