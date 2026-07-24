-- Bloom Scents schema v2: multi-item cart support
-- Run this AFTER migrations.sql, against your existing database.

ALTER TABLE orders RENAME COLUMN amount_kobo TO total_amount_kobo;
ALTER TABLE orders DROP COLUMN IF EXISTS product_id;
ALTER TABLE orders DROP COLUMN IF EXISTS quantity;

CREATE TABLE IF NOT EXISTS order_items (
    id SERIAL PRIMARY KEY,
    order_id INTEGER REFERENCES orders(id) ON DELETE CASCADE,
    product_id INTEGER REFERENCES products(id),
    quantity INTEGER NOT NULL DEFAULT 1,
    price_kobo INTEGER NOT NULL -- price at time of purchase, so later price changes don't affect old orders
);