# TechUI Did Not Run Expiration Worker and Component Build Was Misnamed

Status: fixed
Fixed: 2026-04-27
Commits: [`044db0d`](https://github.com/beastixq/marketplace/commit/044db0d)
Area: techui / payments / build

## Bug

`cmd/techui` не запускал `OrderExpirationWorker`, поэтому pending orders
в этом entrypoint не протухали по payment TTL. Отдельно component build
script имел неправильное имя/выходную папку для `.a` archives.

## Cause

Expiration worker был подключён к API runtime, но не к tech UI runtime.
Build script не соответствовал фактической структуре component artifacts.

## Fix

- `cmd/techui/main.go` создаёт cancellable context и запускает
  `OrderExpirationWorker`.
- Добавлен/исправлен `scripts/build_components.sh`, который кладёт
  archives в `dist/components/`.

## Verification

- Проверка по diff commit: изменены только `cmd/techui/main.go` и
  `scripts/build_components.sh`.
