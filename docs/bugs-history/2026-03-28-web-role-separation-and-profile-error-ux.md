# Web Role Separation and Profile Error UX Gaps

Status: fixed
Fixed: 2026-03-28
Commits: [`6553868`](https://github.com/beastixq/marketplace/commit/6553868)
Area: web / RBAC / UX

## Bug

Web UI смешивал buyer и seller flows: sellers видели cart/orders/addresses,
buyers видели seller dashboard, seller мог видеть add-to-cart/review form,
а profile update показывал raw/непонятные ошибки при duplicate phone/email.

## Cause

Role-specific navigation и page guards ещё не были разделены после
появления полноценного GUI. Backend errors для profile update уже
появлялись, но web mapping не давал пользователю понятный feedback.

## Fix

- Buyer/seller routes и navbar разделены по role.
- Seller dashboard разделён на profile/stats, orders и products.
- Seller может управлять shop info, products и incoming orders.
- Product page скрывает add-to-cart/review controls для sellers.
- Duplicate phone/email в profile update отображаются friendly errors.
- Добавлены order detail и вкладки current/completed orders.

## Verification

- Commit затронул web router, web handler и role-specific templates.
