-- +goose Up
alter table products
    add column reserved_quantity integer not null default 0
        check (reserved_quantity >= 0);

alter table products
    add constraint chk_products_reserved_le_stock
        check (reserved_quantity <= stock_quantity);

-- +goose Down
alter table products drop constraint if exists chk_products_reserved_le_stock;
alter table products drop column if exists reserved_quantity;
