# Database

PostgreSQL is the primary data store. Migrations are managed by goose and live in `migrations/`.

## Local DSN

Default local DSN:

```text
postgres://postgres:postgres@localhost:5432/marketplace?sslmode=disable
```

Set it with:

```bash
export DATABASE_URL='postgres://postgres:postgres@localhost:5432/marketplace?sslmode=disable'
```

## Migration Order

| File | Purpose |
| --- | --- |
| `001_create_tables.sql` | Core tables. |
| `002_create_indexes.sql` | Initial indexes. |
| `003_create_roles.sql` | PostgreSQL roles and grants. |
| `004_create_trigger.sql` | Product price history trigger. |
| `005_create_functions.sql` | Seller statistics function. |
| `006_orders_add_seller_id.sql` | `orders.seller_id`. |
| `007_product_and_seller_rating.sql` | Rating support. |
| `008_system_accounts.sql` | System accounts. |
| `009_add_reserved_quantity.sql` | Product reservation accounting. |
| `010_add_address_house.sql` | Address house/apartment fields. |
| `011_order_cart_item_invariants.sql` | One draft cart per buyer and one product row per order. |

Run:

```bash
goose -dir migrations postgres "$DATABASE_URL" up
```

## Tables

Core tables:

- `users`
- `sellers`
- `categories`
- `products`
- `product_categories`
- `addresses`
- `orders`
- `order_items`
- `reviews`
- `product_price_history`

Keep `docs/db-schema.md` synchronized when schema changes.

## Important Constraints

- `users.email` is unique.
- `users.role` is one of `buyer`, `seller`, `admin`, `analyst`.
- `sellers.user_id` is unique.
- `categories.parent_id` is a self-reference.
- `products.price > 0`.
- `products.stock_quantity >= 0`.
- `products.reserved_quantity >= 0`.
- `products.reserved_quantity <= products.stock_quantity`.
- `orders.status` is one of `draft`, `pending`, `paid`, `shipped`, `delivered`, `cancelled`.
- `orders.address_id` is nullable for draft orders.
- `orders` has one draft cart per buyer via partial unique index on `user_id` where `status = 'draft'`.
- `order_items` has `UNIQUE(order_id, product_id)`.
- `reviews.rating` is between 1 and 5.
- `reviews` has `UNIQUE(user_id, product_id)`.

## Product Reservation

`products.stock_quantity` is total stock. `products.reserved_quantity` tracks stock reserved by pending/paid flow. The invariant `reserved_quantity <= stock_quantity` is enforced by the database and must also be respected by service logic.

## Price History Trigger

Migration `004_create_trigger.sql` creates `insert_price_change_info()` and trigger `catch_price_change`.

When `products.price` changes, the trigger inserts a row into `product_price_history`. Repository code may set:

```sql
SELECT set_config('app.current_user', $1, true)
```

so `changed_by` can record the application actor instead of only the database role.

## Seller Statistics Function

Migration `005_create_functions.sql` creates:

```sql
get_seller_statistics(seller_id, date_from, date_to)
```

It returns:

- `total_orders`
- `total_revenue`
- `avg_order_value`
- `top_product_name`

The function counts paid, shipped, and delivered orders in the requested date interval.

## PostgreSQL Roles

| Role | Intended rights |
| --- | --- |
| `marketplace_buyer` | Read catalog/categories; CRUD own buyer resources through app policy. |
| `marketplace_seller` | Manage own products and view own relevant order data through app policy. |
| `marketplace_admin` | Full administrative access. |
| `marketplace_analyst` | Read-only reporting access. |

Database grants are a safety layer. Application service code still enforces ownership and business policy.

## Repository Rules

- Use parameterized SQL only.
- Use pgx/pgxpool in repository code, not service code.
- Translate known database errors into service errors at repository boundaries.
- Use `pgerrcode.*`, not raw SQLSTATE strings.
- Keep DB row structs private to repository where possible.
- Test SQL, triggers, constraints, and role behavior against a real database.
