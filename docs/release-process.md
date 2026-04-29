# Release Process

This project does not currently have a formal CI/CD release pipeline documented in the repository. Use this checklist for local release candidates, coursework submissions, or portfolio milestones.

## 1. Confirm Scope

- Review the diff.
- Separate unrelated changes from the release branch.
- Confirm whether migrations, API contracts, docs, or diagrams changed.
- Check that generated files were not edited manually.

## 2. Update Documentation

Update relevant docs when behavior changes:

- `docs/architecture.md` for layer or dependency changes.
- `docs/api-contracts.md` for route, DTO, status code, or auth changes.
- `docs/database.md` and `docs/db-schema.md` for migrations/schema changes.
- `docs/setup.md` for config, startup, or dependency changes.
- `docs/testing.md` for test workflow changes.
- `docs/troubleshooting.md` for new recurring failure modes.
- `Readme.md` for user-facing project overview changes.

## 3. Database Review

For every migration:

- Verify goose `Up` and `Down` sections.
- Check constraints and indexes match service behavior.
- Check repository code handles new nullable/non-nullable fields.
- Check seed data still works if the schema changed.
- Run migrations on a clean local database.

Commands:

```bash
export DATABASE_URL='postgres://postgres:postgres@localhost:5432/marketplace?sslmode=disable'
goose -dir migrations postgres "$DATABASE_URL" up
goose -dir migrations postgres "$DATABASE_URL" status
```

## 4. Run Tests

Run all tests:

```bash
GOCACHE=/tmp/marketplace-go-build-cache go test ./...
```

If repository behavior changed, run repository tests with a real database:

```bash
export DATABASE_URL='postgres://postgres:postgres@localhost:5432/marketplace?sslmode=disable'
go test ./internal/repository
```

Check whitespace:

```bash
git diff --check
```

## 5. Manual Smoke Test

Start local services and app:

```bash
docker compose up -d
go run ./cmd/api -config config/config.yaml
```

Smoke-test the main flows affected by the release:

- Register/login.
- Catalog and product detail.
- Buyer cart and checkout.
- Payment link and mock bank flow.
- Seller product/order workflow.
- Admin pages/routes if admin behavior changed.
- Analyst dashboard if reporting changed.

## 6. Security And Config Check

- Do not commit real secrets.
- Ensure `JWT_SECRET` and `DATABASE_URL` are environment-specific outside local dev.
- Confirm logs do not leak password hashes, tokens, or payment secrets.
- Confirm role checks are enforced in service, not only in UI or routes.

## 7. Commit / Tag Guidance

Use focused commits. Suggested message style:

```text
area: concise imperative summary
```

Examples:

```text
service: enforce shipped order cancellation rule
repository: translate duplicate review constraint
web: add seller order dashboard filters
docs: document local release checklist
```

If creating a tag for a milestone, include a short release note with:

- User-visible changes.
- Migration requirements.
- Known limitations.
- Test commands run.
