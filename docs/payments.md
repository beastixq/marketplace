# Payments

The current payment implementation is a mock-bank flow for local development and
coursework demos. It does not persist payment records and does not talk to a
real bank.

## Components

| Component | Path | Responsibility |
| --- | --- | --- |
| Payment service | `internal/service/payment_service.go` | Validates buyer/order/payment state and transitions paid orders. |
| Payment port | `internal/port/payment_gateway.go` | Defines `GetPaymentURL` and `ProcessPayment`. |
| Mock gateway | `internal/adapter/payment/mock_gateway.go` | Encodes/decodes mock bank tokens and returns `/mock_bank/payment` URLs. |
| JSON handler | `internal/handler/payment.go` | Exposes payment link and callback endpoints. |
| Web flow | `internal/web/payment.go`, `internal/web/orders.go` | Redirects buyers to mock bank and processes the mock result. |

## Configuration

| Config | Meaning |
| --- | --- |
| `payment.ttl` | Time window in which a pending order may be paid. Local default is `2m`. |
| `payment.gateway_url` | Base URL used by the mock gateway when building payment links. Local default is `http://localhost:8080`. |
| `orders.expiration_check_interval` | How often the background worker expires old pending orders. |

`payment.gateway_url` must be an absolute URL. Config validation rejects empty,
relative, or malformed values.

## API And Web Routes

| Surface | Route | Purpose |
| --- | --- | --- |
| JSON API | `POST /api/v1/orders/{id}/payment-link` | Buyer creates a payment link for their own pending order. |
| JSON API | `POST /api/v1/payments/callback/mock-bank` | Public mock-bank callback with `{"token":"..."}`. |
| Web | `POST /orders/{id}/pay` | Buyer starts payment and is redirected to the mock bank page. |
| Web | `GET /mock_bank/payment?token=...` | Mock bank payment page. |
| Web | `POST /mock_bank/payment` | Mock bank success/decline submit. |

`OrderService.PayOrder` exists, but the JSON API router does not currently mount
a direct `/api/v1/orders/{id}/pay` route. The tech UI can call this service path.

## Payment Link Flow

1. Checkout creates one or more `pending` orders.
2. Buyer requests a payment link for one order.
3. `PaymentService.GetOrderPaymentURL` verifies:
   - actor has buyer role;
   - order exists;
   - order belongs to the buyer;
   - order status is `pending`;
   - current time is before `order.CreatedAt + payment.ttl`.
4. Mock gateway encodes `{order_id, amount, success}` as URL-safe base64 JSON.
5. The payment URL points to:

```text
{payment.gateway_url}/mock_bank/payment?token={token}
```

The mock token is an implementation detail for local demos. It is base64 JSON,
not a signed or encrypted payment credential.

## Callback Processing

`PaymentService.ProcessOrderPayment` performs the state change:

1. Ask the gateway to process/decode the token.
2. Load the order from PostgreSQL.
3. Return success without changes if the order is already `paid` or
   `cancelled`.
4. Require current status `pending`.
5. Require current time before `order.CreatedAt + payment.ttl`.
6. Require gateway result `Success == true`.
7. Require gateway amount to equal `orders.total_amount`.
8. Atomically transition `pending -> paid` with `UpdateOrderStatus`.

If the final status update loses a race to another payment or cancellation,
service re-reads the order. A resulting `paid` or `cancelled` status is treated
as an idempotent success.

## Errors And Status Codes

| Service Error | Typical Cause | HTTP Status |
| --- | --- | --- |
| `ErrPermissionDenied` | Non-buyer requests payment link. | `403 Forbidden` |
| `ErrNotYourOrder` | Buyer requests another user's order. | `403 Forbidden` |
| `ErrOrderStatusInvalid` | Order is not pending when payment is requested/processed. | `409 Conflict` |
| `ErrPaymentExpired` | Current time is outside the payment TTL. | `410 Gone` |
| `ErrPaymentDeclined` | Mock bank submitted a declined result. | `422 Unprocessable Entity` |
| `ErrInvalidPaymentAmount` | Token amount does not match order total. | `422 Unprocessable Entity` |

The web flow redirects back to the order page with `payment=success` or
`payment=failed&payment_error=...`.

## Current Limitations

- No `payments` table exists; external IDs and failure reasons are not stored.
- The mock token can be modified by a client and is suitable only for local
  development/demo use.
- No webhook signature, retry queue, reconciliation job, or payment audit log is
  implemented.
- Payment TTL is based on `orders.created_at`, not on payment-link creation time.
  For single-seller checkout, the pending order reuses the draft cart row, so a
  long-lived cart can have little or no remaining payment window.
- Expiration and payment race through status compare-and-swap. This prevents
  double finalization but does not record a separate payment attempt.

When replacing the mock bank with a real provider, keep provider-specific
payloads in an adapter and keep order/payment policy in the service layer.
