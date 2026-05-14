-- +goose Up
create table favorite_products (
    user_id bigint not null,
    product_id bigint not null,
    created_at timestamptz not null default now(),

    constraint pk_favorite_products
        primary key (user_id, product_id),

    constraint fk_favorite_products_user_id
        foreign key (user_id)
        references users (id) on delete cascade,

    constraint fk_favorite_products_product_id
        foreign key (product_id)
        references products (id) on delete cascade
);

create index idx_favorite_products_product_id on favorite_products (product_id);

grant all privileges on favorite_products to marketplace_buyer;
grant all privileges on favorite_products to marketplace_seller;
grant all privileges on favorite_products to marketplace_admin;
grant select on favorite_products to marketplace_analyst;

-- +goose Down
revoke all privileges on favorite_products from marketplace_analyst;
revoke all privileges on favorite_products from marketplace_admin;
revoke all privileges on favorite_products from marketplace_seller;
revoke all privileges on favorite_products from marketplace_buyer;

drop index if exists idx_favorite_products_product_id;
drop table if exists favorite_products;
