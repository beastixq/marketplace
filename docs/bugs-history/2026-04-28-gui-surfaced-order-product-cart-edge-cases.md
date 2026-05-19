# GUI-Surfaced Edge Cases in Orders, Products, and Cart

Status: fixed
Fixed: 2026-04-28
Commits: [`378ecaa`](https://github.com/beastixq/marketplace/commit/378ecaa)
Area: orders / products / cart / web

## Bug

Сквозное GUI-тестирование показало несколько edge cases: soft-deleted
товары попадали в cart/checkout, seller мог опустить stock ниже reserve,
duplicate review не показывался как понятная ошибка, expiration loop
останавливался на первом проблемном заказе, draft cart отображал
snapshot price вместо live price, а шаблон рейтинга плохо работал с
`*float64`.

## Cause

Некоторые runtime-инварианты были реализованы не в том слое или не были
проверены для web-пути. Templates также смешивали правила draft cart и
уже размещённых заказов.

## Fix

- `AddItemToCart` и `Checkout` отвергают deleted products.
- `UpdateProduct` возвращает `ErrStockBelowReserved`, если новый stock
  меньше текущего reserve.
- Unique violation на reviews мапится в `ErrDuplicateReview`.
- `ExpireOrders` логирует ошибку конкретного заказа и продолжает цикл.
- `cart.html` использует live prices для draft, `order-detail.html`
  оставляет snapshot prices.
- `product.html` корректно разыменовывает рейтинг и показывает empty state.

## Verification

- `web_handler_test.go`: template parsing, product/cart rendering,
  preserve-snapshot проверка для order items.
