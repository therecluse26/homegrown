-- +goose Up
-- Search domain Phase 2 prep table. [12-search §3.2]
-- Tracks per-collection Typesense sync state for incremental indexing.
-- Phase 1 has no owned tables (uses domain search_vector columns + GIN indexes).
-- This table is created now so the migration sequence is established.

-- ═══════════════════════════════════════════════════════════════════════════════
-- TABLE 1: search_index_state — Typesense index sync tracking (Phase 2)
-- ═══════════════════════════════════════════════════════════════════════════════
CREATE TABLE search_index_state (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    collection_name TEXT        NOT NULL UNIQUE
                    CHECK (collection_name IN (
                        'marketplace_listings', 'social_posts',
                        'social_groups', 'social_events'
                    )),
    last_synced_at  TIMESTAMPTZ NOT NULL DEFAULT '1970-01-01T00:00:00Z',
    last_synced_id  UUID,                   -- last processed entity ID (cursor for incremental sync)
    document_count  BIGINT      NOT NULL DEFAULT 0,
    status          TEXT        NOT NULL DEFAULT 'inactive'
                    CHECK (status IN ('inactive', 'syncing', 'active', 'error')),
    error_message   TEXT,                   -- last error details (internal only, never exposed in API)
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ═══════════════════════════════════════════════════════════════════════════════
-- RLS Policies [12-search §3.3]
-- ═══════════════════════════════════════════════════════════════════════════════
-- search_index_state: system role only (background jobs manage sync state).
-- Phase 2 table — RLS defined in Phase 1 migration for forward compatibility.
ALTER TABLE search_index_state ENABLE ROW LEVEL SECURITY;

CREATE POLICY search_index_state_system_all
    ON search_index_state FOR ALL
    WITH CHECK (current_setting('app.role') = 'system');

-- +goose Down
DROP POLICY IF EXISTS search_index_state_system_all ON search_index_state;
DROP TABLE IF EXISTS search_index_state;
