-- Bloom Scents schema

CREATE TABLE IF NOT EXISTS customers (
    id SERIAL PRIMARY KEY,
    full_name TEXT NOT NULL,
    email TEXT UNIQUE NOT NULL,
    phone TEXT,
    password_hash TEXT NOT NULL,
    delivery_address TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS admins (
    id SERIAL PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS products (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    scent_notes TEXT,
    description TEXT,
    price_kobo INTEGER NOT NULL, -- store in kobo (NGN * 100) for Paystack
    image_url TEXT,
    in_stock BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS orders (
    id SERIAL PRIMARY KEY,
    customer_id INTEGER REFERENCES customers(id),
    product_id INTEGER REFERENCES products(id),
    quantity INTEGER NOT NULL DEFAULT 1,
    amount_kobo INTEGER NOT NULL,
    paystack_reference TEXT UNIQUE,
    status TEXT NOT NULL DEFAULT 'pending', -- pending, paid, failed
    delivery_address TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Seed a few placeholder products (edit freely)
INSERT INTO products (name, scent_notes, description, price_kobo, image_url) VALUES
('Bloom Rose Gold', 'Rose, Amber, Vanilla', 'A warm floral signature scent for evenings.', 1500000, ''),
('Bloom Jasmine Veil', 'Jasmine, White Musk, Sandalwood', 'Soft and elegant, perfect for daily wear.', 1200000, ''),
('Bloom Noir', 'Oud, Black Pepper, Patchouli', 'Bold and luxurious, for the confident woman.', 1800000, '')
ON CONFLICT DO NOTHING;