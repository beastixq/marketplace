# Style Guide

## Go

- Follow standard Go formatting with `gofmt`.
- Keep functions focused and small enough to read without hidden side effects.
- Prefer clear concrete types until an interface is needed for behavior or testing.
- Accept interfaces where behavior is consumed, not where implemented.
- Pass `context.Context` as the first argument on service and repository methods.
- Prefer constructor injection over package-level mutable state.
- Comments should explain why, not restate what the code says.

## Architecture Style

- Do not describe `handler -> service -> repository` as source dependency direction. That is runtime flow.
- Source dependencies are `handler/web/techui -> service <- repository implementation`.
- Service owns repository interfaces.
- Repository implements service-owned interfaces.
- Handlers and web controllers do not import repositories.
- DB rows, domain models, DTOs, and view data are separate concepts.

## Error Style

- Wrap errors with `%w`.
- Check errors with `errors.Is` and `errors.As`.
- Define database-agnostic application errors in service when shared across layers.
- Translate database-specific errors inside repository code.
- Use `pgerrcode.*` instead of raw SQLSTATE strings.
- Use `net/http` constants instead of raw status codes.

## SQL Style

- Use parameterized SQL only.
- Do not concatenate SQL with user input.
- Keep SQL and pgx code in repository.
- Keep transactions explicit.
- Keep mapper logic explicit; avoid magic struct scanning when a local explicit mapper is clearer.

## API Style

- Routes live under `/api/v1`.
- Request validation happens in handlers.
- Business validation happens in service.
- Response DTOs live in `internal/handler`.
- Error responses use `{"error":"message"}`.
- Keep service error mapping centralized in `internal/handler/service_error.go`.

## Web UI Style

- This project is a server-rendered MPA, not a SPA.
- Templates live in `internal/web/templates`.
- Shared CSS lives in `internal/web/static/css/style.css`.
- Pages should define `title` and `content` templates and render through `layout.html`.
- Keep templates mostly presentational. Move complex view shaping into `web_handler.go`.
- Keep business rules in service, not templates or web handlers.
- Use POST forms for mutations and redirect after success.
- Match the current restrained UI: simple surfaces, light borders, blue primary actions, compact tables/lists/forms.

## Testing Style

- Service tests: business rules with doubles/mocks.
- Repository tests: real PostgreSQL.
- Handler tests: transport behavior, DTOs, validation, service error mapping.
- Web tests: template rendering, redirects, form behavior, view data.
- Do not edit generated mocks manually.

## Documentation Style

- Update docs when changing contracts, architecture boundaries, setup steps, migrations, or test workflow.
- Prefer short, actionable docs over exhaustive prose.
- If docs and code conflict, trust current code and update docs.
