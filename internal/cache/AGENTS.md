# Cache rules

- Redis is not source of truth.
- Cache errors must not break normal read path.
- Cache writes may fail without failing the read operation.
- Use explicit TTL.
- Marshal format is JSON unless explicitly changed.
- Do not cache auth-sensitive data unless documented.
