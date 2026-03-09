-- +goose Up
alter table orders add column seller_id bigint references sellers(id) on delete restrict

-- +goose Down
alter table orders drop column seller_id;
