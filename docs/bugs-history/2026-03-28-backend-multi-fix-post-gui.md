# Backend Multi-Fix Post-GUI

Status: fixed
Fixed: 2026-03-28
Commits: [`16126d8`](https://github.com/beastixq/marketplace/commit/16126d8)
Area: orders / products / users

## Bug

После первого подключения GUI всплыла пачка backend-дефектов:
`ShipOrder` не списывал физический stock, `UpdateUser` не мапил
unique-violation в typed errors, пользовательские заказы отдавались без
пагинации и сортировки, каталог нельзя было фильтровать по seller, а
rating пересчитывался в сервисе и мог рассинхронизироваться.

## Cause

Часть правил существовала только на уровне UI-сценариев или сервиса, но
не была закреплена в transaction boundary, repository mapping или DB
trigger. GUI начал проходить реальные end-to-end флоу и показал эти
разрывы.

## Fix

- `ShipOrder` декрементит stock в транзакции и проверяет наличие остатка.
- Unique-violation для phone/email мапится в typed service errors.
- `GetOrdersByUserID` получил `limit`/`offset` и
  `ORDER BY created_at DESC`.
- `CatalogOptions.SellerID` поддержан в product repository.
- Rating продукта и продавца пересчитывается DB trigger из migration 007.

## Verification

- Обновлены `order_service_test.go` под новые сигнатуры и stock logic.
