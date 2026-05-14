# Ideas

Продуктовые, UX и архитектурные идеи, которые еще не являются подтвержденными
багами. Если идея превращается в обязательное требование или баг, перенести ее в
соответствующий документ.

## IDEA-001 Require Product Category On Creation

Status: open
Area: products/categories
Found: 2026-05-13

Observation:
Сейчас при создании продукта категорию задавать необязательно.

Question:
Нужно ли делать хотя бы одну категорию обязательной для публикации товара?

Trade-Offs:
- Обязательная категория улучшает навигацию и качество каталога.
- Необязательная категория упрощает черновое создание товара.
- Если категория станет обязательной, нужно синхронизировать API validation,
  web validation, сервисные правила, тесты и документацию.

Decision:
TBD.

## IDEA-002 Per-User Favorites List Cache

Status: open
Area: cache/favorites
Found: 2026-05-14
Source: specs/002-favorite-products/research.md (Decision 2 follow-up)

Observation:
В v1 фичи `favorites` решено не вводить отдельный кэш-ключ для списка
избранного — чтение идет напрямую в PostgreSQL. Если профиль нагрузки
сместится в сторону горячего чтения списка (например, мобильное приложение
часто запрашивает «избранное» при холодном старте), имеет смысл рассмотреть
кэш `favorites:user:{id}` с инвалидацией при add/remove и каскадных
удалениях.

Trade-Offs:
- Плюс: меньше нагрузки на БД при типичных read-heavy сценариях.
- Минус: новый путь инвалидации (add/remove/product-delete/user-delete) — и
  по AGENTS.md (Constitution II) хендлерам инвалидацию делать нельзя,
  значит нужен сервисный декоратор. Это полноценная архитектурная работа.
- Альтернатива: ничего не делать, пока реальные метрики не покажут проблему.

Decision:
TBD. Триггер для пересмотра — устойчивый рост P95 латенси GET /favorites.

## IDEA-003 Cursor-Based Pagination For Favorites

Status: open
Area: api/favorites
Found: 2026-05-14
Source: specs/002-favorite-products/research.md (Decision 6 follow-up)

Observation:
Listing избранного использует page/page_size пагинацию (default 20, cap 100),
такую же, как в каталоге. Если пользователи будут регулярно упираться в
page_size=100, page-based пагинация начнет деградировать из-за глубоких
offset-ов; cursor-based по `(created_at, product_id)` устойчивее на больших
наборах.

Trade-Offs:
- Плюс: стабильная производительность даже при тысячах избранных у одного
  пользователя; защита от race conditions при пагинации с одновременными
  изменениями.
- Минус: ломает однородность API (каталог тоже page-based) и требует
  выработать общий cursor-формат.

Decision:
TBD. Триггер — наблюдаемые пользователи с page_size=100, повторно листающие
страницы.

## IDEA-004 Seller-Facing Favorites Count As Demand Signal

Status: open
Area: favorites/sellers
Found: 2026-05-14
Source: spec.md Q3 (deferred derived behavior)

Observation:
В Q3 спецификации фичи `favorites` явно отложили все производные поведения,
включая показ продавцу количества пользователей, добавивших его товар в
избранное. Это полезный сигнал спроса до того, как товар начинают активно
покупать.

Open Questions:
- Видимость: только владельцу товара (приватно) или агрегированно в
  публичных метриках? Спека сейчас настаивает на приватности избранного —
  отображение продавцу — это компромисс, нужно явное решение.
- API: новый endpoint `GET /api/v1/sellers/me/products/{id}/favorites-count`
  или поле в существующем DTO товара продавца?

Trade-Offs:
- Плюс: понятный демандр-сигнал для продавцов, минимальная стоимость
  реализации (одна SQL-агрегация `count(*) WHERE product_id=$1`).
- Минус: расширяет приватность модели (raw count раскрывает, что хотя бы
  N пользователей нашли товар), нужно зафиксировать политику.

Decision:
TBD.

## IDEA-005 Notify Users About Changes To Favorited Products

Status: open
Area: favorites/notifications
Found: 2026-05-14
Source: spec.md Q3 (deferred derived behavior)

Observation:
В Q3 фичи `favorites` отдельно отложили нотификации (например, «цена
снижена», «снова в наличии», «товар удален»). Это популярная UX-функция в
маркетплейсах и обычно вторая по востребованности после самого избранного.

Trade-Offs:
- Плюс: повышает retention, превращает избранное в активный поверхностный
  сигнал, а не пассивный bookmark.
- Минус: требует канал доставки (email/push/in-app), event hook на
  изменения товара (price, stock, deleted_at), отдельную очередь — это
  полноценная новая фича, а не довесок к favorites.

Decision:
TBD. Когда возьмем — рассмотреть как отдельный спек (отдельная нумерация
specs/NNN-…), а не как расширение фичи favorites.
