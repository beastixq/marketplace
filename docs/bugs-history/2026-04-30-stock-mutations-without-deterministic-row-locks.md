# Stock Mutations Without Deterministic Row Locks

Status: fixed
Fixed: 2026-04-30
Commits: [`96bb64f`](https://github.com/beastixq/marketplace/commit/96bb64f)
Area: inventory / concurrency

## Bug

Concurrent cancel/expire/ship flows могли менять stock/reserve по
нескольким products без заранее согласованного порядка row locks. Это
создавало риск deadlock и stale validation при пересечении заказов.

## Cause

Stock mutations полагались на последующие update-запросы, но не
захватывали все затрагиваемые product rows через `SELECT FOR UPDATE` в
детерминированном порядке перед изменением остатков.

## Fix

- `CancelOrder`, `ExpireOrders` и `ShipOrder` собирают уникальные product
  IDs, сортируют их и берут `GetProductByIDForUpdate` перед
  `ChangeStockAndReserved`.
- `UpdateProduct` также валидирует deleted-state и stock-vs-reserved
  после row lock. Позднее эта часть была доработана в
  [`Product Update TOCTOU and Nested-Tx Self-Block`](2026-05-01-product-update-toctou-nested-tx-self-block.md).

## Verification

- Обновлены `order_service_test.go`, `product_service_test.go` и
  web-handler tests, связанные с stock/availability.
