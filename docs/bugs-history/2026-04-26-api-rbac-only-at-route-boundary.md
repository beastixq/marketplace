# API RBAC Only Lived at Route Boundary

Status: fixed
Fixed: 2026-04-26
Commits: [`2898b0a`](https://github.com/beastixq/marketplace/commit/2898b0a), [`bc629e7`](https://github.com/beastixq/marketplace/commit/bc629e7)
Area: API / RBAC / service

## Bug

Buyer/seller/admin operations were initially separated mostly by HTTP
route grouping. That protected the JSON API path, but service calls from
web UI, tech UI, tests, or future entrypoints could bypass role checks.

## Cause

Authorization was concentrated at the handler/router boundary. Service
methods accepted raw IDs or domain inputs instead of an `Actor`, so the
business layer could not consistently enforce role permissions itself.

## Fix

- `2898b0a` grouped buyer and seller API endpoints under `RequireRole`.
- `bc629e7` introduced service-level actor checks across address, auth,
  category, order, payment, product, review, seller and user services.
- Handlers were updated to map service `ErrPermissionDenied` to 403.
- TechUI and web paths were updated for the new actor-aware signatures.

## Verification

- `bc629e7` updated service tests, including `rbac_service_test.go`.
