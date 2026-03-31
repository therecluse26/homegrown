-- +goose Up
-- Migration: 20260323000009_create_soc_tables.sql
-- Social & Community domain tables (social::) [05-social §3]
-- 13 tables + RLS policies + indexes

-- ═══════════════════════════════════════════════════════════════════════════════
-- Tables created in dependency order:
--   1. soc_profiles         (FK → iam_families)
--   2. soc_friendships      (FK → iam_families)
--   3. soc_blocks           (FK → iam_families)
--   4. soc_groups           (FK → method_definitions, iam_families)
--   5. soc_group_members    (FK → soc_groups, iam_families)
--   6. soc_posts            (FK → soc_groups, iam_families, iam_parents)
--   7. soc_comments         (FK → soc_posts, iam_families, iam_parents)
--   8. soc_post_likes       (FK → soc_posts, iam_families)
--   9. soc_conversations
--  10. soc_conversation_participants (FK → soc_conversations, iam_parents, iam_families)
--  11. soc_messages          (FK → soc_conversations, iam_parents, iam_families)
--  12. soc_events            (FK → soc_groups, iam_families, iam_parents, method_definitions)
--  13. soc_event_rsvps       (FK → soc_events, iam_families)
-- ═══════════════════════════════════════════════════════════════════════════════

-- 1. Family social profiles [S§7.1]
-- 1:1 with iam_families. Created when FamilyCreated event is received.
CREATE TABLE soc_profiles (
    family_id             UUID PRIMARY KEY REFERENCES iam_families(id) ON DELETE CASCADE,
    bio                   TEXT,
    profile_photo_url     TEXT,
    privacy_settings      JSONB NOT NULL DEFAULT '{
        "display_name": "friends",
        "parent_names": "friends",
        "children_names": "friends",
        "children_ages": "friends",
        "location": "friends",
        "methodology": "friends"
    }',
    location_visible      BOOLEAN NOT NULL DEFAULT false,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- 2. Friendships (bidirectional) [S§7.4]
CREATE TABLE soc_friendships (
    id                    UUID PRIMARY KEY DEFAULT uuidv7(),
    requester_family_id   UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    accepter_family_id    UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    status                TEXT NOT NULL DEFAULT 'pending'
                          CHECK (status IN ('pending', 'accepted')),
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_friendship_pair UNIQUE (requester_family_id, accepter_family_id),
    CONSTRAINT ck_no_self_friend CHECK (requester_family_id != accepter_family_id)
);

CREATE INDEX idx_soc_friendships_requester ON soc_friendships(requester_family_id, status);
CREATE INDEX idx_soc_friendships_accepter ON soc_friendships(accepter_family_id, status);

-- 3. Block lookup table [S§7.4]
CREATE TABLE soc_blocks (
    id                    UUID PRIMARY KEY DEFAULT uuidv7(),
    blocker_family_id     UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    blocked_family_id     UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_block_pair UNIQUE (blocker_family_id, blocked_family_id),
    CONSTRAINT ck_no_self_block CHECK (blocker_family_id != blocked_family_id)
);

CREATE INDEX idx_soc_blocks_blocker ON soc_blocks(blocker_family_id);
CREATE INDEX idx_soc_blocks_blocked ON soc_blocks(blocked_family_id);

-- 4. Groups [S§7.6]
-- NOTE: methodology_slug TEXT (not UUID) per Natural String PK Refactor.
CREATE TABLE soc_groups (
    id                    UUID PRIMARY KEY DEFAULT uuidv7(),
    group_type            TEXT NOT NULL CHECK (group_type IN ('platform', 'user_created')),
    name                  TEXT NOT NULL,
    description           TEXT,
    cover_photo_url       TEXT,
    creator_family_id     UUID REFERENCES iam_families(id),
    methodology_slug      TEXT REFERENCES method_definitions(slug),
    join_policy           TEXT NOT NULL DEFAULT 'open'
                          CHECK (join_policy IN ('open', 'request_to_join', 'invite_only')),
    member_count          INTEGER NOT NULL DEFAULT 0,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_soc_groups_type ON soc_groups(group_type);
CREATE INDEX idx_soc_groups_methodology ON soc_groups(methodology_slug)
    WHERE methodology_slug IS NOT NULL;
CREATE INDEX idx_soc_groups_search ON soc_groups
    USING GIN(to_tsvector('english', coalesce(name, '') || ' ' || coalesce(description, '')));

-- 5. Group members [S§7.6]
CREATE TABLE soc_group_members (
    id                    UUID PRIMARY KEY DEFAULT uuidv7(),
    group_id              UUID NOT NULL REFERENCES soc_groups(id) ON DELETE CASCADE,
    family_id             UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    role                  TEXT NOT NULL DEFAULT 'member'
                          CHECK (role IN ('member', 'moderator', 'owner')),
    status                TEXT NOT NULL DEFAULT 'active'
                          CHECK (status IN ('active', 'pending', 'invited', 'banned')),
    joined_at             TIMESTAMPTZ,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_group_member UNIQUE (group_id, family_id)
);

CREATE INDEX idx_soc_group_members_group ON soc_group_members(group_id, status);
CREATE INDEX idx_soc_group_members_family ON soc_group_members(family_id, status);

-- 6. Posts [S§7.2]
CREATE TABLE soc_posts (
    id                    UUID PRIMARY KEY DEFAULT uuidv7(),
    family_id             UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    author_parent_id      UUID NOT NULL REFERENCES iam_parents(id),
    post_type             TEXT NOT NULL CHECK (post_type IN (
                              'text', 'photo', 'milestone', 'event_share',
                              'marketplace_review', 'resource_share'
                          )),
    content               TEXT,
    attachments           JSONB NOT NULL DEFAULT '[]',
    group_id              UUID REFERENCES soc_groups(id) ON DELETE CASCADE,
    visibility            TEXT NOT NULL DEFAULT 'friends'
                          CHECK (visibility IN ('friends', 'group')),
    CONSTRAINT ck_post_visibility CHECK (
        (group_id IS NULL AND visibility = 'friends') OR
        (group_id IS NOT NULL AND visibility = 'group')
    ),
    likes_count           INTEGER NOT NULL DEFAULT 0,
    comments_count        INTEGER NOT NULL DEFAULT 0,
    is_edited             BOOLEAN NOT NULL DEFAULT false,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_soc_posts_family ON soc_posts(family_id, created_at DESC);
CREATE INDEX idx_soc_posts_group ON soc_posts(group_id, created_at DESC)
    WHERE group_id IS NOT NULL;

-- Full-text search index on posts [S§14.1]
ALTER TABLE soc_posts ADD COLUMN search_vector tsvector
    GENERATED ALWAYS AS (to_tsvector('english', coalesce(content, ''))) STORED;
CREATE INDEX idx_soc_posts_search ON soc_posts USING GIN(search_vector);

-- 7. Comments [S§7.3]
CREATE TABLE soc_comments (
    id                    UUID PRIMARY KEY DEFAULT uuidv7(),
    post_id               UUID NOT NULL REFERENCES soc_posts(id) ON DELETE CASCADE,
    family_id             UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    author_parent_id      UUID NOT NULL REFERENCES iam_parents(id),
    parent_comment_id     UUID REFERENCES soc_comments(id) ON DELETE CASCADE,
    content               TEXT NOT NULL,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_soc_comments_post ON soc_comments(post_id, created_at);

-- 8. Post likes [S§7.2]
CREATE TABLE soc_post_likes (
    id                    UUID PRIMARY KEY DEFAULT uuidv7(),
    post_id               UUID NOT NULL REFERENCES soc_posts(id) ON DELETE CASCADE,
    family_id             UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_post_like UNIQUE (post_id, family_id)
);

CREATE INDEX idx_soc_post_likes_post ON soc_post_likes(post_id);

-- 9. DM conversation abstraction [S§7.5]
CREATE TABLE soc_conversations (
    id                    UUID PRIMARY KEY DEFAULT uuidv7(),
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- 10. Conversation participants [S§7.5]
CREATE TABLE soc_conversation_participants (
    conversation_id       UUID NOT NULL REFERENCES soc_conversations(id) ON DELETE CASCADE,
    parent_id             UUID NOT NULL REFERENCES iam_parents(id) ON DELETE CASCADE,
    family_id             UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    last_read_at          TIMESTAMPTZ,
    deleted_at            TIMESTAMPTZ,
    PRIMARY KEY (conversation_id, parent_id)
);

CREATE INDEX idx_soc_conv_participants_parent ON soc_conversation_participants(parent_id)
    WHERE deleted_at IS NULL;

-- 11. Direct messages [S§7.5]
CREATE TABLE soc_messages (
    id                    UUID PRIMARY KEY DEFAULT uuidv7(),
    conversation_id       UUID NOT NULL REFERENCES soc_conversations(id) ON DELETE CASCADE,
    sender_parent_id      UUID NOT NULL REFERENCES iam_parents(id),
    sender_family_id      UUID NOT NULL REFERENCES iam_families(id),
    content               TEXT NOT NULL,
    attachments           JSONB NOT NULL DEFAULT '[]',
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_soc_messages_conversation ON soc_messages(conversation_id, created_at);

-- 12. Events [S§7.7]
-- NOTE: methodology_slug TEXT (not UUID) per Natural String PK Refactor.
CREATE TABLE soc_events (
    id                    UUID PRIMARY KEY DEFAULT uuidv7(),
    creator_family_id     UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    creator_parent_id     UUID NOT NULL REFERENCES iam_parents(id),
    group_id              UUID REFERENCES soc_groups(id) ON DELETE SET NULL,
    title                 TEXT NOT NULL,
    description           TEXT,
    event_date            TIMESTAMPTZ NOT NULL,
    end_date              TIMESTAMPTZ,
    location_name         TEXT,
    location_region       TEXT,
    is_virtual            BOOLEAN NOT NULL DEFAULT false,
    virtual_url           TEXT,
    capacity              INTEGER,
    visibility            TEXT NOT NULL DEFAULT 'friends'
                          CHECK (visibility IN ('friends', 'group', 'discoverable')),
    status                TEXT NOT NULL DEFAULT 'active'
                          CHECK (status IN ('active', 'cancelled')),
    methodology_slug      TEXT REFERENCES method_definitions(slug),
    attendee_count        INTEGER NOT NULL DEFAULT 0,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_soc_events_creator ON soc_events(creator_family_id);
CREATE INDEX idx_soc_events_group ON soc_events(group_id) WHERE group_id IS NOT NULL;
CREATE INDEX idx_soc_events_date ON soc_events(event_date)
    WHERE status = 'active';
CREATE INDEX idx_soc_events_discoverable ON soc_events(visibility, event_date)
    WHERE visibility = 'discoverable' AND status = 'active';
CREATE INDEX idx_soc_events_search ON soc_events
    USING GIN(to_tsvector('english', coalesce(title, '') || ' ' || coalesce(description, '')));

-- 13. Event RSVPs [S§7.7]
CREATE TABLE soc_event_rsvps (
    id                    UUID PRIMARY KEY DEFAULT uuidv7(),
    event_id              UUID NOT NULL REFERENCES soc_events(id) ON DELETE CASCADE,
    family_id             UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    status                TEXT NOT NULL DEFAULT 'going'
                          CHECK (status IN ('going', 'interested', 'not_going')),
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_event_rsvp UNIQUE (event_id, family_id)
);

CREATE INDEX idx_soc_event_rsvps_event ON soc_event_rsvps(event_id, status);

-- Family scoping is enforced at the GORM level via ScopedTransaction (ADR-008).
-- PostgreSQL RLS is NOT used.

-- +goose Down
DROP TABLE IF EXISTS soc_event_rsvps CASCADE;
DROP TABLE IF EXISTS soc_events CASCADE;
DROP TABLE IF EXISTS soc_messages CASCADE;
DROP TABLE IF EXISTS soc_conversation_participants CASCADE;
DROP TABLE IF EXISTS soc_conversations CASCADE;
DROP TABLE IF EXISTS soc_post_likes CASCADE;
DROP TABLE IF EXISTS soc_comments CASCADE;
DROP TABLE IF EXISTS soc_posts CASCADE;
DROP TABLE IF EXISTS soc_group_members CASCADE;
DROP TABLE IF EXISTS soc_groups CASCADE;
DROP TABLE IF EXISTS soc_blocks CASCADE;
DROP TABLE IF EXISTS soc_friendships CASCADE;
DROP TABLE IF EXISTS soc_profiles CASCADE;
