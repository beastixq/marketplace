# Payment Flow Was Modeled as Direct Status Mutation

Status: fixed
Fixed: 2026-03-31
Commits: [`da34973`](https://github.com/beastixq/marketplace/commit/da34973)
Area: payments / orders

## Bug

Payment was effectively modeled as an in-app order status mutation. There
was no payment link boundary, public bank callback path, or callback-side
processing flow.

## Cause

Order payment logic lived too close to order service status transitions.
The project did not yet separate marketplace checkout from gateway
interaction.

## Fix

- Added payment gateway port and mock gateway adapter.
- Added payment link DTO/handler and public callback route.
- Added `PaymentService` for processing gateway callbacks.
- Added expiration worker tests for unpaid pending orders.

## Verification

- Commit added `payment_service_test.go` and
  `order_expiration_worker_test.go`.
