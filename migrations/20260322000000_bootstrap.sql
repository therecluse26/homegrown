-- +goose Up
-- Migration: 00000000000000_bootstrap.sql
-- Installs PostgreSQL extensions required by the platform.

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";     -- UUID generation
CREATE EXTENSION IF NOT EXISTS "pgcrypto";       -- Cryptographic functions
CREATE EXTENSION IF NOT EXISTS "postgis";        -- Spatial queries [ARCH §5.4]
CREATE EXTENSION IF NOT EXISTS "pg_trgm";        -- Trigram fuzzy matching [ARCH §2.6]

-- +goose Down
DROP EXTENSION IF EXISTS "pg_trgm";
-- CASCADE: the official postgis Docker image pre-installs postgis_topology (and
-- tiger), which depend on postgis. A plain DROP fails with 2BP01 in those
-- databases. CASCADE only runs during a full teardown (down-to 0); on a
-- goose-provisioned DB there are no dependent objects to cascade.
DROP EXTENSION IF EXISTS "postgis" CASCADE;
DROP EXTENSION IF EXISTS "pgcrypto";
DROP EXTENSION IF EXISTS "uuid-ossp";
