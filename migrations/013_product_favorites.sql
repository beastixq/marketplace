-- +goose Up
create table product_favorites (
    user_id bigint not null,
    product_id bigint not null,
    created_at timestamptz not null default now(),

    constraint pk_product_favorites
        primary key (user_id, product_id),

    constraint fk_product_favorites_user_id
        foreign key (user_id)
        references users (id) on delete cascade,

    constraint fk_product_favorites_product_id
        foreign key (product_id)
        references products (id) on delete cascade
);

create index idx_product_favorites_user_created_at
    on product_favorites (user_id, created_at desc);

grant select, insert, delete on product_favorites to marketplace_buyer;
grant select, insert, delete on product_favorites to marketplace_seller;
grant all privileges on product_favorites to marketplace_admin;
grant select on product_favorites to marketplace_analyst;

-- +goose Down
revoke all privileges on product_favorites from marketplace_analyst;
revoke all privileges on product_favorites from marketplace_admin;
revoke all privileges on product_favorites from marketplace_seller;
revoke all privileges on product_favorites from marketplace_buyer;

drop index if exists idx_product_favorites_user_created_at;
drop table if exists product_favorites;
