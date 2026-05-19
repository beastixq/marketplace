# Deleted Product Review and API Gaps

Status: fixed
Fixed: 2026-04-30
Commits: [`ddcb3b8`](https://github.com/beastixq/marketplace/commit/ddcb3b8)
Area: products / reviews / API

## Bug

На soft-deleted product можно было создать новый review, если покупатель
раньше купил товар. API также не отдавал `deleted_at` в product DTO, из-за
чего admin/order contexts не могли явно отличить снятый с продажи товар.

## Cause

`ReviewService` не получал product state перед созданием review, а
`ProductDTO` скрывал soft-delete marker.

## Fix

- `ReviewService` получил narrow `ReviewProductGetter`.
- `CreateReview` загружает product и возвращает `ErrProductDeleted`,
  если `deleted_at` заполнен.
- Existing reviews остаются читаемыми.
- `ProductDTO` получил `deleted_at`.

## Verification

- Обновлены `review_service_test.go` и RBAC tests.
