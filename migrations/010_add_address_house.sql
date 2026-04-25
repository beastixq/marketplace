-- +goose Up
ALTER TABLE addresses
    ADD COLUMN house varchar(20) NOT NULL DEFAULT '',
    ADD COLUMN apartment varchar(20);

ALTER TABLE addresses ALTER COLUMN house DROP DEFAULT;

-- +goose Down
ALTER TABLE addresses
    DROP COLUMN IF EXISTS apartment,
    DROP COLUMN IF EXISTS house;
