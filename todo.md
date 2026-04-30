## GUI

### P0 — important for lab work / use cases / demo

1. **Analyst reports in GUI** — close analyst use cases.
   Platform stats, orders by status, and top products by revenue already exist.
   BL/repository methods for order dynamics and sales by category are already added.
   Remaining work: show these reports in the Web GUI.
   This matters because these use cases are explicitly present in `diagrams/src/UseCases.puml`.

2. **Product categories in seller/admin GUI** — close product categorization flow.
   Seller must choose categories while creating/editing a product.
   Admin must moderate product-category links.
   Current state: `product_categories` schema exists, catalog filter exists, seed creates links, but product create/edit Web UI does not set categories.

### P2 — useful, but not blocking lab work

1. **Category pagination/list refresh**.
   Admin categories currently load a large temporary limit so new categories are visible.
   Need normal page/search/filter for public/admin category lists.

2. **Frontend checklist manual pass**.
   Manually check: admin panel, analyst dashboard, catalog filters, price history, seller flows, buyer flows.
   Check Price Change Graph: after product price change, graph should show old-to-new price transition without seed "initial price" record.
   Important for submission, but does not need heavy e2e framework.

3. **Web cleanup after `ProductService.UpdateProduct` lock**.
   As part of P1 #1 BL fix, `NewProductService` received `TxManager` dependency.
   `internal/web/web_handler_test.go` now has local `passThroughTxManager` mock.
   This works as a stub, but is not good long-term design. Decide:
   - move this mock to shared test helper;
   - or revisit DI so web tests do not depend on tx infrastructure
     (for example, a separate use case for seller stock update).
   Web handlers themselves were not changed; only test factory changed. BL fix is clean.

### P4 — low priority for educational project

1. **CSRF for web forms**.
   Important for production, not blocking current lab work.

2. **E2E GUI tests**.
   Useful for project health, but gives little value for current submission.
   Manual checklist plus unit/template tests is enough.

3. **UI polish**.
   Small copy, spacing, empty states, disabled states.
   Do after P0/P1 are closed.

## BL - business logic (service)

### P1 — important business invariants and concurrency

1. ~~**More informative worker/order lifecycle errors**~~ — done.
   `ExpireOrders` now returns `*ExpireOrderError` with structured context
   (`OrderID`, `ProductID`, `Quantity`, `ReservedQuantity`, `Stage`, `Cause`).
   `OrderExpirationWorker` unwraps joined errors and emits one `slog.Error`
   per failure with named fields. Service stays log-free; logging happens at worker boundary.

2. ~~**Order state machine**~~ — done.
   Allowed order status transitions are centralized in a service-layer table.
   `OrderService`/`PaymentService` use it for validation and CAS update.

3. ~~**Deleted/banned user**~~ — done.
   `Login` rejects soft-deleted users with `ErrAccountDeactivated` (401).
   `ValidateToken` loads the user via `GetAuthUserByID` after JWT/blocklist
   checks and rejects with the same error if `DeletedAt != nil` or the
   user no longer exists. Cost: one DB lookup per authenticated request;
   acceptable for this project. New sentinel mapped to 401 in handler.

### P2 — useful, but not blocking lab work

1. **Transaction boundary in business logic**.
   Service currently opens transactions through `TxManager`.
   Decide whether to keep this or move transaction boundary to separate use case/transaction layer.

### P3 — architectural improvements for later

1. **Use cases instead of god services**.
   Consider splitting services into narrow interfaces:
   ```go
   type CheckoutUseCase interface {
       Execute(ctx context.Context, userID int64, addressID int64) ([]int64, error)
   }
   ```
   Not critical for current lab work.

## DA - data access (repository)

### P1 — important business invariants and concurrency

1. **Repository error boundary**.
   Repositories should translate `pgx.ErrNoRows` and `pgconn.PgError` unique/check/FK errors by constraint name into domain sentinel errors.
   Some errors are already translated, but not everywhere.

### P2 — useful, but not blocking lab work

1. **SQL improvements**.
   Consider where CTE/window queries are appropriate.
   Do not add them for style only.

### P3 — architectural improvements for later

1. **`SET LOCAL ROLE` inside transaction**.
   Investigate whether PostgreSQL roles should really be used at database level.

## Combined BL+DA

### P0 — important for lab work / use cases / demo

1. **Redis TokenBlocklist for logout** — connect real blocklist.
   `TokenBlocklist` interface already exists, but `cmd/api/main.go` passes `nil`.
   Need to add Redis to config/docker-compose, implement blocklist, and wire it into API/techui components.

### P1 — important business invariants and concurrency

1. ~~**Stock update / checkout / cancel / ship / expire concurrency**~~ — done.
   `UpdateProduct` now runs in tx, locks product with `FOR UPDATE` through
   `GetProductByIDForUpdate`, and validates stock-vs-reserved after lock using fresh data.
   Stock semantics stay absolute; delta operations were not introduced.

2. ~~**Cancel/ship/expire concurrency**~~ — done.
   `CancelOrder`, `ExpireOrders`, and `ShipOrder` lock all affected products
   by ascending id inside tx before `ChangeStockAndReserved`.
   This is deadlock-free across all product mutation paths, including checkout.

3. **Draft cart price — DB/API/techui model**.
   Web UI already shows current price from `products`; checkout writes current `price_at_purchase`.
   Remaining work:
   - `GetCart` in service/API/techui still calculates total from `price_at_purchase`;
   - long-term, make `order_items.price_at_purchase` nullable for draft items;
   - at checkout, write snapshot price and calculate `total_amount`.

4. **Address schema drift**.
   `house` is already wired through model/DTO/repository/web.
   `apartment` exists in migration `010_add_address_house.sql`, but is not wired through model/DTO/repository/forms/seed.
   Decide: support `apartment` everywhere, or remove it from schema with separate migration.

### P2 — useful, but not blocking lab work

1. **Shared app errors without repo -> service cycle**.
   Move `ErrNotFound` and common sentinel errors to `internal/apperr` or similar package.
   Repositories currently import `internal/service`; architecturally ugly, acceptable for lab work.

2. **Denormalized rating/total_amount**.
   Check where `rating` and `total_amount` can drift because of manual SQL/seed/bugs.
   This is fine for performance, but invariants should be clear.

### P3 — architectural improvements for later

1. **Port interfaces**.
   Decide whether interfaces dictated by services should stay in `internal/service` or move to `internal/port`.

2. **ACID review of current transactions**.
   Check isolation and invariants for checkout/cancel/ship/expire/update stock.

## Other

### P0 — important for lab work / use cases / demo

1. ~~**Deleted products — remaining edge cases**~~ — done.
   Web shows deleted products where needed, add-to-cart/checkout are forbidden,
   cart blocks checkout with deleted item, `ProductDTO.deleted_at` is exposed
   for order-detail/admin clients, review on deleted product is rejected through
   `ErrProductDeleted` (409). Old pending/paid orders and lifecycle were checked:
   repository joins do not filter `deleted_at`, nothing broke. Seller own-product
   list keeps hiding deleted products (current behavior); admin sees everything.
   Re-delete is idempotent.

### P2 — useful, but not blocking lab work

1. **Load / concurrency testing**.
   Measure how many concurrent requests server handles before latency/error rate degrade.
   Do not call this "hardware testing"; correct terms:
   - **load testing** — expected load, measure latency/throughput;
   - **stress testing** — push above limit, find breaking point;
   - **capacity / concurrency testing** — our "how many RPS" question;
   - **soak testing** — long run, find leaks/degradation.
   Tools: `k6` (JS scripts, convenient for Go services),
   `vegeta` (Go-native, simple CLI), `hey`, `wrk`, JMeter, Locust.
   Metrics: p50/p95/p99 latency, RPS, error rate, DB pool exhaustion,
   `FOR UPDATE` contention on checkout/cancel. Scenarios:
   catalog GET (read-heavy), concurrent checkout of one product
   (write contention), mixed buyer/seller flow.
   Not blocking lab work, but useful for report/defense.

## Needs Decision

1. **Stock UI semantics**.
   Keep absolute `stock_quantity`, or switch to "increase/decrease by N" operations?

2. **Address apartment**.
   Support `apartment` everywhere, or remove it from schema?

3. **Deleted product policy**.
   Allow review on deleted product if buyer actually bought it before?

4. **Sales by category attribution**.
   If product is linked to multiple categories, count its revenue in each category or require one primary category?
   Current BL/repository report counts revenue in every linked category.

---
