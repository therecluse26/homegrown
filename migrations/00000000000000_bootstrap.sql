-- +goose Up
-- Migration: 00000000000000_bootstrap.sql
-- Installs PostgreSQL extensions required by the platform.

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";     -- UUID generation
CREATE EXTENSION IF NOT EXISTS "pgcrypto";       -- Cryptographic functions
CREATE EXTENSION IF NOT EXISTS "postgis";        -- Spatial queries [ARCH §5.4]
CREATE EXTENSION IF NOT EXISTS "pg_trgm";        -- Trigram fuzzy matching [ARCH §2.6]

-- +goose Down
DROP EXTENSION IF EXISTS "pg_trgm";
DROP EXTENSION IF EXISTS "postgis";
DROP EXTENSION IF EXISTS "pgcrypto";
DROP EXTENSION IF EXISTS "uuid-ossp";
