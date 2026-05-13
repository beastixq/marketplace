# Config Validation Allowed Unknown Fields and Weak Values

Status: fixed
Fixed: 2026-04-26
Commits: [`2676cce`](https://github.com/beastixq/marketplace/commit/2676cce)
Area: config / security

## Bug

Config loading allowed unknown YAML fields and weak/invalid values to pass
silently. In particular, bcrypt cost was only checked as positive, and
payment gateway URL was not required to be absolute.

## Cause

YAML was unmarshaled without `KnownFields(true)`, so typos were ignored.
Validation did not use bcrypt min/max bounds and did not parse gateway URL
structure.

## Fix

- YAML decoder now rejects unknown fields.
- `auth.bcrypt_cost` must be within bcrypt `[MinCost, MaxCost]`.
- `payment.gateway_url` must be an absolute URL.
- Secret overrides still come from `DATABASE_URL` and `JWT_SECRET`.

## Verification

- Added `internal/config/config_test.go` for valid load, env overrides,
  unknown fields, bad durations, bcrypt bounds and gateway URL validation.
