# Cart Mutations Race With Checkout

Status: fixed
Fixed: 2026-04-25
Commits: [`02982bf`](https://github.com/beastixq/marketplace/commit/02982bf)
Area: orders / cart / concurrency

## Bug

Операции над корзиной могли продолжить работу после параллельного
checkout и добавить или изменить item уже в `pending`-заказе. В итоге
`price_at_purchase` фиксировался не для того набора строк, который
фактически оставался в заказе.

## Cause

`AddItemToCart`, `ChangeQuantityCartItem`, `DeleteCartItem` и `Checkout`
работали отдельными транзакциями по схеме "прочитал draft, затем поменял".
Координации между операциями одной корзины не было.

## Fix

- В начале транзакций, трогающих корзину пользователя, берётся
  `pg_advisory_xact_lock` с ключом от `userID`.
- Мутации items обёрнуты в транзакцию.
- Добавлены conditional update/delete для `order_items`, требующие
  владельца заказа и статус `draft`.
- `userID` проброшен из handler-слоя для проверки владения.

## Verification

- Обновлены unit-тесты `order_service_test.go` под новые подписи и
  поведение cart-mutations под локом.
