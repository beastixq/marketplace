## P0 — важно для ЛР/use cases/demo

1. **Аналитические отчёты GUI** — закрыть use cases аналитика.
   Уже есть platform stats, orders by status, top products by revenue.
   BL/repo methods для динамики заказов и продаж по категориям добавлены.
   Остаток: вывести эти отчёты в Web GUI.
   Это важно: эти use cases прямо есть в `diagrams/src/UseCases.puml`.

2. **Категории товара в seller/admin GUI** — закрыть сценарий категоризации товара.
   Продавец должен выбирать категории при создании/редактировании товара.
   Админ должен модерировать привязки товара к категориям.
   Сейчас схема `product_categories` есть, фильтр каталога есть, seed связи создаёт, но web UI создания товара категории не задаёт.

3. **Redis TokenBlocklist для logout** — подключить настоящий blocklist.
   Интерфейс `TokenBlocklist` уже есть, но `cmd/api/main.go` передаёт `nil`.
   Нужно добавить Redis в config/docker-compose, реализацию blocklist и wiring в API/techui components.

4. ~~**Удалённые товары — остаточные edge cases**~~ — done.
   Web показывает deleted product, add-to-cart/checkout запрещены,
   cart блокирует checkout с deleted item, `ProductDTO.deleted_at`
   exposed для order-detail/admin клиентов, review на deleted product
   запрещён через `ErrProductDeleted` (409). Старые pending/paid orders
   и lifecycle проверены — repo joins не фильтруют `deleted_at`,
   ничего не сломалось. Seller own-product list остаётся скрывать
   deleted (current behavior); admin видит всё. Re-delete идемпотентен.

## P1 — важные бизнес-инварианты и конкурентность

1. ~~**Конкурентность stock update / checkout / cancel / ship / expire**~~ — done.
   `UpdateProduct` теперь оборачивается в tx, берёт `FOR UPDATE` через
   `GetProductByIDForUpdate` и валидирует stock-vs-reserved уже после
   локирования (свежие данные). Stock semantics остаются абсолютными;
   delta-операции не вводились.

2. ~~**Конкурентность cancel/ship/expire**~~ — done.
   `CancelOrder`, `ExpireOrders`, `ShipOrder` берут row locks на
   все затронутые products в порядке возрастания id внутри tx
   до `ChangeStockAndReserved`. Deadlock-free между всеми путями
   мутации products (включая checkout).

3. **Более информативные ошибки worker/order lifecycle**.
   Сейчас `ExpireOrders` продолжает обработку после ошибки конкретного заказа, но итоговый лог всё ещё бедный.
   Он показывает `order_id`, но не показывает `product_id`, `quantity`, `reserved_quantity`.
   Нужно логировать каждую ошибку отдельно или возвращать структурированный доменный error context.

4. **Order state machine**.
   Вынести допустимые переходы статусов заказа в одну таблицу/методы и использовать её в service/repo status updates.
   Сейчас переходы размазаны по `PayOrder`, `CancelOrder`, `ShipOrder`, `DeliverOrder`, `ExpireOrders`.

5. **Цена в draft-корзине — модель БД/API/techui**.
   Web UI уже показывает актуальную цену из `products`, checkout записывает актуальный `price_at_purchase`.
   Остаток:
   - `GetCart` в service/API/techui всё ещё считает total по `price_at_purchase`;
   - долгосрочно сделать `order_items.price_at_purchase nullable` для draft items;
   - на checkout записывать snapshot price и считать `total_amount`.

6. **Удалённый/забаненный пользователь**.
   Проверять `deleted_at` при login и при token-based auth.
   Сейчас удалённый пользователь может быть проблемным edge case.

7. **Address schema drift**.
   `house` уже протянут в model/DTO/repo/web.
   `apartment` есть в migration `010_add_address_house.sql`, но не протянут в model/DTO/repo/forms/seed.
   Решить: либо поддержать apartment везде, либо убрать из схемы отдельной миграцией.

8. **Repository error boundary**.
   Репозитории должны переводить `pgx.ErrNoRows` и `pgconn.PgError` unique/check/FK по constraint name в доменные sentinel-ошибки.
   Сейчас часть ошибок уже переведена, но не везде.

## P2 — полезно, но не блокирует ЛР

1. **Category pagination/list refresh**.
   Сейчас admin categories временно грузит большой limit, чтобы новые категории были видны.
   Нужно нормальное page/search/filter для public/admin category list.

2. **Frontend checklist manual pass**.
   Проверить руками: admin panel, analyst dashboard, catalog filters, price history, seller flows, buyer flows.
   Проверить Price Change Graph: после изменения цены товара график должен показывать переход старой цены к новой без seed-записи "начальной цены".
   Важно для сдачи, но не требует heavy e2e framework.

3. **Общие app errors без цикла repo -> service**.
   Вынести `ErrNotFound` и общие sentinel-ошибки в `internal/apperr` или похожий пакет.
   Сейчас repositories импортируют `internal/service`, это архитектурно некрасиво, но для ЛР терпимо.

4. **Транзакционная граница в business logic**.
   Сейчас service открывает транзакции через `TxManager`.
   Подумать, стоит ли оставить так или выделить use case/transaction boundary отдельным уровнем.

5. **Денормализация rating/total_amount**.
   Проверить, где `rating` и `total_amount` могут drift из-за ручных SQL/seed/bugs.
   Для performance это нормально, но инварианты должны быть понятны.

6. **Web cleanup после `ProductService.UpdateProduct` lock**.
   В рамках P1 #1 BL-фикса `NewProductService` получил `TxManager` зависимость.
   `internal/web/web_handler_test.go` поэтому теперь несёт локальный
   `passThroughTxManager` mock — это рабочая заглушка, не подходит для
   долгосрочного дизайна. Решить:
   - либо вынести этот mock в общий test helper;
   - либо пересмотреть DI так, чтобы web tests не зависели от tx-инфры
     (например, отдельный use case для seller stock update).
   Web handlers сами не меняли — только тестовая фабрика. BL правка чистая.

7. **Load / concurrency testing**.
   Промерить, сколько concurrent requests держит сервер до деградации
   latency/error rate. Не "hardware tests" — правильные термины:
   - **load testing** — ожидаемая нагрузка, мерим latency/throughput;
   - **stress testing** — давим выше предела, ищем breaking point;
   - **capacity / concurrency testing** — наш вопрос "сколько RPS";
   - **soak testing** — долго гоняем, ищем утечки/деградацию.
   Инструменты: `k6` (JS scripts, удобный для Go-сервисов),
   `vegeta` (Go-нативный, простой CLI), `hey`, `wrk`, JMeter, Locust.
   Что мерить: p50/p95/p99 latency, RPS, error rate, DB pool exhaustion,
   FOR UPDATE contention на checkout/cancel. Сценарии:
   catalog GET (read-heavy), concurrent checkout одного товара
   (write contention), смешанный buyer/seller flow.
   Для ЛР не блокирует, но красиво для отчёта/защиты.

## P3 — архитектурные улучшения на потом

1. **Port interfaces**.
   Решить, оставлять интерфейсы, диктуемые сервисом, в `internal/service` или вынести в `internal/port`.

2. **Use Cases вместо god-сервисов**.
   Подумать о дроблении на узкие interfaces:
   ```go
   type CheckoutUseCase interface {
       Execute(ctx context.Context, userID int64, addressID int64) ([]int64, error)
   }
   ```
   Для текущей ЛР это не критично.

3. **ACID review текущих транзакций**.
   Проверить изоляцию и инварианты для checkout/cancel/ship/expire/update stock.

4. **SQL улучшения**.
   Подумать, где уместны CTE/window queries.
   Не делать ради красоты.

5. **`SET LOCAL ROLE` внутри транзакции**.
   Исследовать, нужно ли реально использовать PostgreSQL roles на уровне БД.

## P4 — низкий приоритет для учебного проекта

1. **CSRF для web forms**.
   Для prod важно, для текущей ЛР не блокирует.

2. **E2E GUI tests**.
   Для проекта полезно, но для сдачи сейчас почти не даёт выгоды.
   Достаточно manual checklist + unit/template tests.

3. **UI polish**.
   Мелкие тексты, spacing, empty states, disabled states.
   Делать после закрытия P0/P1.

## Нужно решение

1. **Stock UI semantics**.
   Оставляем absolute `stock_quantity` или переходим на операции "increase/decrease by N"?

2. **Address apartment**.
   Поддерживаем `apartment` везде или удаляем из схемы?

3. **Deleted product policy**.
   Разрешать ли review на deleted product, если покупатель реально покупал его раньше?

4. **Sales by category attribution**.
   Если товар привязан к нескольким категориям, считать его выручку в каждой категории или нужна одна primary category?
   Сейчас BL/repo report считает выручку в каждой привязанной категории.

---
