# Cart and Checkout Writes Were Not Atomic

Status: fixed
Fixed: 2026-03-27
Commits: [`6df277c`](https://github.com/beastixq/marketplace/commit/6df277c)
Area: orders / transactions

## Bug

`AddItemToCart` and `Checkout` could leave partial writes if one step
failed after earlier steps had already committed. Checkout was especially
risky because it creates seller-split orders, copies items, then deletes
the draft cart.

## Cause

The service executed multi-step order operations as independent repository
calls without one transaction boundary.

## Fix

- Added `TxManager`.
- `AddItemToCart` wraps draft order creation and item insertion in one
  transaction.
- `Checkout` wraps seller-split order creation, item copy and draft cart
  deletion in one transaction.
- Repository connections can reuse the transaction from context.

## Verification

- `order_service_test.go` was updated for the transaction-aware service
  signatures.
