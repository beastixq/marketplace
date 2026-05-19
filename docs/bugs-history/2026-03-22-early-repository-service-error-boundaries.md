# Early Repository and Service Error Boundary Gaps

Status: fixed
Fixed: 2026-03-22
Commits: [`f70d431`](https://github.com/beastixq/marketplace/commit/f70d431), [`d907dc8`](https://github.com/beastixq/marketplace/commit/d907dc8)
Area: repository / service / errors

## Bug

В раннем backend repository/service errors плохо проходили через слои:
часть repository errors нельзя было корректно unwrap, not-found из DB не
всегда становился service `ErrNotFound`, `UpdateUser` не проверял
существование user, а password change не валидировал старый пароль.

## Cause

Repository boundary ещё смешивал технические pgx errors и application
errors. User service также не имел полного набора negative-path checks.

## Fix

- Repository errors начали wrapping через `%w`.
- `GetOrderByID`/`GetUserByEmail` стали мапить `pgx.ErrNoRows` в
  service-level not-found.
- `UpdateUser` сначала проверяет существование user.
- `ChangePasswordUser` проверяет old password и возвращает
  `ErrWrongPassword`.
- User service tests добавлены на новые ветки.

## Verification

- `d907dc8` добавил `user_service_test.go`.
- Позднее [`Repository Update Mislabels Not-Found and SET LOCAL Syntax Error`](2026-04-22-repository-update-not-found-and-set-local.md)
  добил аналогичный класс ошибок для `UpdateX`.
