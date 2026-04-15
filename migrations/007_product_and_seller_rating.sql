-- +goose Up

-- Add rating column to products
ALTER TABLE products ADD COLUMN rating numeric(3,2);

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION update_product_and_seller_rating()
RETURNS TRIGGER AS $$
DECLARE
    v_seller_id bigint;
    v_product_id bigint;
BEGIN
    -- Determine which product_id was affected
    IF TG_OP = 'DELETE' THEN
        v_product_id := OLD.product_id;
    ELSE
        v_product_id := NEW.product_id;
    END IF;

    -- Recalculate product rating
    UPDATE products
    SET rating = (
        SELECT AVG(r.rating)::numeric(3,2)
        FROM reviews r
        WHERE r.product_id = v_product_id
    )
    WHERE id = v_product_id
    RETURNING seller_id INTO v_seller_id;

    IF v_seller_id IS NULL THEN
        RETURN COALESCE(NEW, OLD);
    END IF;

    -- Recalculate seller rating as average of all their products' ratings
    UPDATE sellers
    SET rating = (
        SELECT AVG(p.rating)::numeric(3,2)
        FROM products p
        WHERE p.seller_id = v_seller_id
          AND p.deleted_at IS NULL
          AND p.rating IS NOT NULL
    )
    WHERE id = v_seller_id;

    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER update_ratings_on_review
    AFTER INSERT OR UPDATE OF rating OR DELETE ON reviews
    FOR EACH ROW
    EXECUTE FUNCTION update_product_and_seller_rating();

-- +goose Down
DROP TRIGGER IF EXISTS update_ratings_on_review ON reviews;
DROP FUNCTION IF EXISTS update_product_and_seller_rating;
ALTER TABLE products DROP COLUMN IF EXISTS rating;
