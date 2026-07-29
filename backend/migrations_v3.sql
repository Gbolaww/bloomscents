-- Bloom Scents schema v3: Google Sign-In support
-- Run this against your existing database.

ALTER TABLE customers ADD COLUMN IF NOT EXISTS google_id TEXT UNIQUE;