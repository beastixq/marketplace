-- +goose Up
create unique index ux_orders_one_draft_per_user
    on orders (user_id)
    where status = 'draft';

alter table order_items
    add constraint uq_order_items_order_id_product_id
        unique (order_id, product_id);

-- +goose Down
alter table order_items
    drop constraint if exists uq_order_items_order_id_product_id;

drop index if exists ux_orders_one_draft_per_user;
