# Checkout Did Not Reserve Stock

Status: fixed
Fixed: 2026-04-24
Commits: [`0f9abff`](https://github.com/beastixq/marketplace/commit/0f9abff), [`81605fc`](https://github.com/beastixq/marketplace/commit/81605fc)
Area: inventory / orders

## Bug

Checkout moved a draft cart to pending orders without reserving product
stock. Multiple buyers could pass checkout against the same visible stock
before shipment, making later fulfillment inconsistent.

## Cause

The model had only physical stock quantity. There was no
`reserved_quantity` and no lifecycle step that moved stock from available
to reserved during checkout.

## Fix

- Added `reserved_quantity` to product model/repository/schema.
- Checkout reserves stock for pending orders.
- Cancellation/expiration/payment lifecycle paths were extended to release
  or preserve reservation according to status.
- Added `OrderExpirationWorker` for pending payment TTL handling.

## Verification

- `order_service_test.go` and `order_expiration_worker_test.go` were
  updated around reservation-aware lifecycle behavior.
