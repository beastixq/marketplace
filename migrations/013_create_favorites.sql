-- +goose Up
create table favorites (
    user_id    bigint      not null,
    product_id bigint      not null,
    created_at timestamptz not null default now(),

    constraint pk_favorites primary key (user_id, product_id),

    constraint fk_favorites_user_id
        foreign key (user_id)
        references users (id) on delete cascade,

    constraint fk_favorites_product_id
        foreign key (product_id)
        references products (id) on delete cascade
);

create index idx_favorites_user_created_at_desc
    on favorites (user_id, created_at desc);

grant select, insert, delete on favorites to marketplace_buyer;
grant select, insert, delete on favorites to marketplace_seller;
grant select, insert, delete on favorites to marketplace_admin;
grant select                   on favorites to marketplace_analyst;

-- +goose Down
revoke all privileges on favorites from marketplace_analyst;
revoke all privileges on favorites from marketplace_admin;
revoke all privileges on favorites from marketplace_seller;
revoke all privileges on favorites from marketplace_buyer;

drop index if exists idx_favorites_user_created_at_desc;
drop table if exists favorites;
