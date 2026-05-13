# Repository layer rules

- Persistence only.
- No business policy.
- SQL must be parameterized.
- For stock/order/payment changes, check transaction boundaries and locking.
- Translate database errors into repository/domain-level errors where practical.
- Do not expose DB row structs outside repository.
- Add repository tests for new queries.
