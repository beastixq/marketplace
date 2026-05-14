# Known Issues

Открытые баги и подтвержденные технические риски. Сырые заметки сначала можно
писать в `notes.todo.md`; после первичного разбора переносить сюда.

## BUG-001 Payment TTL Starts From Draft Cart Creation

Status: open
Severity: medium
Area: orders/payments
Found: 2026-05-13

Problem:
`PaymentService` считает дедлайн оплаты как `orders.created_at + payment.ttl`.
При single-seller checkout сервис переиспользует существующую draft cart row,
поэтому pending order наследует `created_at` корзины. Если корзина была создана
давно, окно оплаты может оказаться сильно меньше ожидаемого или уже истекшим.

Expected:
Окно оплаты должно начинаться от момента checkout или от отдельного момента
создания payment window, а не от создания draft cart.

Known Context:
- Multi-seller checkout создает новые pending orders, поэтому там `created_at`
  ближе к моменту checkout.
- Обновлять `created_at` при checkout нежелательно: поле начинает терять смысл
  времени создания записи.

Options:
- Добавить отдельное поле вроде `checked_out_at` или `payment_expires_at`.
- Всегда создавать новую pending order при checkout, даже для одного продавца.
- Обновлять `created_at` при checkout только как временный компромисс.

Decision:
TBD.

## BUG-002 Absolute Stock Update Can Override Seller's Stale Intent

Status: open
Severity: medium
Area: products/orders
Found: 2026-05-13

Problem:
Форма продавца редактирует `stock_quantity` как абсолютное значение. Если
продавец открыл форму при `stock_quantity = 100`, решил уменьшить запас до `90`,
а пока форма была открыта заказ/отгрузка изменили остаток до `96`, отправка формы
запишет `90`. Это безопасно с точки зрения row lock и constraint
`reserved_quantity <= stock_quantity`, но может потерять намерение продавца
"уменьшить на 10" и фактически уменьшить только на 6 относительно свежего
состояния.

Expected:
Нужно явно выбрать бизнес-семантику редактирования склада:

- абсолютное "установить stock_quantity в N";
- относительное "увеличить/уменьшить stock_quantity на delta";
- optimistic concurrency: абсолютное значение принимается только если продавец
  редактировал актуальную версию товара.

Known Context:
- `ProductService.UpdateProduct` берет `FOR UPDATE` lock и проверяет, что новое
  значение не ниже `reserved_quantity`.
- `ProductRepo.UpdateProduct` сейчас делает `SET stock_quantity = $value`.
- Технической гонки записи без lock нет; проблема именно в stale form / lost
  intent.

Options:
- Добавить `version` или `updated_at` check в форму и API update.
- Перевести UI склада на delta operation для пополнения/списания.
- Оставить абсолютное значение, но явно показать продавцу предупреждение и
  свежий `reserved_quantity`/available stock.

Decision:
TBD.

## BUG-003 Category Search Resets Product Creation Form Fields

Status: open
Severity: low
Area: web/products/categories
Found: 2026-05-13

Problem:
На форме создания продукта поиск категории обновляет HTML-страницу, и уже
заполненные поля (`name`, `description`, `price`, `stock_quantity` и другие)
сбрасываются.

Expected:
Поиск/фильтрация категорий не должен уничтожать введенные пользователем данные.

Options:
- Передавать текущие значения формы через query/form state при поиске категории.
- Делать поиск категорий отдельным endpoint/fragment without full form reset.
- Упростить UX: показывать категории без отдельной перезагрузки страницы.

Decision:
TBD.

## BUG-004 Product Cache Wrap Is Disabled While Docs Claim It Is Live

Status: open
Severity: medium
Area: cache/products/docs
Found: 2026-05-14
Source: ultrareview merged_bug_002

Problem:
В `cmd/api/main.go:77-80` вызов `cache.NewProductRepoCache(...)` закомментирован,
поэтому даже при `redis.enabled=true` API-бинарь подключается к Redis, делает
`Ping`, но никогда не оборачивает `ProductRepo`. При этом `docs/cache.md`,
`docs/architecture.md` и `docs/project-map.md` описывают `products:{id}` 5m кэш
как реально работающий, а таблица Cache/Redis в `AGENTS.md` все еще перечисляет
четыре ключа (`products:catalog:page:{n}`, `categories:tree`, `sessions:{token}`
и инвалидацию `products:{id}` при создании/обновлении ревью), которых в коде
нет.

Дополнительно: compile-time проверки интерфейсов в
`internal/cache/product_repo_cache.go:24-30` (`_ svc.ProductRepo` и
`_ svc.ProductCategoryRepo`) тоже закомментированы — теряется страховка,
о которой говорит AGENTS.md.

Expected:
Документация должна совпадать с runtime. Либо включить обертку (и
ассерты), либо вырезать упоминания неимплементированных ключей.

Options:
- Раскомментировать обертку в `cmd/api/main.go` и ассерты в
  `product_repo_cache.go`, привести таблицу AGENTS.md к реальности
  (только `products:{id}` с инвалидацией PATCH/DELETE).
- Оставить обертку выключенной, но убрать `redis.enabled` ветку из main и
  откатить документацию (cache.md / architecture.md / project-map.md) до
  «cache is planned, not implemented».

Decision:
TBD.

## BUG-005 ChangeStockAndReserved Skips Cache Invalidation

Status: open
Severity: medium
Area: cache/products
Found: 2026-05-14
Source: ultrareview bug_001

Problem:
`ProductRepoCache.ChangeStockAndReserved` (internal/cache/product_repo_cache.go)
это чистый pass-through: он не вызывает `invalidateProduct` после успешного
изменения. Кэшированный `m.Product` содержит `stock_quantity`,
`reserved_quantity` и производное `AvailableQuantity()`. Когда обертка
будет включена (см. BUG-004), каждое оформление заказа, отмена, отгрузка и
expire будут оставлять кэш с устаревшим доступным остатком до TTL (5 минут).

Цепочка проявления: `OrderService.AddItemToCart` читает товар через
`productRepo.GetProductByID` (через кэш) и пропускает проверку
`AvailableQuantity() < quantity`, а потом checkout с `FOR UPDATE` отдает
`ErrInsufficientStock` — пользователь видит товар как доступный, а на
checkout получает отказ.

Expected:
После успешного `ChangeStockAndReserved` инвалидировать `products:{id}`,
как это уже делают `UpdateProduct` и `DeleteProductByID`.

Decision:
Fix: одна строка — добавить `c.invalidateProduct(ctx, productID)` после
успешного inner-вызова. Перед фиксом дождаться решения по BUG-006 (порядок
инвалидации относительно транзакции).

## BUG-006 Cache Invalidation Runs Before Outer Transaction Commits

Status: open
Severity: medium
Area: cache/transactions
Found: 2026-05-14
Source: ultrareview bug_007

Problem:
`ProductRepoCache.UpdateProduct` и `DeleteProductByID` вызывают
`invalidateProduct` сразу после возврата внутреннего репо-метода. Но
`ProductService.UpdateProduct` оборачивает этот вызов в
`txManager.WithTransaction`, и репозиторий повторно использует внешнюю
транзакцию (см. 2026-05-01 nested-tx fix). В результате `DEL products:{id}`
уходит в Redis в тот момент, когда `UPDATE products ...` еще не закоммичен.

Гонка:
1. T1 (writer) начинает tx, выполняет `UPDATE`, выполняет `DEL products:42`.
2. T2 (reader) делает `GetProductByID(42)`: cache MISS → loader делает
   `SELECT` в своей tx и читает старую строку (T1 не закоммитил).
3. T2 пишет старое значение обратно в `products:42` с TTL 5m.
4. T1 коммитит. DB=новое, cache=старое до истечения TTL.

Expected:
Инвалидация должна происходить **после** коммита внешней транзакции.

Options:
- Добавить `AfterCommit(func())` хук в `PgxTxManager`, регистрировать
  `c.invalidateProduct` через него; если ambient tx нет — выполнять сразу.
- Поднять инвалидацию на сервисный уровень: сделать кэш-декоратор над
  `ProductService` вместо `ProductRepo`, и вызывать `DEL` уже после
  `txManager.WithTransaction`.
- Double-delete (до и после с задержкой) — частичная мера.

Decision:
TBD. Архитектурное изменение (хук в TxManager) — не однострочник.

## BUG-007 Cache Invalidation Uses Cancellable Request Context

Status: open
Severity: low
Area: cache/products
Found: 2026-05-14
Source: ultrareview bug_005

Problem:
`ProductRepoCache.invalidateProduct(ctx, id)` использует ctx запроса.
Если клиент отвалится или сработает deadline между коммитом БД и
завершением Redis `DEL`, go-redis вернет `context.Canceled`/`DeadlineExceeded`,
`DEL` пропадет, кэш останется с устаревшим значением до TTL. В логах
останется только `slog.Warn("cache invalidation failed", ...)`.

Expected:
Инвалидация должна быть устойчивой к отмене запроса: использовать
`context.WithoutCancel(ctx)` (Go 1.21+) или короткий `context.WithTimeout`
на чистом фоне.

Decision:
Fix: однострочник — `delCtx := context.WithoutCancel(ctx)` и передать его
в `c.rdb.Del`. Можно делать одновременно с BUG-006.

## BUG-008 Corrupted Cache Entry Persists Until TTL

Status: open
Severity: low
Area: cache
Found: 2026-05-14
Source: ultrareview bug_006

Problem:
В `internal/cache/cacheaside.go:62-68` при ошибке `json.Unmarshal` кэшированного
блоба `GetOrLoad` логирует warning и идет в loader, но не удаляет битый ключ и
не перезаписывает его свежим значением. Каждый следующий запрос на тот же
ключ снова попадает в hit → unmarshal fail → warning → loader, и так до
истечения TTL (5 минут).

Реалистичные триггеры: изменение формы структуры `m.Product` между релизами
без сброса Redis, частичная запись, конфликт ключей с другим приложением,
делящим Redis DB.

Expected:
При ошибке unmarshal удалить ключ (fire-and-forget) или перезапустить
miss-ветку (loader → marshal → Set), чтобы один битый блоб давал максимум
один деградированный запрос.

Decision:
Fix: добавить `rdb.Del(ctx, key)` в обработчике unmarshal error.

## BUG-009 Makefile .PHONY Regression

Status: open
Severity: low
Area: build/makefile
Found: 2026-05-14
Source: ultrareview bug_035

Problem:
В `Makefile:9` объявление `.PHONY` было перезаписано новым списком вместо
дополнения. Цели `all`, `png`, `svg`, `clean` больше не помечены phony:
если на корне репозитория случайно появится файл с одним из этих имен,
`make` пропустит рецепт. Отдельно: `run` присутствует в `.PHONY`, но
самого правила `run:` нет — `make run` падает с `No rule to make target 'run'`.

Expected:
Один общий `.PHONY` со всеми phony-целями.

Decision:
Fix: объединить списки —
`.PHONY: fmt vet test test-race lint security check test-repository test-service test-web all png svg clean`,
и либо удалить `run` из `.PHONY`, либо добавить рецепт `run:` (например,
`go run ./cmd/api -config config/config.yaml`).
