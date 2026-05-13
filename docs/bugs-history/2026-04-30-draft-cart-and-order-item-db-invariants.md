# Draft Cart and Order Item Invariants Only Lived in Service

Status: fixed
Fixed: 2026-04-30
Commits: [`85740d9`](https://github.com/beastixq/marketplace/commit/85740d9)
Area: database / orders

## Bug

База позволяла состояние, которое сервис считает невозможным: несколько
draft cart на одного buyer и несколько строк одного `product_id` в одном
order. При обходе сервисного пути или гонке эти инварианты могли
нарушиться на уровне данных.

## Cause

Ограничения существовали в service/repository logic, но не были
закреплены database constraints.

## Fix

- Добавлена migration `011_order_cart_item_invariants.sql`.
- Partial unique index: один `orders(status = 'draft')` на `user_id`.
- Unique constraint: один `(order_id, product_id)` в `order_items`.
- Duplicate order-item insert мапится в domain error
  `ErrProductAlreadyInCart`.

## Verification

- Обновлены docs database/schema.
- Repository insert path теперь переводит constraint violation в typed
  service error.
