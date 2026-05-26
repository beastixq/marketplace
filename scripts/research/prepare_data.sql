\set ON_ERROR_STOP on
\set product_count 40000
\set order_count 80000

\echo 'Preparing additional research data...'

set synchronous_commit = off;

insert into users (email, password_hash, full_name, phone, role, created_at)
select
    'research_seller_' || g || '@example.local',
    'research-password-hash',
    'Research Seller User ' || g,
    '+79001' || lpad(g::text, 9, '0'),
    'seller',
    now()
from generate_series(1, 1000) as g
on conflict (email) do nothing;

insert into sellers (user_id, company_name, description, rating, created_at)
select
    u.id,
    'Research Seller ' || row_number() over (order by u.id),
    'Synthetic seller for RPZ research benchmark',
    4.50,
    now()
from users u
where u.email like 'research_seller_%@example.local'
on conflict (user_id) do nothing;

insert into users (email, password_hash, full_name, phone, role, created_at)
select
    'research_buyer_' || g || '@example.local',
    'research-password-hash',
    'Research Buyer ' || g,
    '+79002' || lpad(g::text, 9, '0'),
    'buyer',
    now()
from generate_series(1, 5000) as g
on conflict (email) do nothing;

create temp table research_seller_array as
select
    array_agg(id order by id) as ids,
    count(*)::integer as cnt
from sellers
where company_name like 'Research Seller %';

insert into products (seller_id, name, description, price, stock_quantity, created_at)
select
    sa.ids[(((g - 1) % sa.cnt) + 1)::integer],
    'Research Product ' || g,
    'Synthetic product for RPZ research benchmark',
    (100 + (g % 50000))::numeric(12, 2) / 10,
    10 + (g % 500),
    timestamp '2025-01-01' + ((g % 420) * interval '1 day')
from generate_series(1, :product_count) as g
cross join research_seller_array sa;

create temp table research_buyer_array as
select
    array_agg(id order by id) as ids,
    count(*)::integer as cnt
from users
where email like 'research_buyer_%@example.local';

create temp table research_product_array as
select
    array_agg(id order by id) as product_ids,
    array_agg(seller_id order by id) as seller_ids,
    array_agg(price order by id) as prices,
    count(*)::integer as cnt
from products
where name like 'Research Product %';

create temp table research_order_source as
select
    nextval(pg_get_serial_sequence('orders', 'id')) as order_id,
    ba.ids[pick.buyer_idx] as user_id,
    pa.seller_ids[pick.product_idx] as seller_id,
    pa.product_ids[pick.product_idx] as product_id,
    (1 + (g % 3))::integer as quantity,
    pa.prices[pick.product_idx] as price_at_purchase,
    (1 + (g % 3)) * pa.prices[pick.product_idx] as total_amount,
    case (g % 5)
        when 0 then 'paid'
        when 1 then 'shipped'
        when 2 then 'delivered'
        when 3 then 'pending'
        else 'cancelled'
    end as status,
    timestamp '2025-01-01'
        + ((g % 480) * interval '1 day')
        + ((g % 86400) * interval '1 second') as created_at
from generate_series(1, :order_count) as g
cross join research_buyer_array ba
cross join research_product_array pa
cross join lateral (
    select
        (((g - 1) % ba.cnt) + 1)::integer as buyer_idx,
        (((g - 1) % pa.cnt) + 1)::integer as product_idx
) pick;

insert into orders (id, user_id, address_id, seller_id, status, total_amount, created_at, updated_at)
select
    order_id,
    user_id,
    null,
    seller_id,
    status,
    total_amount,
    created_at,
    created_at
from research_order_source
on conflict (id) do nothing;

insert into order_items (order_id, product_id, quantity, price_at_purchase)
select
    order_id,
    product_id,
    quantity,
    price_at_purchase
from research_order_source
on conflict (order_id, product_id) do nothing;

analyze users;
analyze sellers;
analyze products;
analyze orders;
analyze order_items;
analyze reviews;

\echo 'Research data prepared.'
