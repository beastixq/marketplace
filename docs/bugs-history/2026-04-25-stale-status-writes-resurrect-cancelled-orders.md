# Stale Status Writes Resurrect Cancelled Orders

Status: fixed
Fixed: 2026-04-25
Commits: [`4403f33`](https://github.com/beastixq/marketplace/commit/4403f33)
Area: orders / payments / concurrency

## Bug

`PayOrder`, платёжный callback или `DeliverOrder` могли записать новый
статус на основании устаревшего чтения. Самый опасный сценарий:
`pending`-заказ отменили или протухли, а поздний платёжный путь вернул
его в `paid`.

## Cause

Не все переходы статуса были переведены на атомарную conditional update
после [`Double Reservation Release on Concurrent Cancel/Ship`](2026-04-25-double-reservation-release-concurrent-cancel-ship.md).
Часть кода всё ещё делала "прочитал - проверил - записал".

## Fix

- `PayOrder`, payment callback и `DeliverOrder` переведены на
  `UpdateOrderStatus` с явным списком разрешённых исходных статусов.
- Payment callback стал идемпотентным для повторных или поздних событий:
  уже `paid`/`cancelled` заказ не считается внутренней ошибкой callback.

## Verification

- Добавлены lost-race тесты в `order_service_test.go` и
  `payment_service_test.go`.
