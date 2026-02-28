-- +goose Up
-- +goose StatementBegin
create or replace function insert_price_change_info()
returns trigger as $$
begin
    insert into product_price_history (product_id, old_price, new_price, changed_at, changed_by) values (NEW.id, OLD.price, NEW.price, now(), current_setting('app.current_user', true));
    return NEW;
end;
$$ language plpgsql;
-- +goose StatementEnd

create trigger catch_price_change
    after update of price on products
    for each row
    when (OLD.price is distinct from NEW.price)
    execute function insert_price_change_info();

-- +goose Down
drop trigger if exists catch_price_change on products;
drop function if exists insert_price_change_info;
