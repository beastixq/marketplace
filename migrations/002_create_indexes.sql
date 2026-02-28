-- +goose Up
create index idx_products_seller_id on products (seller_id);
create index idx_orders_user_id on orders (user_id);
create index idx_orders_status_created_at on orders (status, created_at);
create index idx_product_categories_category_id on product_categories (category_id);
create index idx_reviews_product_id on reviews (product_id);
create index idx_products_id_not_deleted on products (id) where deleted_at is NULL;
create index idx_users_id_not_deleted on users (id) where deleted_at is NULL;
create index idx_products_price on products (price);

-- +goose Down
drop index idx_products_price;
drop index idx_users_id_not_deleted;
drop index idx_products_id_not_deleted;
drop index idx_reviews_product_id;
drop index idx_product_categories_category_id;
drop index idx_orders_status_created_at;
drop index idx_orders_user_id;
drop index idx_products_seller_id;
