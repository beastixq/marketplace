# Review Seeder Retry Loop Could Stall

Status: fixed
Fixed: 2026-03-28
Commits: [`8c6cf3a`](https://github.com/beastixq/marketplace/commit/8c6cf3a)
Area: seed / performance

## Bug

Review seeder мог становиться очень медленным или зависать на больших
объёмах, когда случайный retry-loop всё чаще попадал в уже использованные
`(user_id, product_id)` пары.

## Cause

Генератор пытался набрать уникальные review pairs случайными попытками.
Чем ближе набор к максимуму возможных пар, тем больше retries нужно для
следующей уникальной пары.

## Fix

- Генерация review pairs переведена на deterministic unique pairs через
  `rand.Perm`.
- Batch size увеличен до 500.
- Context timeout увеличен до 3 минут для больших seed datasets.
- Добавлен generator для price history на тех же seed runs.

## Verification

- Commit message фиксирует целевой объём: 50000 reviews и 25000 rows
  price history.
