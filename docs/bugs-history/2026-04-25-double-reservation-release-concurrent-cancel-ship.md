# Double Reservation Release on Concurrent Cancel/Ship

Status: fixed
Fixed: 2026-04-25
Commits: [`623e168`](https://github.com/beastixq/marketplace/commit/623e168)
Area: orders / concurrency

## Bug

Два конкурентных перехода одного заказа, например два cancel или
cancel + expire, могли оба освободить один и тот же резерв товара.
`reserved_quantity` уменьшался дважды для одного заказа.

## Cause

Переход статуса делался как `SELECT` с проверкой в сервисе, затем
безусловный `UPDATE`. Между чтением и записью второй запрос мог увидеть
тот же старый статус и тоже пройти pre-check.

## Fix

- В репозиторий добавлен атомарный `UpdateOrderStatus(id, from[], to)`
  через `UPDATE ... WHERE status IN (...)`.
- `CancelOrder` и `ShipOrder` при проигранной гонке возвращают
  `ErrOrderStatusInvalid`.
- `ExpireOrders` на `ErrNotFound` пропускает заказ: между снимком и
  транзакцией заказ мог стать `paid`, это валидная гонка.

## Verification

- Обновлены unit-тесты `order_service_test.go` под поведение lost-race.
