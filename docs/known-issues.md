# Known Issues

Открытые баги, подтвержденные технические риски и отложенный технический долг документации.

## BUG-004 API Возвращает Пустые Отзывы Без Параметров Пагинации

Status: open
Severity: medium
Area: API/reviews/pagination
Found: 2026-10-07

Проверено на тестовом стенде PostgreSQL 16 с миграциями 1–12, код `ec1d828`.
Требуется товар с отзывом, созданным покупателем после доставки заказа.

1. Зарегистрировать покупателя и продавца, создать профиль, товар и адрес.
2. Купить товар, оплатить через mock-bank, отгрузить и доставить заказ.
3. Создать отзыв: `POST /api/v1/reviews` возвращает `201` и ID.
4. Запросить `GET /api/v1/products/<id>/reviews` без query-параметров.

Фактически: `200` с `[]`, хотя отзыв создан. Репозиторий применяет `LIMIT 0`
при нулевом значении `PaginationOpts.Limit`.
Ожидается: существующий отзыв возвращается с принятой пагинацией по умолчанию,
либо API явно сообщает, что параметры обязательны. Сейчас ошибки нет.

E2E ЛР2 использует явные `page=1&limit=20` и проверяет ID, покупателя, товар,
оценку и текст отзыва. Это обход подтверждённого ограничения, не исправление.
Ретест: сравнить запросы с пагинацией и без неё после согласованного исправления.
GitHub Issue не создан; ссылку на него добавлять только после разрешения публикации.

## RISK-001 Обновление Цены Без ChangedBy Зависит От Состояния Соединения

Status: confirmed risk, open
Area: products/price-history
Found: 2026-10-07

На том же PostgreSQL-стенде вызов `ProductService.UpdateProduct` с новой ценой
и `ChangedBy=nil` завершился нарушением NOT NULL поля `product_price_history.changed_by`.
Триггер читает `current_setting('app.current_user', true)`; на новом соединении
значение отсутствует. HTTP-клиент передаёт actor в `ChangedBy`, но необязательность
поля доменной структуры не выражает это условие. В интеграционных сценариях
передаётся `ChangedBy`, как в текущем HTTP-пути. Контракт и исправление требуют
отдельного решения; бизнес-код и старые миграции при выполнении ЛР2 не менялись.

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

## TD-001 ER Chen Missing Entity Relationships

Status: deferred by student, 2026-09-30
Severity: low
Area: docs/WebLab#1/diagrams

Problem:
ER-диаграмма Чена, опубликованная в README, не показывает связи
«пользователь — профиль продавца», «заказ — продавец», «заказ — адрес» и
«позиция заказа — товар». Эти связи присутствуют в текущей схеме DBML.

Artifacts:
- [Редактируемый источник ER Chen](../diagrams/src/ER_Chen.graphml).
- [Изображение в README](../diagrams/out/ER_Chen.png).
- [Схема DBML для сверки](../diagrams/src/schema.dbml).

Expected:
При последующем исправлении согласовать связи и их кратности с предметной
областью и схемой БД, затем обновить экспорт диаграммы.

Decision:
Студент явно отложил исправление. Наличие диаграммы не означает, что её
согласованность со схемой БД проверена и принята.

## TD-002 Checkout BPMN Stock Reservation

Status: deferred by student, 2026-09-30
Severity: low
Area: docs/WebLab#1/diagrams

Problem:
BPMN оформления показывает проверку наличия товара и переходы статусов,
но не отражает резервирование при checkout и освобождение резерва при
истечении срока оплаты. Эти действия определены в ADR-1 и жизненном цикле
заказа.

Artifacts:
- [Редактируемый источник BPMN](../diagrams/src/BPMN_Checkout.graphml).
- [Текущий экспорт](../diagrams/out/BPMN_Checkout.png).
- [ADR-1](ADR.md#adr-1-резервировать-товар-при-оформлении-заказа).
- [Жизненный цикл заказа](order-lifecycle.md).

Expected:
При последующем исправлении показать резервирование и освобождение резерва,
а также отсутствие частичного оформления при ошибке; сверить процесс с ADR-1
и текущим поведением checkout.

Decision:
Студент явно отложил исправление исходника и экспортов этой BPMN. Новую BPMN
оформлять отдельно; она не закрывает этот технический долг.

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
