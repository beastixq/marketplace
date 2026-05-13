# Address `house` Field Not Persisted End-to-End

Status: fixed
Fixed: 2026-04-26
Commits: [`b28fb7d`](https://github.com/beastixq/marketplace/commit/b28fb7d), [`84f841a`](https://github.com/beastixq/marketplace/commit/84f841a)
Area: address / schema-migration

## Bug

После добавления `addresses.house` поле не проходило весь путь от UI и
DTO до repository/seeder. Seeder падал на новой колонке, а адрес,
введённый через UI, мог сохраняться без номера дома.

## Cause

Схемная колонка была добавлена раньше, чем изменение протянули через
`model.Address`, handler DTO, repository DB mapping, SQL
`INSERT`/`UPDATE`/`SELECT`, templates и seed data.

## Fix

- `house` добавлен в DTO, domain model и repository DB model.
- SQL в `address_repo.go` явно читает и пишет поле.
- `addresses.html`, `cart.html`, `order-detail.html` выводят `house`.
- Seeder адресов подставляет `house` в каждый insert.

## Verification

- Обновлены `address_repo_test.go` и `address_service_test.go`.
