-- +goose Up
create role marketplace_seller;
grant usage, select on all sequences in schema public to marketplace_seller;
grant all privileges on products to marketplace_seller;
grant select on order_items to marketplace_seller;
grant select on categories to marketplace_seller;
grant select on product_categories to marketplace_seller;
grant select on orders to marketplace_seller;

create role marketplace_buyer;
grant select on products to marketplace_buyer;
grant select on categories to marketplace_buyer;
grant all privileges on addresses to marketplace_buyer;
grant all privileges on reviews to marketplace_buyer;
grant all privileges on orders to marketplace_buyer;
grant usage, select on all sequences in schema public to marketplace_buyer;
grant insert, select on order_items to marketplace_buyer;
grant select on product_categories to marketplace_buyer;

create role marketplace_admin;
grant usage, select on all sequences in schema public to marketplace_admin;
grant all privileges on all tables in schema public to marketplace_admin;

create role marketplace_analyst;
grant select on all tables in schema public to marketplace_analyst;

-- +goose Down
revoke all privileges on all tables in schema public from marketplace_analyst;
revoke all privileges on all sequences in schema public from marketplace_analyst;
drop role if exists marketplace_analyst;

revoke all privileges on all tables in schema public from marketplace_admin;
revoke all privileges on all sequences in schema public from marketplace_admin;
drop role if exists marketplace_admin;

revoke all privileges on all tables in schema public from marketplace_buyer;
revoke all privileges on all sequences in schema public from marketplace_buyer;
drop role if exists marketplace_buyer;

revoke all privileges on all tables in schema public from marketplace_seller;
revoke all privileges on all sequences in schema public from marketplace_seller;
drop role if exists marketplace_seller;
