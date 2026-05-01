-- +goose Up
alter table users
    drop constraint if exists users_phone_key;

-- +goose Down
alter table users
    add constraint users_phone_key unique (phone);
