# Migration rules

- Migrations must be deterministic.
- Do not edit old migrations unless explicitly requested.
- Add new migration files with the next numeric prefix.
- For order/payment/stock schema changes, update docs/database.md and tests.
- Consider indexes for new foreign keys and query filters.
