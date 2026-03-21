# Domain Spec 05 — Social & Community (social::)

## §1 Overview

The Social domain owns the platform's **purpose-built private social network** for
homeschooling families. It provides profiles, timeline/feed, comments, friends, direct
messaging, groups, events, and location-based discovery. All user-generated content
defaults to **friends-only** visibility — there is no public visibility option. `[S§7, V§7]`

| Attribute | Value |
|-----------|-------|
| **Module path** | `src/social/` |
| **DB prefix** | `soc_` |
| **Complexity class** | Complex (has `domain/` subdirectory) `[ARCH §4.5]` |
| **CQRS** | Yes — feed reads separated from post writes `[ARCH §4.7]` |
| **External adapter** | None (Redis is shared infrastructure, WebSocket is Axum built-in) |
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
-- Migration: YYYYMMDD_000001_create_soc_tables.rs
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
are available to free-tier users — no `RequirePremium` extractor. `[S§3.2, S§15.1]`

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
- **Response**: `Vec<FriendRequestResponse>` (200 OK)

##### `GET /v1/social/friends/requests/outgoing`

Lists pending friend requests sent by the authenticated family.

- **Auth**: Required (`FamilyScope`)
- **Response**: `Vec<FriendRequestResponse>` (200 OK)

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
- **Response**: `Vec<BlockedFamilyResponse>` (200 OK)

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
- **Response**: `Vec<GroupSummaryResponse>` (200 OK)

##### `GET /v1/social/groups/platform`

Lists platform-managed methodology groups. `[S§7.6]`

- **Auth**: Required (`FamilyScope`)
- **Response**: `Vec<GroupSummaryResponse>` (200 OK)

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
- **Response**: `Vec<GroupMemberResponse>` (200 OK)

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
- **Response**: `Vec<DiscoverableFamilyResponse>` (200 OK)

##### `GET /v1/social/discover/events`

Discovers events by location and/or methodology. `[S§7.7]`

- **Auth**: Required (`FamilyScope`)
- **Query**: `?methodology_id=<uuid>&location_region=<string>`
- **Response**: `Vec<EventSummaryResponse>` (200 OK)

##### `GET /v1/social/discover/groups`

Discovers groups by methodology. `[S§7.6]`

- **Auth**: Required (`FamilyScope`)
- **Query**: `?methodology_id=<uuid>`
- **Response**: `Vec<GroupSummaryResponse>` (200 OK)

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

The `SocialService` trait defines all use cases exposed to handlers and other domains.
Defined in `src/social/ports.rs`. Methods are organized with CQRS separation: command
methods (writes with side effects) are separated from query methods (reads). `[CODING §8.2, ARCH §4.7]`

```rust
// src/social/ports.rs

#[async_trait]
pub trait SocialService: Send + Sync {
    // ═══ COMMAND SIDE (writes with side effects) ════════════════════════

    // ─── Profile Commands ────────────────────────────────────────────────

    /// Creates a social profile for a new family.
    /// Called by FamilyCreated event handler. [S§7.1]
    async fn create_profile(
        &self,
        family_id: FamilyId,
    ) -> Result<(), AppError>;

    /// Updates the authenticated family's social profile. [S§7.1]
    async fn update_profile(
        &self,
        scope: &FamilyScope,
        cmd: UpdateProfileRequest,
    ) -> Result<ProfileResponse, AppError>;

    // ─── Friend Commands ─────────────────────────────────────────────────

    /// Sends a friend request. [S§7.4]
    async fn send_friend_request(
        &self,
        scope: &FamilyScope,
        cmd: SendFriendRequestCommand,
    ) -> Result<FriendshipResponse, AppError>;

    /// Accepts a pending friend request. [S§7.4]
    async fn accept_friend_request(
        &self,
        scope: &FamilyScope,
        friendship_id: Uuid,
    ) -> Result<FriendshipResponse, AppError>;

    /// Rejects (deletes) a pending friend request. [S§7.4]
    async fn reject_friend_request(
        &self,
        scope: &FamilyScope,
        friendship_id: Uuid,
    ) -> Result<(), AppError>;

    /// Removes an existing friendship. Silent, no notification. [S§7.4]
    async fn unfriend(
        &self,
        scope: &FamilyScope,
        target_family_id: Uuid,
    ) -> Result<(), AppError>;

    /// Blocks a family. Removes friendship if exists, purges feed. [S§7.4]
    async fn block_family(
        &self,
        scope: &FamilyScope,
        cmd: BlockFamilyCommand,
    ) -> Result<(), AppError>;

    /// Unblocks a family. [S§7.4]
    async fn unblock_family(
        &self,
        scope: &FamilyScope,
        target_family_id: Uuid,
    ) -> Result<(), AppError>;

    // ─── Post Commands ───────────────────────────────────────────────────

    /// Creates a post and triggers feed fan-out. [S§7.2]
    async fn create_post(
        &self,
        scope: &FamilyScope,
        cmd: CreatePostCommand,
    ) -> Result<PostResponse, AppError>;

    /// Deletes a post. Author's family only. [S§7.2]
    async fn delete_post(
        &self,
        scope: &FamilyScope,
        post_id: Uuid,
    ) -> Result<(), AppError>;

    /// Likes a post. Idempotent. [S§7.2]
    async fn like_post(
        &self,
        scope: &FamilyScope,
        post_id: Uuid,
    ) -> Result<(), AppError>;

    /// Unlikes a post. [S§7.2]
    async fn unlike_post(
        &self,
        scope: &FamilyScope,
        post_id: Uuid,
    ) -> Result<(), AppError>;

    // ─── Comment Commands ────────────────────────────────────────────────

    /// Creates a comment on a post. [S§7.3]
    async fn create_comment(
        &self,
        scope: &FamilyScope,
        post_id: Uuid,
        cmd: CreateCommentCommand,
    ) -> Result<CommentResponse, AppError>;

    /// Deletes a comment. Author or post author can delete. [S§7.3]
    async fn delete_comment(
        &self,
        scope: &FamilyScope,
        post_id: Uuid,
        comment_id: Uuid,
    ) -> Result<(), AppError>;

    // ─── Messaging Commands ──────────────────────────────────────────────

    /// Creates or retrieves a conversation. Friends-only guard. [S§7.5]
    async fn create_conversation(
        &self,
        scope: &FamilyScope,
        cmd: CreateConversationCommand,
    ) -> Result<ConversationResponse, AppError>;

    /// Sends a message in a conversation. [S§7.5]
    async fn send_message(
        &self,
        scope: &FamilyScope,
        conversation_id: Uuid,
        cmd: SendMessageCommand,
    ) -> Result<MessageResponse, AppError>;

    /// Marks a conversation as read. [S§7.5]
    async fn mark_conversation_read(
        &self,
        scope: &FamilyScope,
        conversation_id: Uuid,
    ) -> Result<(), AppError>;

    /// Soft-deletes a conversation for the authenticated user. [S§7.5]
    async fn delete_conversation(
        &self,
        scope: &FamilyScope,
        conversation_id: Uuid,
    ) -> Result<(), AppError>;

    /// Reports a message for moderation review. [S§7.5, S§12.3]
    async fn report_message(
        &self,
        scope: &FamilyScope,
        message_id: Uuid,
        cmd: ReportMessageCommand,
    ) -> Result<(), AppError>;

    // ─── Group Commands ──────────────────────────────────────────────────

    /// Joins a group or submits join request based on join_policy. [S§7.6]
    async fn join_group(
        &self,
        scope: &FamilyScope,
        group_id: Uuid,
    ) -> Result<GroupMemberResponse, AppError>;

    /// Leaves a group. [S§7.6]
    async fn leave_group(
        &self,
        scope: &FamilyScope,
        group_id: Uuid,
    ) -> Result<(), AppError>;

    /// Creates a user-created group. (Phase 2) [S§7.6]
    async fn create_group(
        &self,
        scope: &FamilyScope,
        cmd: CreateGroupCommand,
    ) -> Result<GroupDetailResponse, AppError>;

    /// Updates group settings. Moderator/owner only. (Phase 2) [S§7.6]
    async fn update_group(
        &self,
        scope: &FamilyScope,
        group_id: Uuid,
        cmd: UpdateGroupCommand,
    ) -> Result<GroupDetailResponse, AppError>;

    /// Deletes a user-created group. Owner only. (Phase 2) [S§7.6]
    async fn delete_group(
        &self,
        scope: &FamilyScope,
        group_id: Uuid,
    ) -> Result<(), AppError>;

    /// Approves a pending join request. Moderator/owner only. (Phase 2)
    async fn approve_member(
        &self,
        scope: &FamilyScope,
        group_id: Uuid,
        family_id: Uuid,
    ) -> Result<GroupMemberResponse, AppError>;

    /// Rejects a pending join request. Moderator/owner only. (Phase 2)
    async fn reject_member(
        &self,
        scope: &FamilyScope,
        group_id: Uuid,
        family_id: Uuid,
    ) -> Result<(), AppError>;

    /// Bans a member. Moderator/owner only. (Phase 2)
    async fn ban_member(
        &self,
        scope: &FamilyScope,
        group_id: Uuid,
        family_id: Uuid,
    ) -> Result<(), AppError>;

    /// Promotes a member to moderator. Owner only. (Phase 2)
    async fn promote_member(
        &self,
        scope: &FamilyScope,
        group_id: Uuid,
        family_id: Uuid,
    ) -> Result<GroupMemberResponse, AppError>;

    /// Invites a family to an invite-only group. Moderator/owner only. (Phase 2)
    async fn invite_to_group(
        &self,
        scope: &FamilyScope,
        group_id: Uuid,
        cmd: InviteToGroupCommand,
    ) -> Result<GroupMemberResponse, AppError>;

    // ─── Event Commands ──────────────────────────────────────────────────

    /// Creates an event. [S§7.7]
    async fn create_event(
        &self,
        scope: &FamilyScope,
        cmd: CreateEventCommand,
    ) -> Result<EventResponse, AppError>;

    /// Updates an event. Creator only. [S§7.7]
    async fn update_event(
        &self,
        scope: &FamilyScope,
        event_id: Uuid,
        cmd: UpdateEventCommand,
    ) -> Result<EventResponse, AppError>;

    /// Cancels an event. Creator only. Notifies attendees. [S§7.7]
    async fn cancel_event(
        &self,
        scope: &FamilyScope,
        event_id: Uuid,
    ) -> Result<EventResponse, AppError>;

    /// RSVPs to an event. [S§7.7]
    async fn rsvp_event(
        &self,
        scope: &FamilyScope,
        event_id: Uuid,
        cmd: RsvpCommand,
    ) -> Result<RsvpResponse, AppError>;

    /// Removes RSVP from an event. [S§7.7]
    async fn remove_rsvp(
        &self,
        scope: &FamilyScope,
        event_id: Uuid,
    ) -> Result<(), AppError>;

    // ─── Event Handlers ──────────────────────────────────────────────────

    /// Handles FamilyCreated event — creates social profile.
    async fn handle_family_created(
        &self,
        family_id: FamilyId,
    ) -> Result<(), AppError>;

    /// Handles CoParentRemoved event — disassociates posts from family.
    async fn handle_co_parent_removed(
        &self,
        family_id: FamilyId,
        parent_id: Uuid,
    ) -> Result<(), AppError>;

    /// Handles MilestoneAchieved event — creates optional milestone post.
    async fn handle_milestone_achieved(
        &self,
        family_id: FamilyId,
        milestone: MilestoneData,
    ) -> Result<(), AppError>;

    /// Handles FamilyDeletionScheduled event — prepares for cascade.
    async fn handle_family_deletion_scheduled(
        &self,
        family_id: FamilyId,
    ) -> Result<(), AppError>;

    // ═══ QUERY SIDE (reads, no side effects) ════════════════════════════

    // ─── Profile Queries ─────────────────────────────────────────────────

    /// Returns the authenticated family's profile. [S§7.1]
    async fn get_own_profile(
        &self,
        scope: &FamilyScope,
    ) -> Result<ProfileResponse, AppError>;

    /// Returns another family's profile, filtered by privacy settings. [S§7.1]
    /// CROSS-FAMILY read.
    async fn get_family_profile(
        &self,
        scope: &FamilyScope,
        target_family_id: Uuid,
    ) -> Result<ProfileResponse, AppError>;

    // ─── Friend Queries ──────────────────────────────────────────────────

    /// Lists the authenticated family's friends.
    async fn list_friends(
        &self,
        scope: &FamilyScope,
        pagination: PaginationParams,
    ) -> Result<PaginatedResponse<FriendResponse>, AppError>;

    /// Lists incoming friend requests.
    async fn list_incoming_requests(
        &self,
        scope: &FamilyScope,
    ) -> Result<Vec<FriendRequestResponse>, AppError>;

    /// Lists outgoing friend requests.
    async fn list_outgoing_requests(
        &self,
        scope: &FamilyScope,
    ) -> Result<Vec<FriendRequestResponse>, AppError>;

    /// Lists blocked families.
    async fn list_blocks(
        &self,
        scope: &FamilyScope,
    ) -> Result<Vec<BlockedFamilyResponse>, AppError>;

    // ─── Feed Queries ────────────────────────────────────────────────────

    /// Returns the authenticated family's timeline feed. [S§7.2.3]
    async fn get_feed(
        &self,
        scope: &FamilyScope,
        pagination: CursorPaginationParams,
    ) -> Result<PaginatedResponse<PostResponse>, AppError>;

    /// Returns a single post with comments. Visibility-checked.
    async fn get_post(
        &self,
        scope: &FamilyScope,
        post_id: Uuid,
    ) -> Result<PostDetailResponse, AppError>;

    // ─── Messaging Queries ───────────────────────────────────────────────

    /// Lists the authenticated parent's conversations. [S§7.5]
    async fn list_conversations(
        &self,
        scope: &FamilyScope,
        pagination: CursorPaginationParams,
    ) -> Result<PaginatedResponse<ConversationSummaryResponse>, AppError>;

    /// Returns messages in a conversation. [S§7.5]
    async fn get_conversation_messages(
        &self,
        scope: &FamilyScope,
        conversation_id: Uuid,
        pagination: CursorPaginationParams,
    ) -> Result<PaginatedResponse<MessageResponse>, AppError>;

    // ─── Group Queries ───────────────────────────────────────────────────

    /// Lists groups the authenticated family is a member of. [S§7.6]
    async fn list_my_groups(
        &self,
        scope: &FamilyScope,
    ) -> Result<Vec<GroupSummaryResponse>, AppError>;

    /// Lists platform-managed methodology groups. [S§7.6]
    async fn list_platform_groups(
        &self,
        scope: &FamilyScope,
    ) -> Result<Vec<GroupSummaryResponse>, AppError>;

    /// Returns group details. Membership check for user-created groups. [S§7.6]
    async fn get_group(
        &self,
        scope: &FamilyScope,
        group_id: Uuid,
    ) -> Result<GroupDetailResponse, AppError>;

    /// Lists posts in a group. Requires membership. [S§7.6]
    async fn list_group_posts(
        &self,
        scope: &FamilyScope,
        group_id: Uuid,
        pagination: CursorPaginationParams,
    ) -> Result<PaginatedResponse<PostResponse>, AppError>;

    /// Lists group members. (Phase 2)
    async fn list_group_members(
        &self,
        scope: &FamilyScope,
        group_id: Uuid,
    ) -> Result<Vec<GroupMemberResponse>, AppError>;

    // ─── Event Queries ───────────────────────────────────────────────────

    /// Lists events visible to the authenticated family. [S§7.7]
    async fn list_events(
        &self,
        scope: &FamilyScope,
        filter: EventFilter,
        pagination: CursorPaginationParams,
    ) -> Result<PaginatedResponse<EventSummaryResponse>, AppError>;

    /// Returns event details with RSVP list. [S§7.7]
    async fn get_event(
        &self,
        scope: &FamilyScope,
        event_id: Uuid,
    ) -> Result<EventDetailResponse, AppError>;

    // ─── Discovery Queries (Phase 2) ─────────────────────────────────────

    /// Discovers nearby families with location sharing enabled. [S§7.8]
    async fn discover_families(
        &self,
        scope: &FamilyScope,
        query: DiscoverFamiliesQuery,
    ) -> Result<Vec<DiscoverableFamilyResponse>, AppError>;

    /// Discovers events by location/methodology. [S§7.7]
    async fn discover_events(
        &self,
        scope: &FamilyScope,
        query: DiscoverEventsQuery,
    ) -> Result<Vec<EventSummaryResponse>, AppError>;

    /// Discovers groups by methodology. [S§7.6]
    async fn discover_groups(
        &self,
        scope: &FamilyScope,
        query: DiscoverGroupsQuery,
    ) -> Result<Vec<GroupSummaryResponse>, AppError>;
}
```

**Implementation**: `SocialServiceImpl` in `src/social/service.rs`. Constructor receives:
- `Arc<dyn ProfileRepository>`
- `Arc<dyn FriendshipRepository>`
- `Arc<dyn BlockRepository>`
- `Arc<dyn PostRepository>`
- `Arc<dyn CommentRepository>`
- `Arc<dyn PostLikeRepository>`
- `Arc<dyn ConversationRepository>`
- `Arc<dyn MessageRepository>`
- `Arc<dyn GroupRepository>`
- `Arc<dyn GroupMemberRepository>`
- `Arc<dyn EventRepository>`
- `Arc<dyn EventRsvpRepository>`
- `Arc<dyn IamService>` (for family/parent data lookup)
- `Arc<RedisPool>` (for feed fan-out, caching, WebSocket pub/sub)
- `Arc<EventBus>`

---

## §6 Repository Interfaces

Defined in `src/social/ports.rs`. Social repositories are unique in that several methods
perform **cross-family reads** — friendships, conversations, and feed queries by design
access data from multiple families. These methods are explicitly marked with
`// CROSS-FAMILY:` comments. `[CODING §2.4, CODING §8.2]`

```rust
// src/social/ports.rs (continued)

#[async_trait]
pub trait ProfileRepository: Send + Sync {
    /// Creates a social profile. Called during FamilyCreated handling.
    async fn create(
        &self,
        family_id: FamilyId,
    ) -> Result<Profile, AppError>;

    /// Finds profile by family_id. Family-scoped for own profile.
    async fn find_by_family(
        &self,
        scope: &FamilyScope,
    ) -> Result<Option<Profile>, AppError>;

    /// Finds profile by family_id without family scope.
    /// CROSS-FAMILY: Used to view another family's profile (visibility filtered in service).
    async fn find_by_family_id(
        &self,
        family_id: Uuid,
    ) -> Result<Option<Profile>, AppError>;

    /// Updates a profile.
    async fn update(
        &self,
        scope: &FamilyScope,
        profile: &Profile,
    ) -> Result<Profile, AppError>;
}

#[async_trait]
pub trait FriendshipRepository: Send + Sync {
    /// Creates a friendship record (status = pending).
    async fn create(
        &self,
        requester_family_id: Uuid,
        accepter_family_id: Uuid,
    ) -> Result<Friendship, AppError>;

    /// Finds a friendship between two families (either direction).
    /// CROSS-FAMILY: Friendship involves two families.
    async fn find_between(
        &self,
        family_a: Uuid,
        family_b: Uuid,
    ) -> Result<Option<Friendship>, AppError>;

    /// Finds a friendship by ID.
    /// CROSS-FAMILY: Either participant can access.
    async fn find_by_id(
        &self,
        friendship_id: Uuid,
    ) -> Result<Option<Friendship>, AppError>;

    /// Updates friendship status.
    async fn update_status(
        &self,
        friendship_id: Uuid,
        status: &str,
    ) -> Result<Friendship, AppError>;

    /// Deletes a friendship (unfriend or pre-block cleanup).
    async fn delete_between(
        &self,
        family_a: Uuid,
        family_b: Uuid,
    ) -> Result<(), AppError>;

    /// Lists accepted friends for a family, paginated.
    async fn list_friends(
        &self,
        scope: &FamilyScope,
        pagination: &PaginationParams,
    ) -> Result<Vec<Friendship>, AppError>;

    /// Lists incoming pending requests.
    async fn list_incoming_pending(
        &self,
        scope: &FamilyScope,
    ) -> Result<Vec<Friendship>, AppError>;

    /// Lists outgoing pending requests.
    async fn list_outgoing_pending(
        &self,
        scope: &FamilyScope,
    ) -> Result<Vec<Friendship>, AppError>;

    /// Returns all accepted friend family_ids for a family.
    /// Used by feed fan-out and visibility checks.
    async fn list_friend_family_ids(
        &self,
        family_id: Uuid,
    ) -> Result<Vec<Uuid>, AppError>;
}

#[async_trait]
pub trait BlockRepository: Send + Sync {
    /// Creates a block record.
    async fn create(
        &self,
        blocker_family_id: Uuid,
        blocked_family_id: Uuid,
    ) -> Result<Block, AppError>;

    /// Checks if family_a has blocked family_b. O(1) lookup.
    async fn is_blocked(
        &self,
        blocker_family_id: Uuid,
        blocked_family_id: Uuid,
    ) -> Result<bool, AppError>;

    /// Checks if either family has blocked the other (bidirectional check).
    /// CROSS-FAMILY: Used for visibility checks.
    async fn is_either_blocked(
        &self,
        family_a: Uuid,
        family_b: Uuid,
    ) -> Result<bool, AppError>;

    /// Deletes a block record (unblock).
    async fn delete(
        &self,
        blocker_family_id: Uuid,
        blocked_family_id: Uuid,
    ) -> Result<(), AppError>;

    /// Lists all families blocked by a family.
    async fn list_blocked_by(
        &self,
        scope: &FamilyScope,
    ) -> Result<Vec<Block>, AppError>;

    /// Returns all family_ids blocked by or blocking a family.
    /// Used by feed filtering.
    async fn list_all_blocked_family_ids(
        &self,
        family_id: Uuid,
    ) -> Result<Vec<Uuid>, AppError>;
}

#[async_trait]
pub trait PostRepository: Send + Sync {
    /// Creates a post.
    async fn create(
        &self,
        scope: &FamilyScope,
        post: &CreatePostRecord,
    ) -> Result<Post, AppError>;

    /// Finds a post by ID.
    /// CROSS-FAMILY: Post may belong to another family (visibility checked in service).
    async fn find_by_id(
        &self,
        post_id: Uuid,
    ) -> Result<Option<Post>, AppError>;

    /// Deletes a post. Family-scoped (author's family only).
    async fn delete(
        &self,
        scope: &FamilyScope,
        post_id: Uuid,
    ) -> Result<(), AppError>;

    /// Updates a post (editing). Family-scoped.
    async fn update(
        &self,
        scope: &FamilyScope,
        post_id: Uuid,
        content: &str,
        attachments: &serde_json::Value,
    ) -> Result<Post, AppError>;

    /// Increments likes_count.
    async fn increment_likes(
        &self,
        post_id: Uuid,
    ) -> Result<(), AppError>;

    /// Decrements likes_count.
    async fn decrement_likes(
        &self,
        post_id: Uuid,
    ) -> Result<(), AppError>;

    /// Increments comments_count.
    async fn increment_comments(
        &self,
        post_id: Uuid,
    ) -> Result<(), AppError>;

    /// Decrements comments_count.
    async fn decrement_comments(
        &self,
        post_id: Uuid,
    ) -> Result<(), AppError>;

    /// Lists posts by family_ids (for feed hydration from Redis post IDs).
    /// CROSS-FAMILY: Feed contains posts from multiple families.
    async fn find_by_ids(
        &self,
        post_ids: &[Uuid],
    ) -> Result<Vec<Post>, AppError>;

    /// Lists posts by group, paginated.
    async fn list_by_group(
        &self,
        group_id: Uuid,
        pagination: &CursorPaginationParams,
    ) -> Result<Vec<Post>, AppError>;

    /// Fallback feed query: recent posts from friend family_ids.
    /// CROSS-FAMILY: Reads posts from multiple friend families.
    async fn list_friends_posts(
        &self,
        friend_family_ids: &[Uuid],
        pagination: &CursorPaginationParams,
    ) -> Result<Vec<Post>, AppError>;
}

#[async_trait]
pub trait CommentRepository: Send + Sync {
    /// Creates a comment.
    async fn create(
        &self,
        scope: &FamilyScope,
        comment: &CreateCommentRecord,
    ) -> Result<Comment, AppError>;

    /// Finds a comment by ID.
    async fn find_by_id(
        &self,
        comment_id: Uuid,
    ) -> Result<Option<Comment>, AppError>;

    /// Lists comments for a post, ordered by created_at.
    /// CROSS-FAMILY: Comments come from multiple families.
    async fn list_by_post(
        &self,
        post_id: Uuid,
    ) -> Result<Vec<Comment>, AppError>;

    /// Deletes a comment.
    async fn delete(
        &self,
        comment_id: Uuid,
    ) -> Result<(), AppError>;
}

#[async_trait]
pub trait PostLikeRepository: Send + Sync {
    /// Creates a like (idempotent via UNIQUE constraint).
    async fn create(
        &self,
        scope: &FamilyScope,
        post_id: Uuid,
    ) -> Result<(), AppError>;

    /// Deletes a like.
    async fn delete(
        &self,
        scope: &FamilyScope,
        post_id: Uuid,
    ) -> Result<(), AppError>;

    /// Checks if a family has liked a post.
    async fn exists(
        &self,
        family_id: Uuid,
        post_id: Uuid,
    ) -> Result<bool, AppError>;
}

#[async_trait]
pub trait ConversationRepository: Send + Sync {
    /// Creates a conversation with two participants.
    async fn create_with_participants(
        &self,
        participant_a_parent_id: Uuid,
        participant_a_family_id: Uuid,
        participant_b_parent_id: Uuid,
        participant_b_family_id: Uuid,
    ) -> Result<Conversation, AppError>;

    /// Finds an existing conversation between two parents.
    /// CROSS-FAMILY: Conversations span two families.
    async fn find_between_parents(
        &self,
        parent_a: Uuid,
        parent_b: Uuid,
    ) -> Result<Option<Conversation>, AppError>;

    /// Lists conversations for a parent (excludes soft-deleted).
    async fn list_by_parent(
        &self,
        parent_id: Uuid,
        pagination: &CursorPaginationParams,
    ) -> Result<Vec<ConversationWithParticipants>, AppError>;

    /// Marks conversation read for a participant.
    async fn mark_read(
        &self,
        conversation_id: Uuid,
        parent_id: Uuid,
    ) -> Result<(), AppError>;

    /// Soft-deletes a conversation for a participant.
    async fn soft_delete_for_participant(
        &self,
        conversation_id: Uuid,
        parent_id: Uuid,
    ) -> Result<(), AppError>;

    /// Clears deleted_at for a participant (new message restores conversation).
    async fn restore_for_participant(
        &self,
        conversation_id: Uuid,
        parent_id: Uuid,
    ) -> Result<(), AppError>;

    /// Checks if a parent is a participant in a conversation.
    async fn is_participant(
        &self,
        conversation_id: Uuid,
        parent_id: Uuid,
    ) -> Result<bool, AppError>;
}

#[async_trait]
pub trait MessageRepository: Send + Sync {
    /// Creates a message.
    async fn create(
        &self,
        message: &CreateMessageRecord,
    ) -> Result<Message, AppError>;

    /// Lists messages in a conversation, respecting participant's deleted_at.
    /// CROSS-FAMILY: Messages from both participants are returned.
    async fn list_by_conversation(
        &self,
        conversation_id: Uuid,
        participant_deleted_at: Option<DateTime<Utc>>,
        pagination: &CursorPaginationParams,
    ) -> Result<Vec<Message>, AppError>;

    /// Finds a message by ID.
    async fn find_by_id(
        &self,
        message_id: Uuid,
    ) -> Result<Option<Message>, AppError>;
}

#[async_trait]
pub trait GroupRepository: Send + Sync {
    /// Creates a group.
    async fn create(
        &self,
        group: &CreateGroupRecord,
    ) -> Result<Group, AppError>;

    /// Finds a group by ID.
    async fn find_by_id(
        &self,
        group_id: Uuid,
    ) -> Result<Option<Group>, AppError>;

    /// Updates a group.
    async fn update(
        &self,
        group_id: Uuid,
        group: &UpdateGroupRecord,
    ) -> Result<Group, AppError>;

    /// Deletes a group.
    async fn delete(
        &self,
        group_id: Uuid,
    ) -> Result<(), AppError>;

    /// Lists platform-managed groups.
    async fn list_platform_groups(
    ) -> Result<Vec<Group>, AppError>;

    /// Increments member_count.
    async fn increment_member_count(
        &self,
        group_id: Uuid,
    ) -> Result<(), AppError>;

    /// Decrements member_count.
    async fn decrement_member_count(
        &self,
        group_id: Uuid,
    ) -> Result<(), AppError>;

    /// Lists groups by methodology_id. Used for discovery.
    async fn list_by_methodology(
        &self,
        methodology_id: Uuid,
    ) -> Result<Vec<Group>, AppError>;
}

#[async_trait]
pub trait GroupMemberRepository: Send + Sync {
    /// Creates a group membership record.
    async fn create(
        &self,
        membership: &CreateGroupMemberRecord,
    ) -> Result<GroupMember, AppError>;

    /// Finds membership by group and family.
    async fn find_by_group_and_family(
        &self,
        group_id: Uuid,
        family_id: Uuid,
    ) -> Result<Option<GroupMember>, AppError>;

    /// Updates membership status or role.
    async fn update(
        &self,
        membership_id: Uuid,
        status: Option<&str>,
        role: Option<&str>,
    ) -> Result<GroupMember, AppError>;

    /// Deletes a membership record (leave or reject).
    async fn delete(
        &self,
        group_id: Uuid,
        family_id: Uuid,
    ) -> Result<(), AppError>;

    /// Lists active members of a group.
    async fn list_active_by_group(
        &self,
        group_id: Uuid,
    ) -> Result<Vec<GroupMember>, AppError>;

    /// Lists groups a family is an active member of.
    async fn list_groups_for_family(
        &self,
        scope: &FamilyScope,
    ) -> Result<Vec<Uuid>, AppError>;

    /// Checks if a family is an active member of a group.
    async fn is_active_member(
        &self,
        group_id: Uuid,
        family_id: Uuid,
    ) -> Result<bool, AppError>;

    /// Checks if a family is a moderator or owner of a group.
    async fn is_moderator_or_owner(
        &self,
        group_id: Uuid,
        family_id: Uuid,
    ) -> Result<bool, AppError>;
}

#[async_trait]
pub trait EventRepository: Send + Sync {
    /// Creates an event.
    async fn create(
        &self,
        event: &CreateEventRecord,
    ) -> Result<Event, AppError>;

    /// Finds an event by ID.
    async fn find_by_id(
        &self,
        event_id: Uuid,
    ) -> Result<Option<Event>, AppError>;

    /// Updates an event.
    async fn update(
        &self,
        event_id: Uuid,
        event: &UpdateEventRecord,
    ) -> Result<Event, AppError>;

    /// Updates event status (e.g., cancel).
    async fn update_status(
        &self,
        event_id: Uuid,
        status: &str,
    ) -> Result<Event, AppError>;

    /// Lists events visible to a family (own, friends', group, discoverable).
    /// CROSS-FAMILY: Events from friends and discoverable events.
    async fn list_visible(
        &self,
        family_id: Uuid,
        friend_family_ids: &[Uuid],
        group_ids: &[Uuid],
        filter: &EventFilter,
        pagination: &CursorPaginationParams,
    ) -> Result<Vec<Event>, AppError>;

    /// Lists discoverable events by location/methodology.
    async fn list_discoverable(
        &self,
        methodology_id: Option<Uuid>,
        location_region: Option<&str>,
    ) -> Result<Vec<Event>, AppError>;

    /// Increments attendee_count.
    async fn increment_attendee_count(
        &self,
        event_id: Uuid,
    ) -> Result<(), AppError>;

    /// Decrements attendee_count.
    async fn decrement_attendee_count(
        &self,
        event_id: Uuid,
    ) -> Result<(), AppError>;
}

#[async_trait]
pub trait EventRsvpRepository: Send + Sync {
    /// Creates or updates an RSVP.
    async fn upsert(
        &self,
        scope: &FamilyScope,
        event_id: Uuid,
        status: &str,
    ) -> Result<EventRsvp, AppError>;

    /// Deletes an RSVP.
    async fn delete(
        &self,
        scope: &FamilyScope,
        event_id: Uuid,
    ) -> Result<(), AppError>;

    /// Lists RSVPs for an event.
    /// CROSS-FAMILY: RSVPs come from multiple families.
    async fn list_by_event(
        &self,
        event_id: Uuid,
    ) -> Result<Vec<EventRsvp>, AppError>;

    /// Counts "going" RSVPs for capacity checking.
    async fn count_going(
        &self,
        event_id: Uuid,
    ) -> Result<i64, AppError>;

    /// Lists family_ids with "going" RSVPs (for cancellation notification).
    async fn list_going_family_ids(
        &self,
        event_id: Uuid,
    ) -> Result<Vec<Uuid>, AppError>;
}
```

---

## §7 Adapter Interfaces

None. The social domain has no external third-party service dependencies. Redis is
shared infrastructure (`src/shared/redis.rs`). WebSocket is Axum built-in
(`tokio-tungstenite`). Content scanning and notification delivery are handled via
domain events to `safety::` and `notify::` respectively. `[CODING §8.1]`

---

## §8 Models (DTOs)

All types defined in `src/social/models.rs`. API-facing types derive `serde::Serialize`,
`serde::Deserialize`, and `utoipa::ToSchema`. Request types additionally derive
`validator::Validate`. `[CODING §2.3]`

### §8.1 Request Types

```rust
// src/social/models.rs

/// Profile update. [S§7.1]
#[derive(Debug, Deserialize, Validate, ToSchema)]
pub struct UpdateProfileRequest {
    #[validate(length(max = 2000))]
    pub bio: Option<String>,
    pub profile_photo_url: Option<String>,
    /// Per-field privacy settings. Values: "friends" | "hidden". [S§7.1]
    pub privacy_settings: Option<serde_json::Value>,
    pub location_visible: Option<bool>,
}

/// Send friend request. [S§7.4]
#[derive(Debug, Deserialize, ToSchema)]
pub struct SendFriendRequestCommand {
    pub target_family_id: Uuid,
}

/// Block a family. [S§7.4]
#[derive(Debug, Deserialize, ToSchema)]
pub struct BlockFamilyCommand {
    pub target_family_id: Uuid,
}

/// Create a post. [S§7.2]
#[derive(Debug, Deserialize, Validate, ToSchema)]
pub struct CreatePostCommand {
    pub post_type: String,      // validated against allowed values in service
    #[validate(length(max = 10000))]
    pub content: Option<String>,
    #[serde(default)]
    pub attachments: Vec<AttachmentInput>,
    pub group_id: Option<Uuid>,
}

/// Update a post (Phase 2). [S§7.2]
#[derive(Debug, Deserialize, Validate, ToSchema)]
pub struct UpdatePostCommand {
    #[validate(length(max = 10000))]
    pub content: Option<String>,
    pub attachments: Option<Vec<AttachmentInput>>,
}

/// Attachment input for posts and messages.
#[derive(Debug, Clone, Deserialize, Serialize, ToSchema)]
pub struct AttachmentInput {
    pub url: String,
    pub content_type: String,    // MIME type
    pub filename: Option<String>,
}

/// Create a comment. [S§7.3]
#[derive(Debug, Deserialize, Validate, ToSchema)]
pub struct CreateCommentCommand {
    #[validate(length(min = 1, max = 5000))]
    pub content: String,
    pub parent_comment_id: Option<Uuid>,
}

/// Create or get a conversation. [S§7.5]
#[derive(Debug, Deserialize, ToSchema)]
pub struct CreateConversationCommand {
    pub recipient_parent_id: Uuid,
}

/// Send a message. [S§7.5]
#[derive(Debug, Deserialize, Validate, ToSchema)]
pub struct SendMessageCommand {
    #[validate(length(min = 1, max = 10000))]
    pub content: String,
    #[serde(default)]
    pub attachments: Vec<AttachmentInput>,
}

/// Report a message. [S§12.3]
#[derive(Debug, Deserialize, Validate, ToSchema)]
pub struct ReportMessageCommand {
    #[validate(length(min = 1, max = 2000))]
    pub reason: String,
}

/// Create a group (Phase 2). [S§7.6]
#[derive(Debug, Deserialize, Validate, ToSchema)]
pub struct CreateGroupCommand {
    #[validate(length(min = 1, max = 200))]
    pub name: String,
    #[validate(length(max = 2000))]
    pub description: Option<String>,
    pub join_policy: Option<String>,   // defaults to "open"
    pub methodology_id: Option<Uuid>,
}

/// Update a group (Phase 2). [S§7.6]
#[derive(Debug, Deserialize, Validate, ToSchema)]
pub struct UpdateGroupCommand {
    #[validate(length(min = 1, max = 200))]
    pub name: Option<String>,
    #[validate(length(max = 2000))]
    pub description: Option<String>,
    pub join_policy: Option<String>,
    pub cover_photo_url: Option<String>,
}

/// Invite to group (Phase 2). [S§7.6]
#[derive(Debug, Deserialize, ToSchema)]
pub struct InviteToGroupCommand {
    pub family_id: Uuid,
}

/// Create an event. [S§7.7]
#[derive(Debug, Deserialize, Validate, ToSchema)]
pub struct CreateEventCommand {
    #[validate(length(min = 1, max = 200))]
    pub title: String,
    #[validate(length(max = 5000))]
    pub description: Option<String>,
    pub event_date: DateTime<Utc>,
    pub end_date: Option<DateTime<Utc>>,
    #[validate(length(max = 500))]
    pub location_name: Option<String>,
    #[validate(length(max = 200))]
    pub location_region: Option<String>,
    pub is_virtual: bool,
    pub virtual_url: Option<String>,
    pub capacity: Option<i32>,
    pub visibility: String,           // "friends" | "group" | "discoverable"
    pub group_id: Option<Uuid>,
    pub methodology_id: Option<Uuid>,
}

/// Update an event. [S§7.7]
#[derive(Debug, Deserialize, Validate, ToSchema)]
pub struct UpdateEventCommand {
    #[validate(length(min = 1, max = 200))]
    pub title: Option<String>,
    #[validate(length(max = 5000))]
    pub description: Option<String>,
    pub event_date: Option<DateTime<Utc>>,
    pub end_date: Option<DateTime<Utc>>,
    pub location_name: Option<String>,
    pub location_region: Option<String>,
    pub is_virtual: Option<bool>,
    pub virtual_url: Option<String>,
    pub capacity: Option<i32>,
}

/// RSVP to an event. [S§7.7]
#[derive(Debug, Deserialize, ToSchema)]
pub struct RsvpCommand {
    pub status: String,               // "going" | "interested" | "not_going"
}
```

### §8.2 Response Types

```rust
/// Social profile response. [S§7.1]
#[derive(Debug, Serialize, ToSchema)]
pub struct ProfileResponse {
    pub family_id: Uuid,
    pub display_name: Option<String>,    // from iam_families; hidden if privacy = "hidden"
    pub parent_names: Option<Vec<String>>, // from iam_parents; hidden if privacy = "hidden"
    pub children: Option<Vec<ProfileChildResponse>>, // hidden if privacy = "hidden"
    pub methodology_names: Option<Vec<String>>,      // hidden if privacy = "hidden"
    pub location_region: Option<String>, // hidden if privacy = "hidden" or not location_visible
    pub bio: Option<String>,
    pub profile_photo_url: Option<String>,
    pub privacy_settings: Option<serde_json::Value>, // only included for own profile
    pub location_visible: Option<bool>,              // only included for own profile
    pub is_friend: bool,
    pub friendship_status: Option<String>,           // "pending" | "accepted" | null
}

/// Child info in profile (generic avatar default). [S§7.1]
#[derive(Debug, Serialize, ToSchema)]
pub struct ProfileChildResponse {
    pub display_name: String,
    pub age: Option<i16>,            // computed from birth_year
    pub avatar_url: Option<String>,  // defaults to generic avatar
}

/// Friendship response.
#[derive(Debug, Serialize, ToSchema)]
pub struct FriendshipResponse {
    pub id: Uuid,
    pub requester_family_id: Uuid,
    pub accepter_family_id: Uuid,
    pub status: String,
    pub created_at: DateTime<Utc>,
}

/// Friend list item.
#[derive(Debug, Serialize, ToSchema)]
pub struct FriendResponse {
    pub family_id: Uuid,
    pub display_name: String,
    pub profile_photo_url: Option<String>,
    pub methodology_names: Vec<String>,
    pub friends_since: DateTime<Utc>,
}

/// Incoming/outgoing friend request.
#[derive(Debug, Serialize, ToSchema)]
pub struct FriendRequestResponse {
    pub friendship_id: Uuid,
    pub family_id: Uuid,         // the other family
    pub display_name: String,
    pub profile_photo_url: Option<String>,
    pub created_at: DateTime<Utc>,
}

/// Blocked family.
#[derive(Debug, Serialize, ToSchema)]
pub struct BlockedFamilyResponse {
    pub family_id: Uuid,
    pub display_name: String,
    pub blocked_at: DateTime<Utc>,
}

/// Post response. [S§7.2]
#[derive(Debug, Serialize, ToSchema)]
pub struct PostResponse {
    pub id: Uuid,
    pub family_id: Uuid,
    pub author_name: String,
    pub author_photo_url: Option<String>,
    pub post_type: String,
    pub content: Option<String>,
    pub attachments: Vec<AttachmentInput>,
    pub group_id: Option<Uuid>,
    pub group_name: Option<String>,
    pub visibility: String,
    pub likes_count: i32,
    pub comments_count: i32,
    pub is_liked_by_me: bool,
    pub is_edited: bool,
    pub created_at: DateTime<Utc>,
}

/// Post detail with comments. [S§7.2, S§7.3]
#[derive(Debug, Serialize, ToSchema)]
pub struct PostDetailResponse {
    #[serde(flatten)]
    pub post: PostResponse,
    pub comments: Vec<CommentResponse>,
}

/// Comment response. [S§7.3]
#[derive(Debug, Serialize, ToSchema)]
pub struct CommentResponse {
    pub id: Uuid,
    pub post_id: Uuid,
    pub family_id: Uuid,
    pub author_name: String,
    pub author_photo_url: Option<String>,
    pub parent_comment_id: Option<Uuid>,
    pub content: String,
    pub created_at: DateTime<Utc>,
    pub replies: Vec<CommentResponse>,   // one level only
}

/// Conversation summary. [S§7.5]
#[derive(Debug, Serialize, ToSchema)]
pub struct ConversationSummaryResponse {
    pub id: Uuid,
    pub other_parent_name: String,
    pub other_parent_photo_url: Option<String>,
    pub other_family_id: Uuid,
    pub last_message_preview: Option<String>,
    pub last_message_at: Option<DateTime<Utc>>,
    pub unread_count: i64,
    pub updated_at: DateTime<Utc>,
}

/// Conversation response. [S§7.5]
#[derive(Debug, Serialize, ToSchema)]
pub struct ConversationResponse {
    pub id: Uuid,
    pub participants: Vec<ConversationParticipantResponse>,
    pub created_at: DateTime<Utc>,
}

/// Conversation participant.
#[derive(Debug, Serialize, ToSchema)]
pub struct ConversationParticipantResponse {
    pub parent_id: Uuid,
    pub family_id: Uuid,
    pub display_name: String,
    pub profile_photo_url: Option<String>,
}

/// Message response. [S§7.5]
#[derive(Debug, Serialize, ToSchema)]
pub struct MessageResponse {
    pub id: Uuid,
    pub conversation_id: Uuid,
    pub sender_parent_id: Uuid,
    pub sender_name: String,
    pub content: String,
    pub attachments: Vec<AttachmentInput>,
    pub created_at: DateTime<Utc>,
}

/// Group summary. [S§7.6]
#[derive(Debug, Serialize, ToSchema)]
pub struct GroupSummaryResponse {
    pub id: Uuid,
    pub group_type: String,
    pub name: String,
    pub description: Option<String>,
    pub cover_photo_url: Option<String>,
    pub methodology_name: Option<String>,
    pub member_count: i32,
    pub join_policy: String,
    pub is_member: bool,
}

/// Group detail. [S§7.6]
#[derive(Debug, Serialize, ToSchema)]
pub struct GroupDetailResponse {
    #[serde(flatten)]
    pub summary: GroupSummaryResponse,
    pub creator_family_id: Option<Uuid>,
    pub my_role: Option<String>,       // member/moderator/owner — null if not a member
    pub my_status: Option<String>,     // active/pending/invited — null if not a member
    pub created_at: DateTime<Utc>,
}

/// Group member response. [S§7.6]
#[derive(Debug, Serialize, ToSchema)]
pub struct GroupMemberResponse {
    pub family_id: Uuid,
    pub display_name: String,
    pub profile_photo_url: Option<String>,
    pub role: String,
    pub status: String,
    pub joined_at: Option<DateTime<Utc>>,
}

/// Event summary. [S§7.7]
#[derive(Debug, Serialize, ToSchema)]
pub struct EventSummaryResponse {
    pub id: Uuid,
    pub title: String,
    pub event_date: DateTime<Utc>,
    pub end_date: Option<DateTime<Utc>>,
    pub location_name: Option<String>,
    pub is_virtual: bool,
    pub creator_family_name: String,
    pub attendee_count: i32,
    pub capacity: Option<i32>,
    pub visibility: String,
    pub status: String,
    pub my_rsvp: Option<String>,
}

/// Event detail with RSVPs. [S§7.7]
#[derive(Debug, Serialize, ToSchema)]
pub struct EventDetailResponse {
    pub id: Uuid,
    pub title: String,
    pub description: Option<String>,
    pub event_date: DateTime<Utc>,
    pub end_date: Option<DateTime<Utc>>,
    pub location_name: Option<String>,
    pub location_region: Option<String>,
    pub is_virtual: bool,
    pub virtual_url: Option<String>,
    pub capacity: Option<i32>,
    pub visibility: String,
    pub status: String,
    pub creator_family_id: Uuid,
    pub creator_family_name: String,
    pub group_id: Option<Uuid>,
    pub group_name: Option<String>,
    pub methodology_name: Option<String>,
    pub attendee_count: i32,
    pub my_rsvp: Option<String>,
    pub rsvps: Vec<EventRsvpResponse>,
    pub created_at: DateTime<Utc>,
}

/// Event response (for create/update). [S§7.7]
#[derive(Debug, Serialize, ToSchema)]
pub struct EventResponse {
    pub id: Uuid,
    pub title: String,
    pub description: Option<String>,
    pub event_date: DateTime<Utc>,
    pub end_date: Option<DateTime<Utc>>,
    pub location_name: Option<String>,
    pub location_region: Option<String>,
    pub is_virtual: bool,
    pub virtual_url: Option<String>,
    pub capacity: Option<i32>,
    pub visibility: String,
    pub status: String,
    pub attendee_count: i32,
    pub created_at: DateTime<Utc>,
}

/// RSVP response. [S§7.7]
#[derive(Debug, Serialize, ToSchema)]
pub struct RsvpResponse {
    pub event_id: Uuid,
    pub family_id: Uuid,
    pub status: String,
    pub created_at: DateTime<Utc>,
}

/// Event RSVP in event detail.
#[derive(Debug, Serialize, ToSchema)]
pub struct EventRsvpResponse {
    pub family_id: Uuid,
    pub display_name: String,
    pub status: String,
}

/// Discoverable family (Phase 2). [S§7.8]
#[derive(Debug, Serialize, ToSchema)]
pub struct DiscoverableFamilyResponse {
    pub family_id: Uuid,
    pub display_name: String,
    pub profile_photo_url: Option<String>,
    pub methodology_names: Vec<String>,
    pub location_region: Option<String>,
}
```

### §8.3 Internal Types

```rust
/// Internal profile model (maps to soc_profiles row + iam data).
#[derive(Debug, Clone)]
pub struct Profile {
    pub family_id: Uuid,
    pub bio: Option<String>,
    pub profile_photo_url: Option<String>,
    pub privacy_settings: serde_json::Value,
    pub location_visible: bool,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

/// Internal friendship record.
#[derive(Debug, Clone)]
pub struct Friendship {
    pub id: Uuid,
    pub requester_family_id: Uuid,
    pub accepter_family_id: Uuid,
    pub status: String,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

/// Internal block record.
#[derive(Debug, Clone)]
pub struct Block {
    pub id: Uuid,
    pub blocker_family_id: Uuid,
    pub blocked_family_id: Uuid,
    pub created_at: DateTime<Utc>,
}

/// Internal post record.
#[derive(Debug, Clone)]
pub struct Post {
    pub id: Uuid,
    pub family_id: Uuid,
    pub author_parent_id: Uuid,
    pub post_type: String,
    pub content: Option<String>,
    pub attachments: serde_json::Value,
    pub group_id: Option<Uuid>,
    pub visibility: String,
    pub likes_count: i32,
    pub comments_count: i32,
    pub is_edited: bool,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

/// Internal comment record.
#[derive(Debug, Clone)]
pub struct Comment {
    pub id: Uuid,
    pub post_id: Uuid,
    pub family_id: Uuid,
    pub author_parent_id: Uuid,
    pub parent_comment_id: Option<Uuid>,
    pub content: String,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

/// Internal conversation with participant data.
#[derive(Debug, Clone)]
pub struct ConversationWithParticipants {
    pub id: Uuid,
    pub participants: Vec<ConversationParticipant>,
    pub last_message: Option<Message>,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

/// Internal conversation participant.
#[derive(Debug, Clone)]
pub struct ConversationParticipant {
    pub conversation_id: Uuid,
    pub parent_id: Uuid,
    pub family_id: Uuid,
    pub last_read_at: Option<DateTime<Utc>>,
    pub deleted_at: Option<DateTime<Utc>>,
}

/// Internal conversation record.
#[derive(Debug, Clone)]
pub struct Conversation {
    pub id: Uuid,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

/// Internal message record.
#[derive(Debug, Clone)]
pub struct Message {
    pub id: Uuid,
    pub conversation_id: Uuid,
    pub sender_parent_id: Uuid,
    pub sender_family_id: Uuid,
    pub content: String,
    pub attachments: serde_json::Value,
    pub created_at: DateTime<Utc>,
}

/// Internal group record.
#[derive(Debug, Clone)]
pub struct Group {
    pub id: Uuid,
    pub group_type: String,
    pub name: String,
    pub description: Option<String>,
    pub cover_photo_url: Option<String>,
    pub creator_family_id: Option<Uuid>,
    pub methodology_id: Option<Uuid>,
    pub join_policy: String,
    pub member_count: i32,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

/// Internal group member record.
#[derive(Debug, Clone)]
pub struct GroupMember {
    pub id: Uuid,
    pub group_id: Uuid,
    pub family_id: Uuid,
    pub role: String,
    pub status: String,
    pub joined_at: Option<DateTime<Utc>>,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

/// Internal event record.
#[derive(Debug, Clone)]
pub struct Event {
    pub id: Uuid,
    pub creator_family_id: Uuid,
    pub creator_parent_id: Uuid,
    pub group_id: Option<Uuid>,
    pub title: String,
    pub description: Option<String>,
    pub event_date: DateTime<Utc>,
    pub end_date: Option<DateTime<Utc>>,
    pub location_name: Option<String>,
    pub location_region: Option<String>,
    pub is_virtual: bool,
    pub virtual_url: Option<String>,
    pub capacity: Option<i32>,
    pub visibility: String,
    pub status: String,
    pub methodology_id: Option<Uuid>,
    pub attendee_count: i32,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

/// Internal event RSVP record.
#[derive(Debug, Clone)]
pub struct EventRsvp {
    pub id: Uuid,
    pub event_id: Uuid,
    pub family_id: Uuid,
    pub status: String,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

/// Event filter for list queries.
#[derive(Debug, Clone)]
pub struct EventFilter {
    pub upcoming_only: bool,
    pub past_only: bool,
}

/// Milestone data from learn:: domain events.
#[derive(Debug, Clone)]
pub struct MilestoneData {
    pub student_name: String,
    pub milestone_type: String,
    pub description: String,
}

/// WebSocket message frame.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WsMessage {
    pub msg_type: String,    // "new_message" | "typing" | "read_receipt" | "notification"
    pub payload: serde_json::Value,
}

/// Discovery query params.
#[derive(Debug, Clone)]
pub struct DiscoverFamiliesQuery {
    pub methodology_id: Option<Uuid>,
    pub radius_km: Option<f64>,
}

#[derive(Debug, Clone)]
pub struct DiscoverEventsQuery {
    pub methodology_id: Option<Uuid>,
    pub location_region: Option<String>,
}

#[derive(Debug, Clone)]
pub struct DiscoverGroupsQuery {
    pub methodology_id: Option<Uuid>,
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
2. **Application** (Rust): After hydrating posts from PostgreSQL, a block filter removes
   any posts from families in the viewer's block list (either direction).

### §9.5 Centralized Visibility Module

All visibility logic lives in `src/social/domain/visibility.rs` (see §20). This module
exports pure functions that take content, viewer context, and relationship data, and
return a boolean (visible/not-visible) or filtered content. Service methods delegate
all visibility decisions to this module rather than embedding checks inline.

---

## §10 Friendship State Machine (Domain Deep-Dive 2)

### §10.1 Aggregate Root

The `Friendship` aggregate root lives in `src/social/domain/friendship.rs`. It enforces
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
FanOutPostJob (sidekiq-rs)
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

Real-time message delivery uses Axum's built-in WebSocket support with Redis pub/sub
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

`SocialError` enum defined in `src/social/domain/errors.rs`. Maps to `AppError` via
`From<SocialError> for AppError` `[00-core §6.4]`. `[CODING §2.2, CODING §8.3]`

```rust
#[derive(Debug, thiserror::Error)]
pub enum SocialError {
    // ─── Profile ─────────────────────────────────────────────────────────
    #[error("profile not found")]
    ProfileNotFound,

    #[error("invalid privacy setting value: {0}")]
    InvalidPrivacySetting(String),

    // ─── Friendship ─────────────────────────────────────────────────────
    #[error("cannot send friend request to self")]
    CannotFriendSelf,

    #[error("already friends with this family")]
    AlreadyFriends,

    #[error("friend request already pending")]
    AlreadyPending,

    #[error("friendship not found")]
    FriendshipNotFound,

    #[error("not the accepter of this friend request")]
    NotAccepter,

    #[error("friendship is not pending")]
    NotPending,

    #[error("not friends with this family")]
    NotFriends,

    // ─── Block ───────────────────────────────────────────────────────────
    #[error("blocked by target family")]
    BlockedByTarget,

    #[error("target family is blocked")]
    TargetBlocked,

    #[error("cannot block self")]
    CannotBlockSelf,

    #[error("already blocked")]
    AlreadyBlocked,

    #[error("block not found")]
    BlockNotFound,

    // ─── Post ────────────────────────────────────────────────────────────
    #[error("post not found")]
    PostNotFound,

    #[error("content not visible")]
    ContentNotVisible,

    #[error("not the post author")]
    NotPostAuthor,

    #[error("invalid post type: {0}")]
    InvalidPostType(String),

    #[error("content required for text posts")]
    ContentRequired,

    #[error("attachments required for photo posts")]
    AttachmentsRequired,

    // ─── Comment ─────────────────────────────────────────────────────────
    #[error("comment not found")]
    CommentNotFound,

    #[error("not comment author or post author")]
    NotCommentAuthorOrPostAuthor,

    #[error("cannot reply to a reply (one level only)")]
    CannotReplyToReply,

    // ─── Messaging ───────────────────────────────────────────────────────
    #[error("conversation not found")]
    ConversationNotFound,

    #[error("not a conversation participant")]
    NotParticipant,

    #[error("message not found")]
    MessageNotFound,

    // ─── Group ───────────────────────────────────────────────────────────
    #[error("group not found")]
    GroupNotFound,

    #[error("already a group member")]
    AlreadyGroupMember,

    #[error("group is invite only")]
    InviteOnly,

    #[error("banned from group")]
    BannedFromGroup,

    #[error("not a group member")]
    NotGroupMember,

    #[error("owner cannot leave without transferring ownership")]
    OwnerCannotLeave,

    #[error("insufficient group permissions")]
    InsufficientGroupPermissions,

    #[error("cannot delete platform group")]
    CannotDeletePlatformGroup,

    // ─── Event ───────────────────────────────────────────────────────────
    #[error("event not found")]
    EventNotFound,

    #[error("not the event creator")]
    NotEventCreator,

    #[error("event is cancelled")]
    EventCancelled,

    #[error("event is at capacity")]
    EventAtCapacity,

    #[error("event date must be in the future")]
    EventDateInPast,

    #[error("group visibility requires group_id")]
    GroupVisibilityRequiresGroup,

    // ─── Infrastructure ──────────────────────────────────────────────────
    #[error("database error")]
    DatabaseError(#[from] sea_orm::DbErr),
}
```

### §16.1 Error-to-HTTP Mapping

| SocialError Variant | HTTP Status | Error Code | Notes |
|--------------------|-------------|------------|-------|
| `ProfileNotFound` | 404 | `profile_not_found` | |
| `InvalidPrivacySetting` | 422 | `invalid_privacy_setting` | |
| `CannotFriendSelf` | 422 | `cannot_friend_self` | |
| `AlreadyFriends` | 409 | `already_friends` | |
| `AlreadyPending` | 409 | `already_pending` | |
| `FriendshipNotFound` | 404 | `friendship_not_found` | |
| `NotAccepter` | 403 | `not_accepter` | |
| `NotPending` | 409 | `not_pending` | |
| `NotFriends` | 403 | `not_friends` | |
| `BlockedByTarget` | **404** | `not_found` | **Silent blocking** — maps to 404, not 403 |
| `TargetBlocked` | **404** | `not_found` | **Silent blocking** — maps to 404, not 403 |
| `CannotBlockSelf` | 422 | `cannot_block_self` | |
| `AlreadyBlocked` | 409 | `already_blocked` | |
| `BlockNotFound` | 404 | `block_not_found` | |
| `PostNotFound` | 404 | `post_not_found` | |
| `ContentNotVisible` | **404** | `not_found` | Maps to 404 to avoid revealing existence |
| `NotPostAuthor` | 403 | `not_post_author` | |
| `InvalidPostType` | 422 | `invalid_post_type` | |
| `ContentRequired` | 422 | `content_required` | |
| `AttachmentsRequired` | 422 | `attachments_required` | |
| `CommentNotFound` | 404 | `comment_not_found` | |
| `NotCommentAuthorOrPostAuthor` | 403 | `not_comment_author_or_post_author` | |
| `CannotReplyToReply` | 422 | `cannot_reply_to_reply` | |
| `ConversationNotFound` | 404 | `conversation_not_found` | |
| `NotParticipant` | 403 | `not_participant` | |
| `MessageNotFound` | 404 | `message_not_found` | |
| `GroupNotFound` | 404 | `group_not_found` | |
| `AlreadyGroupMember` | 409 | `already_group_member` | |
| `InviteOnly` | 403 | `invite_only` | |
| `BannedFromGroup` | 403 | `banned_from_group` | |
| `NotGroupMember` | 404 | `not_group_member` | |
| `OwnerCannotLeave` | 422 | `owner_cannot_leave` | |
| `InsufficientGroupPermissions` | 403 | `insufficient_group_permissions` | |
| `CannotDeletePlatformGroup` | 403 | `cannot_delete_platform_group` | |
| `EventNotFound` | 404 | `event_not_found` | |
| `NotEventCreator` | 403 | `not_event_creator` | |
| `EventCancelled` | 409 | `event_cancelled` | |
| `EventAtCapacity` | 422 | `event_at_capacity` | |
| `EventDateInPast` | 422 | `event_date_in_past` | |
| `GroupVisibilityRequiresGroup` | 422 | `group_visibility_requires_group` | |
| `DatabaseError` | 500 | `internal_error` | |

**Critical**: `BlockedByTarget`, `TargetBlocked`, and `ContentNotVisible` all map to
HTTP 404 with a generic `not_found` error code. This prevents information leakage about
whether a block exists or content is being hidden. `[S§7.4]`

---

## §17 Cross-Domain Interactions

### §17.1 social:: Provides (consumed by other domains)

| Export | Consumers | Mechanism |
|--------|-----------|-----------|
| `SocialService` trait methods | `onboard::` (Phase 2) | `Arc<dyn SocialService>` via AppState |
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

Defined in `src/social/events.rs`. `[CODING §8.4]`

```rust
// src/social/events.rs

#[derive(Clone, Debug)]
pub struct PostCreated {
    pub post_id: Uuid,
    pub family_id: FamilyId,
    pub post_type: String,
    pub content: Option<String>,
    pub attachments: serde_json::Value,
    pub group_id: Option<Uuid>,
}
impl DomainEvent for PostCreated {}

#[derive(Clone, Debug)]
pub struct FriendRequestSent {
    pub friendship_id: Uuid,
    pub requester_family_id: FamilyId,
    pub target_family_id: FamilyId,
}
impl DomainEvent for FriendRequestSent {}

#[derive(Clone, Debug)]
pub struct FriendRequestAccepted {
    pub friendship_id: Uuid,
    pub requester_family_id: FamilyId,
    pub accepter_family_id: FamilyId,
}
impl DomainEvent for FriendRequestAccepted {}

#[derive(Clone, Debug)]
pub struct MessageSent {
    pub message_id: Uuid,
    pub conversation_id: Uuid,
    pub sender_parent_id: Uuid,
    pub sender_family_id: FamilyId,
    pub recipient_parent_id: Uuid,
    pub recipient_family_id: FamilyId,
}
impl DomainEvent for MessageSent {}

#[derive(Clone, Debug)]
pub struct EventCancelled {
    pub event_id: Uuid,
    pub creator_family_id: FamilyId,
    pub title: String,
    pub event_date: DateTime<Utc>,
    pub going_family_ids: Vec<FamilyId>,
}
impl DomainEvent for EventCancelled {}

#[derive(Clone, Debug)]
pub struct MessageReported {
    pub message_id: Uuid,
    pub reporter_family_id: FamilyId,
    pub reported_message_sender_id: Uuid,
    pub conversation_id: Uuid,
    pub reason: String,
}
impl DomainEvent for MessageReported {}
```

### §17.4 Events social:: Subscribes To

| Event | Source | Effect |
|-------|--------|--------|
| `FamilyCreated { family_id, parent_id }` | `iam::` | Create `soc_profiles` row for the new family — see `[01-iam §13]` |
| `MilestoneAchieved { family_id, student_id, ... }` | `learn::` | Create an optional `milestone` post if family has opted in `[ARCH §4.6]` |
| `CoParentRemoved { family_id, parent_id }` | `iam::` | Disassociate removed parent's posts from family — retain content but clear `author_parent_id` reference `[S§3.4]` |
| `FamilyDeletionScheduled { family_id, delete_after }` | `iam::` | Mark social data for cascade deletion after grace period `[S§16.3]` |

```rust
// src/social/event_handlers.rs

use crate::iam::events::{FamilyCreated, CoParentRemoved, FamilyDeletionScheduled};
use crate::learn::events::MilestoneAchieved;

pub struct FamilyCreatedHandler {
    social_service: Arc<dyn SocialService>,
}

#[async_trait]
impl DomainEventHandler<FamilyCreated> for FamilyCreatedHandler {
    async fn handle(&self, event: &FamilyCreated) -> Result<(), AppError> {
        self.social_service.handle_family_created(event.family_id).await
    }
}

pub struct CoParentRemovedHandler {
    social_service: Arc<dyn SocialService>,
}

#[async_trait]
impl DomainEventHandler<CoParentRemoved> for CoParentRemovedHandler {
    async fn handle(&self, event: &CoParentRemoved) -> Result<(), AppError> {
        self.social_service.handle_co_parent_removed(
            event.family_id,
            event.parent_id,
        ).await
    }
}

pub struct MilestoneAchievedHandler {
    social_service: Arc<dyn SocialService>,
}

#[async_trait]
impl DomainEventHandler<MilestoneAchieved> for MilestoneAchievedHandler {
    async fn handle(&self, event: &MilestoneAchieved) -> Result<(), AppError> {
        self.social_service.handle_milestone_achieved(
            event.family_id,
            MilestoneData {
                student_name: event.student_name.clone(),
                milestone_type: event.milestone_type.clone(),
                description: event.description.clone(),
            },
        ).await
    }
}

pub struct FamilyDeletionScheduledHandler {
    social_service: Arc<dyn SocialService>,
}

#[async_trait]
impl DomainEventHandler<FamilyDeletionScheduled> for FamilyDeletionScheduledHandler {
    async fn handle(&self, event: &FamilyDeletionScheduled) -> Result<(), AppError> {
        self.social_service.handle_family_deletion_scheduled(event.family_id).await
    }
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
- `SocialService` trait + `SocialServiceImpl`
- 12 repository traits + PostgreSQL implementations
- Domain `domain/` subdirectory: friendship.rs, post.rs, group_membership.rs,
  visibility.rs, errors.rs
- `SocialError` enum + HTTP mapping
- Domain events: `PostCreated`, `FriendRequestSent`, `FriendRequestAccepted`,
  `MessageSent`, `EventCancelled`, `MessageReported`
- Event handlers: `FamilyCreatedHandler`, `CoParentRemovedHandler`,
  `MilestoneAchievedHandler`, `FamilyDeletionScheduledHandler`
- All Phase 1 models (request, response, internal)
- OpenAPI spec + TypeScript type generation

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
- **RLS policies** — AGE does not support row-level security. `domain/visibility.rs`
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
| 3 | **Comment 1-level threading** — AGE supports unlimited `[:REPLY_TO]` depth natively, which could silently relax the threading constraint. | Enforce 1-level limit in `domain/post.rs` validation (application layer), not in the graph schema. Document this constraint explicitly so it is not accidentally relaxed after migration. |

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
src/social/
├── mod.rs                    # Re-exports, domain-level doc comments
├── handlers.rs               # Axum route handlers (thin layer only)
├── service.rs                # SocialServiceImpl — orchestration
├── repository.rs             # PgProfileRepository, PgFriendshipRepository,
│                             # PgBlockRepository, PgPostRepository,
│                             # PgCommentRepository, PgPostLikeRepository,
│                             # PgConversationRepository, PgMessageRepository,
│                             # PgGroupRepository, PgGroupMemberRepository,
│                             # PgEventRepository, PgEventRsvpRepository
├── models.rs                 # Request/response types, internal types
├── ports.rs                  # SocialService trait, all 12 repository traits
├── events.rs                 # PostCreated, FriendRequestSent, FriendRequestAccepted,
│                             # MessageSent, EventCancelled, MessageReported
├── event_handlers.rs         # FamilyCreatedHandler, CoParentRemovedHandler,
│                             # MilestoneAchievedHandler, FamilyDeletionScheduledHandler
├── websocket.rs              # WebSocket upgrade handler, Redis pub/sub integration,
│                             # WsMessage frame handling
├── feed.rs                   # FanOutPostJob, feed read path, feed rebuild,
│                             # Redis sorted set operations
├── domain/
│   ├── mod.rs
│   ├── friendship.rs         # Friendship aggregate root — state machine, invariants
│   ├── post.rs               # Post aggregate — type validation, visibility enforcement
│   ├── group_membership.rs   # GroupMembership aggregate — role transitions, ban logic
│   ├── visibility.rs         # Centralized visibility logic — all access checks
│   └── errors.rs             # SocialError enum
└── entities/                 # SeaORM-generated — never hand-edit [CODING §6.3]
```
