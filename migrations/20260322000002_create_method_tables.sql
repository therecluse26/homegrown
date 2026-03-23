-- +goose Up
-- Migration: 20260322000002_create_method_tables.sql
-- Method domain: methodology definitions, master tool catalog, per-methodology tool activations.
-- These are platform-managed global configuration tables — NO RLS policies. [ARCH §5.1, §5.2]
--
-- Slug is the natural PK for both method_definitions and method_tools.
-- These are immutable seed tables; slug IS the canonical identifier — no UUID indirection needed.
-- [02-method §8.3, ADR: natural-string-pks]

-- Platform-defined methodologies [S§4.1, S§4.5]
-- These are global config rows, NOT user data — no RLS needed.
CREATE TABLE method_definitions (
    slug              TEXT PRIMARY KEY,                        -- e.g., 'charlotte-mason'
    display_name      TEXT NOT NULL,                           -- e.g., 'Charlotte Mason'
    short_desc        TEXT NOT NULL,                           -- one-line summary for lists
    icon_url          TEXT,                                    -- CDN URL for methodology icon
    philosophy        JSONB NOT NULL DEFAULT '{}',             -- philosophy module [S§4.1]
    onboarding_config JSONB NOT NULL DEFAULT '{}',             -- roadmaps, starter recs [S§6.4]
    community_config  JSONB NOT NULL DEFAULT '{}',             -- group IDs, mentor criteria [S§6.6]
    mastery_paths     JSONB NOT NULL DEFAULT '{}',             -- beginner/intermediate/advanced [S§4.1]
    terminology       JSONB NOT NULL DEFAULT '{}',             -- methodology-specific labels [S§4.4]
    display_order     SMALLINT NOT NULL DEFAULT 0,             -- controls listing order
    is_active         BOOLEAN NOT NULL DEFAULT true,           -- soft-disable without deletion
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_method_definitions_active ON method_definitions(is_active, display_order);

-- Master tool catalog [S§4.2, S§8.1]
-- Every learning tool on the platform. Tools exist independently of methodologies.
CREATE TABLE method_tools (
    slug          TEXT PRIMARY KEY,                            -- e.g., 'reading-lists'
    display_name  TEXT NOT NULL,                               -- e.g., 'Reading Lists'
    description   TEXT,                                        -- tool description for UI
    config_schema JSONB NOT NULL DEFAULT '{}',                 -- JSON Schema for tool config
    tier          TEXT NOT NULL DEFAULT 'free'
                  CHECK (tier IN ('free', 'premium')),         -- subscription gating [S§15.2]
    is_active     BOOLEAN NOT NULL DEFAULT true,               -- soft-disable tool globally
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_method_tools_active ON method_tools(is_active);

-- Per-methodology tool activation [S§4.2]
-- Many-to-many: which tools are active for which methodology, with config overrides.
CREATE TABLE method_tool_activations (
    methodology_slug TEXT NOT NULL REFERENCES method_definitions(slug) ON DELETE CASCADE,
    tool_slug        TEXT NOT NULL REFERENCES method_tools(slug) ON DELETE CASCADE,
    config_overrides JSONB NOT NULL DEFAULT '{}',              -- methodology-specific labels, guidance
    sort_order       SMALLINT NOT NULL DEFAULT 0,              -- tool display order within methodology
    PRIMARY KEY (methodology_slug, tool_slug)
);

CREATE INDEX idx_method_activations_tool ON method_tool_activations(tool_slug);

-- +goose Down
DROP TABLE IF EXISTS method_tool_activations;
DROP TABLE IF EXISTS method_tools;
DROP TABLE IF EXISTS method_definitions;
