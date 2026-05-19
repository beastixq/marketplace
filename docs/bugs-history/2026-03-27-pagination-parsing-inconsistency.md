# Pagination Parsing Was Inconsistent Across Endpoints

Status: fixed
Fixed: 2026-03-27
Commits: [`517e9bd`](https://github.com/beastixq/marketplace/commit/517e9bd)
Area: API / pagination

## Bug

Endpoints with `page`/`limit` support did not all parse pagination the
same way. Some handlers duplicated validation manually, increasing the
risk of different 400 behavior for the same invalid query.

## Cause

Pagination parsing was copy-pasted into category, product catalog and
product reviews handlers instead of using one shared helper.

## Fix

- `GetCategories`, `GetCatalog` and `GetProductReviews` now use
  `parsePagination`.
- Catalog passes pagination only when `page` is present, preserving the
  existing optional-pagination behavior.

## Verification

- Commit diff shows all pagination branches in those handlers replaced by
  the shared helper.
