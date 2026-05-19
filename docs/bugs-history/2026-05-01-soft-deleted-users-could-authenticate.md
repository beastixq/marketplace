# Soft-Deleted Users Could Authenticate

Status: fixed
Fixed: 2026-05-01
Commits: [`751ad08`](https://github.com/beastixq/marketplace/commit/751ad08)
Area: auth / users

## Bug

Soft-deleted user мог войти по email/password или продолжить работу с
уже выпущенным token. Для системы такой аккаунт был удалён, но auth path
не учитывал `deleted_at`.

## Cause

`Login` проверял credentials, но не блокировал пользователя с
`DeletedAt != nil`. `ValidateToken` доверял payload токена и не
перечитывал актуальное состояние user.

## Fix

- `Login` возвращает `ErrAccountDeactivated` для soft-deleted users.
- `ValidateToken` загружает user по ID и отвергает отсутствующего или
  soft-deleted пользователя.
- Handler mapping переводит `ErrAccountDeactivated` в 401.

## Verification

- Добавлены/обновлены `auth_service_test.go` на login и token validation
  для deleted users.
