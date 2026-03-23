# Domain Spec 08 — Notifications (notify::)

## §1 Overview

The Notifications domain is the **cross-cutting event-driven dispatch system** — it receives
domain events from every other module and delivers in-app notifications (persisted + WebSocket
push), transactional email (Postmark), and eventually digest compilations. It owns the
notification lifecycle from event receipt through delivery, including user preference enforcement,
CAN-SPAM compliance for all email, and streak detection for learning milestones. `[S§13, V§7, V§8, V§9]`

| Attribute | Value |
|-----------|-------|
| **Module path** | `internal/notify/` |
| **DB prefix** | `notify_` |
| **Complexity class** | Simple (no `domain/` subdirectory) — event-triggered dispatch, no complex invariants `[ARCH §4.5]` |
| **CQRS** | No — read and write paths are straightforward; no separated query model needed |
| **External adapter** | `internal/notify/adapters/postmark.go` (Postmark — transactional + broadcast email) `[ARCH §2.12]` |
| **Key constraint** | System-critical notifications (security alerts, moderation actions) MUST NOT be disableable; all emails MUST comply with CAN-SPAM `[S§13.3]` |

**What notify:: owns**: In-app notification records (persistence, read tracking), notification
preferences (per-type, per-channel), streak detection (Redis counters, milestone thresholds),
Postmark email adapter (transactional and broadcast streams), digest compilation (Phase 2),
one-click email unsubscribe, background tasks (`SendEmailTask`, `PushNotificationTask`,
`CompileDigestTask`), WebSocket notification push (Redis pub/sub fan-out).

**What notify:: does NOT own**: Triggering events (owned by source domains — `social::`,
`learn::`, `mkt::`, `method::`, `onboard::`, `iam::`, `safety::`, `billing::`), WebSocket DM delivery
(owned by `social::`), user accounts and family membership (owned by `iam::`), WebSocket
connection management (shared infrastructure via `social::` WebSocket server), content
moderation decisions (owned by `safety::`), account verification and password reset emails
(owned by `iam::` via Ory Kratos built-in email templates `[ARCH §2.3]`).

**What notify:: delegates**: User/family email lookup → `iam::IamService`. Redis pub/sub
for WebSocket distribution → shared infrastructure `[ARCH §2.16]`. Background task
scheduling → `JobEnqueuer` `[ARCH §12]`. Email template rendering → Postmark server-side
templates `[ARCH §2.12]`.

---

## §2 Requirements Traceability

Every SPEC.md §13 requirement maps to a section in this document. Cross-references from
other spec sections are included where the notifications domain is involved.

| Requirement | SPEC Reference | Domain Spec Section |
|-------------|---------------|---------------------|
| Social notification types: friend requests, messages, event invitations/reminders | `[S§13.1]` | §9 (type registry), §17.1 (social events) |
| Learning notification types: milestone completions, streak achievements, progress | `[S§13.1]` | §9 (type registry), §17.1 (learning events) |
| Marketplace notification types: purchase confirmations, reviews, content updates | `[S§13.1]` | §9 (type registry), §17.1 (marketplace events) |
| System notification types: security alerts, moderation actions, policy updates | `[S§13.1]` | §9 (type registry, system-critical flag) |
| In-app delivery: notification center for all types | `[S§13.2]` | §3 (`notify_notifications`), §4.1, §10, §11 |
| Email delivery: social, marketplace, system notifications configurable | `[S§13.2]` | §7 (`EmailAdapter`), §12 (Postmark), §14 (`SendEmailTask`) |
| Push notifications: architected for future channels without redesign | `[S§13.2]` | §7 (`EmailAdapter` interface), §20 (module structure) |
| Per-type per-channel notification preferences | `[S§13.3]` | §3 (`notify_preferences`), §4.1, §13 |
| Opt out of all non-essential email | `[S§13.3]` | §13 (batch opt-out operation) |
| Email digest options: immediate, daily, weekly, off | `[S§13.3]` | §3 (`digest_frequency` CHECK), §13, §15 |
| CAN-SPAM compliance: unsubscribe link, physical address, honest subjects | `[S§13.3]` | §4.1 (unsubscribe endpoint), §12, §13 |
| System-critical notifications MUST NOT be disableable | `[S§13.3]` | §9 (system-critical flag), §13 (422 on disable attempt) |
| Event cancellation with notification to RSVPed attendees | `[S§7.7]` | §9 (`event_cancelled` type), §17.1 (`EventCancelled` handler) |
| Purchase receipts via email | `[S§9.4]` | §9 (`purchase_completed` type), §12 (Postmark template) |
| Moderation action notifications | `[S§12.3]` | §9 (`content_flagged` type), §17.1 (`ContentFlagged` handler) |
| Subscription created notification | `[S§15.3]` | §9 (`subscription_created` type), §17.1 (`SubscriptionCreated` handler) — Phase 2 |
| Subscription plan change notification | `[S§15.3]` | §9 (`subscription_changed` type), §17.1 (`SubscriptionChanged` handler) — Phase 2 |
| Subscription cancellation notification | `[S§15.3]` | §9 (`subscription_cancelled` type), §17.1 (`SubscriptionCancelled` handler) — Phase 2 |
| Creator payout confirmation | `[S§15.3]` | §9 (`payout_completed` type), §17.1 (`PayoutCompleted` handler) — Phase 2 |

> **Coverage note on `[S§13.1]` examples**: SPEC.md §13.1 lists illustrative notification types
> per category using the word "Examples." This spec implements notification types for all domain
> events that currently exist in domain specs 01-07. Types listed in SPEC.md but not yet
> implemented include: post comments (no `PostCommented` event exists — `social::` §17.3 does
> not publish one), group invitations (groups are Phase 2 in `social::` §19), event
> invitations/reminders (Phase 2 calendar features), mastery path progress (Phase 2 in
> `learn::` §19), reviews on creator content (no `ReviewCreated` subscriber for `notify::` in
> `mkt::` §18.3 — review notification goes to `safety::` only), content updates for purchasers
> (Phase 2 in `mkt::` §19), sale announcements (Phase 3), security alerts beyond moderation
> (Kratos handles authentication emails directly `[ARCH §2.3]`), subscription renewal advance
> notice (Phase 3 — scheduled reminder before auto-renewal), and policy updates (admin tooling,
> Phase 3+). Billing subscription lifecycle events (`SubscriptionCreated`, `SubscriptionChanged`,
> `SubscriptionCancelled`, `PayoutCompleted`) are now covered in Phase 2. Each remaining type will be added
> to the type registry when its source event is defined in the originating domain spec.

> **Note on `CoParentInvited`**: `01-iam.md` §10.2 (line 1162) mentions a `CoParentInvited`
> event in its lifecycle narrative, but this event does not appear in IAM's formal event table
> (§13.3). Until IAM's event table is updated to include `CoParentInvited`, this spec follows
> the formal table and only handles `CoParentAdded` (Phase 2). If `CoParentInvited` is
> ratified in a future IAM spec revision, a `co_parent_invited` notification type and handler
> should be added here.

---

## §3 Database Schema

The notifications domain stores in-app notification records, per-type per-channel preference
overrides, and compiled digest snapshots. All tables are family-scoped via `family_id` foreign
key. `[ARCH §5.1, ARCH §5.2]`

> **Refinement note**: The architecture doc (§15.10) sketches `notify_notifications` and
> `notify_preferences` tables. This spec adds: (1) `notify_digests` table for Phase 2 digest
> compilation, (2) `notification_type` and `category` CHECK constraints with full type registry,
> (3) `metadata` JSONB column for event-specific payload (enabling idempotency via source_event_id),
> (4) partial index on unread notifications for fast badge count.

### §3.1 Enums

All enums use CHECK constraints for consistency with other domain specs and to avoid
PostgreSQL enum migration limitations. `[ARCH §5.2]`

```sql
-- Notification category, notification type, delivery channel, and digest
-- frequency are all enforced via CHECK constraints on their respective
-- columns rather than as PostgreSQL enum types. This avoids ALTER TYPE
-- limitations when adding new values in future migrations. [ARCH §5.2]
--
-- Category values: social, learning, marketplace, system
-- Notification type values: friend_request_sent, friend_request_accepted,
--     message_received, event_cancelled, methodology_changed,
--     onboarding_completed, activity_streak, milestone_achieved,
--     book_completed, data_export_ready, purchase_completed,
--     purchase_refunded, creator_onboarded, content_flagged,
--     co_parent_added, family_deletion_scheduled,
--     subscription_created, subscription_changed,
--     subscription_cancelled, payout_completed
-- Channel values: in_app, email
-- Digest frequency values: immediate, daily, weekly, off
```

### §3.2 Tables

#### `notify_notifications` — In-App Notification Store

```sql
-- ═══════════════════════════════════════════════════════════════════════════════
-- TABLE 1: notify_notifications — In-app notification store [S§13.2]
-- ═══════════════════════════════════════════════════════════════════════════════
-- Stores every in-app notification delivered to a family. Read tracking is
-- per-notification; no per-user read state (families share a notification
-- feed).
-- ═══════════════════════════════════════════════════════════════════════════════
CREATE TABLE notify_notifications (
    id              UUID        PRIMARY KEY DEFAULT uuidv7(),
    family_id       UUID        NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    notification_type TEXT      NOT NULL CHECK (notification_type IN (
        'friend_request_sent', 'friend_request_accepted',
        'message_received', 'event_cancelled',
        'methodology_changed', 'onboarding_completed',
        'activity_streak', 'milestone_achieved',
        'book_completed', 'data_export_ready',
        'purchase_completed', 'purchase_refunded',
        'creator_onboarded', 'content_flagged',
        'co_parent_added', 'family_deletion_scheduled',
        'subscription_created', 'subscription_changed',
        'subscription_cancelled', 'payout_completed'
    )),
    category        TEXT        NOT NULL CHECK (category IN (
        'social', 'learning', 'marketplace', 'system'
    )),
    title           TEXT        NOT NULL,
    body            TEXT        NOT NULL,
    action_url      TEXT,                     -- Deep link into the SPA (e.g., /friends/requests)
    metadata        JSONB       NOT NULL DEFAULT '{}',  -- Event-specific payload; always contains source_event_id
    is_read         BOOLEAN     NOT NULL DEFAULT false,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Primary query: family feed sorted by newest first
CREATE INDEX idx_notify_notifications_family_created
    ON notify_notifications (family_id, created_at DESC);

-- Fast unread count for badge: partial index avoids scanning read notifications
CREATE INDEX idx_notify_notifications_family_unread
    ON notify_notifications (family_id)
    WHERE is_read = false;

-- Idempotency check: prevent duplicate notifications for the same source event
CREATE UNIQUE INDEX idx_notify_notifications_idempotency
    ON notify_notifications (family_id, notification_type, ((metadata->>'source_event_id')))
    WHERE metadata->>'source_event_id' IS NOT NULL;
```

#### `notify_preferences` — Per-Type Per-Channel Settings

```sql
-- ═══════════════════════════════════════════════════════════════════════════════
-- TABLE 2: notify_preferences — Per-type per-channel preference overrides [S§13.3]
-- ═══════════════════════════════════════════════════════════════════════════════
-- Stores per-family overrides for notification delivery. Missing row = enabled
-- (default-enabled semantics). Only rows that deviate from defaults need to
-- exist.
-- ═══════════════════════════════════════════════════════════════════════════════
CREATE TABLE notify_preferences (
    id              UUID        PRIMARY KEY DEFAULT uuidv7(),
    family_id       UUID        NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    notification_type TEXT      NOT NULL CHECK (notification_type IN (
        'friend_request_sent', 'friend_request_accepted',
        'message_received', 'event_cancelled',
        'methodology_changed', 'onboarding_completed',
        'activity_streak', 'milestone_achieved',
        'book_completed', 'data_export_ready',
        'purchase_completed', 'purchase_refunded',
        'creator_onboarded', 'content_flagged',
        'co_parent_added', 'family_deletion_scheduled',
        'subscription_created', 'subscription_changed',
        'subscription_cancelled', 'payout_completed'
    )),
    channel         TEXT        NOT NULL CHECK (channel IN ('in_app', 'email')),
    enabled         BOOLEAN     NOT NULL DEFAULT true,
    digest_frequency TEXT       NOT NULL DEFAULT 'immediate' CHECK (digest_frequency IN (
        'immediate', 'daily', 'weekly', 'off'
    )),
    UNIQUE (family_id, notification_type, channel)
);

-- Query all preferences for a family (preferences page)
CREATE INDEX idx_notify_preferences_family
    ON notify_preferences (family_id);
```

#### `notify_digests` — Compiled Digest Snapshots (Phase 2)

```sql
-- ═══════════════════════════════════════════════════════════════════════════════
-- TABLE 3: notify_digests — Compiled digest snapshots (Phase 2) [ARCH §12]
-- ═══════════════════════════════════════════════════════════════════════════════
-- Phase 2: compiled digest snapshots ready for email delivery. Created in
-- Phase 1 migration but unused until Phase 2.
-- ═══════════════════════════════════════════════════════════════════════════════
CREATE TABLE notify_digests (
    id              UUID        PRIMARY KEY DEFAULT uuidv7(),
    family_id       UUID        NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    digest_type     TEXT        NOT NULL CHECK (digest_type IN ('daily', 'weekly')),
    period_start    TIMESTAMPTZ NOT NULL,
    period_end      TIMESTAMPTZ NOT NULL,
    content_json    JSONB       NOT NULL,     -- Compiled digest content: { notifications: [...] }
    sent_at         TIMESTAMPTZ,              -- NULL until email sent
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Query unsent digests for batch processing
CREATE INDEX idx_notify_digests_unsent
    ON notify_digests (created_at)
    WHERE sent_at IS NULL;

-- One digest per family per period
CREATE UNIQUE INDEX idx_notify_digests_family_period
    ON notify_digests (family_id, digest_type, period_start);
```

### §3.3 RLS Policies

```sql
-- Families can read and update their own notifications (mark read).
-- System role (service account) can INSERT notifications on behalf of any family.
ALTER TABLE notify_notifications ENABLE ROW LEVEL SECURITY;

CREATE POLICY notify_notifications_family_select
    ON notify_notifications FOR SELECT
    USING (family_id = current_setting('app.family_id')::UUID);

CREATE POLICY notify_notifications_family_update
    ON notify_notifications FOR UPDATE
    USING (family_id = current_setting('app.family_id')::UUID);

CREATE POLICY notify_notifications_system_insert
    ON notify_notifications FOR INSERT
    WITH CHECK (current_setting('app.role') = 'system');

-- Families can read and update their own preferences.
ALTER TABLE notify_preferences ENABLE ROW LEVEL SECURITY;

CREATE POLICY notify_preferences_family_select
    ON notify_preferences FOR SELECT
    USING (family_id = current_setting('app.family_id')::UUID);

CREATE POLICY notify_preferences_family_update
    ON notify_preferences FOR UPDATE
    USING (family_id = current_setting('app.family_id')::UUID);

CREATE POLICY notify_preferences_system_insert
    ON notify_preferences FOR INSERT
    WITH CHECK (current_setting('app.role') = 'system');

-- Digests: system role only (background task creates and marks sent).
ALTER TABLE notify_digests ENABLE ROW LEVEL SECURITY;

CREATE POLICY notify_digests_system_all
    ON notify_digests FOR ALL
    WITH CHECK (current_setting('app.role') = 'system');
```

---

## §4 API Endpoints

All endpoints are prefixed with `/v1/notifications`. Auth requirements use extractors defined
in `00-core §13`: `AuthContext` for authenticated users, `FamilyScope` for family-scoped
data access. `[CODING §2.1]`

### §4.1 Phase 1 (5 authenticated + 1 unauthenticated)

#### Notification Feed (3 endpoints)

##### `GET /v1/notifications`

List notifications for the current family with unread count.

- **Auth**: `AuthContext` + `FamilyScope`
- **Query**: `NotificationListParams { cursor?, limit? (default 20, max 100), category?: string, unread_only?: bool }`
- **Response**: `200 OK` → `NotificationListResponse { Notifications: []NotificationResponse, UnreadCount: int64, NextCursor? }`
- **Pagination**: Cursor-based on `(created_at, id)` for stable ordering
- **Error codes**: `401` (unauthenticated), `400` (invalid params)

##### `PATCH /v1/notifications/:id/read`

Mark a single notification as read. Idempotent — marking an already-read notification succeeds.

- **Auth**: `AuthContext` + `FamilyScope`
- **Body**: None
- **Response**: `200 OK` → `NotificationResponse`
- **Side effects**: Sets `is_read = true`. No event published (terminal domain).
- **Error codes**: `404` (not found or not owned — never reveal existence `[S§18]`)

##### `PATCH /v1/notifications/read-all`

Bulk mark all notifications as read, optionally filtered by category.

- **Auth**: `AuthContext` + `FamilyScope`
- **Body**: `{ category?: string }`
- **Response**: `200 OK` → `{ updated_count: int64 }`
- **Side effects**: Sets `is_read = true` on all matching unread notifications for the family.
- **Error codes**: `422` (invalid category)

#### Preferences (2 endpoints)

##### `GET /v1/notifications/preferences`

Return the full type x channel preference matrix with defaults applied.

- **Auth**: `AuthContext` + `FamilyScope`
- **Response**: `200 OK` → `[]PreferenceResponse`
- **Semantics**: Returns one entry per (notification_type, channel) pair. Missing rows are returned as `enabled: true, digest_frequency: "immediate"` (default-enabled). System-critical types include `system_critical: true` flag.
- **Error codes**: `401`

##### `PATCH /v1/notifications/preferences`

Batch upsert notification preferences.

- **Auth**: `AuthContext` + `FamilyScope`
- **Body**: `UpdatePreferencesCommand { Preferences: []PreferenceUpdate }`
- **Validation**: Rejects disabling system-critical types (`content_flagged`, `co_parent_added`, `family_deletion_scheduled`) — returns `422 CannotDisableSystemCritical`.
- **Response**: `200 OK` → `[]PreferenceResponse`
- **Side effects**: Upserts `notify_preferences` rows. No event published.
- **Error codes**: `422` (invalid type/channel, system-critical disable attempt)

#### Unsubscribe (1 unauthenticated endpoint)

##### `GET /v1/notifications/unsubscribe?token=<signed_token>`

One-click email unsubscribe link for CAN-SPAM compliance. No authentication required —
the signed token proves ownership. `[S§13.3]`

- **Auth**: None (signed token verification)
- **Query**: `token` — HMAC-signed token containing `{ family_id, notification_type, channel }`
- **Response**: `200 OK` → HTML page confirming unsubscription
- **Side effects**: Upserts `notify_preferences` row with `enabled = false` for the specified type and channel. System-critical types cannot be unsubscribed (returns error page).
- **Error codes**: `400` (invalid/expired token), `422` (system-critical type)

### §4.2 Phase 2

- `GET /v1/notifications/digest-preview` — Preview next digest content for the family
- Digest frequency changes in preferences become functional (daily/weekly options)

### §4.3 Phase 3+

- Push notification registration endpoints (mobile device tokens)
- Notification grouping/collapsing preferences

---

## §5 Service Interface

The `NotificationService` interface defines all use cases exposed to handlers and event handler
structs. No CQRS separation needed — this is a simple domain with straightforward
read/write paths. `[CODING §8.2]`

```go
// internal/notify/ports.go

package notify

import (
    "context"

    "github.com/google/uuid"
    "github.com/homegrown-academy/homegrown-academy/internal/shared/types"
)

// NotificationService defines all use cases exposed to handlers and event handlers.
type NotificationService interface {
    // ─── Commands (write, has side effects) ─────────────────────────────

    // CreateNotification creates an in-app notification and fans out via WebSocket + email.
    // Called by event handlers, not by HTTP handlers directly.
    CreateNotification(ctx context.Context, cmd CreateNotificationCommand) error

    // MarkRead marks a single notification as read. Idempotent.
    MarkRead(ctx context.Context, notificationID uuid.UUID, scope types.FamilyScope) error

    // MarkAllRead bulk marks all (optionally category-filtered) notifications as read.
    MarkAllRead(ctx context.Context, scope types.FamilyScope, category *string) (int64, error)

    // UpdatePreferences batch upserts notification preferences. Validates system-critical constraints.
    UpdatePreferences(ctx context.Context, cmd UpdatePreferencesCommand, scope types.FamilyScope) error

    // SendEmail enqueues a transactional email via Postmark.
    SendEmail(ctx context.Context, cmd SendEmailCommand) error

    // ProcessUnsubscribe processes a signed unsubscribe token and disables the preference.
    ProcessUnsubscribe(ctx context.Context, token string) error

    // ─── Event handlers (one per subscribed event type) ─────────────────
    // Each method is called by its corresponding DomainEventHandler struct
    // in event_handlers.go. Failures are logged but do not propagate to
    // the source domain. [ARCH §4.6]

    // social:: events
    HandleFriendRequestSent(ctx context.Context, event *FriendRequestSent) error
    HandleFriendRequestAccepted(ctx context.Context, event *FriendRequestAccepted) error
    HandleMessageSent(ctx context.Context, event *MessageSent) error
    HandleEventCancelled(ctx context.Context, event *EventCancelled) error

    // method:: events
    HandleFamilyMethodologyChanged(ctx context.Context, event *FamilyMethodologyChanged) error

    // onboard:: events
    HandleOnboardingCompleted(ctx context.Context, event *OnboardingCompleted) error

    // learn:: events
    HandleActivityLogged(ctx context.Context, event *ActivityLogged) error
    HandleMilestoneAchieved(ctx context.Context, event *MilestoneAchieved) error
    HandleBookCompleted(ctx context.Context, event *BookCompleted) error
    HandleDataExportReady(ctx context.Context, event *DataExportReady) error

    // mkt:: events
    HandlePurchaseCompleted(ctx context.Context, event *PurchaseCompleted) error
    HandlePurchaseRefunded(ctx context.Context, event *PurchaseRefunded) error
    HandleCreatorOnboarded(ctx context.Context, event *CreatorOnboarded) error

    // safety:: events
    HandleContentFlagged(ctx context.Context, event *ContentFlagged) error

    // iam:: events (Phase 2)
    HandleCoParentAdded(ctx context.Context, event *CoParentAdded) error
    HandleFamilyDeletionScheduled(ctx context.Context, event *FamilyDeletionScheduled) error

    // billing:: events (Phase 2)
    HandleSubscriptionCreated(ctx context.Context, event *SubscriptionCreated) error
    HandleSubscriptionChanged(ctx context.Context, event *SubscriptionChanged) error
    HandleSubscriptionCancelled(ctx context.Context, event *SubscriptionCancelled) error
    HandlePayoutCompleted(ctx context.Context, event *PayoutCompleted) error

    // ─── Queries (read, no side effects) ────────────────────────────────

    // ListNotifications returns a paginated notification list with unread count.
    ListNotifications(ctx context.Context, params NotificationListParams, scope types.FamilyScope) (*NotificationListResponse, error)

    // GetPreferences returns the full type x channel preference matrix with defaults applied.
    GetPreferences(ctx context.Context, scope types.FamilyScope) ([]PreferenceResponse, error)
}
```

### `NotificationServiceImpl`

```go
// internal/notify/service.go

package notify

// NotificationServiceImpl implements NotificationService.
type NotificationServiceImpl struct {
    notificationRepo NotificationRepository
    preferenceRepo   PreferenceRepository
    digestRepo       DigestRepository        // Phase 2
    emailAdapter     EmailAdapter
    iamService       IamService              // Email lookup
    cache            shared.Cache            // WebSocket pub/sub + streak counters
    jobEnqueuer      shared.JobEnqueuer      // Background job enqueuing [CODING §8.1b]
}
```

---

## §6 Repository Interfaces

All notification and preference repositories are family-scoped via `FamilyScope` parameter.
Digest repository is system-scoped (background task access). `[CODING §8.2]`

```go
// internal/notify/ports.go (continued)

// ─── NotificationRepository ────────────────────────────────────────────
// Family-scoped — all reads and writes are per-family. [00-core §8]
type NotificationRepository interface {
    Create(ctx context.Context, cmd CreateNotification) (*NotifyNotification, error)

    GetByID(ctx context.Context, notificationID uuid.UUID, scope types.FamilyScope) (*NotifyNotification, error)

    List(ctx context.Context, params *NotificationListParams, scope types.FamilyScope) ([]NotifyNotification, error)

    CountUnread(ctx context.Context, scope types.FamilyScope) (int64, error)

    MarkRead(ctx context.Context, notificationID uuid.UUID, scope types.FamilyScope) (bool, error)

    MarkAllRead(ctx context.Context, scope types.FamilyScope, category *string) (int64, error)

    // ExistsBySourceEvent checks idempotency: does a notification with this source_event_id already exist?
    ExistsBySourceEvent(ctx context.Context, familyID types.FamilyID, notificationType string, sourceEventID string) (bool, error)

    // DeleteByFamily performs cascade delete for family deletion.
    DeleteByFamily(ctx context.Context, familyID types.FamilyID) error
}

// ─── PreferenceRepository ──────────────────────────────────────────────
// Family-scoped. Default-enabled semantics: missing row = enabled.
type PreferenceRepository interface {
    // GetAll returns all explicit preference overrides for a family.
    GetAll(ctx context.Context, scope types.FamilyScope) ([]NotifyPreference, error)

    // UpsertBatch batch upserts preferences (INSERT ON CONFLICT UPDATE).
    UpsertBatch(ctx context.Context, scope types.FamilyScope, updates []PreferenceUpsert) error

    // IsEnabled checks if a specific type+channel is enabled for a family.
    // Returns true if no row exists (default-enabled).
    IsEnabled(ctx context.Context, familyID types.FamilyID, notificationType string, channel string) (bool, error)

    // DeleteByFamily performs cascade delete for family deletion.
    DeleteByFamily(ctx context.Context, familyID types.FamilyID) error
}

// ─── DigestRepository (Phase 2) ────────────────────────────────────────
// System-scoped — background task creates and processes digests.
type DigestRepository interface {
    Create(ctx context.Context, cmd CreateDigest) (*NotifyDigest, error)

    GetUnsent(ctx context.Context, limit int64) ([]NotifyDigest, error)

    MarkSent(ctx context.Context, digestID uuid.UUID) error
}
```

---

## §7 Adapter Interfaces

### EmailAdapter (Postmark)

The `EmailAdapter` interface wraps email delivery. It is provider-agnostic by name — the only
implementation is `PostmarkEmailAdapter`, but the interface boundary allows testing with a mock
and swapping providers if needed. `[ARCH §2.12]`

```go
// internal/notify/ports.go (continued)

// EmailAdapter is a provider-agnostic email delivery interface.
// Phase 1: PostmarkEmailAdapter (transactional stream).
// Phase 2: Adds broadcast stream for digests.
type EmailAdapter interface {
    // SendTransactional sends a single transactional email using a Postmark template.
    //
    // templateAlias maps to a Postmark template (e.g., "purchase-receipt").
    // templateModel is a JSON object of template variables.
    SendTransactional(ctx context.Context, to string, templateAlias string, templateModel map[string]any) error

    // SendBatch sends up to 500 emails in a single Postmark batch API call.
    // Used when a single event triggers notifications to multiple families
    // (e.g., EventCancelled with many RSVPs).
    SendBatch(ctx context.Context, messages []BatchEmailMessage) error

    // SendBroadcast sends a broadcast email (Phase 2 — digest stream).
    // Uses Postmark's broadcast message stream to protect transactional
    // deliverability from digest volume. [ARCH §2.12]
    SendBroadcast(ctx context.Context, to string, templateAlias string, templateModel map[string]any) error
}

// BatchEmailMessage represents a single message in a batch send.
type BatchEmailMessage struct {
    To             string         `json:"to"`
    TemplateAlias  string         `json:"template_alias"`
    TemplateModel  map[string]any `json:"template_model"`
}
```

---

## §8 Models (DTOs)

### §8.1 Request Types

```go
// internal/notify/models.go

package notify

// NotificationListParams holds query parameters for GET /v1/notifications.
type NotificationListParams struct {
    Cursor    *string `query:"cursor"`
    Limit     *uint8  `query:"limit"`          // Default 20, max 100
    Category  *string `query:"category"`       // Filter by category
    UnreadOnly *bool  `query:"unread_only"`
}

// UpdatePreferencesCommand holds the body for PATCH /v1/notifications/preferences.
type UpdatePreferencesCommand struct {
    Preferences []PreferenceUpdate `json:"preferences" validate:"required,dive"`
}

// PreferenceUpdate represents a single preference change within a batch update.
type PreferenceUpdate struct {
    NotificationType string  `json:"notification_type" validate:"required"`
    Channel          string  `json:"channel" validate:"required"`            // "in_app" | "email"
    Enabled          bool    `json:"enabled"`
    DigestFrequency  *string `json:"digest_frequency,omitempty"`            // "immediate" | "daily" | "weekly" | "off"
}
```

### §8.2 Response Types

```go
// NotificationResponse represents a single notification in the feed.
type NotificationResponse struct {
    ID               uuid.UUID `json:"id"`
    NotificationType string    `json:"notification_type"`
    Category         string    `json:"category"`
    Title            string    `json:"title"`
    Body             string    `json:"body"`
    ActionURL        *string   `json:"action_url,omitempty"`
    IsRead           bool      `json:"is_read"`
    CreatedAt        time.Time `json:"created_at"`
}

// NotificationListResponse is a paginated notification list with unread badge count.
type NotificationListResponse struct {
    Notifications []NotificationResponse `json:"notifications"`
    UnreadCount   int64                  `json:"unread_count"`
    NextCursor    *string                `json:"next_cursor,omitempty"`
}

// PreferenceResponse represents a single entry in the preference matrix.
type PreferenceResponse struct {
    NotificationType string `json:"notification_type"`
    Channel          string `json:"channel"`
    Enabled          bool   `json:"enabled"`
    DigestFrequency  string `json:"digest_frequency"`
    SystemCritical   bool   `json:"system_critical"`      // true = cannot be disabled
}
```

### §8.3 Internal Types

```go
// CreateNotificationCommand is an internal command to create a notification (used by event handlers).
// Not exposed via API — event handlers construct this.
type CreateNotificationCommand struct {
    FamilyID         types.FamilyID
    NotificationType string
    Category         string
    Title            string
    Body             string
    ActionURL        *string
    Metadata         map[string]any   // Must contain "source_event_id"
}

// SendEmailCommand is an internal command to send an email (enqueued as SendEmailTask).
type SendEmailCommand struct {
    To             string         `json:"to"`
    TemplateAlias  string         `json:"template_alias"`
    TemplateModel  map[string]any `json:"template_model"`
}

// WebSocketFrame is the frame pushed to connected clients.
type WebSocketFrame struct {
    MsgType string               `json:"msg_type"`     // "notification"
    Data    NotificationResponse `json:"data"`
}

// notification_types contains all registered notification types.
// Defined as constants rather than an enum to allow easy iteration and
// lookup without switch exhaustiveness overhead in a simple domain.
const (
    // Social
    TypeFriendRequestSent     = "friend_request_sent"
    TypeFriendRequestAccepted = "friend_request_accepted"
    TypeMessageReceived       = "message_received"
    TypeEventCancelled        = "event_cancelled"

    // Learning
    TypeMethodologyChanged  = "methodology_changed"
    TypeOnboardingCompleted = "onboarding_completed"
    TypeActivityStreak      = "activity_streak"
    TypeMilestoneAchieved   = "milestone_achieved"
    TypeBookCompleted       = "book_completed"
    TypeDataExportReady     = "data_export_ready"

    // Marketplace
    TypePurchaseCompleted = "purchase_completed"
    TypePurchaseRefunded  = "purchase_refunded"
    TypeCreatorOnboarded  = "creator_onboarded"
    TypePayoutCompleted   = "payout_completed"

    // System
    TypeContentFlagged          = "content_flagged"
    TypeCoParentAdded           = "co_parent_added"
    TypeFamilyDeletionScheduled = "family_deletion_scheduled"

    // Billing (Phase 2)
    TypeSubscriptionCreated   = "subscription_created"
    TypeSubscriptionChanged   = "subscription_changed"
    TypeSubscriptionCancelled = "subscription_cancelled"
)

// SystemCriticalTypes are types that cannot be disabled via preferences. [S§13.3]
var SystemCriticalTypes = []string{
    TypeContentFlagged,
    TypeCoParentAdded,
    TypeFamilyDeletionScheduled,
}
```

---

## §9 Notification Types & Category System (Domain Deep-Dive 1)

### Category Taxonomy

Notifications are grouped into four categories matching `[S§13.1]`. Categories determine
default email behavior and provide a coarse filter for the notification feed.

| Category | Purpose | Default Email | Examples |
|----------|---------|---------------|----------|
| `social` | Friend and community interactions | Enabled | Friend requests, messages, event cancellations |
| `learning` | Student progress and milestones | Enabled | Streaks, milestones, book completions, export ready |
| `marketplace` | Commerce lifecycle | Enabled | Purchase receipts, refund confirmations, creator welcome |
| `system` | Platform operations and safety | **Always enabled** | Content moderation, co-parent changes, account deletion |

### Complete Type Registry

Each notification type maps to a source event, belongs to a category, and has a title template
used by `CreateNotification`. System-critical types cannot be disabled.

| Type | Category | Source Event | Source Domain | Title Template | System-Critical | Phase |
|------|----------|-------------|---------------|----------------|:---------------:|-------|
| `friend_request_sent` | social | `FriendRequestSent` | `social::` | "{family_name} sent you a friend request" | No | 1 |
| `friend_request_accepted` | social | `FriendRequestAccepted` | `social::` | "{family_name} accepted your friend request" | No | 1 |
| `message_received` | social | `MessageSent` | `social::` | "New message from {sender_name}" | No | 1 |
| `event_cancelled` | social | `EventCancelled` | `social::` | "'{event_title}' has been cancelled" | No | 1 |
| `methodology_changed` | learning | `FamilyMethodologyChanged` | `method::` | "Your methodology has been updated" | No | 1 |
| `onboarding_completed` | learning | `OnboardingCompleted` | `onboard::` | "Welcome to Homegrown Academy!" | No | 1 |
| `activity_streak` | learning | `ActivityLogged` | `learn::` | "{student_name} reached a {days}-day streak!" | No | 1 |
| `milestone_achieved` | learning | `MilestoneAchieved` | `learn::` | "{student_name} achieved: {description}" | No | 1 |
| `book_completed` | learning | `BookCompleted` | `learn::` | "{student_name} finished '{book_title}'" | No | 1 |
| `data_export_ready` | learning | `DataExportReady` | `learn::` | "Your data export is ready to download" | No | 1 |
| `purchase_completed` | marketplace | `PurchaseCompleted` | `mkt::` | "Your purchase is confirmed" | No | 1 |
| `purchase_refunded` | marketplace | `PurchaseRefunded` | `mkt::` | "Your refund has been processed" | No | 1 |
| `creator_onboarded` | marketplace | `CreatorOnboarded` | `mkt::` | "Welcome to the Creator Program!" | No | 1 |
| `content_flagged` | system | `ContentFlagged` | `safety::` | "Content moderation update" | **Yes** | 1 |
| `co_parent_added` | system | `CoParentAdded` | `iam::` | "A co-parent has been added to your family" | **Yes** | 2 |
| `family_deletion_scheduled` | system | `FamilyDeletionScheduled` | `iam::` | "Your account is scheduled for deletion" | **Yes** | 2 |
| `subscription_created` | system | `SubscriptionCreated` | `billing::` | "Welcome to Homegrown Premium!" | No | 2 |
| `subscription_changed` | system | `SubscriptionChanged` | `billing::` | "Your subscription has been updated" | No | 2 |
| `subscription_cancelled` | system | `SubscriptionCancelled` | `billing::` | "Your subscription has ended" | No | 2 |
| `payout_completed` | marketplace | `PayoutCompleted` | `billing::` | "Your payout of {amount} has been sent" | No | 2 |

### Streak Detection Logic

Activity streaks are detected by the `HandleActivityLogged` event handler using Redis
counters — not computed from the database.

```
ActivityLogged event received
  -> INCR notify:streak:{student_id}:{activity_date}  (SET with NX, expire 48h)
  -> Check consecutive days via Redis: get keys for last N days
  -> If streak count in {7, 14, 30, 60, 100}:
      -> CreateNotification(activity_streak) with streak days in metadata
      -> Publish MilestoneAchieved event (delegated to learn:: to decide)
```

**Redis key pattern**: `notify:streak:{student_id}:{YYYY-MM-DD}` with 48-hour TTL.
The gap tolerance (48h TTL vs checking consecutive dates) ensures timezone edge cases
don't break streaks.

### Extensibility

Adding a new notification type requires:
1. Add the type string to the CHECK constraint (append-only migration)
2. Add constant to notification type constants
3. Add to type registry table (category, title template, system-critical flag)
4. Implement the event handler in `event_handlers.go`
5. Add Postmark template alias if email delivery is needed

No existing code needs modification — the dispatch is type-string-driven, not switch-arm-driven.

---

## §10 Delivery Pipeline (Domain Deep-Dive 2)

### Event-to-Notification Flow

```
+--------------+
| Source Domain |
| publishes     |
| DomainEvent   |
+------+-------+
       |
       v
+----------------------------------------------------------+
| internal/notify/event_handlers.go                          |
|                                                            |
|  DomainEventHandler.Handle()                               |
|    -> notificationService.Handle<EventType>()              |
+----------------------+-----------------------------------+
                       |
                       v
+----------------------------------------------------------+
| internal/notify/service.go -- Handle<EventType>() impl     |
|                                                            |
|  1. Build CreateNotificationCommand from event payload     |
|  2. Check idempotency (source_event_id)                    |
|  3. Call CreateNotification(cmd)                           |
+----------------------+-----------------------------------+
                       |
                       v
+----------------------------------------------------------+
| CreateNotification(cmd)                                    |
|                                                            |
|  +- In-App Path ------------------------------------------+
|  | 1. Check preference: IsEnabled(type, "in_app")?         |
|  | 2. INSERT into notify_notifications                     |
|  | 3. Redis PUBLISH notifications:{family_id}              |
|  |    -> WebSocket push to connected clients               |
|  +--------------------------------------------------------+
|                                                            |
|  +- Email Path -------------------------------------------+
|  | 1. Check preference: IsEnabled(type, "email")?          |
|  | 2. If digest_frequency == "immediate":                  |
|  |    -> Look up family email via iam::IamService          |
|  |    -> Enqueue SendEmailTask via JobEnqueuer              |
|  | 3. If digest_frequency in {"daily", "weekly"}:          |
|  |    -> Skip (CompileDigestTask handles it -- Phase 2)    |
|  +--------------------------------------------------------+
+----------------------------------------------------------+
```

### Key Pipeline Properties

**In-app and email are independent paths**: Failure in email delivery does not prevent in-app
notification creation. Both paths check preferences independently — a family can have email
disabled but in-app enabled for a given type.

**Idempotency**: Every `CreateNotificationCommand` includes `Metadata["source_event_id"]` (typically
the source entity's UUID). The unique index on `(family_id, notification_type, source_event_id)`
prevents duplicate notifications. If a duplicate is detected, `CreateNotification` returns `nil`
silently.

**Error isolation**: Event handler failures are logged at `slog.Error` level but do not propagate to
the source domain operation. The source event is not retried — notification delivery is
best-effort from the source domain's perspective. `[ARCH §4.6]`

**System-critical override**: For system-critical notification types, the preference check is
skipped entirely — both in-app and email paths always execute regardless of preference settings.

---

## §11 WebSocket Real-Time Push (Domain Deep-Dive 3)

### Architecture

WebSocket notification push shares the same WebSocket infrastructure established by `social::`
for direct messages. `[ARCH §2.16, 05-social §11]`

```
+--------------+      +--------------+      +--------------------------+
| notify::      |      | Redis        |      | WebSocket Server         |
| service.go    |----->| PUBLISH      |----->| (shared with social::)   |
|               |      | channel:     |      |                          |
| Create        |      | notifications|      | Delivers to all          |
| Notification  |      | :{family_id} |      | connections for family   |
+--------------+      +--------------+      +--------------------------+
```

### Redis Pub/Sub Channel

- **Channel name**: `notifications:{family_id}`
- **Payload**: JSON-serialized `WebSocketFrame`
- **Shared with**: `social::` uses `messages:{family_id}` on the same Redis pub/sub infrastructure

### Frame Format

```json
{
    "msg_type": "notification",
    "data": {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "notification_type": "friend_request_sent",
        "category": "social",
        "title": "The Smith Family sent you a friend request",
        "body": "Would you like to connect?",
        "action_url": "/friends/requests",
        "is_read": false,
        "created_at": "2026-03-20T15:30:00Z"
    }
}
```

### Frontend Integration

```typescript
// Pseudocode — actual implementation in social:: WebSocket infrastructure

// The shared useWebSocket() hook dispatches on msg_type:
// - "message" -> social:: DM handling
// - "notification" -> invalidate notification queries

const { lastMessage } = useWebSocket();

useEffect(() => {
    if (lastMessage?.msg_type === "notification") {
        // Invalidate notification list + unread count queries
        queryClient.invalidateQueries({ queryKey: ["notifications"] });
    }
}, [lastMessage]);
```

### Multi-Connection Handling

A family may have multiple parents connected simultaneously (e.g., both parents on different
devices). Redis pub/sub delivers to all subscribers on the `notifications:{family_id}` channel,
so all connected clients receive the push. Each client independently handles query invalidation.

### Offline Handling

If no WebSocket connection is active for a family, the Redis PUBLISH has zero subscribers
and the message is silently dropped. This is correct behavior — the notification persists in
`notify_notifications` and will appear when the user next opens the app and fetches the
notification list.

---

## §12 Postmark Email Adapter & Template System (Domain Deep-Dive 4)

### Postmark Stream Separation

Two separate Postmark message streams protect transactional email deliverability: `[ARCH §2.12]`

| Stream | Purpose | Phase | Volume Profile |
|--------|---------|-------|---------------|
| **Transactional** | Account verification, purchase receipts, friend requests, moderation alerts | 1 | Low volume, high priority |
| **Broadcast** | Daily/weekly digest compilations, platform announcements | 2 | Higher volume, batched |

### Template Alias Registry

Each notification type that triggers email maps to a Postmark template alias and a set of
template variables. Templates are managed in the Postmark dashboard, not in code.

| Notification Type | Template Alias | Template Variables | Phase |
|-------------------|---------------|-------------------|-------|
| `friend_request_sent` | `friend-request` | `{ requester_family_name, action_url, unsubscribe_url }` | 1 |
| `friend_request_accepted` | `friend-accepted` | `{ accepter_family_name, action_url, unsubscribe_url }` | 1 |
| `message_received` | `new-message` | `{ sender_name, preview_text, action_url, unsubscribe_url }` | 1 |
| `event_cancelled` | `event-cancelled` | `{ event_title, event_date, creator_family_name, unsubscribe_url }` | 1 |
| `onboarding_completed` | `welcome` | `{ family_name, methodology_name, getting_started_url }` | 1 |
| `activity_streak` | `streak-milestone` | `{ student_name, streak_days, action_url, unsubscribe_url }` | 1 |
| `milestone_achieved` | `milestone` | `{ student_name, milestone_description, action_url, unsubscribe_url }` | 1 |
| `data_export_ready` | `export-ready` | `{ download_url, expires_at, unsubscribe_url }` | 1 |
| `purchase_completed` | `purchase-receipt` | `{ listing_title, amount_formatted, download_url, unsubscribe_url }` | 1 |
| `purchase_refunded` | `refund-confirmation` | `{ listing_title, refund_amount_formatted, unsubscribe_url }` | 1 |
| `creator_onboarded` | `creator-welcome` | `{ store_name, dashboard_url }` | 1 |
| `content_flagged` | `content-flagged` | `{ content_description, action_taken, appeal_url }` | 1 |
| `co_parent_added` | `co-parent-added` | `{ family_name, co_parent_name }` | 2 |
| `family_deletion_scheduled` | `deletion-scheduled` | `{ delete_after_date, cancellation_url }` | 2 |

### Email Personalization

Notification event handlers call `iam::IamService` to look up the primary parent's email
address and display name for the target family. The lookup is:

```
iamService.GetFamilyPrimaryEmail(ctx, familyID) -> (email string, displayName string, err error)
```

This is the only cross-domain service call notify:: makes. All other interactions are via
domain events.

### SendEmailTask Structure

```go
// internal/notify/tasks.go

// SendEmailTask is a background task payload for transactional email delivery. [ARCH §12]
type SendEmailTask struct {
    To             string         `json:"to"`
    TemplateAlias  string         `json:"template_alias"`
    TemplateModel  map[string]any `json:"template_model"`
    // For deduplication: Postmark ignores duplicate sends with same MessageStream
    // + Tag within a short window. We also use this for logging correlation.
    IdempotencyKey string         `json:"idempotency_key"`
}
```

**Queue**: Default (process within 5 minutes) `[ARCH §12]`

**Retry policy**: 3 retries with exponential backoff (30s, 2m, 10m). After 3 failures,
the task moves to the dead letter queue for manual inspection.

**Failure handling**: Email delivery failures are logged at `slog.Error` level with the
idempotency key and template alias. The in-app notification is unaffected — it was already
created before the task was enqueued.

### CAN-SPAM Compliance

Every transactional email includes: `[S§13.3]`

1. **`List-Unsubscribe` header**: Points to `GET /v1/notifications/unsubscribe?token=<signed>`
2. **`List-Unsubscribe-Post` header**: Supports RFC 8058 one-click unsubscribe
3. **Physical mailing address**: In Postmark template footer (managed in dashboard)
4. **Honest subject lines**: Template subjects match notification titles — no deceptive content

---

## §13 User Preferences & CAN-SPAM Compliance (Domain Deep-Dive 5)

### Preference Matrix

The full matrix has `16 notification types x 2 channels = 32 cells`. The GET preferences
endpoint returns all 32 entries with defaults applied (missing row = enabled).

```
                          in_app    email
                         -------- --------
 friend_request_sent     |  Y   |   Y   |
 friend_request_accepted |  Y   |   Y   |
 message_received        |  Y   |   Y   |
 event_cancelled         |  Y   |   Y   |
 methodology_changed     |  Y   |   Y   |
 onboarding_completed    |  Y   |   Y   |
 activity_streak         |  Y   |   Y   |
 milestone_achieved      |  Y   |   Y   |
 book_completed          |  Y   |   Y   |
 data_export_ready       |  Y   |   Y   |
 purchase_completed      |  Y   |   Y   |
 purchase_refunded       |  Y   |   Y   |
 creator_onboarded       |  Y   |   Y   |
 content_flagged         |  L   |   L   |  <- system-critical
 co_parent_added         |  L   |   L   |  <- system-critical
 family_deletion_sched.  |  L   |   L   |  <- system-critical

 Y = enabled (default), toggleable
 L = always enabled, cannot be disabled
```

### Default-Enabled Semantics

The `notify_preferences` table only stores **deviations** from the default. A family that
has never changed any preferences has zero rows in this table, and all notifications are
delivered. This means:

- `IsEnabled(familyID, type, channel)` → `SELECT enabled FROM notify_preferences WHERE family_id = $1 AND notification_type = $2 AND channel = $3` → if no row, return `true`
- New notification types are automatically enabled for all families without migration

### System-Critical Override

Three notification types are hardcoded as system-critical in `SystemCriticalTypes`:

```go
var SystemCriticalTypes = []string{
    TypeContentFlagged,
    TypeCoParentAdded,
    TypeFamilyDeletionScheduled,
}
```

**Enforcement points**:
1. `UpdatePreferences()` — returns `ErrCannotDisableSystemCritical` (HTTP 422) if any preference update attempts to set `enabled = false` for a system-critical type
2. `ProcessUnsubscribe()` — returns an error page for system-critical types
3. `CreateNotification()` — skips preference check entirely for system-critical types (always delivers both in-app and email)

### Batch Opt-Out

"Opt out of all non-essential email" `[S§13.3]` is implemented as a batch preference update:

```
PATCH /v1/notifications/preferences
{
    "preferences": [
        // One entry per non-system-critical type with channel "email", enabled false
        { "notification_type": "friend_request_sent", "channel": "email", "enabled": false },
        { "notification_type": "friend_request_accepted", "channel": "email", "enabled": false },
        // ... (13 entries total — all non-system-critical types)
    ]
}
```

The frontend provides a "Mute all non-essential emails" toggle that generates this batch request.
System-critical types are excluded from the batch — attempting to include them returns 422.

### Digest Frequency

| Value | Phase | Behavior |
|-------|-------|----------|
| `immediate` | 1 | Email sent immediately via `SendEmailTask` |
| `off` | 1 | Email suppressed entirely |
| `daily` | 2 | Notification accumulated; `CompileDigestTask` sends at 6:00 AM UTC |
| `weekly` | 2 | Notification accumulated; `CompileDigestTask` sends Monday 6:00 AM UTC |

In Phase 1, `daily` and `weekly` values are accepted and stored in preferences but behave
identically to `immediate` (no digest compilation task exists yet). This ensures preferences
set in Phase 1 become functional when Phase 2 ships without requiring migration.

### One-Click Unsubscribe

The unsubscribe token is an HMAC-SHA256 signed payload:

```
payload = base64url({ family_id, notification_type, channel, exp })
signature = HMAC-SHA256(payload, UNSUBSCRIBE_SECRET)
token = payload.signature
```

The `GET /v1/notifications/unsubscribe` endpoint:
1. Verifies the HMAC signature
2. Checks token expiration (30-day TTL)
3. Rejects system-critical types
4. Upserts `notify_preferences` with `enabled = false`
5. Returns a simple HTML confirmation page (no SPA — must work without JavaScript)

---

## §14 Background Tasks (Domain Deep-Dive 6)

### SendEmailTask (Default Queue, Phase 1)

| Property | Value |
|----------|-------|
| **Queue** | `default` (process within 5 minutes) `[ARCH §12]` |
| **Payload** | `SendEmailTask { To, TemplateAlias, TemplateModel, IdempotencyKey }` |
| **Retry policy** | 3 retries, exponential backoff: 30s -> 2m -> 10m |
| **Dead letter** | After 3 failures, task moves to dead letter queue |
| **Idempotency** | Postmark deduplicates by `MessageStream` + `Tag` within a window; `IdempotencyKey` is set as the Postmark `Tag` header |

```go
// internal/notify/tasks.go

package notify

import (
    "context"
    "encoding/json"
    "fmt"
    "log/slog"

    "github.com/hibiken/asynq"
)

const (
    TypeSendEmail    = "notify:send_email"
    TypeCompileDigest = "notify:compile_digest"
)

// NewSendEmailTask creates a new asynq task for email delivery.
func NewSendEmailTask(payload SendEmailTask) (*asynq.Task, error) {
    data, err := json.Marshal(payload)
    if err != nil {
        return nil, fmt.Errorf("marshalling send email task: %w", err)
    }
    return asynq.NewTask(TypeSendEmail, data,
        asynq.MaxRetry(3),
        asynq.Queue("default"),
    ), nil
}

// HandleSendEmailTask processes a send email task.
func HandleSendEmailTask(adapter EmailAdapter) asynq.HandlerFunc {
    return func(ctx context.Context, t *asynq.Task) error {
        var payload SendEmailTask
        if err := json.Unmarshal(t.Payload(), &payload); err != nil {
            return fmt.Errorf("unmarshalling send email task: %w", err)
        }
        if err := adapter.SendTransactional(ctx, payload.To, payload.TemplateAlias, payload.TemplateModel); err != nil {
            slog.Error("Email delivery failed",
                "idempotency_key", payload.IdempotencyKey,
                "template", payload.TemplateAlias,
                "error", err,
            )
            return err // asynq retries automatically
        }
        return nil
    }
}
```

### PushNotificationTask (Default Queue, Phase 1 inline -> Phase 2 as task)

In Phase 1, WebSocket push is performed inline within `CreateNotification()` (Redis PUBLISH
is fast and non-blocking). In Phase 2, if push notification delivery to mobile devices is
added, this becomes a proper background task.

| Property | Value |
|----------|-------|
| **Queue** | `default` |
| **Phase 1** | Inline Redis PUBLISH (not a background task) |
| **Phase 2+** | Background task for mobile push delivery (APNs/FCM) |

### CompileDigestTask (Low Queue, Phase 2)

| Property | Value |
|----------|-------|
| **Queue** | `low` (process within 1 hour) `[ARCH §12]` |
| **Schedule** | Daily at 6:00 AM UTC, weekly on Mondays at 6:00 AM UTC `[ARCH §12]` |
| **Payload** | `CompileDigestTask { DigestType: "daily" | "weekly" }` |

```go
// CompileDigestPayload is the payload for the compile digest task.
type CompileDigestPayload struct {
    DigestType string `json:"digest_type"` // "daily" | "weekly"
}

// NewCompileDigestTask creates a new asynq task for digest compilation.
func NewCompileDigestTask(payload CompileDigestPayload) (*asynq.Task, error) {
    data, err := json.Marshal(payload)
    if err != nil {
        return nil, fmt.Errorf("marshalling compile digest task: %w", err)
    }
    return asynq.NewTask(TypeCompileDigest, data,
        asynq.MaxRetry(2),
        asynq.Queue("low"),
    ), nil
}

// HandleCompileDigestTask processes a compile digest task.
func HandleCompileDigestTask( /* deps */ ) asynq.HandlerFunc {
    return func(ctx context.Context, t *asynq.Task) error {
        // See §15 for compilation algorithm
        // Phase 2 implementation
        return nil
    }
}
```

---

## §15 Digest Compilation (Domain Deep-Dive 7, Phase 2)

### Schedule

| Digest Type | Schedule | Period |
|-------------|----------|--------|
| Daily | 6:00 AM UTC every day | Previous 24 hours |
| Weekly | 6:00 AM UTC every Monday | Previous 7 days |

### Compilation Algorithm

```
CompileDigestTask.Handle(digestType):
  1. period = calculatePeriod(digestType)
       daily:  (yesterday 6:00 AM UTC, today 6:00 AM UTC)
       weekly: (last Monday 6:00 AM UTC, this Monday 6:00 AM UTC)

  2. families = preferenceRepo.GetFamiliesWithDigest(digestType)
       -> SELECT DISTINCT family_id FROM notify_preferences
         WHERE digest_frequency = $1 AND channel = 'email' AND enabled = true

  3. FOR EACH familyID IN families (batched, 100 at a time):
       a. notifications = notificationRepo.ListInPeriod(familyID, period)
       b. IF len(notifications) == 0 -> SKIP (empty digest suppression)
       c. contentJSON = compileDigestContent(notifications)
            -> Group by category, sort by created_at DESC within each group
       d. digestRepo.Create({ familyID, digestType, period, contentJSON })
       e. email, _ = iamService.GetFamilyPrimaryEmail(familyID)
       f. taskClient.Enqueue(NewSendEmailTask(SendEmailTask{
              To:             email,
              TemplateAlias:  fmt.Sprintf("%s-digest", digestType),
              TemplateModel:  contentJSON,
              IdempotencyKey: fmt.Sprintf("digest:%s:%s", familyID, periodStart),
          }))
            -> Uses BROADCAST stream (not transactional)
       g. digestRepo.MarkSent(digestID)
```

### Empty Digest Suppression

If a family has no notifications in the digest period, no digest email is sent. This prevents
"nothing happened this week" emails that train users to ignore digests.

### Postmark Broadcast Stream

Digest emails use the Postmark broadcast stream `[ARCH §2.12]` rather than the transactional
stream. This separation ensures that digest email volume and engagement metrics do not affect
the deliverability reputation of transactional emails (purchase receipts, security alerts, etc.).

---

## §16 Error Types

All notification errors use custom error types with `errors.Is`/`errors.As` and map to HTTP status
codes via `AppError`. Internal details are logged but never exposed in API responses. `[CODING §5.2, S§18]`

```go
// internal/notify/errors.go

package notify

import "errors"

var (
    // ─── Notification Errors ─────────────────────────────────────────────
    ErrNotificationNotFound = errors.New("notification not found")
    ErrNotificationNotOwned = errors.New("notification not owned by this family")

    // ─── Preference Errors ───────────────────────────────────────────────
    ErrCannotDisableSystemCritical = errors.New("cannot disable system-critical notification type")
    ErrInvalidNotificationType     = errors.New("invalid notification type")
    ErrInvalidChannel              = errors.New("invalid delivery channel")
    ErrInvalidDigestFrequency      = errors.New("invalid digest frequency")

    // ─── Unsubscribe Errors ──────────────────────────────────────────────
    ErrInvalidUnsubscribeToken = errors.New("invalid or expired unsubscribe token")

    // ─── Adapter Errors ──────────────────────────────────────────────────
    ErrEmailDeliveryFailed = errors.New("email delivery failed")
    ErrEmailAdapterError   = errors.New("email adapter error")

    // ─── Infrastructure ──────────────────────────────────────────────────
    ErrDatabaseError = errors.New("database error")
    ErrRedisError    = errors.New("redis error")
)
```

### Error-to-HTTP Mapping

| Error Variant | HTTP Status | Response Code |
|--------------|-------------|---------------|
| `ErrNotificationNotFound` | `404 Not Found` | `notification_not_found` |
| `ErrNotificationNotOwned` | `404 Not Found` | `notification_not_found` (never reveal existence) |
| `ErrCannotDisableSystemCritical` | `422 Unprocessable` | `cannot_disable_system_critical` |
| `ErrInvalidNotificationType` | `422 Unprocessable` | `invalid_notification_type` |
| `ErrInvalidChannel` | `422 Unprocessable` | `invalid_channel` |
| `ErrInvalidDigestFrequency` | `422 Unprocessable` | `invalid_digest_frequency` |
| `ErrInvalidUnsubscribeToken` | `400 Bad Request` | `invalid_unsubscribe_token` |
| `ErrEmailDeliveryFailed` | `502 Bad Gateway` | `email_delivery_failed` |
| `ErrDatabaseError` | `500 Internal` | `internal_error` (no details exposed) |
| `ErrRedisError` | `500 Internal` | `internal_error` (no details exposed) |
| `ErrEmailAdapterError` | `500 Internal` | `internal_error` (no details exposed) |

**Security note**: `ErrNotificationNotOwned` maps to `404` (not `403`) to prevent enumeration.
Callers cannot distinguish "does not exist" from "exists but belongs to another family".
`[S§18]`

---

## §17 Cross-Domain Interactions

### §17.1 notify:: Provides (consumed by other domains)

**Nothing.** `notify::` is a **pure consumer / terminal domain**. It subscribes to events
from other domains but publishes no events and exposes no service interface for other domains
to call. This is by design — notification delivery is a side effect, not a dependency.

No domain should import anything from `notify::`. The dependency graph is strictly one-way.

### §17.2 notify:: Consumes

| Dependency | Source | Purpose |
|-----------|--------|---------|
| `AuthContext` | `iam::` middleware | User identity on every request `[00-core §7.2]` |
| `FamilyScope` | `iam::` middleware | Family-scoped data access `[00-core §8]` |
| `IamService.GetFamilyPrimaryEmail()` | `iam::` | Email address lookup for transactional email personalization |
| `EmailAdapter` | Postmark (self-hosted adapter) | Transactional + broadcast email delivery `[ARCH §2.12]` |
| Redis pub/sub | shared infrastructure | WebSocket notification push `[ARCH §2.16]` |
| Redis counters | shared infrastructure | Streak detection counters |
| asynq | shared infrastructure | Background task scheduling `[ARCH §12]` |

### §17.3 Events notify:: Subscribes To

`notify::` subscribes to 20 domain events from 8 source domains. Each event maps to a
handler struct in `internal/notify/event_handlers.go`. `[ARCH §4.6]`

```go
// internal/notify/event_handlers.go

package notify

import (
    "context"
    "log/slog"

    socialevents "github.com/homegrown-academy/homegrown-academy/internal/social/events"
    methodevents "github.com/homegrown-academy/homegrown-academy/internal/method/events"
    onboardevents "github.com/homegrown-academy/homegrown-academy/internal/onboard/events"
    learnevents "github.com/homegrown-academy/homegrown-academy/internal/learn/events"
    mktevents "github.com/homegrown-academy/homegrown-academy/internal/mkt/events"
    safetyevents "github.com/homegrown-academy/homegrown-academy/internal/safety/events"
    iamevents "github.com/homegrown-academy/homegrown-academy/internal/iam/events"
    billingevents "github.com/homegrown-academy/homegrown-academy/internal/billing/events"
)

// ─── social:: events ─────────────────────────────────────────────────────

type FriendRequestSentHandler struct {
    NotificationService NotificationService
}

func (h *FriendRequestSentHandler) Handle(ctx context.Context, event *socialevents.FriendRequestSent) error {
    return h.NotificationService.HandleFriendRequestSent(ctx, event)
}

type FriendRequestAcceptedHandler struct {
    NotificationService NotificationService
}

func (h *FriendRequestAcceptedHandler) Handle(ctx context.Context, event *socialevents.FriendRequestAccepted) error {
    return h.NotificationService.HandleFriendRequestAccepted(ctx, event)
}

type MessageSentHandler struct {
    NotificationService NotificationService
}

func (h *MessageSentHandler) Handle(ctx context.Context, event *socialevents.MessageSent) error {
    return h.NotificationService.HandleMessageSent(ctx, event)
}

type EventCancelledHandler struct {
    NotificationService NotificationService
}

func (h *EventCancelledHandler) Handle(ctx context.Context, event *socialevents.EventCancelled) error {
    return h.NotificationService.HandleEventCancelled(ctx, event)
}

// ─── method:: events ─────────────────────────────────────────────────────

type FamilyMethodologyChangedHandler struct {
    NotificationService NotificationService
}

func (h *FamilyMethodologyChangedHandler) Handle(ctx context.Context, event *methodevents.FamilyMethodologyChanged) error {
    return h.NotificationService.HandleFamilyMethodologyChanged(ctx, event)
}

// ─── onboard:: events ────────────────────────────────────────────────────

type OnboardingCompletedHandler struct {
    NotificationService NotificationService
}

func (h *OnboardingCompletedHandler) Handle(ctx context.Context, event *onboardevents.OnboardingCompleted) error {
    return h.NotificationService.HandleOnboardingCompleted(ctx, event)
}

// ─── learn:: events ──────────────────────────────────────────────────────

type ActivityLoggedHandler struct {
    NotificationService NotificationService
}

func (h *ActivityLoggedHandler) Handle(ctx context.Context, event *learnevents.ActivityLogged) error {
    return h.NotificationService.HandleActivityLogged(ctx, event)
}

type MilestoneAchievedHandler struct {
    NotificationService NotificationService
}

func (h *MilestoneAchievedHandler) Handle(ctx context.Context, event *learnevents.MilestoneAchieved) error {
    return h.NotificationService.HandleMilestoneAchieved(ctx, event)
}

type BookCompletedHandler struct {
    NotificationService NotificationService
}

func (h *BookCompletedHandler) Handle(ctx context.Context, event *learnevents.BookCompleted) error {
    return h.NotificationService.HandleBookCompleted(ctx, event)
}

type DataExportReadyHandler struct {
    NotificationService NotificationService
}

func (h *DataExportReadyHandler) Handle(ctx context.Context, event *learnevents.DataExportReady) error {
    return h.NotificationService.HandleDataExportReady(ctx, event)
}

// ─── mkt:: events ────────────────────────────────────────────────────────

type PurchaseCompletedHandler struct {
    NotificationService NotificationService
}

func (h *PurchaseCompletedHandler) Handle(ctx context.Context, event *mktevents.PurchaseCompleted) error {
    return h.NotificationService.HandlePurchaseCompleted(ctx, event)
}

type PurchaseRefundedHandler struct {
    NotificationService NotificationService
}

func (h *PurchaseRefundedHandler) Handle(ctx context.Context, event *mktevents.PurchaseRefunded) error {
    return h.NotificationService.HandlePurchaseRefunded(ctx, event)
}

type CreatorOnboardedHandler struct {
    NotificationService NotificationService
}

func (h *CreatorOnboardedHandler) Handle(ctx context.Context, event *mktevents.CreatorOnboarded) error {
    return h.NotificationService.HandleCreatorOnboarded(ctx, event)
}

// ─── safety:: events ─────────────────────────────────────────────────────

type ContentFlaggedHandler struct {
    NotificationService NotificationService
}

func (h *ContentFlaggedHandler) Handle(ctx context.Context, event *safetyevents.ContentFlagged) error {
    return h.NotificationService.HandleContentFlagged(ctx, event)
}

// ─── iam:: events (Phase 2) ──────────────────────────────────────────────

type CoParentAddedHandler struct {
    NotificationService NotificationService
}

func (h *CoParentAddedHandler) Handle(ctx context.Context, event *iamevents.CoParentAdded) error {
    return h.NotificationService.HandleCoParentAdded(ctx, event)
}

type FamilyDeletionScheduledHandler struct {
    NotificationService NotificationService
}

func (h *FamilyDeletionScheduledHandler) Handle(ctx context.Context, event *iamevents.FamilyDeletionScheduled) error {
    return h.NotificationService.HandleFamilyDeletionScheduled(ctx, event)
}

// ─── billing:: events (Phase 2) ─────────────────────────────────────────

type SubscriptionCreatedHandler struct {
    NotificationService NotificationService
}

func (h *SubscriptionCreatedHandler) Handle(ctx context.Context, event *billingevents.SubscriptionCreated) error {
    return h.NotificationService.HandleSubscriptionCreated(ctx, event)
}

type SubscriptionChangedHandler struct {
    NotificationService NotificationService
}

func (h *SubscriptionChangedHandler) Handle(ctx context.Context, event *billingevents.SubscriptionChanged) error {
    return h.NotificationService.HandleSubscriptionChanged(ctx, event)
}

type SubscriptionCancelledHandler struct {
    NotificationService NotificationService
}

func (h *SubscriptionCancelledHandler) Handle(ctx context.Context, event *billingevents.SubscriptionCancelled) error {
    return h.NotificationService.HandleSubscriptionCancelled(ctx, event)
}

type PayoutCompletedHandler struct {
    NotificationService NotificationService
}

func (h *PayoutCompletedHandler) Handle(ctx context.Context, event *billingevents.PayoutCompleted) error {
    return h.NotificationService.HandlePayoutCompleted(ctx, event)
}
```

### §17.4 Event Handler Detail

Each event handler follows the same pattern: build a `CreateNotificationCommand` from the
event payload, then call `CreateNotification()`. Special cases are noted below.

| Event | Target Family | Special Behavior |
|-------|--------------|------------------|
| `FriendRequestSent` | `target_family_id` | — |
| `FriendRequestAccepted` | `requester_family_id` | — |
| `MessageSent` | `recipient_family_id` | Only if recipient has no active WebSocket (offline check via Redis) |
| `EventCancelled` | Each `going_family_ids[i]` | Batch: one notification per RSVP'd family; batch email via `SendBatch()` |
| `FamilyMethodologyChanged` | `family_id` | — |
| `OnboardingCompleted` | `family_id` | Skip if `skipped == true` (no welcome email for skipped onboarding) |
| `ActivityLogged` | `family_id` | No in-app notification created directly; only triggers streak check (§9) |
| `MilestoneAchieved` | `family_id` | — |
| `BookCompleted` | `family_id` | Also triggers streak check for reading-specific milestones |
| `DataExportReady` | `family_id` | Email includes signed download URL with expiration |
| `PurchaseCompleted` | `family_id` | Email includes purchase receipt details from `content_metadata` |
| `PurchaseRefunded` | `family_id` | — |
| `CreatorOnboarded` | Looked up via `parent_id` | Uses `iam::IamService` to resolve `parent_id -> family_id` |
| `ContentFlagged` | Content owner's `family_id` | System-critical: always delivered regardless of preferences |
| `CoParentAdded` (Phase 2) | `family_id` | System-critical: always delivered |
| `FamilyDeletionScheduled` (Phase 2) | `family_id` | System-critical: always delivered; includes cancellation URL |
| `SubscriptionCreated` (Phase 2) | `family_id` | Email includes tier and billing interval |
| `SubscriptionChanged` (Phase 2) | `family_id` | Email includes `change_type` (interval_change, renewal, reactivation) |
| `SubscriptionCancelled` (Phase 2) | `family_id` | Email includes `effective_at` date; mentions data preservation |
| `PayoutCompleted` (Phase 2) | Looked up via `creator_id` | Uses `iam::IamService` to resolve `creator_id -> family_id`; email includes amount and currency |

---

## §18 Phase Scope

### Phase 1 — Foundation `[S§19, ARCH §15.10]`

**In scope**:
- 3 database tables (`notify_notifications`, `notify_preferences`, `notify_digests`)
- 5 authenticated API endpoints (notification list, mark read, mark all read, get preferences, update preferences)
- 1 unauthenticated endpoint (email unsubscribe)
- 14 event handlers (all except `CoParentAdded`, `FamilyDeletionScheduled`)
- In-app notification creation with WebSocket push via Redis pub/sub
- Postmark transactional email stream with template-based delivery
- `SendEmailTask` background task with retry policy
- Streak detection via Redis counters (7/14/30/60/100-day milestones)
- User preference enforcement with default-enabled semantics
- System-critical notification override (cannot disable `content_flagged`)
- CAN-SPAM compliance (one-click unsubscribe, `List-Unsubscribe` headers)
- **5+1 endpoints, 3 tables, 14 event handlers**

**Out of scope for Phase 1**: Digest compilation, broadcast email stream, `CoParentAdded`
handler, `FamilyDeletionScheduled` handler, notification retention purge, mobile push
notifications.

### Phase 2 — Digests & Depth `[S§19]`

**In scope**:
- `CompileDigestTask` (daily at 6:00 AM UTC, weekly Monday 6:00 AM UTC)
- Postmark broadcast stream for digest delivery
- `CoParentAdded` and `FamilyDeletionScheduled` event handlers
- `SubscriptionCreated`, `SubscriptionChanged`, `SubscriptionCancelled`, `PayoutCompleted` event handlers (`billing::` events)
- Digest frequency preferences become functional (daily/weekly actually batch)
- Notification retention purge (delete notifications older than 90 days)
- Digest preview endpoint
- **1 additional endpoint, 6 additional event handlers**

### Phase 3+ — Scale `[S§19]`

**In scope**:
- Mobile push notifications (APNs/FCM) via `PushNotificationTask`
- Rich HTML email templates (Phase 1 uses simple Postmark templates)
- Notification grouping/collapsing (e.g., "3 new messages" instead of 3 separate notifications)
- `billing::` subscription renewal advance notice (scheduled reminder before auto-renewal)

---

## §19 Verification Checklist

Each item is a testable assertion. Implementation is not complete until all assertions pass.

### Notification CRUD

1. `GET /v1/notifications` returns paginated results sorted by `created_at DESC`
2. `GET /v1/notifications?category=social` filters to social notifications only
3. `GET /v1/notifications?unread_only=true` returns only unread notifications
4. Response includes `unread_count` computed from partial index
5. Cursor-based pagination returns stable results across pages
6. `PATCH /v1/notifications/:id/read` sets `is_read = true`
7. Marking an already-read notification returns `200 OK` (idempotent)
8. Marking a notification owned by another family returns `404` (not `403`)
9. `PATCH /v1/notifications/read-all` updates all unread notifications for the family
10. `PATCH /v1/notifications/read-all` with `category` filter only updates matching notifications

### Preference Enforcement

11. `GET /v1/notifications/preferences` returns all 32 cells (16 types x 2 channels)
12. Missing preference rows are returned as `enabled: true, digest_frequency: "immediate"`
13. System-critical types are returned with `system_critical: true`
14. `PATCH /v1/notifications/preferences` upserts preference rows
15. Attempting to disable a system-critical type returns `422 CannotDisableSystemCritical`
16. Batch opt-out of all non-essential email succeeds (13 types disabled)
17. Preferences for `content_flagged`, `co_parent_added`, `family_deletion_scheduled` cannot be set to `enabled: false`

### Event Handlers

18. `FriendRequestSent` creates notification for `target_family_id`
19. `FriendRequestAccepted` creates notification for `requester_family_id`
20. `EventCancelled` creates one notification per family in `going_family_ids`
21. `OnboardingCompleted` with `skipped: true` does NOT create a notification
22. `ActivityLogged` performs streak detection; milestone at 7 days creates `activity_streak` notification
23. Duplicate events (same `source_event_id`) do not create duplicate notifications (idempotency)
24. Event handler failure does not propagate to source domain

### WebSocket Push

25. `CreateNotification()` publishes to `notifications:{family_id}` Redis channel
26. WebSocket frame format matches `{ msg_type: "notification", data: NotificationResponse }`
27. Multiple connected clients for the same family all receive the push

### Email Delivery

28. `SendEmailTask` calls `emailAdapter.SendTransactional()` with correct template alias and variables
29. Failed email delivery retries 3 times with exponential backoff
30. Email includes `List-Unsubscribe` header with signed token URL
31. `GET /v1/notifications/unsubscribe?token=` with valid token disables the preference
32. `GET /v1/notifications/unsubscribe?token=` with expired or invalid token returns `400`

### Error Handling

33. Internal errors (DB, Redis, email adapter) return `500` with generic message — no details exposed
34. Email delivery failures return `502 Bad Gateway`
35. All error types map to documented HTTP status codes

---

## §20 Module Structure

```
internal/notify/
+-- handlers.go              # 5 Echo route handlers (thin layer only)
+-- service.go               # NotificationServiceImpl — event handling,
|                            # preference checks, delivery orchestration
+-- repository.go            # PgNotificationRepo, PgPreferenceRepo,
|                            # PgDigestRepo (Phase 2)
|                            # (GORM models defined in models.go)
+-- models.go                # Request/response types, notification type
|                            # constants, WebSocketFrame, GORM models
+-- ports.go                 # NotificationService interface, all repository
|                            # interfaces, EmailAdapter interface
+-- errors.go                # Sentinel error variables
+-- event_handlers.go        # 20 DomainEventHandler structs (one per
|                            # subscribed event type) [ARCH §4.6]
+-- tasks.go                 # SendEmailTask, PushNotificationTask (Phase 2),
|                            # CompileDigestTask (Phase 2) [ARCH §12]
+-- adapters/
    +-- postmark.go          # PostmarkEmailAdapter — wraps Postmark HTTP
                             # API, returns domain types only [ARCH §2.12]
```

---

## §21 Addendum: Email Operations Detail `[S§13.2, ARCH §2.12]`

*Added to address spec gaps in email template management, deliverability, and bounce handling.*

### §21.1 Transactional Email Templates

All transactional emails use Postmark message streams with server-side templates.
Templates are managed as code (version-controlled) and synced to Postmark via CI.

**Email template catalog** (Phase 1):

| Template ID | Trigger | Category |
|-------------|---------|----------|
| `welcome` | Account creation | Account |
| `email-verification` | Registration / email change | Account |
| `password-reset` | Password reset request (via Kratos) | Account |
| `coppa-consent` | COPPA consent request | Account |
| `co-parent-invitation` | Co-parent invited to family | Account |
| `account-deletion-confirm` | Account deletion requested | Account |
| `account-deletion-cancelled` | Account deletion cancelled | Account |
| `data-export-ready` | Data export completed | Account |
| `purchase-receipt` | Marketplace purchase completed | Marketplace |
| `creator-sale-notification` | Creator's content was purchased | Marketplace |
| `payout-processed` | Creator payout sent | Marketplace |
| `friend-request` | New friend request received | Social |
| `message-received` | New direct message (if email pref enabled) | Social |
| `event-reminder` | Upcoming event reminder | Social |
| `quiz-completed` | Student completed a quiz | Learning |
| `assignment-completed` | Student completed an assignment | Learning |
| `moderation-action` | Content removed / account warned | Safety |
| `security-alert` | Suspicious session detected | Security |
| `session-revoked` | "Sign out everywhere" completed | Security |

### §21.2 Template Management

```
frontend/emails/
+-- templates/
|   +-- welcome.html          # Postmark template (HTML + text)
|   +-- purchase-receipt.html
|   +-- ...
+-- layouts/
|   +-- base.html             # Shared header/footer with brand
+-- sync.ts                   # Script to sync templates to Postmark API
```

Templates follow these rules:
- All templates MUST have both HTML and plaintext versions
- HTML templates MUST be responsive (mobile-friendly)
- Templates MUST NOT include tracking pixels or external image loads (privacy)
- Template variables use Postmark's `{{variable}}` syntax
- Templates are synced to Postmark via `npm run sync-email-templates` (CI step)

### §21.3 Deliverability Monitoring

| Metric | Source | Alert Threshold |
|--------|--------|-----------------|
| **Bounce rate** | Postmark webhook | > 5% in 24 hours |
| **Spam complaint rate** | Postmark webhook | > 0.1% in 24 hours |
| **Delivery rate** | Postmark dashboard | < 95% in 24 hours |
| **Open rate** (transactional) | Postmark dashboard | < 40% (investigate subject lines) |

### §21.4 Bounce & Complaint Handling

Postmark sends webhooks for bounces and spam complaints. The `PostmarkEmailAdapter`
processes these:

```go
// internal/notify/adapters/postmark.go

// HandlePostmarkBounce processes webhook notifications for bounces.
func (a *PostmarkEmailAdapter) HandlePostmarkBounce(ctx context.Context, bounce PostmarkBounceWebhook) error {
    switch bounce.BounceType {
    case "HardBounce":
        // Hard bounce: email address is invalid
        // Mark email as undeliverable in notify_email_status
        if err := a.repo.MarkEmailUndeliverable(ctx, bounce.Email); err != nil {
            return fmt.Errorf("marking email undeliverable: %w", err)
        }
        // After 3 hard bounces, suppress all emails to this address
        count, err := a.repo.CountHardBounces(ctx, bounce.Email)
        if err != nil {
            return fmt.Errorf("counting hard bounces: %w", err)
        }
        if count >= 3 {
            if err := a.repo.SuppressEmail(ctx, bounce.Email); err != nil {
                return fmt.Errorf("suppressing email: %w", err)
            }
            slog.Warn("Email suppressed after 3 hard bounces",
                "email", bounce.Email,
            )
        }
    case "SpamComplaint":
        // Spam complaint: user marked email as spam
        // Immediately suppress all emails to this address (CAN-SPAM)
        if err := a.repo.SuppressEmail(ctx, bounce.Email); err != nil {
            return fmt.Errorf("suppressing email: %w", err)
        }
        slog.Warn("Email suppressed due to spam complaint",
            "email", bounce.Email,
        )
    default:
        // Soft bounce: temporary issue (mailbox full, etc.)
        // Log but don't suppress — Postmark handles retries
        slog.Info("Soft bounce received",
            "email", bounce.Email,
            "bounce_type", bounce.BounceType,
        )
    }
    return nil
}
```

### §21.5 CAN-SPAM Compliance

- All marketing/digest emails MUST include an unsubscribe link (one-click via Postmark)
- Transactional emails (receipts, security alerts, COPPA) are exempt from unsubscribe
  requirements but SHOULD still include preference management links
- Physical mailing address MUST be included in marketing emails (platform's business address)
- Unsubscribe requests MUST be honored within 10 business days (Postmark handles immediately)
- The email preference system (§13) provides granular opt-out per notification category

### §21.6 Schema Addition

```sql
-- Track email delivery status for bounce management
CREATE TABLE notify_email_status (
    email           VARCHAR(255) PRIMARY KEY,
    is_suppressed   BOOLEAN NOT NULL DEFAULT false,
    hard_bounce_count INT NOT NULL DEFAULT 0,
    last_bounce_at  TIMESTAMPTZ,
    suppressed_at   TIMESTAMPTZ,
    suppression_reason VARCHAR(30)
                    CHECK (suppression_reason IN ('hard_bounce', 'spam_complaint', 'manual')),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

Before sending any email, the `SendEmailTask` checks `notify_email_status` — if the
email is suppressed, the task completes without sending (logged but not delivered).
