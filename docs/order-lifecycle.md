# Order Lifecycle

Current lifecycle behavior is implemented in `internal/service/order_service.go`
and `internal/service/order_state.go`. PostgreSQL stores status in
`orders.status`; service code owns business policy.

## Statuses

| Status | Meaning | Stock Effect |
| --- | --- | --- |
| `draft` | Buyer cart. There is no separate cart table. | No reservation. Add/update only checks available quantity. |
| `pending` | Checked out order waiting for payment. | Checkout reserves requested quantity in `products.reserved_quantity`. |
| `paid` | Payment accepted. | Stock is still reserved; physical stock is not decremented yet. |
| `shipped` | Seller marked the order as shipped. | Stock and reserved quantity are both decremented. |
| `delivered` | Seller marked the shipped order as delivered. | No additional stock change. |
| `cancelled` | Pending/paid order was cancelled or an expired pending order was closed by the worker. | Reserved quantity is released for `pending` and `paid` cancellations/expiration. |

## State Machine

```text
draft --checkout--> pending --payment--> paid --ship--> shipped --deliver--> delivered
                      |             |
                      |             +--cancel--> cancelled
                      +--cancel/expire--------> cancelled
```

Allowed transitions are centralized in `orderTransitionRules`:

| Transition | From | To | Caller |
| --- | --- | --- | --- |
| checkout | `draft` | `pending` | Buyer through `OrderService.Checkout`. |
| pay | `pending` | `paid` | `PaymentService.ProcessOrderPayment`; `OrderService.PayOrder` also exists for tech UI/service tests. |
| cancel | `pending`, `paid` | `cancelled` | Buyer who owns the order, or admin. |
| expire | `pending` | `cancelled` | `OrderExpirationWorker` through `OrderService.ExpireOrders`. |
| ship | `paid` | `shipped` | Seller who owns `orders.seller_id`. |
| deliver | `shipped` | `delivered` | Seller who owns `orders.seller_id`. |

There is no current seller cancel endpoint or seller cancel service path.

## Cart Behavior

- A cart is an `orders` row with status `draft`.
- `orders.address_id` and `orders.seller_id` may be `NULL` while the order is a
  draft cart.
- Migration `011_order_cart_item_invariants.sql` enforces one draft cart per
  buyer with `ux_orders_one_draft_per_user`.
- The same migration enforces one row per product per order with
  `uq_order_items_order_id_product_id`.
- Cart mutations and checkout take a transaction-scoped PostgreSQL advisory
  lock for the buyer via `OrderRepo.LockUserCart`.
- Add/update cart operations check `Product.AvailableQuantity()`, but do not
  reserve stock until checkout.

## Checkout

`OrderService.Checkout` is buyer-only and runs in one transaction:

1. Lock the buyer cart with an advisory transaction lock.
2. Load the selected address and verify that it belongs to the buyer.
3. Load the current draft cart.
4. Atomically claim the cart with `draft -> pending` using
   `UpdateOrderStatus`. If no row matches, another transaction already claimed
   or changed it.
5. Re-read order items in the transaction and reject an empty cart.
6. Lock product rows with `FOR UPDATE` in ascending product ID order.
7. Reject deleted products and insufficient stock.
8. Reserve stock with `ChangeStockAndReserved(productID, 0, +qty)`.
9. Fix `price_at_purchase` to the current product price.
10. Group items by product seller.

If all items belong to one seller, the existing cart row becomes the pending
order: service sets `seller_id`, `address_id`, and `total_amount`.

If items belong to multiple sellers, service creates one pending order per
seller, copies the relevant items, and deletes the original draft cart. The
original cart's items are removed by the `order_items.order_id` foreign key
`ON DELETE CASCADE`. Returned order IDs are not a stable ordering contract.

## Payment And Expiration

Payments are documented in `docs/payments.md`.

The current payment deadline is computed as:

```text
orders.created_at + payment.ttl
```

This is important because single-seller checkout reuses the draft cart row, so
its deadline is based on cart creation time. Multi-seller checkout creates new
pending orders, so those deadlines start at checkout time.

`OrderExpirationWorker` runs every `orders.expiration_check_interval`, computes
`deadline = now - payment.ttl`, selects pending orders with
`created_at <= deadline`, transitions them to `cancelled`, and releases reserved
stock. If a concurrent payment already changed the status, the worker skips that
order.

## Stock Reservation

| Operation | Stock Delta | Reserved Delta | Notes |
| --- | ---: | ---: | --- |
| Add/update draft item | 0 | 0 | Checks availability only. |
| Checkout | 0 | `+qty` | Reserves product quantity for pending/paid order. |
| Payment | 0 | 0 | Status only. |
| Buyer/admin cancel pending or paid | 0 | `-qty` | Releases reservation. |
| Expire pending | 0 | `-qty` | Releases reservation. |
| Ship paid order | `-qty` | `-qty` | Converts reservation into real stock decrement. |
| Deliver shipped order | 0 | 0 | Status only. |

The database constraint `reserved_quantity <= stock_quantity` is a safety net.
Service code still locks products before stock mutations and uses sorted lock
ordering to avoid deadlocks across checkout/cancel/expire/ship.

## Access Rules

- Buyers can list and read only their own orders.
- Sellers can read/ship/deliver orders whose `seller_id` matches their seller
  profile.
- Admin can read orders and can cancel pending/paid orders through the service.
- Review purchase checks count only `paid`, `shipped`, and `delivered` orders.

Handlers and web controllers delegate these rules to services. Do not duplicate
or bypass lifecycle policy in transport code.
