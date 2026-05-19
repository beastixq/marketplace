# Service layer rules

- This layer owns business logic.
- Order status transitions belong here.
- Repository layer must not decide business transitions.
- Payment processing must be idempotent.
- Stock reservation/release must be transactionally safe.
- Add or update service tests for business rule changes.
- Prefer table-driven tests.
