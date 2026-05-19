# Checkout Address and Review Purchase Rules Missing

Status: fixed
Fixed: 2026-04-30
Commits: [`ff116b2`](https://github.com/beastixq/marketplace/commit/ff116b2)
Area: orders / reviews / authorization

## Bug

Checkout мог принять `address_id`, который не принадлежит текущему
покупателю. Review creation также не требовал доказательства покупки
товара пользователем.

## Cause

`OrderService.Checkout` не имел зависимости для проверки адреса и
доверял переданному `address_id`. `ReviewService.CreateReview` проверял
только review-level данные, но не purchase history по paid/shipped/
delivered заказам.

## Fix

- В `OrderService` добавлен address repository interface для проверки
  владельца адреса перед checkout.
- В `ReviewRepo` добавлен `UserPurchasedProduct`.
- `ReviewService.CreateReview` разрешает review только если пользователь
  покупал товар в заказе со статусом `paid`, `shipped` или `delivered`.
- Handler error mapping и API docs обновлены под новые ошибки.

## Verification

- Обновлены `order_service_test.go`, `review_service_test.go`,
  `review_repo_test.go` и RBAC tests.
