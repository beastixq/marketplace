-- +goose Up
create table users (
    id bigserial primary key,
    email varchar(255) not null unique,
    password_hash varchar(255) not null,
    full_name varchar(255) not null,
    phone varchar(20) unique,
    role varchar(20) not null check (role in ('buyer', 'seller', 'admin', 'analyst')),
    created_at timestamptz not null default now(),
    deleted_at timestamptz
);

create table sellers (
    id bigserial primary key,
    user_id bigint not null unique,
    company_name varchar(255) not null,
    description text,
    rating numeric(3,2),
    created_at timestamptz not null default now(),

    constraint fk_sellers_user_id
        foreign key (user_id)
        references users (id) on delete restrict
);

create table categories (
    id bigserial primary key,
    parent_id bigint,
    name varchar(255) not null unique,
    description text,

    constraint fk_categories_parent_id
        foreign key (parent_id)
        references categories (id) on delete set null
);

create table products (
    id bigserial primary key,
    seller_id bigint not null,
    name varchar(255) not null,
    description text,
    price numeric(12,2) not null check (price > 0),
    stock_quantity integer not null default 0 check (stock_quantity >= 0),
    created_at timestamptz not null default now(),
    deleted_at timestamptz,

    constraint fk_products_seller_id
        foreign key (seller_id)
        references sellers (id) on delete restrict
);

create table product_categories (
    product_id bigint not null,
    category_id bigint not null,

    constraint pk_product_categories
        primary key (product_id, category_id),

    constraint fk_product_categories_product_id
        foreign key (product_id)
        references products (id) on delete cascade,

    constraint fk_product_categories_category_id
        foreign key (category_id)
        references categories (id) on delete cascade
);

create table addresses (
    id bigserial primary key,
    user_id bigint not null,
    city varchar(100) not null,
    street varchar(255) not null,
    zip_code varchar(20) not null,
    is_default boolean not null default false,
    created_at timestamptz not null default now(),

    constraint fk_addresses_user_id
        foreign key (user_id)
        references users (id) on delete cascade
);

create table orders (
    id bigserial primary key,
    user_id bigint not null,
    address_id bigint,
    status varchar(30) not null check (status in (
        'draft',
        'pending',
        'paid',
        'shipped',
        'delivered',         -- actually received by buyer
        'cancelled'
        )),
    total_amount numeric(12,2) not null check (total_amount >= 0),
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),

    constraint fk_orders_user_id
        foreign key (user_id)
        references users (id) on delete restrict,

    constraint fk_orders_address_id
        foreign key (address_id)
        references addresses (id) on delete restrict
);

create table order_items (
    id bigserial primary key,
    order_id bigint not null,
    product_id bigint not null,
    quantity integer not null check (quantity > 0),
    price_at_purchase numeric(12,2) not null check (price_at_purchase > 0),

    constraint fk_order_items_order_id
        foreign key (order_id)
        references orders (id) on delete restrict,

    constraint fk_order_items_product_id
        foreign key (product_id)
        references products (id) on delete restrict
);

create table reviews (
    id bigserial primary key,
    user_id bigint not null,
    product_id bigint not null,
    rating smallint not null check (rating between 1 and 5),
    comment text,
    created_at timestamptz not null default now(),
    unique (user_id, product_id),

    constraint fk_reviews_user_id
        foreign key (user_id)
        references users (id) on delete cascade,

    constraint fk_reviews_product_id
        foreign key (product_id)
        references products (id) on delete cascade
);

create table product_price_history (
    id bigserial primary key,
    product_id bigint not null,
    old_price numeric(12,2) not null,
    new_price numeric(12,2) not null,
    changed_at timestamptz not null default now(),
    changed_by text not null default current_user,

    constraint fk_product_price_history_product_id
        foreign key (product_id)
        references products (id) on delete cascade
);

-- +goose Down
drop table if exists product_price_history;
drop table if exists reviews;
drop table if exists order_items;
drop table if exists orders;
drop table if exists addresses;
drop table if exists product_categories;
drop table if exists products;
drop table if exists categories;
drop table if exists sellers;
drop table if exists users;
