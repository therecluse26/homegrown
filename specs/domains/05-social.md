# Domain Spec 05 — Social & Community (social::)

## §1 Overview

The Social domain owns the platform's **purpose-built private social network** for
homeschooling families. It provides profiles, timeline/feed, comments, friends, direct
messaging, groups, events, and location-based discovery. All user-generated content
defaults to **friends-only** visibility — there is no public visibility option. `[S§7, V§7]`

| Attribute | Value |
|-----------|-------|
| **Module path** | `internal/social/` |
| **DB prefix** | `soc_` |
| **Complexity class** | Complex (has `domain/` subdirectory) `[ARCH §4.5]` |
| **CQRS** | Yes — feed reads separated from post writes `[ARCH §4.7]` |
| **External adapter** | None (Redis is shared infrastructure, WebSocket via gorilla/websocket) |
| **Key constraint** | No public visibility; blocking maps to 404 (silent); friends-only messaging `[V§7]` |

**What social:: owns**: Family social profiles (1:1 with `iam_families`), friendships
(bidirectional, with blocking), posts (6 types, friends/group visibility), comments
(one-level threading), post likes, direct messaging conversations (parent-to-parent),
groups (platform-managed and user-created), group membership, events (with RSVP),
feed fan-out (Redis sorted sets), WebSocket real-time delivery, visibility enforcement,
and block enforcement.

**What social:: does NOT own**: Family accounts or parent records (owned by `iam::`),
methodology definitions (owned by `method::`), file uploads or media processing
(owned by `media::`), content moderation pipeline or CSAM scanning (owned by `safety::`),
search indexing (owned by `search::`), notification delivery (owned by `notify::`),
learning milestones (owned by `learn::`).

**What social:: delegates**: Media upload/validation → `media::` (via domain events or
direct service call). Content scanning → `safety::` (via `PostCreated`, `MessageSent`
events). Search indexing → `search::` (via `PostCreated` event). Notification delivery →
`notify::` (via domain events). Family/parent data lookup → `iam::IamService`.
Methodology display names → `method::MethodologyService`.

---

## §2 Requirements Traceability

Every SPEC.md §7 requirement maps to a section in this document. Cross-references from
other spec sections are included where the social domain is involved.

| Requirement | SPEC Reference | Domain Spec Section |
|-------------|---------------|---------------------|
| Family profiles visible to friends | `[S§7.1]` | §3.2 (`soc_profiles`), §4.1 |
| Per-field privacy controls (friends-only, hidden) | `[S§7.1]` | §3.2 (`privacy_settings` JSONB), §9 |
| No public profiles | `[S§7.1, V§7]` | §3.2 (no `public` visibility), §9 |
| Profile photos MAY be supported (family/parent) | `[S§7.1]` | §18 (Phase 2 — deferred) |
| Student photos default to generic avatar | `[S§7.1]` | §8.2 (`ProfileResponse`) |
| 6 post types (text, photo, milestone, event_share, marketplace_review, resource_share) | `[S§7.2.1]` | §3.2 (`soc_posts`), §8.1 |
| Friends-only default visibility for posts | `[S§7.2.2]` | §3.2 (CHECK constraint), §9 |
| Group posts visible to group members only | `[S§7.2.2]` | §3.2 (CHECK constraint), §9 |
| No public visibility for user-generated content | `[S§7.2.2, V§7]` | §3.1 (no `public` enum value), §9 |
| Reverse-chronological feed | `[S§7.2.3]` | §11 (feed architecture) |
| Algorithmic ranking deferred to post-MVP | `[S§7.2.3]` | §18 (Phase 3+) |
| Comment threading (one level of reply) | `[S§7.3]` | §3.2 (`soc_comments.parent_comment_id`) |
| Comments inherit parent post visibility | `[S§7.3]` | §9 |
| Comment deletion by author and post author | `[S§7.3]` | §4.1, §5 |
| Bidirectional friend requests with mutual acceptance | `[S§7.4]` | §3.2 (`soc_friendships`), §10 |
| Friend discovery (methodology, groups, location, name) | `[S§7.4]` | §4.2, §15 |
| Silent blocking (blocked user not notified) | `[S§7.4]` | §3.2 (`soc_blocks`), §9, §16 |
| Blocking prevents all interaction including profile view | `[S§7.4]` | §9, §16 |
| Unfriending without notification | `[S§7.4]` | §10 |
| Parent-to-parent direct messaging only | `[S§7.5]` | §3.2 (`soc_messages`), §12 |
| Friends-only messaging for MVP | `[S§7.5]` | §12 |
| Text and image attachments in messages | `[S§7.5]` | §3.2 (`soc_messages.attachments`), §12 |
| Message reporting for moderation | `[S§7.5]` | §17.3 (`MessageReported` event) |
| Per-user message deletion | `[S§7.5]` | §3.2 (`soc_conversation_participants.deleted_at`), §12 |
| Platform-managed methodology groups | `[S§7.6]` | §3.2 (`soc_groups.group_type = 'platform'`), §13 |
| User-created groups with customizable rules | `[S§7.6]` | §3.2 (`soc_groups.group_type = 'user_created'`), §13 |
| Group join policies: open, request-to-join, invite-only | `[S§7.6]` | §3.2 (`soc_groups.join_policy`), §13 |
| Group moderators | `[S§7.6]` | §3.2 (`soc_group_members.role`), §13 |
| Group-specific events | `[S§7.6]` | §3.2 (`soc_events.group_id`), §14 |
| Pinned posts in groups | `[S§7.6]` | §4.2, §18 (Phase 2) |
| Event creation with title, description, date/time, location, capacity | `[S§7.7]` | §3.2 (`soc_events`), §14 |
| RSVP (going, interested, not_going) | `[S§7.7]` | §3.2 (`soc_event_rsvps`), §14 |
| Recurring events | `[S§7.7]` | §18 (Phase 2) |
| Event discovery by location, group, methodology | `[S§7.7]` | §4.2, §14, §15 |
| Event visibility: friends-only, group-only, discoverable | `[S§7.7]` | §3.2 (`soc_events.visibility`), §14 |
| Event cancellation with notification | `[S§7.7]` | §14, §17.3 (`EventCancelled` event) |
| Location-based discovery is opt-in | `[S§7.8]` | §3.2 (`soc_profiles.location_visible`), §15 |
| Coarse-grained location (city/region, never coordinates) | `[S§7.8]` | §15 (PostGIS on `iam_families`, not `soc_`) |
| Location-based discovery: families, groups, events | `[S§7.8]` | §15 |
| Disable location sharing removes from results immediately | `[S§7.8]` | §15 |
| No precise geolocation stored | `[S§7.8, ARCH §1.5]` | §15 |
| Permission matrix: all social features free tier | `[S§3.2, S§15.1]` | §4.1 (no `RequirePremium`) |
| Parent-mediated access model | `[S§3.3]` | §12 (messaging is parent-to-parent only) |
| Content moderation pipeline for social content | `[S§12.2]` | §17.3 (`PostCreated`, `MessageReported` events → `safety::`) |
| User reporting for posts, comments, messages, profiles, groups, events | `[S§12.3]` | §17.3 (`MessageReported` event) |
| Social notifications (friend requests, messages, comments, group/event invitations) | `[S§13.1]` | §17.3 (events → `notify::`) |
| Social search (users by name, groups, events) | `[S§14.1]` | §3.2 (`search_vector`), §17.3 (`PostCreated` → `search::`) |
| Social discovery respects privacy | `[S§14.2]` | §9, §15 |
| Co-parent removal preserves but disassociates posts | `[S§3.4]` | §17.4 (`CoParentRemoved` subscriber) |
| Family deletion cascades social data | `[S§16.3]` | §17.4 (`FamilyDeletionScheduled` subscriber) |

---

## §3 Database Schema

All tables use the `soc_` prefix. Social tables contain **cross-family data** — friendships,
groups, and events involve multiple families. RLS policies are nuanced: some tables use
standard family-scoping, others require participant-based or membership-based policies.
`[ARCH §5.1]`

### §3.1 Enums

All enums use CHECK constraints for consistency with other domain specs and to avoid
PostgreSQL enum migration limitations. `[ARCH §5.2]`

```sql
-- Post type, friendship status, group type, join policy, member role/status,
-- event visibility, event status, RSVP status, and message type are all
-- enforced via CHECK constraints on their respective columns rather than
-- as PostgreSQL enum types. This avoids ALTER TYPE limitations when adding
-- new values in future migrations. [ARCH §5.2]
```

### §3.2 Tables

```sql
-- =============================================================================
-- Migration: YYYYMMDD_000001_create_soc_tables.sql (goose migration)
-- =============================================================================

-- Family social profiles [S§7.1]
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
    }',                                              -- per-field: "friends" | "hidden" [S§7.1]
    location_visible      BOOLEAN NOT NULL DEFAULT false, -- opt-in for location discovery [S§7.8]
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Friendships (bidirectional) [S§7.4]
-- Status: pending → accepted. Block is tracked separately in soc_blocks.
CREATE TABLE soc_friendships (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
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

-- Block lookup table [S§7.4]
-- Separate from friendships for O(1) block checks.
-- Blocking can happen without a prior friendship.
CREATE TABLE soc_blocks (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    blocker_family_id     UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    blocked_family_id     UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_block_pair UNIQUE (blocker_family_id, blocked_family_id),
    CONSTRAINT ck_no_self_block CHECK (blocker_family_id != blocked_family_id)
);

CREATE INDEX idx_soc_blocks_blocker ON soc_blocks(blocker_family_id);
CREATE INDEX idx_soc_blocks_blocked ON soc_blocks(blocked_family_id);

-- Posts [S§7.2]
-- 6 post types, visibility friends/group only (no public). [S§7.2.2]
CREATE TABLE soc_posts (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id             UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    author_parent_id      UUID NOT NULL REFERENCES iam_parents(id),
    post_type             TEXT NOT NULL CHECK (post_type IN (
                              'text', 'photo', 'milestone', 'event_share',
                              'marketplace_review', 'resource_share'
                          )),                        -- [S§7.2.1]
    content               TEXT,
    attachments           JSONB NOT NULL DEFAULT '[]', -- array of media references
    group_id              UUID REFERENCES soc_groups(id) ON DELETE CASCADE,
    visibility            TEXT NOT NULL DEFAULT 'friends'
                          CHECK (visibility IN ('friends', 'group')),
    -- Enforce: group posts must have group visibility, non-group posts must have friends
    CONSTRAINT ck_post_visibility CHECK (
        (group_id IS NULL AND visibility = 'friends') OR
        (group_id IS NOT NULL AND visibility = 'group')
    ),
    likes_count           INTEGER NOT NULL DEFAULT 0,
    comments_count        INTEGER NOT NULL DEFAULT 0,
    is_edited             BOOLEAN NOT NULL DEFAULT false,      -- Phase 2: post editing
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

-- Comments [S§7.3]
-- One level of threading via parent_comment_id.
CREATE TABLE soc_comments (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    post_id               UUID NOT NULL REFERENCES soc_posts(id) ON DELETE CASCADE,
    family_id             UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    author_parent_id      UUID NOT NULL REFERENCES iam_parents(id),
    parent_comment_id     UUID REFERENCES soc_comments(id) ON DELETE CASCADE,
    content               TEXT NOT NULL,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_soc_comments_post ON soc_comments(post_id, created_at);

-- Post likes [S§7.2]
-- Deduplicated: one like per family per post.
CREATE TABLE soc_post_likes (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    post_id               UUID NOT NULL REFERENCES soc_posts(id) ON DELETE CASCADE,
    family_id             UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_post_like UNIQUE (post_id, family_id)
);

CREATE INDEX idx_soc_post_likes_post ON soc_post_likes(post_id);

-- DM conversation abstraction [S§7.5]
-- Groups messages between two parents. Supports per-user deletion.
CREATE TABLE soc_conversations (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Conversation participants [S§7.5]
-- Per-user deletion via deleted_at, per-user read tracking via last_read_at.
CREATE TABLE soc_conversation_participants (
    conversation_id       UUID NOT NULL REFERENCES soc_conversations(id) ON DELETE CASCADE,
    parent_id             UUID NOT NULL REFERENCES iam_parents(id) ON DELETE CASCADE,
    family_id             UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    last_read_at          TIMESTAMPTZ,
    deleted_at            TIMESTAMPTZ,                -- per-user conversation deletion
    PRIMARY KEY (conversation_id, parent_id)
);

CREATE INDEX idx_soc_conv_participants_parent ON soc_conversation_participants(parent_id)
    WHERE deleted_at IS NULL;

-- Direct messages [S§7.5]
-- Parent-to-parent only. Messages reference a conversation.
CREATE TABLE soc_messages (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id       UUID NOT NULL REFERENCES soc_conversations(id) ON DELETE CASCADE,
    sender_parent_id      UUID NOT NULL REFERENCES iam_parents(id),
    sender_family_id      UUID NOT NULL REFERENCES iam_families(id),
    content               TEXT NOT NULL,
    attachments           JSONB NOT NULL DEFAULT '[]', -- array of media references
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_soc_messages_conversation ON soc_messages(conversation_id, created_at);

-- Groups [S§7.6]
-- Platform-managed (one per methodology) and user-created.
CREATE TABLE soc_groups (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_type            TEXT NOT NULL CHECK (group_type IN ('platform', 'user_created')),
    name                  TEXT NOT NULL,
    description           TEXT,
    cover_photo_url       TEXT,
    creator_family_id     UUID REFERENCES iam_families(id), -- NULL for platform groups
    methodology_id        UUID REFERENCES method_definitions(id),
                          -- app-level ref only for platform groups; FK because methodology
                          -- drives group discovery and matching
    join_policy           TEXT NOT NULL DEFAULT 'open'
                          CHECK (join_policy IN ('open', 'request_to_join', 'invite_only')),
    member_count          INTEGER NOT NULL DEFAULT 0,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_soc_groups_type ON soc_groups(group_type);
CREATE INDEX idx_soc_groups_methodology ON soc_groups(methodology_id)
    WHERE methodology_id IS NOT NULL;

-- Group members [S§7.6]
CREATE TABLE soc_group_members (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id              UUID NOT NULL REFERENCES soc_groups(id) ON DELETE CASCADE,
    family_id             UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    role                  TEXT NOT NULL DEFAULT 'member'
                          CHECK (role IN ('member', 'moderator', 'owner')),
    status                TEXT NOT NULL DEFAULT 'active'
                          CHECK (status IN ('active', 'pending', 'invited', 'banned')),
    joined_at             TIMESTAMPTZ,                -- set when status → active
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_group_member UNIQUE (group_id, family_id)
);

CREATE INDEX idx_soc_group_members_group ON soc_group_members(group_id, status);
CREATE INDEX idx_soc_group_members_family ON soc_group_members(family_id, status);

-- Events [S§7.7]
-- Visibility includes 'discoverable' (unique to events, not posts).
CREATE TABLE soc_events (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    creator_family_id     UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    creator_parent_id     UUID NOT NULL REFERENCES iam_parents(id),
    group_id              UUID REFERENCES soc_groups(id) ON DELETE SET NULL,
    title                 TEXT NOT NULL,
    description           TEXT,
    event_date            TIMESTAMPTZ NOT NULL,
    end_date              TIMESTAMPTZ,
    location_name         TEXT,                       -- human-readable location
    location_region       TEXT,                       -- city/region for discovery
    is_virtual            BOOLEAN NOT NULL DEFAULT false,
    virtual_url           TEXT,
    capacity              INTEGER,                    -- NULL = unlimited
    visibility            TEXT NOT NULL DEFAULT 'friends'
                          CHECK (visibility IN ('friends', 'group', 'discoverable')),
    status                TEXT NOT NULL DEFAULT 'active'
                          CHECK (status IN ('active', 'cancelled')),
    methodology_id        UUID REFERENCES method_definitions(id),
                          -- methodology tagging for discovery; app-level ref
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

-- Event RSVPs [S§7.7]
CREATE TABLE soc_event_rsvps (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id              UUID NOT NULL REFERENCES soc_events(id) ON DELETE CASCADE,
    family_id             UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    status                TEXT NOT NULL DEFAULT 'going'
                          CHECK (status IN ('going', 'interested', 'not_going')),
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_event_rsvp UNIQUE (event_id, family_id)
);

CREATE INDEX idx_soc_event_rsvps_event ON soc_event_rsvps(event_id, status);
```

### §3.3 RLS Policies

Social tables require nuanced RLS policies because they contain **cross-family data**.
Unlike IAM or Learning tables where every row belongs to exactly one family, social data
involves two or more families (friendships, conversations, groups). RLS provides
defense-in-depth alongside application-level access control. `[CODING §3.2]`

```sql
-- soc_profiles: owner can read/write their own profile.
-- Friends can read via application-level friendship check (not RLS — too complex for RLS).
ALTER TABLE soc_profiles ENABLE ROW LEVEL SECURITY;
CREATE POLICY soc_profiles_owner_policy ON soc_profiles
    USING (family_id = current_setting('app.current_family_id')::uuid);

-- soc_friendships: either participant can read/modify.
-- CROSS-FAMILY: both requester and accepter need access.
ALTER TABLE soc_friendships ENABLE ROW LEVEL SECURITY;
CREATE POLICY soc_friendships_participant_policy ON soc_friendships
    USING (
        requester_family_id = current_setting('app.current_family_id')::uuid
        OR accepter_family_id = current_setting('app.current_family_id')::uuid
    );

-- soc_blocks: only the blocker can see their blocks.
-- The blocked user MUST NOT be able to see that they are blocked (silent blocking).
ALTER TABLE soc_blocks ENABLE ROW LEVEL SECURITY;
CREATE POLICY soc_blocks_blocker_policy ON soc_blocks
    USING (blocker_family_id = current_setting('app.current_family_id')::uuid);

-- soc_posts: family-scoped for writes; reads handled at application level
-- (visibility check involves friendship/group membership).
ALTER TABLE soc_posts ENABLE ROW LEVEL SECURITY;
CREATE POLICY soc_posts_family_policy ON soc_posts
    USING (family_id = current_setting('app.current_family_id')::uuid);

-- soc_comments: family-scoped for writes; reads via post visibility.
ALTER TABLE soc_comments ENABLE ROW LEVEL SECURITY;
CREATE POLICY soc_comments_family_policy ON soc_comments
    USING (family_id = current_setting('app.current_family_id')::uuid);

-- soc_post_likes: family-scoped.
ALTER TABLE soc_post_likes ENABLE ROW LEVEL SECURITY;
CREATE POLICY soc_post_likes_family_policy ON soc_post_likes
    USING (family_id = current_setting('app.current_family_id')::uuid);

-- soc_conversation_participants: family-scoped for participant access.
ALTER TABLE soc_conversation_participants ENABLE ROW LEVEL SECURITY;
CREATE POLICY soc_conv_participants_family_policy ON soc_conversation_participants
    USING (family_id = current_setting('app.current_family_id')::uuid);

-- soc_messages: sender family-scoped for writes.
-- Reads are via conversation participant check at application level.
ALTER TABLE soc_messages ENABLE ROW LEVEL SECURITY;
CREATE POLICY soc_messages_sender_policy ON soc_messages
    USING (sender_family_id = current_setting('app.current_family_id')::uuid);

-- soc_groups: NO RLS.
-- Groups are accessed via membership checks at application level. Platform groups
-- are readable by all authenticated users. User-created groups are accessed via
-- soc_group_members.
-- No family_id column on soc_groups — access is membership-based.

-- soc_group_members: family-scoped.
ALTER TABLE soc_group_members ENABLE ROW LEVEL SECURITY;
CREATE POLICY soc_group_members_family_policy ON soc_group_members
    USING (family_id = current_setting('app.current_family_id')::uuid);

-- soc_events: NO RLS.
-- Events with 'discoverable' visibility are intentionally visible to non-friends.
-- Access control is enforced at application level via visibility + friendship +
-- group membership checks.

-- soc_event_rsvps: family-scoped.
ALTER TABLE soc_event_rsvps ENABLE ROW LEVEL SECURITY;
CREATE POLICY soc_event_rsvps_family_policy ON soc_event_rsvps
    USING (family_id = current_setting('app.current_family_id')::uuid);
```

---

## §4 API Endpoints

All social endpoints require authentication unless otherwise noted. All social features
are available to free-tier users — no `RequirePremium` middleware. `[S§3.2, S§15.1]`

### §4.1 Phase 1 Endpoints

#### Profile

##### `GET /v1/social/profile`

Returns the authenticated family's social profile.

- **Auth**: Required (`FamilyScope`)
- **Response**: `ProfileResponse` (200 OK)
- **404**: Profile not found (should not happen — created on `FamilyCreated`)

##### `PATCH /v1/social/profile`

Updates the authenticated family's social profile.

- **Auth**: Required (`FamilyScope`)
- **Body**: `UpdateProfileRequest` (`bio`, `profile_photo_url`, `privacy_settings`, `location_visible`)
- **Validation**: `bio` max 2000 chars; `privacy_settings` values must be `"friends"` or `"hidden"`
- **Response**: `ProfileResponse` (200 OK)

##### `GET /v1/social/families/:family_id/profile`

Returns another family's social profile, filtered by privacy settings.

- **Auth**: Required (`FamilyScope`)
- **Response**: `ProfileResponse` (200 OK) — fields filtered by privacy settings and friendship status
- **404**: Profile not found OR blocked (silent block → 404) `[S§7.4]`
- **Note**: CROSS-FAMILY read. Block check is performed first; if blocked, returns 404.

#### Friends

##### `GET /v1/social/friends`

Lists the authenticated family's friends.

- **Auth**: Required (`FamilyScope`)
- **Query**: `?cursor=<uuid>&limit=20`
- **Response**: `PaginatedResponse<FriendResponse>` (200 OK)

##### `POST /v1/social/friends/request`

Sends a friend request to another family.

- **Auth**: Required (`FamilyScope`)
- **Body**: `SendFriendRequestCommand` (`target_family_id`)
- **Validation**: Cannot friend self; cannot send if blocked; cannot send if already friends or pending
- **Response**: `FriendshipResponse` (201 Created)
- **Events**: `FriendRequestSent`
- **Error codes**: `already_friends` (409), `already_pending` (409), `blocked` (404, silent)

##### `POST /v1/social/friends/request/:friendship_id/accept`

Accepts a pending friend request.

- **Auth**: Required (`FamilyScope`)
- **Precondition**: Caller must be the accepter (not the requester)
- **Response**: `FriendshipResponse` (200 OK)
- **Events**: `FriendRequestAccepted`
- **Error codes**: `not_pending` (409), `not_accepter` (403)

##### `POST /v1/social/friends/request/:friendship_id/reject`

Rejects (deletes) a pending friend request.

- **Auth**: Required (`FamilyScope`)
- **Precondition**: Caller must be the accepter
- **Response**: 204 No Content

##### `GET /v1/social/friends/requests/incoming`

Lists pending friend requests received by the authenticated family.

- **Auth**: Required (`FamilyScope`)
- **Response**: `[]FriendRequestResponse` (200 OK)

##### `GET /v1/social/friends/requests/outgoing`

Lists pending friend requests sent by the authenticated family.

- **Auth**: Required (`FamilyScope`)
- **Response**: `[]FriendRequestResponse` (200 OK)

##### `DELETE /v1/social/friends/:family_id`

Unfriends a family. Silent — no notification sent. `[S§7.4]`

- **Auth**: Required (`FamilyScope`)
- **Response**: 204 No Content
- **Error codes**: `not_friends` (404)

##### `POST /v1/social/blocks`

Blocks a family. Silent — blocked user is not notified. `[S§7.4]`

- **Auth**: Required (`FamilyScope`)
- **Body**: `BlockFamilyCommand` (`target_family_id`)
- **Side effects**: Removes any existing friendship; purges blocked family's posts from blocker's feed (inline)
- **Response**: 201 Created
- **Error codes**: `already_blocked` (409), `cannot_block_self` (422)

##### `DELETE /v1/social/blocks/:family_id`

Unblocks a family.

- **Auth**: Required (`FamilyScope`)
- **Response**: 204 No Content

##### `GET /v1/social/blocks`

Lists families blocked by the authenticated family.

- **Auth**: Required (`FamilyScope`)
- **Response**: `[]BlockedFamilyResponse` (200 OK)

#### Feed & Posts

##### `GET /v1/social/feed`

Returns the authenticated family's timeline feed (reverse chronological). `[S§7.2.3]`

- **Auth**: Required (`FamilyScope`)
- **Query**: `?cursor=<timestamp>&limit=20`
- **Response**: `PaginatedResponse<PostResponse>` (200 OK)
- **Note**: Reads from Redis sorted set first; falls back to PostgreSQL. Block filter applied in application layer.

##### `POST /v1/social/posts`

Creates a new post. `[S§7.2]`

- **Auth**: Required (`FamilyScope`)
- **Body**: `CreatePostCommand` (`post_type`, `content`, `attachments`, `group_id`)
- **Validation**: `content` required for text posts; `attachments` required for photo posts; `group_id` requires active membership
- **Response**: `PostResponse` (201 Created)
- **Events**: `PostCreated`
- **Side effects**: Fan-out to friends' feeds via Redis (async job)

##### `GET /v1/social/posts/:post_id`

Returns a single post with comments.

- **Auth**: Required (`FamilyScope`)
- **Response**: `PostDetailResponse` (200 OK)
- **404**: Post not found OR not visible to caller (visibility/block check)

##### `DELETE /v1/social/posts/:post_id`

Deletes a post. Only the author's family can delete.

- **Auth**: Required (`FamilyScope`)
- **Response**: 204 No Content
- **Error codes**: `not_post_author` (403)

##### `POST /v1/social/posts/:post_id/like`

Likes a post. Idempotent — re-liking is a no-op.

- **Auth**: Required (`FamilyScope`)
- **Response**: 201 Created (or 200 OK if already liked)

##### `DELETE /v1/social/posts/:post_id/like`

Unlikes a post.

- **Auth**: Required (`FamilyScope`)
- **Response**: 204 No Content

##### `POST /v1/social/posts/:post_id/comments`

Adds a comment to a post. `[S§7.3]`

- **Auth**: Required (`FamilyScope`)
- **Body**: `CreateCommentCommand` (`content`, `parent_comment_id?`)
- **Validation**: `parent_comment_id` must reference a top-level comment on the same post (one level only)
- **Response**: `CommentResponse` (201 Created)

##### `DELETE /v1/social/posts/:post_id/comments/:comment_id`

Deletes a comment. Author or post author can delete. `[S§7.3]`

- **Auth**: Required (`FamilyScope`)
- **Response**: 204 No Content
- **Error codes**: `not_comment_author_or_post_author` (403)

#### Messaging

##### `GET /v1/social/conversations`

Lists the authenticated parent's conversations.

- **Auth**: Required (`FamilyScope`)
- **Query**: `?cursor=<timestamp>&limit=20`
- **Response**: `PaginatedResponse<ConversationSummaryResponse>` (200 OK)
- **Note**: Excludes conversations where `deleted_at` is set for this participant.

##### `POST /v1/social/conversations`

Creates or retrieves a conversation with another parent. Friends-only guard. `[S§7.5]`

- **Auth**: Required (`FamilyScope`)
- **Body**: `CreateConversationCommand` (`recipient_parent_id`)
- **Validation**: Recipient must be a friend; not blocked
- **Response**: `ConversationResponse` (201 Created or 200 OK if exists)
- **Error codes**: `not_friends` (403), `blocked` (404, silent)

##### `GET /v1/social/conversations/:conversation_id/messages`

Returns messages in a conversation.

- **Auth**: Required (`FamilyScope`)
- **Query**: `?cursor=<timestamp>&limit=50`
- **Response**: `PaginatedResponse<MessageResponse>` (200 OK)
- **Note**: Only returns messages created after participant's `deleted_at` (if set).

##### `POST /v1/social/conversations/:conversation_id/messages`

Sends a message in a conversation. `[S§7.5]`

- **Auth**: Required (`FamilyScope`)
- **Body**: `SendMessageCommand` (`content`, `attachments`)
- **Validation**: Caller must be a participant; friendship still active; not blocked
- **Response**: `MessageResponse` (201 Created)
- **Events**: `MessageSent`
- **Side effects**: WebSocket push to recipient if connected; clears recipient's `deleted_at` (new message restores conversation)

##### `POST /v1/social/conversations/:conversation_id/read`

Marks all messages in a conversation as read (updates `last_read_at`).

- **Auth**: Required (`FamilyScope`)
- **Response**: 204 No Content

##### `DELETE /v1/social/conversations/:conversation_id`

Soft-deletes a conversation for the authenticated user (sets `deleted_at`). `[S§7.5]`

- **Auth**: Required (`FamilyScope`)
- **Response**: 204 No Content
- **Note**: Does not delete messages. Other participant still sees the conversation.

##### `POST /v1/social/messages/:message_id/report`

Reports a message for moderation review. `[S§7.5, S§12.3]`

- **Auth**: Required (`FamilyScope`)
- **Body**: `ReportMessageCommand` (`reason`)
- **Response**: 201 Created
- **Events**: `MessageReported`

#### Groups

##### `GET /v1/social/groups`

Lists groups the authenticated family is a member of.

- **Auth**: Required (`FamilyScope`)
- **Response**: `[]GroupSummaryResponse` (200 OK)

##### `GET /v1/social/groups/platform`

Lists platform-managed methodology groups. `[S§7.6]`

- **Auth**: Required (`FamilyScope`)
- **Response**: `[]GroupSummaryResponse` (200 OK)

##### `GET /v1/social/groups/:group_id`

Returns group details. Requires membership for user-created groups.

- **Auth**: Required (`FamilyScope`)
- **Response**: `GroupDetailResponse` (200 OK)
- **404**: Group not found or not a member (for non-platform groups)

##### `POST /v1/social/groups/:group_id/join`

Joins a group or submits a join request. `[S§7.6]`

- **Auth**: Required (`FamilyScope`)
- **Behavior by join_policy**:
  - `open`: Immediately becomes `active` member
  - `request_to_join`: Creates `pending` membership; moderator approval required
  - `invite_only`: Returns 403
- **Response**: `GroupMemberResponse` (201 Created)
- **Error codes**: `already_member` (409), `invite_only` (403), `banned` (403)

##### `DELETE /v1/social/groups/:group_id/leave`

Leaves a group. Owners cannot leave without transferring ownership.

- **Auth**: Required (`FamilyScope`)
- **Response**: 204 No Content
- **Error codes**: `owner_cannot_leave` (422)

##### `GET /v1/social/groups/:group_id/posts`

Lists posts in a group. Requires membership. `[S§7.6]`

- **Auth**: Required (`FamilyScope`)
- **Query**: `?cursor=<timestamp>&limit=20`
- **Response**: `PaginatedResponse<PostResponse>` (200 OK)

#### Events

##### `POST /v1/social/events`

Creates an event. `[S§7.7]`

- **Auth**: Required (`FamilyScope`)
- **Body**: `CreateEventCommand` (`title`, `description`, `event_date`, `end_date?`, `location_name?`, `location_region?`, `is_virtual`, `virtual_url?`, `capacity?`, `visibility`, `group_id?`, `methodology_id?`)
- **Validation**: `event_date` must be in the future; `group_id` requires membership; `visibility = 'group'` requires `group_id`
- **Response**: `EventResponse` (201 Created)

##### `GET /v1/social/events`

Lists events visible to the authenticated family (own events, friend events, group events, discoverable events).

- **Auth**: Required (`FamilyScope`)
- **Query**: `?filter=upcoming|past&cursor=<timestamp>&limit=20`
- **Response**: `PaginatedResponse<EventSummaryResponse>` (200 OK)

##### `GET /v1/social/events/:event_id`

Returns event details with RSVP list.

- **Auth**: Required (`FamilyScope`)
- **Response**: `EventDetailResponse` (200 OK)
- **404**: Event not found or not visible

##### `PATCH /v1/social/events/:event_id`

Updates an event. Creator only.

- **Auth**: Required (`FamilyScope`)
- **Body**: `UpdateEventCommand` (partial fields)
- **Response**: `EventResponse` (200 OK)
- **Error codes**: `not_event_creator` (403)

##### `POST /v1/social/events/:event_id/cancel`

Cancels an event. Creator only. Notifies RSVPed attendees. `[S§7.7]`

- **Auth**: Required (`FamilyScope`)
- **Response**: `EventResponse` (200 OK)
- **Events**: `EventCancelled`
- **Error codes**: `not_event_creator` (403), `already_cancelled` (409)

##### `POST /v1/social/events/:event_id/rsvp`

RSVPs to an event. `[S§7.7]`

- **Auth**: Required (`FamilyScope`)
- **Body**: `RsvpCommand` (`status`: `going` | `interested` | `not_going`)
- **Validation**: Event must be active; if capacity set, `going` count must be under capacity
- **Response**: `RsvpResponse` (200 OK)
- **Error codes**: `event_at_capacity` (422), `event_cancelled` (409)

##### `DELETE /v1/social/events/:event_id/rsvp`

Removes RSVP from an event.

- **Auth**: Required (`FamilyScope`)
- **Response**: 204 No Content

#### WebSocket

##### `GET /v1/social/ws`

WebSocket upgrade endpoint for real-time messaging and notifications. `[ARCH §2.16]`

- **Auth**: Required (session cookie validated on upgrade)
- **Protocol**: WebSocket (RFC 6455)
- **Frames**: JSON-encoded `WsMessage` (type: `new_message`, `typing`, `read_receipt`, `notification`)
- **Distribution**: Redis pub/sub for multi-connection support

### §4.2 Phase 2 Endpoints

#### User-Created Groups

##### `POST /v1/social/groups`

Creates a user-created group. `[S§7.6]`

- **Auth**: Required (`FamilyScope`)
- **Body**: `CreateGroupCommand` (`name`, `description`, `join_policy`, `methodology_id?`)
- **Response**: `GroupDetailResponse` (201 Created)

##### `PATCH /v1/social/groups/:group_id`

Updates group settings. Owner or moderator only.

- **Auth**: Required (`FamilyScope`)
- **Body**: `UpdateGroupCommand` (partial fields)
- **Response**: `GroupDetailResponse` (200 OK)

##### `DELETE /v1/social/groups/:group_id`

Deletes a user-created group. Owner only. Platform groups cannot be deleted.

- **Auth**: Required (`FamilyScope`)
- **Response**: 204 No Content
- **Error codes**: `cannot_delete_platform_group` (403)

#### Group Member Management

##### `GET /v1/social/groups/:group_id/members`

Lists group members. Requires membership.

- **Auth**: Required (`FamilyScope`)
- **Response**: `[]GroupMemberResponse` (200 OK)

##### `POST /v1/social/groups/:group_id/members/:family_id/approve`

Approves a pending join request. Moderator/owner only.

- **Auth**: Required (`FamilyScope`)
- **Response**: `GroupMemberResponse` (200 OK)

##### `POST /v1/social/groups/:group_id/members/:family_id/reject`

Rejects a pending join request. Moderator/owner only.

- **Auth**: Required (`FamilyScope`)
- **Response**: 204 No Content

##### `POST /v1/social/groups/:group_id/members/:family_id/ban`

Bans a member from the group. Moderator/owner only.

- **Auth**: Required (`FamilyScope`)
- **Response**: 204 No Content

##### `POST /v1/social/groups/:group_id/members/:family_id/promote`

Promotes a member to moderator. Owner only.

- **Auth**: Required (`FamilyScope`)
- **Response**: `GroupMemberResponse` (200 OK)

##### `POST /v1/social/groups/:group_id/invite`

Invites a family to an invite-only group. Moderator/owner only.

- **Auth**: Required (`FamilyScope`)
- **Body**: `InviteToGroupCommand` (`family_id`)
- **Response**: `GroupMemberResponse` (201 Created)

#### Discovery

##### `GET /v1/social/discover/families`

Discovers nearby families (opt-in location). `[S§7.8]`

- **Auth**: Required (`FamilyScope`)
- **Query**: `?methodology_id=<uuid>&radius_km=50`
- **Response**: `[]DiscoverableFamilyResponse` (200 OK)

##### `GET /v1/social/discover/events`

Discovers events by location and/or methodology. `[S§7.7]`

- **Auth**: Required (`FamilyScope`)
- **Query**: `?methodology_id=<uuid>&location_region=<string>`
- **Response**: `[]EventSummaryResponse` (200 OK)

##### `GET /v1/social/discover/groups`

Discovers groups by methodology. `[S§7.6]`

- **Auth**: Required (`FamilyScope`)
- **Query**: `?methodology_id=<uuid>`
- **Response**: `[]GroupSummaryResponse` (200 OK)

#### Post Editing

##### `PATCH /v1/social/posts/:post_id`

Edits a post. Author only. Sets `is_edited = true`.

- **Auth**: Required (`FamilyScope`)
- **Body**: `UpdatePostCommand` (`content`, `attachments`)
- **Response**: `PostResponse` (200 OK)

#### Pinned Posts

##### `POST /v1/social/groups/:group_id/posts/:post_id/pin`

Pins a post in a group. Moderator/owner only.

- **Auth**: Required (`FamilyScope`)
- **Response**: 200 OK

##### `DELETE /v1/social/groups/:group_id/posts/:post_id/pin`

Unpins a post. Moderator/owner only.

- **Auth**: Required (`FamilyScope`)
- **Response**: 204 No Content

**Phase 1 total**: ~39 endpoints (Profile: 3, Friends: 10, Feed/Posts: 8, Messaging: 7,
Groups: 5, Events: 7, WebSocket: 1).

**Phase 2 total**: ~13 endpoints (User groups: 3, Member management: 5, Discovery: 3,
Post editing: 1, Pinned posts: 2 — some overlap with Phase 1 group endpoints that gain
additional capabilities).

---

## §5 Service Interface

The `SocialService` interface defines all use cases exposed to handlers and other domains.
Defined in `internal/social/ports.go`. Methods are organized with CQRS separation: command
methods (writes with side effects) are separated from query methods (reads). `[CODING §8.2, ARCH §4.7]`

```go
// internal/social/ports.go

type SocialService interface {
    // === COMMAND SIDE (writes with side effects) ========================

    // --- Profile Commands -----------------------------------------------

    // CreateProfile creates a social profile for a new family.
    // Called by FamilyCreated event handler. [S§7.1]
    CreateProfile(ctx context.Context, familyID FamilyID) error

    // UpdateProfile updates the authenticated family's social profile. [S§7.1]
    UpdateProfile(ctx context.Context, scope *FamilyScope, cmd UpdateProfileRequest) (*ProfileResponse, error)

    // --- Friend Commands ------------------------------------------------

    // SendFriendRequest sends a friend request. [S§7.4]
    SendFriendRequest(ctx context.Context, scope *FamilyScope, cmd SendFriendRequestCommand) (*FriendshipResponse, error)

    // AcceptFriendRequest accepts a pending friend request. [S§7.4]
    AcceptFriendRequest(ctx context.Context, scope *FamilyScope, friendshipID uuid.UUID) (*FriendshipResponse, error)

    // RejectFriendRequest rejects (deletes) a pending friend request. [S§7.4]
    RejectFriendRequest(ctx context.Context, scope *FamilyScope, friendshipID uuid.UUID) error

    // Unfriend removes an existing friendship. Silent, no notification. [S§7.4]
    Unfriend(ctx context.Context, scope *FamilyScope, targetFamilyID uuid.UUID) error

    // BlockFamily blocks a family. Removes friendship if exists, purges feed. [S§7.4]
    BlockFamily(ctx context.Context, scope *FamilyScope, cmd BlockFamilyCommand) error

    // UnblockFamily unblocks a family. [S§7.4]
    UnblockFamily(ctx context.Context, scope *FamilyScope, targetFamilyID uuid.UUID) error

    // --- Post Commands --------------------------------------------------

    // CreatePost creates a post and triggers feed fan-out. [S§7.2]
    CreatePost(ctx context.Context, scope *FamilyScope, cmd CreatePostCommand) (*PostResponse, error)

    // DeletePost deletes a post. Author's family only. [S§7.2]
    DeletePost(ctx context.Context, scope *FamilyScope, postID uuid.UUID) error

    // LikePost likes a post. Idempotent. [S§7.2]
    LikePost(ctx context.Context, scope *FamilyScope, postID uuid.UUID) error

    // UnlikePost unlikes a post. [S§7.2]
    UnlikePost(ctx context.Context, scope *FamilyScope, postID uuid.UUID) error

    // --- Comment Commands ------------------------------------------------

    // CreateComment creates a comment on a post. [S§7.3]
    CreateComment(ctx context.Context, scope *FamilyScope, postID uuid.UUID, cmd CreateCommentCommand) (*CommentResponse, error)

    // DeleteComment deletes a comment. Author or post author can delete. [S§7.3]
    DeleteComment(ctx context.Context, scope *FamilyScope, postID uuid.UUID, commentID uuid.UUID) error

    // --- Messaging Commands ----------------------------------------------

    // CreateConversation creates or retrieves a conversation. Friends-only guard. [S§7.5]
    CreateConversation(ctx context.Context, scope *FamilyScope, cmd CreateConversationCommand) (*ConversationResponse, error)

    // SendMessage sends a message in a conversation. [S§7.5]
    SendMessage(ctx context.Context, scope *FamilyScope, conversationID uuid.UUID, cmd SendMessageCommand) (*MessageResponse, error)

    // MarkConversationRead marks a conversation as read. [S§7.5]
    MarkConversationRead(ctx context.Context, scope *FamilyScope, conversationID uuid.UUID) error

    // DeleteConversation soft-deletes a conversation for the authenticated user. [S§7.5]
    DeleteConversation(ctx context.Context, scope *FamilyScope, conversationID uuid.UUID) error

    // ReportMessage reports a message for moderation review. [S§7.5, S§12.3]
    ReportMessage(ctx context.Context, scope *FamilyScope, messageID uuid.UUID, cmd ReportMessageCommand) error

    // --- Group Commands ------------------------------------------------──

    // JoinGroup joins a group or submits join request based on join_policy. [S§7.6]
    JoinGroup(ctx context.Context, scope *FamilyScope, groupID uuid.UUID) (*GroupMemberResponse, error)

    // LeaveGroup leaves a group. [S§7.6]
    LeaveGroup(ctx context.Context, scope *FamilyScope, groupID uuid.UUID) error

    // CreateGroup creates a user-created group. (Phase 2) [S§7.6]
    CreateGroup(ctx context.Context, scope *FamilyScope, cmd CreateGroupCommand) (*GroupDetailResponse, error)

    // UpdateGroup updates group settings. Moderator/owner only. (Phase 2) [S§7.6]
    UpdateGroup(ctx context.Context, scope *FamilyScope, groupID uuid.UUID, cmd UpdateGroupCommand) (*GroupDetailResponse, error)

    // DeleteGroup deletes a user-created group. Owner only. (Phase 2) [S§7.6]
    DeleteGroup(ctx context.Context, scope *FamilyScope, groupID uuid.UUID) error

    // ApproveMember approves a pending join request. Moderator/owner only. (Phase 2)
    ApproveMember(ctx context.Context, scope *FamilyScope, groupID uuid.UUID, familyID uuid.UUID) (*GroupMemberResponse, error)

    // RejectMember rejects a pending join request. Moderator/owner only. (Phase 2)
    RejectMember(ctx context.Context, scope *FamilyScope, groupID uuid.UUID, familyID uuid.UUID) error

    // BanMember bans a member. Moderator/owner only. (Phase 2)
    BanMember(ctx context.Context, scope *FamilyScope, groupID uuid.UUID, familyID uuid.UUID) error

    // PromoteMember promotes a member to moderator. Owner only. (Phase 2)
    PromoteMember(ctx context.Context, scope *FamilyScope, groupID uuid.UUID, familyID uuid.UUID) (*GroupMemberResponse, error)

    // InviteToGroup invites a family to an invite-only group. Moderator/owner only. (Phase 2)
    InviteToGroup(ctx context.Context, scope *FamilyScope, groupID uuid.UUID, cmd InviteToGroupCommand) (*GroupMemberResponse, error)

    // --- Event Commands ------------------------------------------------──

    // CreateEvent creates an event. [S§7.7]
    CreateEvent(ctx context.Context, scope *FamilyScope, cmd CreateEventCommand) (*EventResponse, error)

    // UpdateEvent updates an event. Creator only. [S§7.7]
    UpdateEvent(ctx context.Context, scope *FamilyScope, eventID uuid.UUID, cmd UpdateEventCommand) (*EventResponse, error)

    // CancelEvent cancels an event. Creator only. Notifies attendees. [S§7.7]
    CancelEvent(ctx context.Context, scope *FamilyScope, eventID uuid.UUID) (*EventResponse, error)

    // RsvpEvent RSVPs to an event. [S§7.7]
    RsvpEvent(ctx context.Context, scope *FamilyScope, eventID uuid.UUID, cmd RsvpCommand) (*RsvpResponse, error)

    // RemoveRsvp removes RSVP from an event. [S§7.7]
    RemoveRsvp(ctx context.Context, scope *FamilyScope, eventID uuid.UUID) error

    // --- Event Handlers ------------------------------------------------──

    // HandleFamilyCreated handles FamilyCreated event — creates social profile.
    HandleFamilyCreated(ctx context.Context, familyID FamilyID) error

    // HandleCoParentRemoved handles CoParentRemoved event — disassociates posts from family.
    HandleCoParentRemoved(ctx context.Context, familyID FamilyID, parentID uuid.UUID) error

    // HandleMilestoneAchieved handles MilestoneAchieved event — creates optional milestone post.
    HandleMilestoneAchieved(ctx context.Context, familyID FamilyID, milestone MilestoneData) error

    // HandleFamilyDeletionScheduled handles FamilyDeletionScheduled event — prepares for cascade.
    HandleFamilyDeletionScheduled(ctx context.Context, familyID FamilyID) error

    // === QUERY SIDE (reads, no side effects) ════════════════════════════

    // --- Profile Queries ------------------------------------------------─

    // GetOwnProfile returns the authenticated family's profile. [S§7.1]
    GetOwnProfile(ctx context.Context, scope *FamilyScope) (*ProfileResponse, error)

    // GetFamilyProfile returns another family's profile, filtered by privacy settings. [S§7.1]
    // CROSS-FAMILY read.
    GetFamilyProfile(ctx context.Context, scope *FamilyScope, targetFamilyID uuid.UUID) (*ProfileResponse, error)

    // --- Friend Queries ------------------------------------------------──

    // ListFriends lists the authenticated family's friends.
    ListFriends(ctx context.Context, scope *FamilyScope, pagination *PaginationParams) (*PaginatedResponse[FriendResponse], error)

    // ListIncomingRequests lists incoming friend requests.
    ListIncomingRequests(ctx context.Context, scope *FamilyScope) ([]FriendRequestResponse, error)

    // ListOutgoingRequests lists outgoing friend requests.
    ListOutgoingRequests(ctx context.Context, scope *FamilyScope) ([]FriendRequestResponse, error)

    // ListBlocks lists blocked families.
    ListBlocks(ctx context.Context, scope *FamilyScope) ([]BlockedFamilyResponse, error)

    // --- Feed Queries ------------------------------------------------────

    // GetFeed returns the authenticated family's timeline feed. [S§7.2.3]
    GetFeed(ctx context.Context, scope *FamilyScope, pagination *CursorPaginationParams) (*PaginatedResponse[PostResponse], error)

    // GetPost returns a single post with comments. Visibility-checked.
    GetPost(ctx context.Context, scope *FamilyScope, postID uuid.UUID) (*PostDetailResponse, error)

    // --- Messaging Queries -----------------------------------------------

    // ListConversations lists the authenticated parent's conversations. [S§7.5]
    ListConversations(ctx context.Context, scope *FamilyScope, pagination *CursorPaginationParams) (*PaginatedResponse[ConversationSummaryResponse], error)

    // GetConversationMessages returns messages in a conversation. [S§7.5]
    GetConversationMessages(ctx context.Context, scope *FamilyScope, conversationID uuid.UUID, pagination *CursorPaginationParams) (*PaginatedResponse[MessageResponse], error)

    // --- Group Queries ------------------------------------------------───

    // ListMyGroups lists groups the authenticated family is a member of. [S§7.6]
    ListMyGroups(ctx context.Context, scope *FamilyScope) ([]GroupSummaryResponse, error)

    // ListPlatformGroups lists platform-managed methodology groups. [S§7.6]
    ListPlatformGroups(ctx context.Context, scope *FamilyScope) ([]GroupSummaryResponse, error)

    // GetGroup returns group details. Membership check for user-created groups. [S§7.6]
    GetGroup(ctx context.Context, scope *FamilyScope, groupID uuid.UUID) (*GroupDetailResponse, error)

    // ListGroupPosts lists posts in a group. Requires membership. [S§7.6]
    ListGroupPosts(ctx context.Context, scope *FamilyScope, groupID uuid.UUID, pagination *CursorPaginationParams) (*PaginatedResponse[PostResponse], error)

    // ListGroupMembers lists group members. (Phase 2)
    ListGroupMembers(ctx context.Context, scope *FamilyScope, groupID uuid.UUID) ([]GroupMemberResponse, error)

    // --- Event Queries ------------------------------------------------───

    // ListEvents lists events visible to the authenticated family. [S§7.7]
    ListEvents(ctx context.Context, scope *FamilyScope, filter *EventFilter, pagination *CursorPaginationParams) (*PaginatedResponse[EventSummaryResponse], error)

    // GetEvent returns event details with RSVP list. [S§7.7]
    GetEvent(ctx context.Context, scope *FamilyScope, eventID uuid.UUID) (*EventDetailResponse, error)

    // --- Discovery Queries (Phase 2) -------------------------------------

    // DiscoverFamilies discovers nearby families with location sharing enabled. [S§7.8]
    DiscoverFamilies(ctx context.Context, scope *FamilyScope, query DiscoverFamiliesQuery) ([]DiscoverableFamilyResponse, error)

    // DiscoverEvents discovers events by location/methodology. [S§7.7]
    DiscoverEvents(ctx context.Context, scope *FamilyScope, query DiscoverEventsQuery) ([]EventSummaryResponse, error)

    // DiscoverGroups discovers groups by methodology. [S§7.6]
    DiscoverGroups(ctx context.Context, scope *FamilyScope, query DiscoverGroupsQuery) ([]GroupSummaryResponse, error)
}
```

**Implementation**: `SocialServiceImpl` in `internal/social/service.go`. Constructor receives:
- `ProfileRepository (interface)`
- `FriendshipRepository (interface)`
- `BlockRepository (interface)`
- `PostRepository (interface)`
- `CommentRepository (interface)`
- `PostLikeRepository (interface)`
- `ConversationRepository (interface)`
- `MessageRepository (interface)`
- `GroupRepository (interface)`
- `GroupMemberRepository (interface)`
- `EventRepository (interface)`
- `EventRsvpRepository (interface)`
- `IamService (interface)` (for family/parent data lookup)
- `RedisPool` (for feed fan-out, caching, WebSocket pub/sub)
- `EventBus (interface)`

---

## §6 Repository Interfaces

Defined in `internal/social/ports.go`. Social repositories are unique in that several methods
perform **cross-family reads** — friendships, conversations, and feed queries by design
access data from multiple families. These methods are explicitly marked with
`// CROSS-FAMILY:` comments. `[CODING §2.4, CODING §8.2]`

```go
// internal/social/ports.go (continued)

type ProfileRepository interface {
    // Create creates a social profile. Called during FamilyCreated handling.
    Create(ctx context.Context, familyID FamilyID) (*Profile, error)

    // FindByFamily finds profile by family_id. Family-scoped for own profile.
    FindByFamily(ctx context.Context, scope *FamilyScope) (*Profile, error)

    // FindByFamilyID finds profile by family_id without family scope.
    // CROSS-FAMILY: Used to view another family's profile (visibility filtered in service).
    FindByFamilyID(ctx context.Context, familyID uuid.UUID) (*Profile, error)

    // Update updates a profile.
    Update(ctx context.Context, scope *FamilyScope, profile *Profile) (*Profile, error)
}

type FriendshipRepository interface {
    // Create creates a friendship record (status = pending).
    Create(ctx context.Context, requesterFamilyID uuid.UUID, accepterFamilyID uuid.UUID) (*Friendship, error)

    // FindBetween finds a friendship between two families (either direction).
    // CROSS-FAMILY: Friendship involves two families.
    FindBetween(ctx context.Context, familyA uuid.UUID, familyB uuid.UUID) (*Friendship, error)

    // FindByID finds a friendship by ID.
    // CROSS-FAMILY: Either participant can access.
    FindByID(ctx context.Context, friendshipID uuid.UUID) (*Friendship, error)

    // UpdateStatus updates friendship status.
    UpdateStatus(ctx context.Context, friendshipID uuid.UUID, status string) (*Friendship, error)

    // DeleteBetween deletes a friendship (unfriend or pre-block cleanup).
    DeleteBetween(ctx context.Context, familyA uuid.UUID, familyB uuid.UUID) error

    // ListFriends lists accepted friends for a family, paginated.
    ListFriends(ctx context.Context, scope *FamilyScope, pagination *PaginationParams) ([]Friendship, error)

    // ListIncomingPending lists incoming pending requests.
    ListIncomingPending(ctx context.Context, scope *FamilyScope) ([]Friendship, error)

    // ListOutgoingPending lists outgoing pending requests.
    ListOutgoingPending(ctx context.Context, scope *FamilyScope) ([]Friendship, error)

    // ListFriendFamilyIDs returns all accepted friend family_ids for a family.
    // Used by feed fan-out and visibility checks.
    ListFriendFamilyIDs(ctx context.Context, familyID uuid.UUID) ([]uuid.UUID, error)
}

type BlockRepository interface {
    // Creates a block record.
    Create(ctx context.Context, blockerFamilyID uuid.UUID, blockedFamilyID uuid.UUID) (Block, error)

    // Checks if family_a has blocked family_b. O(1) lookup.
    IsBlocked(ctx context.Context, blockerFamilyID uuid.UUID, blockedFamilyID uuid.UUID) (bool, error)

    // Checks if either family has blocked the other (bidirectional check).
    // CROSS-FAMILY: Used for visibility checks.
    IsEitherBlocked(ctx context.Context, familyA uuid.UUID, familyB uuid.UUID) (bool, error)

    // Deletes a block record (unblock).
    Delete(ctx context.Context, blockerFamilyID uuid.UUID, blockedFamilyID uuid.UUID) error

    // Lists all families blocked by a family.
    ListBlockedBy(ctx context.Context, scope *FamilyScope) ([]Block, error)

    // Returns all family_ids blocked by or blocking a family.
    // Used by feed filtering.
    ListAllBlockedFamilyIDs(ctx context.Context, familyID uuid.UUID) ([]uuid.UUID, error)
}

type PostRepository interface {
    // Creates a post.
    Create(ctx context.Context, scope *FamilyScope, post CreatePostRecord) (Post, error)

    // Finds a post by ID.
    // CROSS-FAMILY: Post may belong to another family (visibility checked in service).
    FindByID(ctx context.Context, postID uuid.UUID) (*Post, error)

    // Deletes a post. Family-scoped (author's family only).
    Delete(ctx context.Context, scope *FamilyScope, postID uuid.UUID) error

    // Updates a post (editing). Family-scoped.
    Update(ctx context.Context, scope *FamilyScope, postID uuid.UUID, content string, attachments json.RawMessage) (Post, error)

    // Increments likes_count.
    IncrementLikes(ctx context.Context, postID uuid.UUID) error

    // Decrements likes_count.
    DecrementLikes(ctx context.Context, postID uuid.UUID) error

    // Increments comments_count.
    IncrementComments(ctx context.Context, postID uuid.UUID) error

    // Decrements comments_count.
    DecrementComments(ctx context.Context, postID uuid.UUID) error

    // Lists posts by family_ids (for feed hydration from Redis post IDs).
    // CROSS-FAMILY: Feed contains posts from multiple families.
    FindByIDs(ctx context.Context, postIDs []uuid.UUID) ([]Post, error)

    // Lists posts by group, paginated.
    ListByGroup(ctx context.Context, groupID uuid.UUID, pagination *CursorPaginationParams) ([]Post, error)

    // Fallback feed query: recent posts from friend family_ids.
    // CROSS-FAMILY: Reads posts from multiple friend families.
    ListFriendsPosts(ctx context.Context, friendFamilyIDs []uuid.UUID, pagination *CursorPaginationParams) ([]Post, error)
}

type CommentRepository interface {
    // Creates a comment.
    Create(ctx context.Context, scope *FamilyScope, comment CreateCommentRecord) (Comment, error)

    // Finds a comment by ID.
    FindByID(ctx context.Context, commentID uuid.UUID) (*Comment, error)

    // Lists comments for a post, ordered by created_at.
    // CROSS-FAMILY: Comments come from multiple families.
    ListByPost(ctx context.Context, postID uuid.UUID) ([]Comment, error)

    // Deletes a comment.
    Delete(ctx context.Context, commentID uuid.UUID) error
}

type PostLikeRepository interface {
    // Creates a like (idempotent via UNIQUE constraint).
    Create(ctx context.Context, scope *FamilyScope, postID uuid.UUID) error

    // Deletes a like.
    Delete(ctx context.Context, scope *FamilyScope, postID uuid.UUID) error

    // Checks if a family has liked a post.
    Exists(ctx context.Context, familyID uuid.UUID, postID uuid.UUID) (bool, error)
}

type ConversationRepository interface {
    // Creates a conversation with two participants.
    CreateWithParticipants(ctx context.Context, participantAParentID uuid.UUID, participantAFamilyID uuid.UUID, participantBParentID uuid.UUID, participantBFamilyID uuid.UUID) (Conversation, error)

    // Finds an existing conversation between two parents.
    // CROSS-FAMILY: Conversations span two families.
    FindBetweenParents(ctx context.Context, parentA uuid.UUID, parentB uuid.UUID) (*Conversation, error)

    // Lists conversations for a parent (excludes soft-deleted).
    ListByParent(ctx context.Context, parentID uuid.UUID, pagination *CursorPaginationParams) ([]ConversationWithParticipants, error)

    // Marks conversation read for a participant.
    MarkRead(ctx context.Context, conversationID uuid.UUID, parentID uuid.UUID) error

    // Soft-deletes a conversation for a participant.
    SoftDeleteForParticipant(ctx context.Context, conversationID uuid.UUID, parentID uuid.UUID) error

    // Clears deleted_at for a participant (new message restores conversation).
    RestoreForParticipant(ctx context.Context, conversationID uuid.UUID, parentID uuid.UUID) error

    // Checks if a parent is a participant in a conversation.
    IsParticipant(ctx context.Context, conversationID uuid.UUID, parentID uuid.UUID) (bool, error)
}

type MessageRepository interface {
    // Creates a message.
    Create(ctx context.Context, message CreateMessageRecord) (Message, error)

    // Lists messages in a conversation, respecting participant's deleted_at.
    // CROSS-FAMILY: Messages from both participants are returned.
    ListByConversation(ctx context.Context, conversationID uuid.UUID, participantDeletedAt *time.Time, pagination *CursorPaginationParams) ([]Message, error)

    // Finds a message by ID.
    FindByID(ctx context.Context, messageID uuid.UUID) (*Message, error)
}

type GroupRepository interface {
    // Creates a group.
    Create(ctx context.Context, group CreateGroupRecord) (Group, error)

    // Finds a group by ID.
    FindByID(ctx context.Context, groupID uuid.UUID) (*Group, error)

    // Updates a group.
    Update(ctx context.Context, groupID uuid.UUID, group UpdateGroupRecord) (Group, error)

    // Deletes a group.
    Delete(ctx context.Context, groupID uuid.UUID) error

    // Lists platform-managed groups.
    ListPlatformGroups(ctx context.Context) ([]Group, error)

    // Increments member_count.
    IncrementMemberCount(ctx context.Context, groupID uuid.UUID) error

    // Decrements member_count.
    DecrementMemberCount(ctx context.Context, groupID uuid.UUID) error

    // Lists groups by methodology_id. Used for discovery.
    ListByMethodology(ctx context.Context, methodologyID uuid.UUID) ([]Group, error)
}

type GroupMemberRepository interface {
    // Creates a group membership record.
    Create(ctx context.Context, membership CreateGroupMemberRecord) (GroupMember, error)

    // Finds membership by group and family.
    FindByGroupAndFamily(ctx context.Context, groupID uuid.UUID, familyID uuid.UUID) (*GroupMember, error)

    // Updates membership status or role.
    Update(ctx context.Context, membershipID uuid.UUID, status *string, role *string) (GroupMember, error)

    // Deletes a membership record (leave or reject).
    Delete(ctx context.Context, groupID uuid.UUID, familyID uuid.UUID) error

    // Lists active members of a group.
    ListActiveByGroup(ctx context.Context, groupID uuid.UUID) ([]GroupMember, error)

    // Lists groups a family is an active member of.
    ListGroupsForFamily(ctx context.Context, scope *FamilyScope) ([]uuid.UUID, error)

    // Checks if a family is an active member of a group.
    IsActiveMember(ctx context.Context, groupID uuid.UUID, familyID uuid.UUID) (bool, error)

    // Checks if a family is a moderator or owner of a group.
    IsModeratorOrOwner(ctx context.Context, groupID uuid.UUID, familyID uuid.UUID) (bool, error)
}

type EventRepository interface {
    // Creates an event.
    Create(ctx context.Context, event CreateEventRecord) (Event, error)

    // Finds an event by ID.
    FindByID(ctx context.Context, eventID uuid.UUID) (*Event, error)

    // Updates an event.
    Update(ctx context.Context, eventID uuid.UUID, event UpdateEventRecord) (Event, error)

    // Updates event status (e.g., cancel).
    UpdateStatus(ctx context.Context, eventID uuid.UUID, status string) (Event, error)

    // Lists events visible to a family (own, friends', group, discoverable).
    // CROSS-FAMILY: Events from friends and discoverable events.
    ListVisible(ctx context.Context, familyID uuid.UUID, friendFamilyIDs []uuid.UUID, groupIDs []uuid.UUID, filter *EventFilter, pagination *CursorPaginationParams) ([]Event, error)

    // Lists discoverable events by location/methodology.
    ListDiscoverable(ctx context.Context, methodologyID *uuid.UUID, locationRegion *string) ([]Event, error)

    // Increments attendee_count.
    IncrementAttendeeCount(ctx context.Context, eventID uuid.UUID) error

    // Decrements attendee_count.
    DecrementAttendeeCount(ctx context.Context, eventID uuid.UUID) error
}

type EventRsvpRepository interface {
    // Creates or updates an RSVP.
    Upsert(ctx context.Context, scope *FamilyScope, eventID uuid.UUID, status string) (EventRsvp, error)

    // Deletes an RSVP.
    Delete(ctx context.Context, scope *FamilyScope, eventID uuid.UUID) error

    // Lists RSVPs for an event.
    // CROSS-FAMILY: RSVPs come from multiple families.
    ListByEvent(ctx context.Context, eventID uuid.UUID) ([]EventRsvp, error)

    // Counts "going" RSVPs for capacity checking.
    CountGoing(ctx context.Context, eventID uuid.UUID) (int64, error)

    // Lists family_ids with "going" RSVPs (for cancellation notification).
    ListGoingFamilyIDs(ctx context.Context, eventID uuid.UUID) ([]uuid.UUID, error)
}
```

---

## §7 Adapter Interfaces

None. The social domain has no external third-party service dependencies. Redis is
shared infrastructure (`internal/shared/redis.go`). WebSocket is handled via gorilla/websocket.
Content scanning and notification delivery are handled via
domain events to `safety::` and `notify::` respectively. `[CODING §8.1]`

---

## §8 Models (DTOs)

All types defined in `internal/social/models.go`. API-facing types use `json:"field"` struct tags and swaggo/swag annotations.
Request types additionally use go-playground/validator tags. `[CODING §2.3]`

### §8.1 Request Types

```go
// internal/social/models.go

// Profile update. [S§7.1]

type UpdateProfileRequest struct {
    Bio *string `json:"bio"`
    ProfilePhotoURL *string `json:"profile_photo_url"`
    // / Per-field privacy settings. Values: "friends" | "hidden". [S§7.1]
    PrivacySettings *json.RawMessage `json:"privacy_settings"`
    LocationVisible *bool `json:"location_visible"`
}

// Send friend request. [S§7.4]

type SendFriendRequestCommand struct {
    TargetFamilyID uuid.UUID `json:"target_family_id"`
}

// Block a family. [S§7.4]

type BlockFamilyCommand struct {
    TargetFamilyID uuid.UUID `json:"target_family_id"`
}

// Create a post. [S§7.2]

type CreatePostCommand struct {
    PostType string `json:"post_type"` // validated against allowed values in service
    Content *string `json:"content"`
    Attachments []AttachmentInput `json:"attachments"`
    GroupID *uuid.UUID `json:"group_id"`
}

// Update a post (Phase 2). [S§7.2]

type UpdatePostCommand struct {
    Content *string `json:"content"`
    Attachments *[]AttachmentInput `json:"attachments"`
}

// Attachment input for posts and messages.

type AttachmentInput struct {
    URL string `json:"url"`
    ContentType string `json:"content_type"` // MIME type
    Filename *string `json:"filename"`
}

// Create a comment. [S§7.3]

type CreateCommentCommand struct {
    Content string `json:"content"`
    ParentCommentID *uuid.UUID `json:"parent_comment_id"`
}

// Create or get a conversation. [S§7.5]

type CreateConversationCommand struct {
    RecipientParentID uuid.UUID `json:"recipient_parent_id"`
}

// Send a message. [S§7.5]

type SendMessageCommand struct {
    Content string `json:"content"`
    Attachments []AttachmentInput `json:"attachments"`
}

// Report a message. [S§12.3]

type ReportMessageCommand struct {
    Reason string `json:"reason"`
}

// Create a group (Phase 2). [S§7.6]

type CreateGroupCommand struct {
    Name string `json:"name"`
    Description *string `json:"description"`
    JoinPolicy *string `json:"join_policy"` // defaults to "open"
    MethodologyID *uuid.UUID `json:"methodology_id"`
}

// Update a group (Phase 2). [S§7.6]

type UpdateGroupCommand struct {
    Name *string `json:"name"`
    Description *string `json:"description"`
    JoinPolicy *string `json:"join_policy"`
    CoverPhotoURL *string `json:"cover_photo_url"`
}

// Invite to group (Phase 2). [S§7.6]

type InviteToGroupCommand struct {
    FamilyID uuid.UUID `json:"family_id"`
}

// Create an event. [S§7.7]

type CreateEventCommand struct {
    Title string `json:"title"`
    Description *string `json:"description"`
    EventDate time.Time `json:"event_date"`
    EndDate *time.Time `json:"end_date"`
    LocationName *string `json:"location_name"`
    LocationRegion *string `json:"location_region"`
    IsVirtual bool `json:"is_virtual"`
    VirtualURL *string `json:"virtual_url"`
    Capacity *int32 `json:"capacity"`
    Visibility string `json:"visibility"` // "friends" | "group" | "discoverable"
    GroupID *uuid.UUID `json:"group_id"`
    MethodologyID *uuid.UUID `json:"methodology_id"`
}

// Update an event. [S§7.7]

type UpdateEventCommand struct {
    Title *string `json:"title"`
    Description *string `json:"description"`
    EventDate *time.Time `json:"event_date"`
    EndDate *time.Time `json:"end_date"`
    LocationName *string `json:"location_name"`
    LocationRegion *string `json:"location_region"`
    IsVirtual *bool `json:"is_virtual"`
    VirtualURL *string `json:"virtual_url"`
    Capacity *int32 `json:"capacity"`
}

// RSVP to an event. [S§7.7]

type RsvpCommand struct {
    Status string `json:"status"` // "going" | "interested" | "not_going"
}
```

### §8.2 Response Types

```go
// Social profile response. [S§7.1]

type ProfileResponse struct {
    FamilyID uuid.UUID `json:"family_id"`
    DisplayName *string `json:"display_name"` // from iam_families; hidden if privacy = "hidden"
    ParentNames *[]string `json:"parent_names"` // from iam_parents; hidden if privacy = "hidden"
    Children *[]ProfileChildResponse `json:"children"` // hidden if privacy = "hidden"
    MethodologyNames *[]string `json:"methodology_names"` // hidden if privacy = "hidden"
    LocationRegion *string `json:"location_region"` // hidden if privacy = "hidden" or not location_visible
    Bio *string `json:"bio"`
    ProfilePhotoURL *string `json:"profile_photo_url"`
    PrivacySettings *json.RawMessage `json:"privacy_settings"` // only included for own profile
    LocationVisible *bool `json:"location_visible"` // only included for own profile
    IsFriend bool `json:"is_friend"`
    FriendshipStatus *string `json:"friendship_status"` // "pending" | "accepted" | null
}

// Child info in profile (generic avatar default). [S§7.1]

type ProfileChildResponse struct {
    DisplayName string `json:"display_name"`
    Age *int16 `json:"age"` // computed from birth_year
    AvatarURL *string `json:"avatar_url"` // defaults to generic avatar
}

// Friendship response.

type FriendshipResponse struct {
    ID uuid.UUID `json:"id"`
    RequesterFamilyID uuid.UUID `json:"requester_family_id"`
    AccepterFamilyID uuid.UUID `json:"accepter_family_id"`
    Status string `json:"status"`
    CreatedAt time.Time `json:"created_at"`
}

// Friend list item.

type FriendResponse struct {
    FamilyID uuid.UUID `json:"family_id"`
    DisplayName string `json:"display_name"`
    ProfilePhotoURL *string `json:"profile_photo_url"`
    MethodologyNames []string `json:"methodology_names"`
    FriendsSince time.Time `json:"friends_since"`
}

// Incoming/outgoing friend request.

type FriendRequestResponse struct {
    FriendshipID uuid.UUID `json:"friendship_id"`
    FamilyID uuid.UUID `json:"family_id"` // the other family
    DisplayName string `json:"display_name"`
    ProfilePhotoURL *string `json:"profile_photo_url"`
    CreatedAt time.Time `json:"created_at"`
}

// Blocked family.

type BlockedFamilyResponse struct {
    FamilyID uuid.UUID `json:"family_id"`
    DisplayName string `json:"display_name"`
    BlockedAt time.Time `json:"blocked_at"`
}

// Post response. [S§7.2]

type PostResponse struct {
    ID uuid.UUID `json:"id"`
    FamilyID uuid.UUID `json:"family_id"`
    AuthorName string `json:"author_name"`
    AuthorPhotoURL *string `json:"author_photo_url"`
    PostType string `json:"post_type"`
    Content *string `json:"content"`
    Attachments []AttachmentInput `json:"attachments"`
    GroupID *uuid.UUID `json:"group_id"`
    GroupName *string `json:"group_name"`
    Visibility string `json:"visibility"`
    LikesCount int32 `json:"likes_count"`
    CommentsCount int32 `json:"comments_count"`
    IsLikedByMe bool `json:"is_liked_by_me"`
    IsEdited bool `json:"is_edited"`
    CreatedAt time.Time `json:"created_at"`
}

// Post detail with comments. [S§7.2, S§7.3]

type PostDetailResponse struct {
    Post PostResponse `json:"post"`
    Comments []CommentResponse `json:"comments"`
}

// Comment response. [S§7.3]

type CommentResponse struct {
    ID uuid.UUID `json:"id"`
    PostID uuid.UUID `json:"post_id"`
    FamilyID uuid.UUID `json:"family_id"`
    AuthorName string `json:"author_name"`
    AuthorPhotoURL *string `json:"author_photo_url"`
    ParentCommentID *uuid.UUID `json:"parent_comment_id"`
    Content string `json:"content"`
    CreatedAt time.Time `json:"created_at"`
    Replies []CommentResponse `json:"replies"` // one level only
}

// Conversation summary. [S§7.5]

type ConversationSummaryResponse struct {
    ID uuid.UUID `json:"id"`
    OtherParentName string `json:"other_parent_name"`
    OtherParentPhotoURL *string `json:"other_parent_photo_url"`
    OtherFamilyID uuid.UUID `json:"other_family_id"`
    LastMessagePreview *string `json:"last_message_preview"`
    LastMessageAt *time.Time `json:"last_message_at"`
    UnreadCount int64 `json:"unread_count"`
    UpdatedAt time.Time `json:"updated_at"`
}

// Conversation response. [S§7.5]

type ConversationResponse struct {
    ID uuid.UUID `json:"id"`
    Participants []ConversationParticipantResponse `json:"participants"`
    CreatedAt time.Time `json:"created_at"`
}

// Conversation participant.

type ConversationParticipantResponse struct {
    ParentID uuid.UUID `json:"parent_id"`
    FamilyID uuid.UUID `json:"family_id"`
    DisplayName string `json:"display_name"`
    ProfilePhotoURL *string `json:"profile_photo_url"`
}

// Message response. [S§7.5]

type MessageResponse struct {
    ID uuid.UUID `json:"id"`
    ConversationID uuid.UUID `json:"conversation_id"`
    SenderParentID uuid.UUID `json:"sender_parent_id"`
    SenderName string `json:"sender_name"`
    Content string `json:"content"`
    Attachments []AttachmentInput `json:"attachments"`
    CreatedAt time.Time `json:"created_at"`
}

// Group summary. [S§7.6]

type GroupSummaryResponse struct {
    ID uuid.UUID `json:"id"`
    GroupType string `json:"group_type"`
    Name string `json:"name"`
    Description *string `json:"description"`
    CoverPhotoURL *string `json:"cover_photo_url"`
    MethodologyName *string `json:"methodology_name"`
    MemberCount int32 `json:"member_count"`
    JoinPolicy string `json:"join_policy"`
    IsMember bool `json:"is_member"`
}

// Group detail. [S§7.6]

type GroupDetailResponse struct {
    Summary GroupSummaryResponse `json:"summary"`
    CreatorFamilyID *uuid.UUID `json:"creator_family_id"`
    MyRole *string `json:"my_role"` // member/moderator/owner — null if not a member
    MyStatus *string `json:"my_status"` // active/pending/invited — null if not a member
    CreatedAt time.Time `json:"created_at"`
}

// Group member response. [S§7.6]

type GroupMemberResponse struct {
    FamilyID uuid.UUID `json:"family_id"`
    DisplayName string `json:"display_name"`
    ProfilePhotoURL *string `json:"profile_photo_url"`
    Role string `json:"role"`
    Status string `json:"status"`
    JoinedAt *time.Time `json:"joined_at"`
}

// Event summary. [S§7.7]

type EventSummaryResponse struct {
    ID uuid.UUID `json:"id"`
    Title string `json:"title"`
    EventDate time.Time `json:"event_date"`
    EndDate *time.Time `json:"end_date"`
    LocationName *string `json:"location_name"`
    IsVirtual bool `json:"is_virtual"`
    CreatorFamilyName string `json:"creator_family_name"`
    AttendeeCount int32 `json:"attendee_count"`
    Capacity *int32 `json:"capacity"`
    Visibility string `json:"visibility"`
    Status string `json:"status"`
    MyRsvp *string `json:"my_rsvp"`
}

// Event detail with RSVPs. [S§7.7]

type EventDetailResponse struct {
    ID uuid.UUID `json:"id"`
    Title string `json:"title"`
    Description *string `json:"description"`
    EventDate time.Time `json:"event_date"`
    EndDate *time.Time `json:"end_date"`
    LocationName *string `json:"location_name"`
    LocationRegion *string `json:"location_region"`
    IsVirtual bool `json:"is_virtual"`
    VirtualURL *string `json:"virtual_url"`
    Capacity *int32 `json:"capacity"`
    Visibility string `json:"visibility"`
    Status string `json:"status"`
    CreatorFamilyID uuid.UUID `json:"creator_family_id"`
    CreatorFamilyName string `json:"creator_family_name"`
    GroupID *uuid.UUID `json:"group_id"`
    GroupName *string `json:"group_name"`
    MethodologyName *string `json:"methodology_name"`
    AttendeeCount int32 `json:"attendee_count"`
    MyRsvp *string `json:"my_rsvp"`
    Rsvps []EventRsvpResponse `json:"rsvps"`
    CreatedAt time.Time `json:"created_at"`
}

// Event response (for create/update). [S§7.7]

type EventResponse struct {
    ID uuid.UUID `json:"id"`
    Title string `json:"title"`
    Description *string `json:"description"`
    EventDate time.Time `json:"event_date"`
    EndDate *time.Time `json:"end_date"`
    LocationName *string `json:"location_name"`
    LocationRegion *string `json:"location_region"`
    IsVirtual bool `json:"is_virtual"`
    VirtualURL *string `json:"virtual_url"`
    Capacity *int32 `json:"capacity"`
    Visibility string `json:"visibility"`
    Status string `json:"status"`
    AttendeeCount int32 `json:"attendee_count"`
    CreatedAt time.Time `json:"created_at"`
}

// RSVP response. [S§7.7]

type RsvpResponse struct {
    EventID uuid.UUID `json:"event_id"`
    FamilyID uuid.UUID `json:"family_id"`
    Status string `json:"status"`
    CreatedAt time.Time `json:"created_at"`
}

// Event RSVP in event detail.

type EventRsvpResponse struct {
    FamilyID uuid.UUID `json:"family_id"`
    DisplayName string `json:"display_name"`
    Status string `json:"status"`
}

// Discoverable family (Phase 2). [S§7.8]

type DiscoverableFamilyResponse struct {
    FamilyID uuid.UUID `json:"family_id"`
    DisplayName string `json:"display_name"`
    ProfilePhotoURL *string `json:"profile_photo_url"`
    MethodologyNames []string `json:"methodology_names"`
    LocationRegion *string `json:"location_region"`
}
```

### §8.3 Internal Types

```go
// Internal profile model (maps to soc_profiles row + iam data).

type Profile struct {
    FamilyID uuid.UUID `json:"family_id"`
    Bio *string `json:"bio"`
    ProfilePhotoURL *string `json:"profile_photo_url"`
    PrivacySettings json.RawMessage `json:"privacy_settings"`
    LocationVisible bool `json:"location_visible"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}

// Internal friendship record.

type Friendship struct {
    ID uuid.UUID `json:"id"`
    RequesterFamilyID uuid.UUID `json:"requester_family_id"`
    AccepterFamilyID uuid.UUID `json:"accepter_family_id"`
    Status string `json:"status"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}

// Internal block record.

type Block struct {
    ID uuid.UUID `json:"id"`
    BlockerFamilyID uuid.UUID `json:"blocker_family_id"`
    BlockedFamilyID uuid.UUID `json:"blocked_family_id"`
    CreatedAt time.Time `json:"created_at"`
}

// Internal post record.

type Post struct {
    ID uuid.UUID `json:"id"`
    FamilyID uuid.UUID `json:"family_id"`
    AuthorParentID uuid.UUID `json:"author_parent_id"`
    PostType string `json:"post_type"`
    Content *string `json:"content"`
    Attachments json.RawMessage `json:"attachments"`
    GroupID *uuid.UUID `json:"group_id"`
    Visibility string `json:"visibility"`
    LikesCount int32 `json:"likes_count"`
    CommentsCount int32 `json:"comments_count"`
    IsEdited bool `json:"is_edited"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}

// Internal comment record.

type Comment struct {
    ID uuid.UUID `json:"id"`
    PostID uuid.UUID `json:"post_id"`
    FamilyID uuid.UUID `json:"family_id"`
    AuthorParentID uuid.UUID `json:"author_parent_id"`
    ParentCommentID *uuid.UUID `json:"parent_comment_id"`
    Content string `json:"content"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}

// Internal conversation with participant data.

type ConversationWithParticipants struct {
    ID uuid.UUID `json:"id"`
    Participants []ConversationParticipant `json:"participants"`
    LastMessage *Message `json:"last_message"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}

// Internal conversation participant.

type ConversationParticipant struct {
    ConversationID uuid.UUID `json:"conversation_id"`
    ParentID uuid.UUID `json:"parent_id"`
    FamilyID uuid.UUID `json:"family_id"`
    LastReadAt *time.Time `json:"last_read_at"`
    DeletedAt *time.Time `json:"deleted_at"`
}

// Internal conversation record.

type Conversation struct {
    ID uuid.UUID `json:"id"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}

// Internal message record.

type Message struct {
    ID uuid.UUID `json:"id"`
    ConversationID uuid.UUID `json:"conversation_id"`
    SenderParentID uuid.UUID `json:"sender_parent_id"`
    SenderFamilyID uuid.UUID `json:"sender_family_id"`
    Content string `json:"content"`
    Attachments json.RawMessage `json:"attachments"`
    CreatedAt time.Time `json:"created_at"`
}

// Internal group record.

type Group struct {
    ID uuid.UUID `json:"id"`
    GroupType string `json:"group_type"`
    Name string `json:"name"`
    Description *string `json:"description"`
    CoverPhotoURL *string `json:"cover_photo_url"`
    CreatorFamilyID *uuid.UUID `json:"creator_family_id"`
    MethodologyID *uuid.UUID `json:"methodology_id"`
    JoinPolicy string `json:"join_policy"`
    MemberCount int32 `json:"member_count"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}

// Internal group member record.

type GroupMember struct {
    ID uuid.UUID `json:"id"`
    GroupID uuid.UUID `json:"group_id"`
    FamilyID uuid.UUID `json:"family_id"`
    Role string `json:"role"`
    Status string `json:"status"`
    JoinedAt *time.Time `json:"joined_at"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}

// Internal event record.

type Event struct {
    ID uuid.UUID `json:"id"`
    CreatorFamilyID uuid.UUID `json:"creator_family_id"`
    CreatorParentID uuid.UUID `json:"creator_parent_id"`
    GroupID *uuid.UUID `json:"group_id"`
    Title string `json:"title"`
    Description *string `json:"description"`
    EventDate time.Time `json:"event_date"`
    EndDate *time.Time `json:"end_date"`
    LocationName *string `json:"location_name"`
    LocationRegion *string `json:"location_region"`
    IsVirtual bool `json:"is_virtual"`
    VirtualURL *string `json:"virtual_url"`
    Capacity *int32 `json:"capacity"`
    Visibility string `json:"visibility"`
    Status string `json:"status"`
    MethodologyID *uuid.UUID `json:"methodology_id"`
    AttendeeCount int32 `json:"attendee_count"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}

// Internal event RSVP record.

type EventRsvp struct {
    ID uuid.UUID `json:"id"`
    EventID uuid.UUID `json:"event_id"`
    FamilyID uuid.UUID `json:"family_id"`
    Status string `json:"status"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}

// Event filter for list queries.

type EventFilter struct {
    UpcomingOnly bool `json:"upcoming_only"`
    PastOnly bool `json:"past_only"`
}

// Milestone data from learn:: domain events.

type MilestoneData struct {
    StudentName string `json:"student_name"`
    MilestoneType string `json:"milestone_type"`
    Description string `json:"description"`
}

// WebSocket message frame.
type WsMessage struct {
    MsgType string `json:"msg_type"` // "new_message" | "typing" | "read_receipt" | "notification"
    Payload json.RawMessage `json:"payload"`
}

// Discovery query params.

type DiscoverFamiliesQuery struct {
    MethodologyID *uuid.UUID `json:"methodology_id"`
    RadiusKm *float64 `json:"radius_km"`
}

type DiscoverEventsQuery struct {
    MethodologyID *uuid.UUID `json:"methodology_id"`
    LocationRegion *string `json:"location_region"`
}

type DiscoverGroupsQuery struct {
    MethodologyID *uuid.UUID `json:"methodology_id"`
}
```

---

## §9 Visibility & Access Control (Domain Deep-Dive 1)

Visibility is the central privacy mechanism of the social domain. Every content access
decision flows through visibility rules. `[V§7, ARCH §1.5]`

### §9.1 Visibility Matrix

| Content Type | Own Family | Friend | Non-Friend | Blocked |
|-------------|:---:|:---:|:---:|:---:|
| Profile (friends fields) | ✓ | ✓ | ✗ | 404 |
| Profile (hidden fields) | ✓ | ✗ | ✗ | 404 |
| Post (friends) | ✓ | ✓ | ✗ | 404 |
| Post (group) | ✓* | ✓* | ✓* | 404 |
| Comment | inherits post | inherits post | inherits post | 404 |
| Conversation | ✓ (participant) | ✓ (participant) | ✗ | 404 |
| Group (platform) | ✓ | ✓ | ✓ | 404 |
| Group (user-created) | ✓* | ✓* | ✗ | 404 |
| Event (friends) | ✓ | ✓ | ✗ | 404 |
| Event (group) | ✓* | ✓* | ✓* | 404 |
| Event (discoverable) | ✓ | ✓ | ✓ | 404 |

\* = requires active group membership

### §9.2 Block Enforcement Rules

Block enforcement has the **highest priority** in all access checks. When either party
has blocked the other, all interactions return 404 (not 403) to avoid revealing that a
block exists. `[S§7.4]`

1. **Check order**: Block check MUST happen before any other visibility logic.
2. **Bidirectional**: If A blocks B, then B→A interactions also return 404.
3. **Silent**: The blocked user receives 404, identical to "content not found."
4. **Scope**: Blocks affect profiles, posts, comments, messages, friend requests, groups
   (blocked user's posts hidden), and events.
5. **Feed purge**: When A blocks B, B's posts are immediately removed from A's Redis feed
   (inline, not async).
6. **Friendship removal**: Blocking removes any existing friendship between the families.

### §9.3 Profile Privacy Settings

Profile privacy uses per-field JSONB settings with two levels: `[S§7.1]`

| Value | Meaning |
|-------|---------|
| `"friends"` | Visible to accepted friends only (default for all fields) |
| `"hidden"` | Not visible to anyone except the profile owner |

There is **no `"public"` value**. This is enforced by validation in the service layer.

Fields controlled by privacy settings:
- `display_name` — family display name
- `parent_names` — names of parents in the family
- `children_names` — first names of children
- `children_ages` — ages of children
- `location` — location region
- `methodology` — selected methodology names

### §9.4 Feed Visibility Filtering

The feed pipeline applies visibility filtering at two levels:

1. **Structural** (Redis): Feed sorted set contains only post IDs from accepted friends
   (via fan-out-on-write). Non-friends' posts never enter the feed.
2. **Application** (Go): After hydrating posts from PostgreSQL, a block filter removes
   any posts from families in the viewer's block list (either direction).

### §9.5 Centralized Visibility Module

All visibility logic lives in `internal/social/domain/visibility.go` (see §20). This module
exports pure functions that take content, viewer context, and relationship data, and
return a boolean (visible/not-visible) or filtered content. Service methods delegate
all visibility decisions to this module rather than embedding checks inline.

---

## §10 Friendship State Machine (Domain Deep-Dive 2)

### §10.1 Aggregate Root

The `Friendship` aggregate root lives in `internal/social/domain/friendship.go`. It enforces
all friendship state transitions and invariants. `[ARCH §4.5]`

### §10.2 State Diagram

```
    ┌──────┐  send_request()  ┌─────────┐
    │ None │─────────────────→│ Pending │
    └──────┘                  └────┬────┘
                                   │
                    ┌──────────────┼──────────────┐
                    │ accept()     │ reject()     │
                    ▼              ▼              │
              ┌──────────┐   ┌──────┐            │
              │ Accepted │   │ None │←───────────┘
              └────┬─────┘   └──────┘
                   │
          ┌────────┼────────┐
          │ unfriend()      │ block()
          ▼                 ▼
    ┌──────┐          ┌──────────┐
    │ None │          │ Blocked  │  (tracked in soc_blocks, not soc_friendships)
    └──────┘          └──────────┘
```

### §10.3 Transition Table

| Current State | Action | Next State | Preconditions | Side Effects |
|--------------|--------|------------|---------------|-------------|
| None | `send_request(target)` | Pending | Not self; not blocked (either direction); no existing friendship | Creates `soc_friendships` row; publishes `FriendRequestSent` |
| Pending | `accept()` | Accepted | Caller is accepter (not requester) | Updates status; publishes `FriendRequestAccepted`; triggers feed rebuild for both |
| Pending | `reject()` | None | Caller is accepter | Deletes `soc_friendships` row |
| Pending | `cancel()` | None | Caller is requester | Deletes `soc_friendships` row |
| Accepted | `unfriend(target)` | None | Either participant | Deletes `soc_friendships` row; no notification; no feed purge |
| Accepted | `block(target)` | Blocked | Either participant | Deletes friendship; creates `soc_blocks` row; purges feed |
| None | `block(target)` | Blocked | Not self | Creates `soc_blocks` row |
| Blocked | `unblock()` | None | Only the blocker | Deletes `soc_blocks` row; does NOT restore friendship |

### §10.4 Invariants

1. **No duplicate friendships**: The UNIQUE constraint on `(requester_family_id, accepter_family_id)` prevents duplicate requests. The service checks both directions before creating.
2. **No self-friendship**: CHECK constraint `requester_family_id != accepter_family_id`.
3. **Block prevents friendship**: `send_request()` checks `soc_blocks` for either-direction block before creating.
4. **Only accepter can accept/reject**: The service verifies the caller is the `accepter_family_id`.
5. **Unblock does not restore friendship**: After unblocking, families start from "None" state — they must send a new friend request.

---

## §11 Feed Architecture (Domain Deep-Dive 3)

### §11.1 Fan-Out-on-Write

When a post is created, the service enqueues a `FanOutPostJob` that writes the post ID
to each friend's Redis feed sorted set. `[ARCH §2.7]`

```
Post Created
    │
    ▼
FanOutPostJob (hibiken/asynq)
    │
    ├── Get author's friend list (FriendshipRepository::list_friend_family_ids)
    │
    ├── For each friend:
    │   └── ZADD feed:{friend_family_id} {timestamp} {post_id}
    │
    └── ZADD feed:{author_family_id} {timestamp} {post_id}  (own posts in own feed)
```

**Redis key**: `feed:{family_id}` — sorted set, score = Unix timestamp (milliseconds).

**TTL**: Feed entries are trimmed to the most recent 500 posts per family. Older posts
fall back to PostgreSQL query.

### §11.2 Feed Read Path

```
GET /v1/social/feed
    │
    ▼
1. ZREVRANGE feed:{family_id} {cursor} {limit}
    │         → returns [post_id, ...]
    │
    ▼
2. PostRepository::find_by_ids(post_ids)
    │         → returns [Post, ...]
    │
    ▼
3. Block filter: remove posts from blocked families
    │         → uses BlockRepository::list_all_blocked_family_ids
    │
    ▼
4. Enrich: resolve author names, photos, like status, group names
    │         → IamService + PostLikeRepository
    │
    ▼
5. Return PaginatedResponse<PostResponse>
```

### §11.3 Feed Rebuild (PostgreSQL Fallback)

If a family's Redis feed is empty (new user, Redis flush, or cold start), the service
falls back to a PostgreSQL query:

```sql
SELECT p.* FROM soc_posts p
WHERE p.family_id = ANY($1)       -- $1 = friend family IDs + own family ID
  AND p.group_id IS NULL          -- only friends-visibility posts
  AND p.visibility = 'friends'
ORDER BY p.created_at DESC
LIMIT $2
```

The results are written back to Redis for subsequent reads. This ensures Redis is an
**optimization, not a source of truth**. `[ARCH §1.4]`

### §11.4 Feed Purge on Block

When A blocks B, B's posts are **immediately** removed from A's feed:

```
ZRANGEBYSCORE feed:{A_family_id} -inf +inf
  → filter post_ids where post.family_id = B_family_id
  → ZREM feed:{A_family_id} {matching_post_ids}
```

This happens **inline** (not async) because the user expects blocked content to disappear
immediately.

### §11.5 Feed on Unfriend

When A unfriends B, B's posts are **not** immediately purged from A's feed. They will
naturally age out as A scrolls past them and new posts push old ones out of the 500-post
window. This is intentional — unfriending is a lighter action than blocking.

---

## §12 Direct Messaging Architecture (Domain Deep-Dive 4)

### §12.1 Conversation Model

Direct messaging uses a **conversation abstraction** rather than simple sender/recipient
on each message: `[S§7.5]`

```
soc_conversations (1)
    │
    ├── soc_conversation_participants (2) — one per parent
    │   ├── last_read_at — per-user read tracking
    │   └── deleted_at — per-user conversation deletion
    │
    └── soc_messages (N) — all messages in the conversation
        └── sender_parent_id — who sent this message
```

**Why conversations**: This abstraction supports per-user deletion (one user deletes
their view of the conversation, the other still sees it), read tracking per participant,
and potential future extension to group messaging (Phase 3+).

### §12.2 Per-User Deletion Semantics

When a user deletes a conversation:
1. Their `soc_conversation_participants.deleted_at` is set to `now()`.
2. Messages are **not deleted** from the database.
3. Subsequent `GET /v1/social/conversations` excludes conversations where `deleted_at IS NOT NULL`.
4. Subsequent `GET /v1/social/conversations/:id/messages` returns only messages with `created_at > deleted_at`.
5. If the other participant sends a new message, `deleted_at` is cleared — the conversation reappears with only the new message visible.

### §12.3 WebSocket Integration

Real-time message delivery uses gorilla/websocket with Redis pub/sub
for multi-connection distribution. `[ARCH §2.16]`

```
Message Sent
    │
    ├── 1. Insert into soc_messages
    │
    ├── 2. PUBLISH ws:parent:{recipient_parent_id} {WsMessage JSON}
    │
    └── 3. Recipient's WebSocket connection receives via Redis SUBSCRIBE
```

**Channel key**: `ws:parent:{parent_id}` — each authenticated parent subscribes to their
own channel on WebSocket connect.

**Connection lifecycle**:
1. Client connects to `GET /v1/social/ws` with session cookie
2. Server validates session, extracts `parent_id`
3. Server subscribes to `ws:parent:{parent_id}` Redis channel
4. Incoming Redis messages are forwarded to the WebSocket connection
5. On disconnect, Redis subscription is cleaned up

### §12.4 Friends-Only Guard

All messaging operations check friendship status: `[S§7.5]`

1. `create_conversation`: Verifies recipient is a friend and not blocked.
2. `send_message`: Verifies friendship still active and not blocked. If friendship has
   been removed since conversation creation, returns `NotFriends` error.

---

## §13 Group Lifecycle (Domain Deep-Dive 5)

### §13.1 Platform-Managed vs User-Created Groups

| Aspect | Platform-Managed | User-Created |
|--------|:---:|:---:|
| Created by | Database seed / admin | Any user |
| Deletable | No | Yes (owner only) |
| `group_type` | `'platform'` | `'user_created'` |
| `methodology_id` | Required (one per methodology) | Optional |
| `join_policy` | Always `'open'` | User's choice |
| Moderation | Platform moderators | Creator (owner) + promoted moderators |
| Visibility | All authenticated users | Members only |

### §13.2 Membership States

| Status | Meaning | Can Post | Visible in Member List |
|--------|---------|:---:|:---:|
| `active` | Full member | ✓ | ✓ |
| `pending` | Submitted join request (request_to_join) | ✗ | ✗ |
| `invited` | Moderator/owner invited (invite_only) | ✗ | ✗ |
| `banned` | Removed by moderator/owner | ✗ | ✗ |

### §13.3 Moderation Roles

| Action | Member | Moderator | Owner |
|--------|:---:|:---:|:---:|
| Post in group | ✓ | ✓ | ✓ |
| View members | ✓ | ✓ | ✓ |
| Approve join requests | ✗ | ✓ | ✓ |
| Reject join requests | ✗ | ✓ | ✓ |
| Ban members | ✗ | ✓ | ✓ |
| Invite members | ✗ | ✓ | ✓ |
| Promote to moderator | ✗ | ✗ | ✓ |
| Delete group | ✗ | ✗ | ✓ |
| Update group settings | ✗ | ✓ | ✓ |
| Pin/unpin posts | ✗ | ✓ | ✓ |

---

## §14 Event Lifecycle (Domain Deep-Dive 6)

### §14.1 RSVP States and Capacity

| RSVP Status | Meaning | Counts Toward Capacity |
|-------------|---------|:---:|
| `going` | Confirmed attendance | ✓ |
| `interested` | May attend | ✗ |
| `not_going` | Declined | ✗ |

**Capacity logic**: If `soc_events.capacity` is set, the service checks
`EventRsvpRepository::count_going(event_id)` before accepting a `going` RSVP. If
at capacity, returns `EventAtCapacity` error. Changing from `going` to
`interested`/`not_going` frees the spot. `[S§7.7]`

### §14.2 Cancellation

When an event is cancelled: `[S§7.7]`
1. Service sets `soc_events.status = 'cancelled'`
2. Publishes `EventCancelled` event with list of `going` family IDs
3. `notify::` subscriber sends cancellation notification to all `going` RSVPs
4. Cancelled events remain visible in listings (greyed out) but cannot accept new RSVPs

### §14.3 Event Visibility

Events have a third visibility option not available to posts: `[S§7.7]`

| Visibility | Who Can See | Use Case |
|------------|-------------|----------|
| `friends` | Creator's friends only | Small gatherings |
| `group` | Group members only | Group-organized events |
| `discoverable` | Anyone searching by location/methodology | Co-op days, workshops |

**Constraint**: `visibility = 'group'` requires `group_id IS NOT NULL`.

---

## §15 Location-Based Discovery (Domain Deep-Dive 7)

### §15.1 PostGIS Data Architecture

Location data lives on `iam_families` (owned by `iam::`) as a PostGIS point column.
The social domain reads this data through `iam::IamService` — it does not write to
IAM tables. `[ARCH §1.5, S§7.8]`

```sql
-- Defined in IAM migrations (see 01-iam §3.2)
-- iam_families.location_point GEOGRAPHY(POINT, 4326)
-- Populated from geocoded location_region (coarse — city centroid, not address)
```

**Opt-in**: Location discovery is controlled by `soc_profiles.location_visible`.
Families with `location_visible = false` are excluded from all location-based queries.

### §15.2 Privacy Guarantees

1. **No precise coordinates stored**: `location_point` is geocoded to the **city centroid**,
   not a street address. `[S§7.8]`
2. **Opt-in only**: Users explicitly enable `location_visible` in their profile. Default
   is `false`.
3. **Immediate removal**: Setting `location_visible = false` immediately removes the
   family from all location-based query results.
4. **Block filtering**: Location discovery results are filtered through the block list —
   blocked families never appear in discovery results.

### §15.3 Discovery Queries

**Nearby families** (Phase 2):
```sql
SELECT f.id, f.display_name, p.profile_photo_url
FROM iam_families f
JOIN soc_profiles p ON p.family_id = f.id
WHERE p.location_visible = true
  AND f.location_point IS NOT NULL
  AND ST_DWithin(f.location_point, $1::geography, $2)  -- $2 = radius in meters
  AND f.id != $3                                        -- exclude self
  AND f.id NOT IN (SELECT blocked_family_id FROM soc_blocks WHERE blocker_family_id = $3
                   UNION
                   SELECT blocker_family_id FROM soc_blocks WHERE blocked_family_id = $3)
ORDER BY ST_Distance(f.location_point, $1::geography)
LIMIT 50;
```

**Nearby events** (Phase 2): Filter `soc_events` where `visibility = 'discoverable'`
and `location_region` matches or `methodology_id` matches.

---

## §16 Error Types

`SocialError` enum defined in `internal/social/domain/errors.go`. Maps to `AppError` via
the `ToAppError()` method `[00-core §6.4]`. `[CODING §2.2, CODING §8.3]`

```go
// Sentinel errors — use errors.Is(err, ErrXxx) for matching
var (
    // --- Profile ------------------------------------------------─────────
    ErrProfileNotFound      = errors.New("social: profile not found")
    ErrInvalidPrivacySetting = errors.New("social: invalid privacy setting value")

    // --- Friendship ------------------------------------------------─────
    ErrCannotFriendSelf = errors.New("social: cannot send friend request to self")
    ErrAlreadyFriends   = errors.New("social: already friends with this family")
    ErrAlreadyPending   = errors.New("social: friend request already pending")
    ErrFriendshipNotFound = errors.New("social: friendship not found")
    ErrNotAccepter      = errors.New("social: not the accepter of this friend request")
    ErrNotPending       = errors.New("social: friendship is not pending")
    ErrNotFriends       = errors.New("social: not friends with this family")

    // --- Block ------------------------------------------------───────────
    ErrBlockedByTarget  = errors.New("social: blocked by target family")
    ErrTargetBlocked    = errors.New("social: target family is blocked")
    ErrCannotBlockSelf  = errors.New("social: cannot block self")
    ErrAlreadyBlocked   = errors.New("social: already blocked")
    ErrBlockNotFound    = errors.New("social: block not found")

    // --- Post ------------------------------------------------────────────
    ErrPostNotFound     = errors.New("social: post not found")
    ErrContentNotVisible = errors.New("social: content not visible")
    ErrNotPostAuthor    = errors.New("social: not the post author")
    ErrInvalidPostType  = errors.New("social: invalid post type")
    ErrContentRequired  = errors.New("social: content required for text posts")
    ErrAttachmentsRequired = errors.New("social: attachments required for photo posts")

    // --- Comment ------------------------------------------------─────────
    ErrCommentNotFound  = errors.New("social: comment not found")
    ErrNotCommentAuthorOrPostAuthor = errors.New("social: not comment author or post author")
    ErrCannotReplyToReply = errors.New("social: cannot reply to a reply (one level only)")

    // --- Messaging ------------------------------------------------───────
    ErrConversationNotFound = errors.New("social: conversation not found")
    ErrNotParticipant   = errors.New("social: not a conversation participant")
    ErrMessageNotFound  = errors.New("social: message not found")

    // --- Group ------------------------------------------------───────────
    ErrGroupNotFound    = errors.New("social: group not found")
    ErrAlreadyGroupMember = errors.New("social: already a group member")
    ErrInviteOnly       = errors.New("social: group is invite only")
    ErrBannedFromGroup  = errors.New("social: banned from group")
    ErrNotGroupMember   = errors.New("social: not a group member")
    ErrOwnerCannotLeave = errors.New("social: owner cannot leave without transferring ownership")
    ErrInsufficientGroupPermissions = errors.New("social: insufficient group permissions")
    ErrCannotDeletePlatformGroup = errors.New("social: cannot delete platform group")

    // --- Event ------------------------------------------------───────────
    ErrEventNotFound    = errors.New("social: event not found")
    ErrNotEventCreator  = errors.New("social: not the event creator")
    ErrEventCancelled   = errors.New("social: event is cancelled")
    ErrEventAtCapacity  = errors.New("social: event is at capacity")
    ErrEventDateInPast  = errors.New("social: event date must be in the future")
    ErrGroupVisibilityRequiresGroup = errors.New("social: group visibility requires group_id")

    // --- Infrastructure ------------------------------------------------──
    ErrDatabase         = errors.New("social: database error")
)
```

### §16.1 Error-to-HTTP Mapping

| SocialError Type | HTTP Status | Error Code | Notes |
|--------------------|-------------|------------|-------|
| `ErrProfileNotFound` | 404 | `profile_not_found` | |
| `ErrInvalidPrivacySetting` | 422 | `invalid_privacy_setting` | |
| `ErrCannotFriendSelf` | 422 | `cannot_friend_self` | |
| `ErrAlreadyFriends` | 409 | `already_friends` | |
| `ErrAlreadyPending` | 409 | `already_pending` | |
| `ErrFriendshipNotFound` | 404 | `friendship_not_found` | |
| `ErrNotAccepter` | 403 | `not_accepter` | |
| `ErrNotPending` | 409 | `not_pending` | |
| `ErrNotFriends` | 403 | `not_friends` | |
| `ErrBlockedByTarget` | **404** | `not_found` | **Silent blocking** — maps to 404, not 403 |
| `ErrTargetBlocked` | **404** | `not_found` | **Silent blocking** — maps to 404, not 403 |
| `ErrCannotBlockSelf` | 422 | `cannot_block_self` | |
| `ErrAlreadyBlocked` | 409 | `already_blocked` | |
| `ErrBlockNotFound` | 404 | `block_not_found` | |
| `ErrPostNotFound` | 404 | `post_not_found` | |
| `ErrContentNotVisible` | **404** | `not_found` | Maps to 404 to avoid revealing existence |
| `ErrNotPostAuthor` | 403 | `not_post_author` | |
| `ErrInvalidPostType` | 422 | `invalid_post_type` | |
| `ErrContentRequired` | 422 | `content_required` | |
| `ErrAttachmentsRequired` | 422 | `attachments_required` | |
| `ErrCommentNotFound` | 404 | `comment_not_found` | |
| `ErrNotCommentAuthorOrPostAuthor` | 403 | `not_comment_author_or_post_author` | |
| `ErrCannotReplyToReply` | 422 | `cannot_reply_to_reply` | |
| `ErrConversationNotFound` | 404 | `conversation_not_found` | |
| `ErrNotParticipant` | 403 | `not_participant` | |
| `ErrMessageNotFound` | 404 | `message_not_found` | |
| `ErrGroupNotFound` | 404 | `group_not_found` | |
| `ErrAlreadyGroupMember` | 409 | `already_group_member` | |
| `ErrInviteOnly` | 403 | `invite_only` | |
| `ErrBannedFromGroup` | 403 | `banned_from_group` | |
| `ErrNotGroupMember` | 404 | `not_group_member` | |
| `ErrOwnerCannotLeave` | 422 | `owner_cannot_leave` | |
| `ErrInsufficientGroupPermissions` | 403 | `insufficient_group_permissions` | |
| `ErrCannotDeletePlatformGroup` | 403 | `cannot_delete_platform_group` | |
| `ErrEventNotFound` | 404 | `event_not_found` | |
| `ErrNotEventCreator` | 403 | `not_event_creator` | |
| `ErrEventCancelled` | 409 | `event_cancelled` | |
| `ErrEventAtCapacity` | 422 | `event_at_capacity` | |
| `ErrEventDateInPast` | 422 | `event_date_in_past` | |
| `ErrGroupVisibilityRequiresGroup` | 422 | `group_visibility_requires_group` | |
| `ErrDatabase` / GORM errors | 500 | `internal_error` | |

**Critical**: `ErrBlockedByTarget`, `ErrTargetBlocked`, and `ErrContentNotVisible` all map to
HTTP 404 with a generic `not_found` error code. This prevents information leakage about
whether a block exists or content is being hidden. `[S§7.4]`

---

## §17 Cross-Domain Interactions

### §17.1 social:: Provides (consumed by other domains)

| Export | Consumers | Mechanism |
|--------|-----------|-----------|
| `SocialService` interface methods | `onboard::` (Phase 2) | `SocialService (interface)` via AppState |
| `PostCreated` event | `safety::`, `search::` | Domain event — content scan, search index |
| `FriendRequestAccepted` event | `notify::` | Domain event — notification to requester |
| `FriendRequestSent` event | `notify::` | Domain event — notification to target |
| `MessageSent` event | `notify::`, `safety::` (Phase 2) | Domain event — notification if recipient offline, grooming detection `[11-safety §14.2]` |
| `EventCancelled` event | `notify::` | Domain event — notification to RSVPed attendees |
| `MessageReported` event | `safety::` | Domain event — moderation queue entry |

### §17.2 social:: Consumes

| Dependency | Source | Purpose |
|-----------|--------|---------|
| Family data (display_name, parents, students) | `iam::IamService` | Populate profile views, enrich posts/comments with author info |
| Parent lookup by ID | `iam::IamService` | Resolve parent for messaging, authorship |
| Methodology display names | `method::MethodologyService` | Show methodology names in profiles, groups, events |
| Family location_point | `iam::IamService` (read) | Location-based discovery queries (Phase 2) |

### §17.3 Events social:: Publishes

Defined in `internal/social/events.go`. `[CODING §8.4]`

```go
// internal/social/events.go

type PostCreated struct {
    PostID uuid.UUID `json:"post_id"`
    FamilyID FamilyID `json:"family_id"`
    PostType string `json:"post_type"`
    Content *string `json:"content"`
    Attachments json.RawMessage `json:"attachments"`
    GroupID *uuid.UUID `json:"group_id"`
}
// Implements DomainEvent interface

type FriendRequestSent struct {
    FriendshipID uuid.UUID `json:"friendship_id"`
    RequesterFamilyID FamilyID `json:"requester_family_id"`
    TargetFamilyID FamilyID `json:"target_family_id"`
}
// Implements DomainEvent interface

type FriendRequestAccepted struct {
    FriendshipID uuid.UUID `json:"friendship_id"`
    RequesterFamilyID FamilyID `json:"requester_family_id"`
    AccepterFamilyID FamilyID `json:"accepter_family_id"`
}
// Implements DomainEvent interface

type MessageSent struct {
    MessageID uuid.UUID `json:"message_id"`
    ConversationID uuid.UUID `json:"conversation_id"`
    SenderParentID uuid.UUID `json:"sender_parent_id"`
    SenderFamilyID FamilyID `json:"sender_family_id"`
    RecipientParentID uuid.UUID `json:"recipient_parent_id"`
    RecipientFamilyID FamilyID `json:"recipient_family_id"`
}
// Implements DomainEvent interface

type EventCancelled struct {
    EventID uuid.UUID `json:"event_id"`
    CreatorFamilyID FamilyID `json:"creator_family_id"`
    Title string `json:"title"`
    EventDate time.Time `json:"event_date"`
    GoingFamilyIDs []FamilyID `json:"going_family_ids"`
}
// Implements DomainEvent interface

type MessageReported struct {
    MessageID uuid.UUID `json:"message_id"`
    ReporterFamilyID FamilyID `json:"reporter_family_id"`
    ReportedMessageSenderID uuid.UUID `json:"reported_message_sender_id"`
    ConversationID uuid.UUID `json:"conversation_id"`
    Reason string `json:"reason"`
}
// Implements DomainEvent interface
```

### §17.4 Events social:: Subscribes To

| Event | Source | Effect |
|-------|--------|--------|
| `FamilyCreated { family_id, parent_id }` | `iam::` | Create `soc_profiles` row for the new family — see `[01-iam §13]` |
| `MilestoneAchieved { family_id, student_id, ... }` | `learn::` | Create an optional `milestone` post if family has opted in `[ARCH §4.6]` |
| `CoParentRemoved { family_id, parent_id }` | `iam::` | Disassociate removed parent's posts from family — retain content but clear `author_parent_id` reference `[S§3.4]` |
| `FamilyDeletionScheduled { family_id, delete_after }` | `iam::` | Mark social data for cascade deletion after grace period `[S§16.3]` |

```go
// internal/social/event_handlers.go

// import "homegrown-academy/internal/iam"
// import "homegrown-academy/internal/learn"

type FamilyCreatedHandler struct {
    socialService SocialService
}

func (h *FamilyCreatedHandler) Handle(ctx context.Context, event *FamilyCreated) error {
    return h.socialService.HandleFamilyCreated(ctx, event.FamilyID)
}

type CoParentRemovedHandler struct {
    socialService SocialService
}

func (h *CoParentRemovedHandler) Handle(ctx context.Context, event *CoParentRemoved) error {
    return h.socialService.HandleCoParentRemoved(ctx, event.FamilyID, event.ParentID)
}

type MilestoneAchievedHandler struct {
    socialService SocialService
}

func (h *MilestoneAchievedHandler) Handle(ctx context.Context, event *MilestoneAchieved) error {
    return h.socialService.HandleMilestoneAchieved(ctx, event.FamilyID, MilestoneData{
        StudentName:   event.StudentName,
        MilestoneType: event.MilestoneType,
        Description:   event.Description,
    })
}

type FamilyDeletionScheduledHandler struct {
    socialService SocialService
}

func (h *FamilyDeletionScheduledHandler) Handle(ctx context.Context, event *FamilyDeletionScheduled) error {
    return h.socialService.HandleFamilyDeletionScheduled(ctx, event.FamilyID)
}
```

---

## §18 Phase Scope

### Phase 1 — Foundation

**In scope**:
- All `soc_` tables: profiles, friendships, blocks, posts, comments, post_likes,
  conversations, conversation_participants, messages, groups, group_members, events,
  event_rsvps (13 tables)
- RLS policies for applicable tables
- Core CRUD for all entities
- Feed architecture: fan-out-on-write via Redis, PostgreSQL fallback
- Direct messaging with conversation abstraction, per-user deletion
- Platform-managed groups (join, leave, post)
- Basic events (create, RSVP, cancel)
- WebSocket endpoint for real-time messaging
- Block enforcement (silent → 404)
- ~39 Phase 1 endpoints
- `SocialService` interface + `SocialServiceImpl`
- 12 repository interfaces + PostgreSQL implementations
- Domain `domain/` subdirectory: friendship.go, post.go, group_membership.go,
  visibility.go, errors.go
- `SocialError` types + HTTP mapping
- Domain events: `PostCreated`, `FriendRequestSent`, `FriendRequestAccepted`,
  `MessageSent`, `EventCancelled`, `MessageReported`
- Event handlers: `FamilyCreatedHandler`, `CoParentRemovedHandler`,
  `MilestoneAchievedHandler`, `FamilyDeletionScheduledHandler`
- All Phase 1 models (request, response, internal)
- OpenAPI spec (swaggo/swag) + TypeScript type generation

### Phase 2 — Depth

**In scope**:
- User-created groups (create, update, delete)
- Group member management (approve, reject, ban, promote, invite)
- Location-based discovery (families, events, groups via PostGIS)
- Friend suggestions (methodology-based, mutual friends)
- Post editing (`PATCH /v1/social/posts/:post_id`, `is_edited` flag)
- Pinned posts in groups
- Recurring events
- Milestone posts (auto-generated from `MilestoneAchieved` events)
- ~13 Phase 2 endpoints

### Phase 3+ — Expansion

**Deferred**:
- Algorithmic feed ranking (with user toggle for chronological) `[S§7.2.3]`
- Message reactions (emoji)
- Group chat (multi-participant conversations)
- Apache AGE graph queries for friend-of-friend suggestions
- Rich media embeds in posts (link previews, video embeds)
- Post sharing / reposting
- Friend suggestions based on mutual friends

### §18.4 Apache AGE Migration Contract

Per `ARCHITECTURE.md` ADR-009, Apache AGE will be adopted when friend-of-friend queries
exceed 100 ms or recommendation features require multi-hop traversals. This section defines
the **binding contract** for migrating `soc_` relational data to a hybrid
PostgreSQL + AGE architecture. No data may be lost during migration.

#### Node / Edge Classification

Every `soc_` table maps to exactly one AGE role. External tables that appear as graph nodes
are marked "shadow" — the relational table remains the source of truth.

| Table | AGE Role | Label | Notes |
|-------|----------|-------|-------|
| `iam_families` (external) | Node | `:Family` | Source of truth stays in `iam_` — AGE gets a shadow node |
| `soc_profiles` | Node property | — | Merged as properties on `:Family` node |
| `soc_friendships` | Edge | `[:FRIENDS_WITH]` | Undirected; store `requester_family_id` as `requested_by` and `status`, `created_at`, `updated_at` as edge props |
| `soc_blocks` | Edge | `[:BLOCKS]` | Directed; `blocker → blocked`; `created_at` as edge prop |
| `soc_posts` | Node | `:Post` | Content stays in PostgreSQL; AGE node holds `id`, `family_id`, `author_parent_id`, `post_type`, `visibility`, `created_at` |
| `soc_comments` | Node + Edge | `:Comment` + `[:REPLY_TO]` | Self-referential threading becomes `[:REPLY_TO]` edge; 1-level limit enforced in app. Also linked to parent post via `[:COMMENT_ON]` edge |
| `soc_post_likes` | Edge | `[:LIKES]` | `Family → Post`; `created_at` as edge prop |
| `soc_conversations` | Node | `:Conversation` | Minimal node (`id`, `created_at`, `updated_at`) |
| `soc_conversation_participants` | Edge | `[:PARTICIPANT_IN]` | `Parent → Conversation`; `last_read_at`, `deleted_at` as edge properties |
| `soc_messages` | Node | `:Message` | Content stays in PostgreSQL; attached to `:Conversation` via `[:IN_CONVERSATION]` edge |
| `soc_groups` | Node | `:Group` | Full properties on node |
| `soc_group_members` | Edge | `[:MEMBER_OF]` | `Family → Group`; `role`, `status`, `joined_at`, `created_at`, `updated_at` as edge properties |
| `soc_events` | Node | `:Event` | Full properties on node |
| `soc_event_rsvps` | Edge | `[:RSVP]` | `Family → Event`; `status`, `created_at`, `updated_at` as edge properties |

#### What Stays Relational (Hybrid Architecture)

Phase 3 is **hybrid** — AGE handles graph traversal queries (shortest path, friend-of-friend,
community detection), PostgreSQL handles everything else:

- **PostGIS spatial queries** — AGE has no spatial support; location-based discovery for
  families, events, and groups stays in PostgreSQL.
- **Full-text search** — `search_vector` / GIN indexes on `soc_posts` stay in PostgreSQL.
- **Content storage** — Post body (`content`), message text, comment text, `attachments`
  JSONB all stay in relational tables. AGE nodes hold only identifiers and metadata needed
  for traversal.
- **RLS policies** — AGE does not support row-level security. `domain/visibility.go`
  application-layer enforcement is already the primary access-control gate (RLS is
  defense-in-depth per §3.3), so this transitions cleanly.
- **CASCADE deletion** — AGE has no FK cascades. `FamilyDeletionScheduledHandler` already
  handles cascade at application level — extend it to delete corresponding AGE nodes/edges
  when a family is purged.
- **Counters** — `likes_count`, `comments_count`, `member_count`, `attendee_count` stay in
  relational tables. AGE `COUNT` queries are too expensive for hot-read paths.

#### Known Friction Points

| # | Issue | Resolution |
|---|-------|------------|
| 1 | **Friendship asymmetry** — `(requester_family_id, accepter_family_id)` stores a symmetric relationship asymmetrically. | Preserve as edge properties `{requested_by, created_at, updated_at}` on a single undirected `[:FRIENDS_WITH]` edge. Query both directions with `()-[:FRIENDS_WITH]-()`. |
| 2 | **Conversation participant soft-delete** — `deleted_at` on `soc_conversation_participants` is a soft-delete pattern with no AGE equivalent. | Store as edge property. Filter `WHERE rel.deleted_at IS NULL` in Cypher queries. Restore by setting `rel.deleted_at = NULL`. |
| 3 | **Comment 1-level threading** — AGE supports unlimited `[:REPLY_TO]` depth natively, which could silently relax the threading constraint. | Enforce 1-level limit in `domain/post.go` validation (application layer), not in the graph schema. Document this constraint explicitly so it is not accidentally relaxed after migration. |

#### Edge Property Preservation Guarantee

**Contract**: every column on every join table (edge table) MUST be preserved as a named
property on the corresponding AGE edge. No data loss during migration.

| Relational column | AGE edge property | Notes |
|-------------------|-------------------|-------|
| `id` (PK) | `original_id` | AGE assigns its own internal `graphid`; preserve the UUID for cross-reference |
| `created_at` | `created_at` | — |
| `updated_at` | `updated_at` | — |
| `status` | `status` | On `soc_friendships`, `soc_group_members`, `soc_event_rsvps` |
| `role` | `role` | On `soc_group_members` |
| `joined_at` | `joined_at` | On `soc_group_members` |
| `last_read_at` | `last_read_at` | On `soc_conversation_participants` |
| `deleted_at` | `deleted_at` | On `soc_conversation_participants` |

#### Example Migration: Friendships

```sql
-- Load Apache AGE extension
LOAD 'age';
SET search_path = ag_catalog, "$user", public;

-- Create graph
SELECT create_graph('social');

-- Migrate accepted friendships as undirected edges
SELECT * FROM cypher('social', $$
    MATCH (a:Family {id: $requester}), (b:Family {id: $accepter})
    CREATE (a)-[:FRIENDS_WITH {
        original_id: $id,
        requested_by: $requester,
        status: $status,
        created_at: $created_at,
        updated_at: $updated_at
    }]->(b)
$$) AS (result agtype);

-- Friend-of-friend query (the trigger for AGE adoption per ADR-009)
SELECT * FROM cypher('social', $$
    MATCH (me:Family {id: $my_family_id})-[:FRIENDS_WITH]-(:Family)-[:FRIENDS_WITH]-(suggestion:Family)
    WHERE NOT (me)-[:FRIENDS_WITH]-(suggestion)
      AND NOT (me)-[:BLOCKS]-(suggestion)
      AND suggestion.id <> $my_family_id
    RETURN DISTINCT suggestion.id, count(*) AS mutual_count
    ORDER BY mutual_count DESC
    LIMIT 20
$$) AS (family_id agtype, mutual_count agtype);
```

---

## §19 Verification Checklist

Numbered assertions that MUST be true when the social:: implementation is correct. Use these
as acceptance criteria for code review and integration testing.

### Profile

1. `FamilyCreated` event creates a `soc_profiles` row with default privacy settings (all fields = "friends")
2. `GET /v1/social/profile` returns the authenticated family's full profile
3. `PATCH /v1/social/profile` updates bio, photo, privacy_settings, location_visible
4. `GET /v1/social/families/:id/profile` returns friend's profile filtered by privacy settings
5. `GET /v1/social/families/:id/profile` for a blocked family returns 404 (not 403)
6. Profile privacy settings only accept values "friends" or "hidden" — no "public"

### Friendship

7. `POST /v1/social/friends/request` creates a pending friendship
8. Sending a friend request to yourself returns 422
9. Sending a request when already friends returns 409
10. Sending a request when blocked (either direction) returns 404
11. Only the accepter can accept or reject a friend request
12. Accepting a friend request publishes `FriendRequestAccepted` event
13. `DELETE /v1/social/friends/:family_id` removes the friendship without notification
14. Unfriending does not purge the unfriended family's posts from the feed

### Blocking

15. `POST /v1/social/blocks` creates a block and removes any existing friendship
16. Blocking purges the blocked family's posts from the blocker's feed immediately
17. All interactions with a blocked family return 404 (not 403)
18. Unblocking does not restore the previous friendship

### Feed & Posts

19. Creating a post triggers fan-out to all friends' Redis feeds
20. `GET /v1/social/feed` returns posts in reverse chronological order
21. Feed falls back to PostgreSQL when Redis feed is empty
22. Posts from blocked families are filtered out of the feed
23. `POST /v1/social/posts` with `group_id` sets visibility to "group" automatically
24. Posts without `group_id` always have visibility "friends"
25. Only the author's family can delete a post

### Comments

26. Comments support one level of threading (reply to a comment, but not reply to a reply)
27. Comment authors and post authors can both delete comments
28. Comments inherit the visibility of their parent post

### Messaging

29. Creating a conversation requires friendship (friends-only guard)
30. Per-user deletion sets `deleted_at` without deleting messages
31. New messages after deletion clear `deleted_at` and restore the conversation
32. Messages are delivered via WebSocket when recipient is connected
33. Reporting a message publishes `MessageReported` event for `safety::` moderation

### Groups

34. Platform groups are listed separately and cannot be deleted
35. `POST /v1/social/groups/:id/join` respects join_policy (open → active, request_to_join → pending, invite_only → 403)
36. Owners cannot leave their group without transferring ownership
37. Banned members cannot rejoin the group

### Events

38. Event cancellation publishes `EventCancelled` event with list of `going` family IDs
39. RSVP `going` respects capacity limit — returns 422 when at capacity
40. Events with `discoverable` visibility appear in discovery queries
41. Events with `group` visibility require a group_id

### Privacy Invariants

42. No CHECK constraint or JSONB field allows a `"public"` visibility value
43. All block-related errors map to HTTP 404 with generic "not_found" code
44. `soc_blocks` RLS allows only the blocker to see their block records
45. Location discovery only includes families with `location_visible = true`

### AGE Migration Contract

46. Every `soc_` join table column appears in the §18.4 node/edge classification and edge property preservation tables
47. No `soc_` table is classified as both a node and an edge except `soc_comments` (which is both a `:Comment` node and has a self-referential `[:REPLY_TO]` edge)

---

## §20 Module Structure

```
internal/social/
├── handlers.go               # Echo route handlers (thin layer only)
├── service.go                # SocialServiceImpl — orchestration
├── repository.go             # PgProfileRepository, PgFriendshipRepository,
│                             # PgBlockRepository, PgPostRepository,
│                             # PgCommentRepository, PgPostLikeRepository,
│                             # PgConversationRepository, PgMessageRepository,
│                             # PgGroupRepository, PgGroupMemberRepository,
│                             # PgEventRepository, PgEventRsvpRepository
├── models.go                 # Request/response types, internal types, GORM models
├── ports.go                  # SocialService interface, all 12 repository interfaces
├── events.go                 # PostCreated, FriendRequestSent, FriendRequestAccepted,
│                             # MessageSent, EventCancelled, MessageReported
├── event_handlers.go         # FamilyCreatedHandler, CoParentRemovedHandler,
│                             # MilestoneAchievedHandler, FamilyDeletionScheduledHandler
├── websocket.go              # WebSocket upgrade handler (gorilla/websocket),
│                             # Redis pub/sub integration, WsMessage frame handling
├── feed.go                   # FanOutPostJob, feed read path, feed rebuild,
│                             # Redis sorted set operations
└── domain/
    ├── friendship.go         # Friendship aggregate root — state machine, invariants
    ├── post.go               # Post aggregate — type validation, visibility enforcement
    ├── group_membership.go   # GroupMembership aggregate — role transitions, ban logic
    ├── visibility.go         # Centralized visibility logic — all access checks
    └── errors.go             # SocialError sentinel errors
```
