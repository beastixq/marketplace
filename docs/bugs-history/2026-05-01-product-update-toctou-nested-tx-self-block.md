# Product Update TOCTOU and Nested-Tx Self-Block

Status: fixed
Fixed: 2026-05-01
Commits: [`2e683cd`](https://github.com/beastixq/marketplace/commit/2e683cd)
Area: products / transactions

## Bug

`UpdateProduct` мог валидировать владение, deleted-state и
`stock_quantity >= reserved_quantity` по устаревшему snapshot. Вторая
проблема: repository update открывал вложенную транзакцию и мог
самоблокироваться на строке, которую уже залочила внешняя транзакция.

## Cause

Сервис сначала делал `GetProductByID`, проверял права и stock-guard, а
только потом открывал транзакцию и брал `SELECT FOR UPDATE`. При этом
`ProductRepo.UpdateProduct` всегда открывал собственную транзакцию, даже
когда уже был вызван из сервисной transaction boundary.

## Fix

- Snapshot-чтение перед транзакцией убрано.
- Авторизация, проверка `DeletedAt` и `stock_quantity >= reserved_quantity`
  выполняются после `GetProductByIDForUpdate` на свежей залоченной строке.
- `ProductRepo.UpdateProduct` переиспользует транзакцию из контекста,
  если она уже есть.

## Verification

- Обновлены unit-тесты `product_service_test.go` под порядок
  `lock -> authorize -> validate -> update`.
