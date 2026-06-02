\set ON_ERROR_STOP on

-- Параметры можно переопределить через psql -v ... (run.sh гоняет лестницу размеров).
-- Значения по умолчанию задаются только если переменная не передана извне.
\if :{?product_count}
\else
  \set product_count 40000
\endif
\if :{?order_count}
\else
  \set order_count 80000
\endif
\if :{?review_count}
\else
  \set review_count 50000
\endif
-- Отзывы концентрируются на первых review_focus товарах, чтобы у части товаров
-- было много отзывов (запрос reviews_by_product и эндпоинт отзывов становятся показательными).
\if :{?review_focus}
\else
  \set review_focus 2000
\endif
-- Заказы концентрируются на первых order_focus покупателях, чтобы у активного
-- покупателя было много заказов. Тогда для orders_by_user составной индекс
-- (user_id, created_at) обслуживает сортировку (узел Sort исчезает), а простой —
-- нет. На равномерном распределении эффект не виден.
\if :{?order_focus}
\else
  \set order_focus 100
\endif

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
    -- Имя из словаря частотных токенов (~1/10 на токен для ILIKE '%phone%')
    -- плюс редкий токен Zephyr (~0.1%) для демонстрации эффекта GIN/pg_trgm
    -- против последовательного сканирования при низкой селективности.
    (array['Phone','Case','Cable','Laptop','Mouse','Keyboard','Monitor','Charger','Headset','Speaker'])[(g % 10) + 1]
        || case when g % 1000 = 0 then ' Zephyr' else '' end
        || ' Research Product ' || g,
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
where name like '%Research Product %';

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
        (((g - 1) % least(:order_focus, ba.cnt)) + 1)::integer as buyer_idx,
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

-- Отзывы. Триггер update_ratings_on_review (миграция 007) срабатывает на каждую
-- строку и делает два UPDATE с агрегатами, поэтому на время массовой вставки он
-- отключается. Рейтинги для исследования не измеряются.
-- Покупатель меняется каждые review_focus строк, товар берётся из первых
-- review_focus товаров — пары (user_id, product_id) уникальны до исчерпания.
alter table reviews disable trigger user;

insert into reviews (user_id, product_id, rating, comment, created_at)
select
    ba.ids[(((((g - 1) / :review_focus) % ba.cnt) + 1))::integer],
    pa.product_ids[(((g - 1) % least(:review_focus, pa.cnt)) + 1)::integer],
    (g % 5) + 1,
    'Synthetic review for RPZ research benchmark',
    timestamp '2025-01-01' + ((g % 400) * interval '1 day')
from generate_series(1, :review_count) as g
cross join research_buyer_array ba
cross join research_product_array pa
on conflict (user_id, product_id) do nothing;

alter table reviews enable trigger user;

analyze users;
analyze sellers;
analyze products;
analyze orders;
analyze order_items;
analyze reviews;

\echo 'Research data prepared.'
