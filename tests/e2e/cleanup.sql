-- psql quotes the JSON variable as a SQL literal. No user strings are interpolated into SQL syntax.
\set QUIET on
BEGIN;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '15s';
CREATE TEMP TABLE run_manifest AS SELECT :'manifest'::jsonb AS value;
CREATE TEMP TABLE run_users AS
SELECT u.id FROM users u, run_manifest m
WHERE u.email IN (SELECT jsonb_each_text.value FROM jsonb_each_text(m.value->'emails'))
AND u.full_name = 'Test ' || (m.value->>'marker');
SELECT id FROM users WHERE id IN (SELECT id FROM run_users) ORDER BY id FOR UPDATE;
CREATE TEMP TABLE run_sellers AS SELECT id FROM sellers WHERE user_id IN (SELECT id FROM run_users);
SELECT id FROM sellers WHERE id IN (SELECT id FROM run_sellers) ORDER BY id FOR UPDATE;
CREATE TEMP TABLE run_products AS
SELECT p.id FROM products p, run_manifest m WHERE p.seller_id IN (SELECT id FROM run_sellers) AND p.name = m.value->>'marker';
SELECT id FROM products WHERE id IN (SELECT id FROM run_products) ORDER BY id FOR UPDATE;
CREATE TEMP TABLE run_addresses AS SELECT id FROM addresses WHERE user_id IN (SELECT id FROM run_users);
SELECT id FROM addresses WHERE id IN (SELECT id FROM run_addresses) ORDER BY id FOR UPDATE;
CREATE TEMP TABLE run_orders AS SELECT id FROM orders WHERE user_id IN (SELECT id FROM run_users);
SELECT id FROM orders WHERE id IN (SELECT id FROM run_orders) ORDER BY id FOR UPDATE;
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM users u, run_manifest m
               WHERE u.email IN (SELECT jsonb_each_text.value FROM jsonb_each_text(m.value->'emails'))
               AND u.id NOT IN (SELECT id FROM run_users)) THEN
        RAISE EXCEPTION 'cleanup refused: email exists without run ownership marker';
    END IF;
    IF EXISTS (SELECT 1 FROM products WHERE seller_id IN (SELECT id FROM run_sellers) AND id NOT IN (SELECT id FROM run_products))
       OR EXISTS (SELECT 1 FROM orders WHERE user_id NOT IN (SELECT id FROM run_users)
                  AND (seller_id IN (SELECT id FROM run_sellers) OR address_id IN (SELECT id FROM run_addresses)))
       OR EXISTS (SELECT 1 FROM order_items WHERE product_id IN (SELECT id FROM run_products) AND order_id NOT IN (SELECT id FROM run_orders))
       OR EXISTS (SELECT 1 FROM order_items WHERE order_id IN (SELECT id FROM run_orders) AND product_id NOT IN (SELECT id FROM run_products))
       OR EXISTS (SELECT 1 FROM reviews WHERE product_id IN (SELECT id FROM run_products) AND user_id NOT IN (SELECT id FROM run_users)) THEN
        RAISE EXCEPTION 'cleanup refused: foreign references to run data';
    END IF;
END $$;
\pset format unaligned
\pset tuples_only on
SELECT 'CACHE_PRODUCT=' || id FROM run_products ORDER BY id;
DELETE FROM orders WHERE id IN (SELECT id FROM run_orders);
DELETE FROM reviews WHERE user_id IN (SELECT id FROM run_users);
DELETE FROM products WHERE id IN (SELECT id FROM run_products);
DELETE FROM sellers WHERE id IN (SELECT id FROM run_sellers);
DELETE FROM users WHERE id IN (SELECT id FROM run_users);
COMMIT;
