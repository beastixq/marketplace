# Bugs History

История закрытых багов и их решений. Открытые баги хранятся в
[`docs/known-issues.md`](../known-issues.md), продуктовые идеи — в
[`docs/ideas.md`](../ideas.md).

Одна запись = один исправленный баг или одна связанная группа багов. Формат
файла:

```md
# Short Title

Status: fixed
Fixed: YYYY-MM-DD
Commits: [`hash`](https://github.com/beastixq/marketplace/commit/hash)
Area: orders/payments/cache/web/etc.

## Bug
<что ломалось>

## Cause
<почему баг возник>

## Fix
<что изменили>

## Verification
- <какая проверка подтвердила исправление>
```

Имена файлов держим стабильными: `YYYY-MM-DD-short-slug.md`, где дата — дата
финального фикса. Если баг закрывался несколькими коммитами, ранние коммиты
остаются в поле `Commits`, а имя файла берёт дату последнего фикс-коммита.

## Closed Bugs

| Fixed | Area | Title |
| --- | --- | --- |
| 2026-03-22 | repository / service / errors | [Early Repository and Service Error Boundary Gaps](2026-03-22-early-repository-service-error-boundaries.md) |
| 2026-03-25 | seed / migrations / seller-orders | [Seed Seller Orders and Migration Syntax Issues](2026-03-25-seed-seller-orders-and-migration-syntax.md) |
| 2026-03-27 | orders / transactions | [Cart and Checkout Writes Were Not Atomic](2026-03-27-cart-and-checkout-writes-not-atomic.md) |
| 2026-03-27 | API / pagination | [Pagination Parsing Was Inconsistent Across Endpoints](2026-03-27-pagination-parsing-inconsistency.md) |
| 2026-03-28 | orders / products / users | [Backend Multi-Fix Post-GUI](2026-03-28-backend-multi-fix-post-gui.md) |
| 2026-03-28 | seed / performance | [Review Seeder Retry Loop Could Stall](2026-03-28-review-seeder-retry-loop-performance.md) |
| 2026-03-28 | web / RBAC / UX | [Web Role Separation and Profile Error UX Gaps](2026-03-28-web-role-separation-and-profile-error-ux.md) |
| 2026-03-31 | payments / orders | [Payment Flow Was Modeled as Direct Status Mutation](2026-03-31-payment-flow-direct-status-mutation.md) |
| 2026-04-22 | repository / errors | [Repository Update Mislabels Not-Found and SET LOCAL Syntax Error](2026-04-22-repository-update-not-found-and-set-local.md) |
| 2026-04-24 | inventory / orders | [Checkout Did Not Reserve Stock](2026-04-24-checkout-did-not-reserve-stock.md) |
| 2026-04-25 | orders / concurrency | [Double Reservation Release on Concurrent Cancel/Ship](2026-04-25-double-reservation-release-concurrent-cancel-ship.md) |
| 2026-04-25 | orders / cart / concurrency | [Cart Mutations Race With Checkout](2026-04-25-cart-mutations-race-with-checkout.md) |
| 2026-04-25 | orders / payments / concurrency | [Stale Status Writes Resurrect Cancelled Orders](2026-04-25-stale-status-writes-resurrect-cancelled-orders.md) |
| 2026-04-26 | address / schema-migration | [Address `house` Field Not Persisted End-to-End](2026-04-26-address-house-field-not-persisted.md) |
| 2026-04-26 | API / RBAC / service | [API RBAC Only Lived at Route Boundary](2026-04-26-api-rbac-only-at-route-boundary.md) |
| 2026-04-26 | config / security | [Config Validation Allowed Unknown Fields and Weak Values](2026-04-26-config-validation-and-insecure-defaults.md) |
| 2026-04-26 | docs / diagrams | [README and Diagram Artifact Drift](2026-04-26-readme-and-diagram-artifact-drift.md) |
| 2026-04-27 | techui / payments / build | [TechUI Did Not Run Expiration Worker and Component Build Was Misnamed](2026-04-27-techui-expiration-worker-and-component-build.md) |
| 2026-04-28 | orders / products / cart / web | [GUI-Surfaced Edge Cases in Orders, Products, and Cart](2026-04-28-gui-surfaced-order-product-cart-edge-cases.md) |
| 2026-04-30 | seed / data-integrity | [Seed Data Reservation Invariants](2026-04-30-seed-data-reservation-invariants.md) |
| 2026-04-30 | orders / reviews / authorization | [Checkout Address and Review Purchase Rules Missing](2026-04-30-checkout-address-and-review-purchase-rules.md) |
| 2026-04-30 | database / orders | [Draft Cart and Order Item Invariants Only Lived in Service](2026-04-30-draft-cart-and-order-item-db-invariants.md) |
| 2026-04-30 | products / reviews / API | [Deleted Product Review and API Gaps](2026-04-30-deleted-product-review-and-api-gaps.md) |
| 2026-04-30 | inventory / concurrency | [Stock Mutations Without Deterministic Row Locks](2026-04-30-stock-mutations-without-deterministic-row-locks.md) |
| 2026-05-01 | products / transactions | [Product Update TOCTOU and Nested-Tx Self-Block](2026-05-01-product-update-toctou-nested-tx-self-block.md) |
| 2026-05-01 | auth / users | [Soft-Deleted Users Could Authenticate](2026-05-01-soft-deleted-users-could-authenticate.md) |
| 2026-05-07 | docs / diagrams | [BPMN Checkout Diagram Link Used Wrong Artifact Type](2026-05-07-bpmn-checkout-diagram-link.md) |
