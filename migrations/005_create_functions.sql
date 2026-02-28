-- +goose Up
-- +goose StatementBegin
create or replace function get_seller_statistics(
    p_seller_id bigint,
    p_date_from date,
    p_date_to date
)
returns table(
    total_orders bigint,
    total_revenue numeric,
    avg_order_value numeric,
    top_product_name varchar
) as $$
begin
    return query
    WITH general_select AS (
        SELECT ords.id as order_id, ords.status, ords.created_at, oi.quantity as oi_quantity, oi.price_at_purchase as oi_price_at_purchase, p.id as product_id, p.name as product_name
        from orders ords
                    JOIN order_items oi on ords.id = oi.order_id
                    JOIN products p on oi.product_id = p.id
                where p.seller_id = p_seller_id
                and ords.created_at >= p_date_from and ords.created_at < p_date_to
                and ords.status in ('paid', 'shipped', 'delivered')
    )
    select count(distinct order_id) as total_orders, sum(oi_quantity * oi_price_at_purchase) as total_revenue, sum(oi_quantity * oi_price_at_purchase) / NULLIF(count(distinct order_id), 0) as avg_order_value,
        (select product_name as top_product_name
            from general_select
            group by product_id, product_name
            order by sum(oi_quantity) desc
            limit 1)
        from general_select;
end;
$$ language plpgsql;
-- +goose StatementEnd

-- +goose Down
drop function if exists get_seller_statistics;
