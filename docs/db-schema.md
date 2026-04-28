# Схема базы данных (10 таблиц)

## `users` — Пользователи системы
| Поле | Тип | Ограничения |
|---|---|---|
| id | BIGSERIAL | PRIMARY KEY |
| email | VARCHAR(255) | NOT NULL, UNIQUE |
| password_hash | VARCHAR(255) | NOT NULL |
| full_name | VARCHAR(255) | NOT NULL |
| phone | VARCHAR(20) | UNIQUE |
| role | VARCHAR(20) | NOT NULL, CHECK IN ('buyer','seller','admin','analyst') |
| created_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() |
| deleted_at | TIMESTAMPTZ | мягкое удаление |

## `sellers` — Профили продавцов (1:1 с users)
| Поле | Тип | Ограничения |
|---|---|---|
| id | BIGSERIAL | PRIMARY KEY |
| user_id | BIGINT | NOT NULL, FK → users(id) ON DELETE RESTRICT, UNIQUE |
| company_name | VARCHAR(255) | NOT NULL |
| description | TEXT | |
| rating | NUMERIC(3,2) | |
| created_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() |

## `categories` — Категории товаров (иерархия через самоссылку)
| Поле | Тип | Ограничения |
|---|---|---|
| id | BIGSERIAL | PRIMARY KEY |
| parent_id | BIGINT | FK → categories(id) ON DELETE SET NULL |
| name | VARCHAR(255) | NOT NULL, UNIQUE |
| description | TEXT | |

## `products` — Товары
| Поле | Тип | Ограничения |
|---|---|---|
| id | BIGSERIAL | PRIMARY KEY |
| seller_id | BIGINT | NOT NULL, FK → sellers(id) ON DELETE RESTRICT |
| name | VARCHAR(255) | NOT NULL |
| description | TEXT | |
| price | NUMERIC(12,2) | NOT NULL, CHECK (price > 0) |
| stock_quantity | INTEGER | NOT NULL DEFAULT 0, CHECK (>= 0) |
| created_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() |
| deleted_at | TIMESTAMPTZ | мягкое удаление |

## `product_categories` — Связь M:N (товары ↔ категории)
| Поле | Тип | Ограничения |
|---|---|---|
| product_id | BIGINT | NOT NULL, FK → products(id) ON DELETE CASCADE |
| category_id | BIGINT | NOT NULL, FK → categories(id) ON DELETE CASCADE |
| | | PRIMARY KEY (product_id, category_id) |

## `addresses` — Адреса доставки
| Поле | Тип | Ограничения |
|---|---|---|
| id | BIGSERIAL | PRIMARY KEY |
| user_id | BIGINT | NOT NULL, FK → users(id) ON DELETE CASCADE |
| city | VARCHAR(100) | NOT NULL |
| street | VARCHAR(255) | NOT NULL |
| zip_code | VARCHAR(20) | NOT NULL |
| is_default | BOOLEAN | NOT NULL DEFAULT FALSE |
| created_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() |

## `orders` — Заказы
| Поле | Тип | Ограничения |
|---|---|---|
| id | BIGSERIAL | PRIMARY KEY |
| user_id | BIGINT | NOT NULL, FK → users(id) ON DELETE RESTRICT |
| address_id | BIGINT | FK → addresses(id) ON DELETE RESTRICT (nullable для draft) |
| status | VARCHAR(30) | NOT NULL, CHECK IN ('draft','pending','paid','shipped','delivered','cancelled') |
| seller_id | BIGINT | FK → sellers(id) ON DELETE RESTRICT (добавлено миграцией 006) |
| total_amount | NUMERIC(12,2) | NOT NULL, CHECK (>= 0) |
| created_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() |
| updated_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() |

### Жизненный цикл заказа
```
draft → pending → paid → shipped → delivered
                    ↘       ↘
                  cancelled  cancelled
```
| Переход | Кто | Действие |
|---|---|---|
| draft → pending | Покупатель | Оформляет заказ (выбирает адрес) |
| pending → paid | Покупатель | Оплачивает |
| paid → shipped | Продавец | Отмечает отправку |
| shipped → delivered | Продавец | Отмечает доставку |
| pending/paid → cancelled | Покупатель | Отменяет (до отправки) |
| paid → cancelled | Продавец | Отклоняет |

## `order_items` — Позиции заказа
| Поле | Тип | Ограничения |
|---|---|---|
| id | BIGSERIAL | PRIMARY KEY |
| order_id | BIGINT | NOT NULL, FK → orders(id) ON DELETE RESTRICT |
| product_id | BIGINT | NOT NULL, FK → products(id) ON DELETE RESTRICT |
| quantity | INTEGER | NOT NULL, CHECK (> 0) |
| price_at_purchase | NUMERIC(12,2) | NOT NULL, CHECK (> 0) |

## `reviews` — Отзывы покупателей
| Поле | Тип | Ограничения |
|---|---|---|
| id | BIGSERIAL | PRIMARY KEY |
| user_id | BIGINT | NOT NULL, FK → users(id) ON DELETE CASCADE |
| product_id | BIGINT | NOT NULL, FK → products(id) ON DELETE CASCADE |
| rating | SMALLINT | NOT NULL, CHECK BETWEEN 1 AND 5 |
| comment | TEXT | |
| created_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() |
| | | UNIQUE (user_id, product_id) |

## `product_price_history` — Журнал изменений цен (заполняется триггером)
| Поле | Тип | Ограничения |
|---|---|---|
| id | BIGSERIAL | PRIMARY KEY |
| product_id | BIGINT | NOT NULL, FK → products(id) ON DELETE CASCADE |
| old_price | NUMERIC(12,2) | NOT NULL |
| new_price | NUMERIC(12,2) | NOT NULL |
| changed_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() |
| changed_by | TEXT | NOT NULL DEFAULT current_user |

## Индексы
```sql
CREATE INDEX ON products (seller_id);
CREATE INDEX ON products (price);
CREATE INDEX ON orders (user_id);
CREATE INDEX ON orders (status, created_at);
CREATE INDEX ON reviews (product_id);
CREATE INDEX ON product_categories (category_id);
CREATE INDEX ON products (id) WHERE deleted_at IS NULL;
CREATE INDEX ON users (id) WHERE deleted_at IS NULL;
```
