# Known Issues

Открытые баги и подтвержденные технические риски.

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
