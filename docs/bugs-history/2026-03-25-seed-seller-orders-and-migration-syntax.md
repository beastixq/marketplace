# Seed Seller Orders and Migration Syntax Issues

Status: fixed
Fixed: 2026-03-25
Commits: [`b88181d`](https://github.com/beastixq/marketplace/commit/b88181d)
Area: seed / migrations / seller-orders

## Bug

Seeded orders могли не иметь корректного `seller_id`, а order items могли
попадать в заказ не того seller. Migration 006 также имела syntax issue.
Дополнительно удаление заказа требовало cascade behavior для order items.

## Cause

После добавления seller ownership в orders генераторы не были полностью
согласованы с новым инвариантом "один order принадлежит одному seller".
Миграция и cascade rules отставали от новой модели данных.

## Fix

- Генератор orders выставляет `seller_id`.
- Генератор order items выбирает products того же seller для non-draft
  orders.
- Исправлен синтаксис `migrations/006_orders_add_seller_id.sql`.
- В `migrations/001_create_tables.sql` закреплён cascade для order items.

## Verification

- Commit затронул seed generators, order repository и migrations, то есть
  источник данных и schema path были исправлены вместе.
