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

3. **Seed-инварианты заказов и резервов** — исправить демо-данные.
   Сейчас seeder напрямую создаёт `pending` orders и `order_items`, но не увеличивает `products.reserved_quantity`.
   Потом `OrderExpirationWorker` отменяет просроченный `pending` заказ и пытается снять резерв, из-за чего возможен `Stock invariant violated`.
   Также seed может создать пустые non-draft orders и повторяющиеся items с одинаковым `(order_id, product_id)`.
   Варианты:
   - не генерировать `pending` заказы через raw seed;
   - после генерации `order_items` пересчитывать `reserved_quantity` для всех `pending`/`paid`;
   - генерировать такие заказы через service checkout path.

4. **DB-инварианты корзины/order_items** — добавить constraints.
   Нужно:
   - partial unique index на одну draft-корзину на пользователя;
   - unique constraint/index на `order_items(order_id, product_id)`.
   Это важно: сейчас часть защиты есть в service, но БД должна держать инвариант.

5. **Ownership `address_id` при checkout** — запретить checkout на чужой адрес.
   Перед созданием заказов проверять, что address принадлежит `actor.UserID`.
   Чужой address должен давать доменную ошибку.

6. **Review только после покупки** — запретить отзывы без покупки.
   В `CreateReview` нужна проверка, что buyer покупал этот product в `paid`/`shipped`/`delivered` order.
   Сейчас duplicate review обработан, но purchase check ещё нет.

7. **Redis TokenBlocklist для logout** — подключить настоящий blocklist.
   Интерфейс `TokenBlocklist` уже есть, но `cmd/api/main.go` передаёт `nil`.
   Нужно добавить Redis в config/docker-compose, реализацию blocklist и wiring в API/techui components.

8. **Удалённые товары — остаточные edge cases**.
   Уже сделано: web показывает deleted product, add-to-cart/checkout запрещены, cart блокирует checkout с deleted item.
   Нужно ещё проверить/доделать:
   - API DTO/ответы для deleted product;
   - policy для review на deleted product;
   - старые pending/paid orders с deleted product;
   - seller/admin операции над deleted product.

## P1 — важные бизнес-инварианты и конкурентность

1. **Конкурентность stock update / checkout / cancel / ship / expire**.
   `UpdateProduct(stock_quantity)` сейчас работает абсолютным значением.
   Нужна стратегия:
   - блокировать product rows через `FOR UPDATE`;
   - считать delta stock;
   - не давать гонкам ломать `stock_quantity`/`reserved_quantity`.
   Спорный дизайн: оставить абсолютный `stock_quantity` в UI или перейти на операции "увеличить/уменьшить на N".

2. **Конкурентность cancel/ship/expire**.
   Как в checkout, блокировать products в детерминированном порядке (`FOR UPDATE`), затем release/deduct stock.
   Сейчас БД constraint страхует, но ошибки могут всплывать поздно.

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

не добавляется запись по первой цене продукта
