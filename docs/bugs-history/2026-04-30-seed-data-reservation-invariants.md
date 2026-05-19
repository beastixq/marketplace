# Seed Data Reservation Invariants

Status: fixed
Fixed: 2026-04-30
Commits: [`4c57edb`](https://github.com/beastixq/marketplace/commit/4c57edb)
Area: seed / data-integrity

## Bug

Seeder создавал состояние, недостижимое через сервисный слой: дубликаты
товаров внутри одного заказа, non-draft заказы без items, неверный
`total_amount` и несинхронизированный `product.reserved_quantity`.

## Cause

Генератор order items выбирал продукты независимо и мог повторять
`product_id`. После генерации items не было обязательного пересчёта
сумм и reserve-агрегации для pending/paid заказов.

## Fix

- Items выбираются без дубликатов через deterministic sampling.
- Для всех non-draft заказов гарантируются items.
- `total_amount` пересчитывается после генерации строк.
- `reserved_quantity` синхронизируется пост-проходом по pending/paid
  заказам.

## Verification

- Seeder запускался на чистой БД, затем вручную проверялись агрегаты по
  orders/order_items/products.

## Notes

Это bug seed data, а не runtime-кода. Но битое стартовое состояние
маскировало настоящие reservation bugs.
