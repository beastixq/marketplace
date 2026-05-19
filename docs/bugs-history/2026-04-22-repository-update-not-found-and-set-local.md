# Repository Update Mislabels Not-Found and SET LOCAL Syntax Error

Status: fixed
Fixed: 2026-04-22
Commits: [`f9f7218`](https://github.com/beastixq/marketplace/commit/f9f7218)
Area: repository / errors

## Bug

`UpdateX` в репозиториях возвращал scan/internal error вместо
domain-level not-found. Отдельно `SET LOCAL app.current_user = $1` ломал
price-history tests из-за конфликта с зарезервированным SQL-именем.

## Cause

При `UPDATE ... RETURNING` отсутствие строки приходит как `pgx.ErrNoRows`
из `Scan`, но код оборачивал это как `ErrToScan`. Для audit-контекста
использовалась форма `SET LOCAL` с ключом `current_user`, который
PostgreSQL воспринимал как зарезервированное имя.

## Fix

- Во всех update-методах добавлена проверка
  `errors.Is(err, pgx.ErrNoRows)` и возврат `ErrNotFound`.
- `SET LOCAL` заменён на
  `SELECT set_config('app.current_user', $1, true)`.

## Verification

- Добавлены негативные repository-тесты для `UpdateX_NotFound`,
  `GetByID_NotFound`, duplicate constraints и пагинации.
- `TestProductRepo_PriceHistory` разблокирован.
