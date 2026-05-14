-- +goose Up
create table favorites (
    user_id bigint not null,
    product_id bigint not null,
    created_at timestamptz not null default now(),

    constraint pk_favorites
        primary key (user_id, product_id),

    constraint fk_favorites_user_id
        foreign key (user_id)
        references users (id) on delete cascade,

    constraint fk_favorites_product_id
        foreign key (product_id)
        references products (id) on delete cascade
);

create index idx_favorites_user_id_created_at on favorites (user_id, created_at desc);
create index idx_favorites_product_id on favorites (product_id);

grant select, insert, delete on favorites to marketplace_buyer;

-- +goose Down
revoke select, insert, delete on favorites from marketplace_buyer;
drop index if exists idx_favorites_product_id;
drop index if exists idx_favorites_user_id_created_at;
drop table if exists favorites;
