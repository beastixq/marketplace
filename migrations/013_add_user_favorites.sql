-- +goose Up
create table users_favorites (
    user_id bigint not null,
    product_id bigint not null,

    constraint fk_users_favorites_user_id
        foreign key (user_id)
        references users (id) on delete cascade,

    constraint fk_users_favorites_product_id
        foreign key (product_id)
        references products (id) on delete cascade
);

-- +goose Down
drop table if exists users_favorites;
