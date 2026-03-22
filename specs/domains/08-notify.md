# Domain Spec 08 — Notifications (notify::)

## §1 Overview

The Notifications domain is the **cross-cutting event-driven dispatch system** — it receives
domain events from every other module and delivers in-app notifications (persisted + WebSocket
push), transactional email (Postmark), and eventually digest compilations. It owns the
notification lifecycle from event receipt through delivery, including user preference enforcement,
CAN-SPAM compliance for all email, and streak detection for learning milestones. `[S§13, V§7, V§8, V§9]`

| Attribute | Value |
|-----------|-------|
| **Module path** | `src/notify/` |
| **DB prefix** | `notify_` |
| **Complexity class** | Simple (no `domain/` subdirectory) — event-triggered dispatch, no complex invariants `[ARCH §4.5]` |
| **CQRS** | No — read and write paths are straightforward; no separated query model needed |
| **External adapter** | `src/notify/adapters/postmark.rs` (Postmark — transactional + broadcast email) `[ARCH §2.12]` |
| **Key constraint** | System-critical notifications (security alerts, moderation actions) MUST NOT be disableable; all emails MUST comply with CAN-SPAM `[S§13.3]` |

**What notify:: owns**: In-app notification records (persistence, read tracking), notification
preferences (per-type, per-channel), streak detection (Redis counters, milestone thresholds),
Postmark email adapter (transactional and broadcast streams), digest compilation (Phase 2),
one-click email unsubscribe, background jobs (`SendEmailJob`, `PushNotificationJob`,
`CompileDigestJob`), WebSocket notification push (Redis pub/sub fan-out).

**What notify:: does NOT own**: Triggering events (owned by source domains — `social::`,
`learn::`, `mkt::`, `method::`, `onboard::`, `iam::`, `safety::`, `billing::`), WebSocket DM delivery
(owned by `social::`), user accounts and family membership (owned by `iam::`), WebSocket
connection management (shared infrastructure via `social::` WebSocket server), content
moderation decisions (owned by `safety::`), account verification and password reset emails
(owned by `iam::` via Ory Kratos built-in email templates `[ARCH §2.3]`).

**What notify:: delegates**: User/family email lookup → `iam::IamService`. Redis pub/sub
for WebSocket distribution → shared infrastructure `[ARCH §2.16]`. Background job
scheduling → sidekiq-rs `[ARCH §12]`. Email template rendering → Postmark server-side
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
| Email delivery: social, marketplace, system notifications configurable | `[S§13.2]` | §7 (`EmailAdapter`), §12 (Postmark), §14 (`SendEmailJob`) |
| Push notifications: architected for future channels without redesign | `[S§13.2]` | §7 (`EmailAdapter` trait), §20 (module structure) |
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
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
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
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
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
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
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

-- Digests: system role only (background job creates and marks sent).
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
- **Query**: `NotificationListParams { cursor?, limit? (default 20, max 100), category?: String, unread_only?: bool }`
- **Response**: `200 OK` → `NotificationListResponse { notifications: Vec<NotificationResponse>, unread_count: i64, next_cursor? }`
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
- **Body**: `{ category?: String }`
- **Response**: `200 OK` → `{ updated_count: i64 }`
- **Side effects**: Sets `is_read = true` on all matching unread notifications for the family.
- **Error codes**: `422` (invalid category)

#### Preferences (2 endpoints)

##### `GET /v1/notifications/preferences`

Return the full type × channel preference matrix with defaults applied.

- **Auth**: `AuthContext` + `FamilyScope`
- **Response**: `200 OK` → `Vec<PreferenceResponse>`
- **Semantics**: Returns one entry per (notification_type, channel) pair. Missing rows are returned as `enabled: true, digest_frequency: "immediate"` (default-enabled). System-critical types include `system_critical: true` flag.
- **Error codes**: `401`

##### `PATCH /v1/notifications/preferences`

Batch upsert notification preferences.

- **Auth**: `AuthContext` + `FamilyScope`
- **Body**: `UpdatePreferencesCommand { preferences: Vec<PreferenceUpdate> }`
- **Validation**: Rejects disabling system-critical types (`content_flagged`, `co_parent_added`, `family_deletion_scheduled`) — returns `422 CannotDisableSystemCritical`.
- **Response**: `200 OK` → `Vec<PreferenceResponse>`
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

The `NotificationService` trait defines all use cases exposed to handlers and event handler
structs. No CQRS separation needed — this is a simple domain with straightforward
read/write paths. `[CODING §8.2]`

```rust
// src/notify/ports.rs

use crate::shared::types::{FamilyId, FamilyScope};
use crate::shared::error::AppError;

#[async_trait]
pub trait NotificationService: Send + Sync {
    // ─── Commands (write, has side effects) ─────────────────────────────

    /// Create an in-app notification and fan out via WebSocket + email.
    /// Called by event handlers, not by HTTP handlers directly.
    async fn create_notification(
        &self,
        cmd: CreateNotificationCommand,
    ) -> Result<(), AppError>;

    /// Mark a single notification as read. Idempotent.
    async fn mark_read(
        &self,
        notification_id: Uuid,
        scope: FamilyScope,
    ) -> Result<(), AppError>;

    /// Bulk mark all (optionally category-filtered) notifications as read.
    async fn mark_all_read(
        &self,
        scope: FamilyScope,
        category: Option<String>,
    ) -> Result<i64, AppError>;

    /// Batch upsert notification preferences. Validates system-critical constraints.
    async fn update_preferences(
        &self,
        cmd: UpdatePreferencesCommand,
        scope: FamilyScope,
    ) -> Result<(), AppError>;

    /// Enqueue a transactional email via Postmark.
    async fn send_email(
        &self,
        cmd: SendEmailCommand,
    ) -> Result<(), AppError>;

    /// Process a signed unsubscribe token and disable the preference.
    async fn process_unsubscribe(
        &self,
        token: &str,
    ) -> Result<(), AppError>;

    // ─── Event handlers (one per subscribed event type) ─────────────────
    // Each method is called by its corresponding DomainEventHandler struct
    // in event_handlers.rs. Failures are logged but do not propagate to
    // the source domain. [ARCH §4.6]

    // social:: events
    async fn handle_friend_request_sent(&self, event: &FriendRequestSent) -> Result<(), AppError>;
    async fn handle_friend_request_accepted(&self, event: &FriendRequestAccepted) -> Result<(), AppError>;
    async fn handle_message_sent(&self, event: &MessageSent) -> Result<(), AppError>;
    async fn handle_event_cancelled(&self, event: &EventCancelled) -> Result<(), AppError>;

    // method:: events
    async fn handle_family_methodology_changed(&self, event: &FamilyMethodologyChanged) -> Result<(), AppError>;

    // onboard:: events
    async fn handle_onboarding_completed(&self, event: &OnboardingCompleted) -> Result<(), AppError>;

    // learn:: events
    async fn handle_activity_logged(&self, event: &ActivityLogged) -> Result<(), AppError>;
    async fn handle_milestone_achieved(&self, event: &MilestoneAchieved) -> Result<(), AppError>;
    async fn handle_book_completed(&self, event: &BookCompleted) -> Result<(), AppError>;
    async fn handle_data_export_ready(&self, event: &DataExportReady) -> Result<(), AppError>;

    // mkt:: events
    async fn handle_purchase_completed(&self, event: &PurchaseCompleted) -> Result<(), AppError>;
    async fn handle_purchase_refunded(&self, event: &PurchaseRefunded) -> Result<(), AppError>;
    async fn handle_creator_onboarded(&self, event: &CreatorOnboarded) -> Result<(), AppError>;

    // safety:: events
    async fn handle_content_flagged(&self, event: &ContentFlagged) -> Result<(), AppError>;

    // iam:: events (Phase 2)
    async fn handle_co_parent_added(&self, event: &CoParentAdded) -> Result<(), AppError>;
    async fn handle_family_deletion_scheduled(&self, event: &FamilyDeletionScheduled) -> Result<(), AppError>;

    // billing:: events (Phase 2)
    async fn handle_subscription_created(&self, event: &SubscriptionCreated) -> Result<(), AppError>;
    async fn handle_subscription_changed(&self, event: &SubscriptionChanged) -> Result<(), AppError>;
    async fn handle_subscription_cancelled(&self, event: &SubscriptionCancelled) -> Result<(), AppError>;
    async fn handle_payout_completed(&self, event: &PayoutCompleted) -> Result<(), AppError>;

    // ─── Queries (read, no side effects) ────────────────────────────────

    /// Paginated notification list with unread count.
    async fn list_notifications(
        &self,
        params: NotificationListParams,
        scope: FamilyScope,
    ) -> Result<NotificationListResponse, AppError>;

    /// Full type × channel preference matrix with defaults applied.
    async fn get_preferences(
        &self,
        scope: FamilyScope,
    ) -> Result<Vec<PreferenceResponse>, AppError>;
}
```

### `NotificationServiceImpl`

```rust
// src/notify/service.rs

pub struct NotificationServiceImpl {
    notification_repo: Arc<dyn NotificationRepository>,
    preference_repo: Arc<dyn PreferenceRepository>,
    digest_repo: Arc<dyn DigestRepository>,      // Phase 2
    email_adapter: Arc<dyn EmailAdapter>,
    iam_service: Arc<dyn IamService>,            // Email lookup
    redis: Arc<RedisPool>,                        // WebSocket pub/sub + streak counters
    job_queue: Arc<dyn JobQueue>,                 // sidekiq-rs
}
```

---

## §6 Repository Interfaces

All notification and preference repositories are family-scoped via `FamilyScope` parameter.
Digest repository is system-scoped (background job access). `[CODING §8.2]`

```rust
// src/notify/ports.rs (continued)

// ─── NotificationRepository ────────────────────────────────────────────
// Family-scoped — all reads and writes are per-family. [00-core §8]
#[async_trait]
pub trait NotificationRepository: Send + Sync {
    async fn create(
        &self,
        cmd: CreateNotification,
    ) -> Result<NotifyNotification, AppError>;

    async fn get_by_id(
        &self,
        notification_id: Uuid,
        scope: FamilyScope,
    ) -> Result<Option<NotifyNotification>, AppError>;

    async fn list(
        &self,
        params: &NotificationListParams,
        scope: FamilyScope,
    ) -> Result<Vec<NotifyNotification>, AppError>;

    async fn count_unread(
        &self,
        scope: FamilyScope,
    ) -> Result<i64, AppError>;

    async fn mark_read(
        &self,
        notification_id: Uuid,
        scope: FamilyScope,
    ) -> Result<bool, AppError>;

    async fn mark_all_read(
        &self,
        scope: FamilyScope,
        category: Option<&str>,
    ) -> Result<i64, AppError>;

    /// Check idempotency: does a notification with this source_event_id already exist?
    async fn exists_by_source_event(
        &self,
        family_id: FamilyId,
        notification_type: &str,
        source_event_id: &str,
    ) -> Result<bool, AppError>;

    /// Cascade delete for family deletion.
    async fn delete_by_family(
        &self,
        family_id: FamilyId,
    ) -> Result<(), AppError>;
}

// ─── PreferenceRepository ──────────────────────────────────────────────
// Family-scoped. Default-enabled semantics: missing row = enabled.
#[async_trait]
pub trait PreferenceRepository: Send + Sync {
    /// Get all explicit preference overrides for a family.
    async fn get_all(
        &self,
        scope: FamilyScope,
    ) -> Result<Vec<NotifyPreference>, AppError>;

    /// Batch upsert preferences (INSERT ON CONFLICT UPDATE).
    async fn upsert_batch(
        &self,
        scope: FamilyScope,
        updates: Vec<PreferenceUpsert>,
    ) -> Result<(), AppError>;

    /// Check if a specific type+channel is enabled for a family.
    /// Returns true if no row exists (default-enabled).
    async fn is_enabled(
        &self,
        family_id: FamilyId,
        notification_type: &str,
        channel: &str,
    ) -> Result<bool, AppError>;

    /// Cascade delete for family deletion.
    async fn delete_by_family(
        &self,
        family_id: FamilyId,
    ) -> Result<(), AppError>;
}

// ─── DigestRepository (Phase 2) ────────────────────────────────────────
// System-scoped — background job creates and processes digests.
#[async_trait]
pub trait DigestRepository: Send + Sync {
    async fn create(
        &self,
        cmd: CreateDigest,
    ) -> Result<NotifyDigest, AppError>;

    async fn get_unsent(
        &self,
        limit: i64,
    ) -> Result<Vec<NotifyDigest>, AppError>;

    async fn mark_sent(
        &self,
        digest_id: Uuid,
    ) -> Result<(), AppError>;
}
```

---

## §7 Adapter Interfaces

### EmailAdapter (Postmark)

The `EmailAdapter` trait wraps email delivery. It is provider-agnostic by name — the only
implementation is `PostmarkEmailAdapter`, but the trait boundary allows testing with a mock
and swapping providers if needed. `[ARCH §2.12]`

```rust
// src/notify/ports.rs (continued)

/// Provider-agnostic email delivery trait.
/// Phase 1: PostmarkEmailAdapter (transactional stream).
/// Phase 2: Adds broadcast stream for digests.
#[async_trait]
pub trait EmailAdapter: Send + Sync {
    /// Send a single transactional email using a Postmark template.
    ///
    /// `template_alias` maps to a Postmark template (e.g., "purchase-receipt").
    /// `template_model` is a JSON object of template variables.
    async fn send_transactional(
        &self,
        to: &str,
        template_alias: &str,
        template_model: serde_json::Value,
    ) -> Result<(), AppError>;

    /// Send up to 500 emails in a single Postmark batch API call.
    /// Used when a single event triggers notifications to multiple families
    /// (e.g., EventCancelled with many RSVPs).
    async fn send_batch(
        &self,
        messages: Vec<BatchEmailMessage>,
    ) -> Result<(), AppError>;

    /// Send a broadcast email (Phase 2 — digest stream).
    /// Uses Postmark's broadcast message stream to protect transactional
    /// deliverability from digest volume. [ARCH §2.12]
    async fn send_broadcast(
        &self,
        to: &str,
        template_alias: &str,
        template_model: serde_json::Value,
    ) -> Result<(), AppError>;
}

/// A single message in a batch send.
pub struct BatchEmailMessage {
    pub to: String,
    pub template_alias: String,
    pub template_model: serde_json::Value,
}
```

---

## §8 Models (DTOs)

### §8.1 Request Types

```rust
// src/notify/models.rs

use serde::Deserialize;

/// Query parameters for GET /v1/notifications
#[derive(Debug, Deserialize)]
pub struct NotificationListParams {
    pub cursor: Option<String>,
    pub limit: Option<u8>,          // Default 20, max 100
    pub category: Option<String>,   // Filter by category
    pub unread_only: Option<bool>,
}

/// Body for PATCH /v1/notifications/preferences
#[derive(Debug, Deserialize)]
pub struct UpdatePreferencesCommand {
    pub preferences: Vec<PreferenceUpdate>,
}

/// A single preference change within a batch update.
#[derive(Debug, Deserialize)]
pub struct PreferenceUpdate {
    pub notification_type: String,
    pub channel: String,              // "in_app" | "email"
    pub enabled: bool,
    pub digest_frequency: Option<String>,  // "immediate" | "daily" | "weekly" | "off"
}
```

### §8.2 Response Types

```rust
use serde::Serialize;

/// A single notification in the feed.
#[derive(Debug, Serialize)]
pub struct NotificationResponse {
    pub id: Uuid,
    pub notification_type: String,
    pub category: String,
    pub title: String,
    pub body: String,
    pub action_url: Option<String>,
    pub is_read: bool,
    pub created_at: DateTime<Utc>,
}

/// Paginated notification list with unread badge count.
#[derive(Debug, Serialize)]
pub struct NotificationListResponse {
    pub notifications: Vec<NotificationResponse>,
    pub unread_count: i64,
    pub next_cursor: Option<String>,
}

/// A single entry in the preference matrix.
#[derive(Debug, Serialize)]
pub struct PreferenceResponse {
    pub notification_type: String,
    pub channel: String,
    pub enabled: bool,
    pub digest_frequency: String,
    pub system_critical: bool,        // true = cannot be disabled
}
```

### §8.3 Internal Types

```rust
/// Internal command to create a notification (used by event handlers).
/// Not exposed via API — event handlers construct this.
#[derive(Debug, Clone)]
pub struct CreateNotificationCommand {
    pub family_id: FamilyId,
    pub notification_type: String,
    pub category: String,
    pub title: String,
    pub body: String,
    pub action_url: Option<String>,
    pub metadata: serde_json::Value,   // Must contain "source_event_id"
}

/// Internal command to send an email (enqueued as SendEmailJob).
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SendEmailCommand {
    pub to: String,
    pub template_alias: String,
    pub template_model: serde_json::Value,
}

/// WebSocket frame pushed to connected clients.
#[derive(Debug, Serialize)]
pub struct WebSocketFrame {
    pub msg_type: String,              // "notification"
    pub data: NotificationResponse,
}

/// All registered notification types.
/// Defined as constants rather than an enum to allow easy iteration and
/// lookup without match exhaustiveness overhead in a simple domain.
pub mod notification_types {
    // Social
    pub const FRIEND_REQUEST_SENT: &str = "friend_request_sent";
    pub const FRIEND_REQUEST_ACCEPTED: &str = "friend_request_accepted";
    pub const MESSAGE_RECEIVED: &str = "message_received";
    pub const EVENT_CANCELLED: &str = "event_cancelled";

    // Learning
    pub const METHODOLOGY_CHANGED: &str = "methodology_changed";
    pub const ONBOARDING_COMPLETED: &str = "onboarding_completed";
    pub const ACTIVITY_STREAK: &str = "activity_streak";
    pub const MILESTONE_ACHIEVED: &str = "milestone_achieved";
    pub const BOOK_COMPLETED: &str = "book_completed";
    pub const DATA_EXPORT_READY: &str = "data_export_ready";

    // Marketplace
    pub const PURCHASE_COMPLETED: &str = "purchase_completed";
    pub const PURCHASE_REFUNDED: &str = "purchase_refunded";
    pub const CREATOR_ONBOARDED: &str = "creator_onboarded";
    pub const PAYOUT_COMPLETED: &str = "payout_completed";

    // System
    pub const CONTENT_FLAGGED: &str = "content_flagged";
    pub const CO_PARENT_ADDED: &str = "co_parent_added";
    pub const FAMILY_DELETION_SCHEDULED: &str = "family_deletion_scheduled";

    // Billing (Phase 2)
    pub const SUBSCRIPTION_CREATED: &str = "subscription_created";
    pub const SUBSCRIPTION_CHANGED: &str = "subscription_changed";
    pub const SUBSCRIPTION_CANCELLED: &str = "subscription_cancelled";

    /// Types that cannot be disabled via preferences. [S§13.3]
    pub const SYSTEM_CRITICAL: &[&str] = &[
        CONTENT_FLAGGED,
        CO_PARENT_ADDED,
        FAMILY_DELETION_SCHEDULED,
    ];
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
used by `create_notification`. System-critical types cannot be disabled.

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

Activity streaks are detected by the `handle_activity_logged` event handler using Redis
counters — not computed from the database.

```
ActivityLogged event received
  → INCR notify:streak:{student_id}:{activity_date}  (SET with NX, expire 48h)
  → Check consecutive days via Redis: get keys for last N days
  → If streak count ∈ {7, 14, 30, 60, 100}:
      → create_notification(activity_streak) with streak days in metadata
      → Publish MilestoneAchieved event (delegated to learn:: to decide)
```

**Redis key pattern**: `notify:streak:{student_id}:{YYYY-MM-DD}` with 48-hour TTL.
The gap tolerance (48h TTL vs checking consecutive dates) ensures timezone edge cases
don't break streaks.

### Extensibility

Adding a new notification type requires:
1. Add the type string to the CHECK constraint (append-only migration)
2. Add constant to `notification_types` module
3. Add to type registry table (category, title template, system-critical flag)
4. Implement the event handler in `event_handlers.rs`
5. Add Postmark template alias if email delivery is needed

No existing code needs modification — the dispatch is type-string-driven, not match-arm-driven.

---

## §10 Delivery Pipeline (Domain Deep-Dive 2)

### Event-to-Notification Flow

```
┌──────────────┐
│ Source Domain │
│ publishes     │
│ DomainEvent   │
└──────┬───────┘
       │
       ▼
┌──────────────────────────────────────────────────────────┐
│ notify::event_handlers.rs                                 │
│                                                           │
│  DomainEventHandler<E>::handle()                          │
│    → notification_service.handle_<event_type>()           │
└──────────────────────┬───────────────────────────────────┘
                       │
                       ▼
┌──────────────────────────────────────────────────────────┐
│ notify::service.rs — handle_<event_type>() impl           │
│                                                           │
│  1. Build CreateNotificationCommand from event payload    │
│  2. Check idempotency (source_event_id)                   │
│  3. Call create_notification(cmd)                          │
└──────────────────────┬───────────────────────────────────┘
                       │
                       ▼
┌──────────────────────────────────────────────────────────┐
│ create_notification(cmd)                                  │
│                                                           │
│  ┌─ In-App Path ──────────────────────────────────────┐  │
│  │ 1. Check preference: is_enabled(type, "in_app")?    │  │
│  │ 2. INSERT into notify_notifications                  │  │
│  │ 3. Redis PUBLISH notifications:{family_id}           │  │
│  │    → WebSocket push to connected clients             │  │
│  └─────────────────────────────────────────────────────┘  │
│                                                           │
│  ┌─ Email Path ───────────────────────────────────────┐  │
│  │ 1. Check preference: is_enabled(type, "email")?     │  │
│  │ 2. If digest_frequency == "immediate":               │  │
│  │    → Look up family email via iam::IamService        │  │
│  │    → Enqueue SendEmailJob to sidekiq-rs              │  │
│  │ 3. If digest_frequency ∈ {"daily", "weekly"}:        │  │
│  │    → Skip (CompileDigestJob handles it — Phase 2)    │  │
│  └─────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────┘
```

### Key Pipeline Properties

**In-app and email are independent paths**: Failure in email delivery does not prevent in-app
notification creation. Both paths check preferences independently — a family can have email
disabled but in-app enabled for a given type.

**Idempotency**: Every `CreateNotificationCommand` includes `metadata.source_event_id` (typically
the source entity's UUID). The unique index on `(family_id, notification_type, source_event_id)`
prevents duplicate notifications. If a duplicate is detected, `create_notification` returns `Ok(())`
silently.

**Error isolation**: Event handler failures are logged at `error!` level but do not propagate to
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
┌──────────────┐      ┌──────────────┐      ┌──────────────────────────┐
│ notify::      │      │ Redis        │      │ WebSocket Server         │
│ service.rs    │─────▶│ PUBLISH      │─────▶│ (shared with social::)   │
│               │      │ channel:     │      │                          │
│ create_       │      │ notifications│      │ Delivers to all          │
│ notification  │      │ :{family_id} │      │ connections for family   │
└──────────────┘      └──────────────┘      └──────────────────────────┘
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
// - "message" → social:: DM handling
// - "notification" → invalidate notification queries

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
iam_service.get_family_primary_email(family_id) -> Result<(String, String), AppError>
//                                                         email   display_name
```

This is the only cross-domain service call notify:: makes. All other interactions are via
domain events.

### SendEmailJob Structure

```rust
/// Background job payload for transactional email delivery. [ARCH §12]
#[derive(Debug, Serialize, Deserialize)]
pub struct SendEmailJob {
    pub to: String,
    pub template_alias: String,
    pub template_model: serde_json::Value,
    /// For deduplication: Postmark ignores duplicate sends with same MessageStream
    /// + Tag within a short window. We also use this for logging correlation.
    pub idempotency_key: String,
}
```

**Queue**: Default (process within 5 minutes) `[ARCH §12]`

**Retry policy**: 3 retries with exponential backoff (30s, 2m, 10m). After 3 failures,
the job moves to the dead letter queue for manual inspection.

**Failure handling**: Email delivery failures are logged at `error!` level with the
idempotency key and template alias. The in-app notification is unaffected — it was already
created before the job was enqueued.

### CAN-SPAM Compliance

Every transactional email includes: `[S§13.3]`

1. **`List-Unsubscribe` header**: Points to `GET /v1/notifications/unsubscribe?token=<signed>`
2. **`List-Unsubscribe-Post` header**: Supports RFC 8058 one-click unsubscribe
3. **Physical mailing address**: In Postmark template footer (managed in dashboard)
4. **Honest subject lines**: Template subjects match notification titles — no deceptive content

---

## §13 User Preferences & CAN-SPAM Compliance (Domain Deep-Dive 5)

### Preference Matrix

The full matrix has `16 notification types × 2 channels = 32 cells`. The GET preferences
endpoint returns all 32 entries with defaults applied (missing row = enabled).

```
                          in_app    email
                         ──────── ────────
 friend_request_sent     │  ✓   │   ✓   │
 friend_request_accepted │  ✓   │   ✓   │
 message_received        │  ✓   │   ✓   │
 event_cancelled         │  ✓   │   ✓   │
 methodology_changed     │  ✓   │   ✓   │
 onboarding_completed    │  ✓   │   ✓   │
 activity_streak         │  ✓   │   ✓   │
 milestone_achieved      │  ✓   │   ✓   │
 book_completed          │  ✓   │   ✓   │
 data_export_ready       │  ✓   │   ✓   │
 purchase_completed      │  ✓   │   ✓   │
 purchase_refunded       │  ✓   │   ✓   │
 creator_onboarded       │  ✓   │   ✓   │
 content_flagged         │  🔒  │   🔒  │  ← system-critical
 co_parent_added         │  🔒  │   🔒  │  ← system-critical
 family_deletion_sched.  │  🔒  │   🔒  │  ← system-critical

 ✓ = enabled (default), toggleable
 🔒 = always enabled, cannot be disabled
```

### Default-Enabled Semantics

The `notify_preferences` table only stores **deviations** from the default. A family that
has never changed any preferences has zero rows in this table, and all notifications are
delivered. This means:

- `is_enabled(family_id, type, channel)` → `SELECT enabled FROM notify_preferences WHERE family_id = $1 AND notification_type = $2 AND channel = $3` → if no row, return `true`
- New notification types are automatically enabled for all families without migration

### System-Critical Override

Three notification types are hardcoded as system-critical in `notification_types::SYSTEM_CRITICAL`:

```rust
pub const SYSTEM_CRITICAL: &[&str] = &[
    CONTENT_FLAGGED,
    CO_PARENT_ADDED,
    FAMILY_DELETION_SCHEDULED,
];
```

**Enforcement points**:
1. `update_preferences()` — returns `NotifyError::CannotDisableSystemCritical` (HTTP 422) if any preference update attempts to set `enabled = false` for a system-critical type
2. `process_unsubscribe()` — returns an error page for system-critical types
3. `create_notification()` — skips preference check entirely for system-critical types (always delivers both in-app and email)

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
| `immediate` | 1 | Email sent immediately via `SendEmailJob` |
| `off` | 1 | Email suppressed entirely |
| `daily` | 2 | Notification accumulated; `CompileDigestJob` sends at 6:00 AM UTC |
| `weekly` | 2 | Notification accumulated; `CompileDigestJob` sends Monday 6:00 AM UTC |

In Phase 1, `daily` and `weekly` values are accepted and stored in preferences but behave
identically to `immediate` (no digest compilation job exists yet). This ensures preferences
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

## §14 Background Jobs (Domain Deep-Dive 6)

### SendEmailJob (Default Queue, Phase 1)

| Property | Value |
|----------|-------|
| **Queue** | `default` (process within 5 minutes) `[ARCH §12]` |
| **Payload** | `SendEmailJob { to, template_alias, template_model, idempotency_key }` |
| **Retry policy** | 3 retries, exponential backoff: 30s → 2m → 10m |
| **Dead letter** | After 3 failures, job moves to dead letter queue |
| **Idempotency** | Postmark deduplicates by `MessageStream` + `Tag` within a window; `idempotency_key` is set as the Postmark `Tag` header |

```rust
// src/notify/jobs.rs

#[derive(Debug, Serialize, Deserialize)]
pub struct SendEmailJob {
    pub to: String,
    pub template_alias: String,
    pub template_model: serde_json::Value,
    pub idempotency_key: String,
}

#[async_trait]
impl Job for SendEmailJob {
    const QUEUE: &'static str = "default";
    const MAX_RETRIES: u32 = 3;

    async fn perform(&self, ctx: &JobContext) -> Result<(), JobError> {
        let adapter = ctx.get::<Arc<dyn EmailAdapter>>()?;
        adapter.send_transactional(
            &self.to,
            &self.template_alias,
            self.template_model.clone(),
        ).await.map_err(|e| {
            tracing::error!(
                idempotency_key = %self.idempotency_key,
                template = %self.template_alias,
                "Email delivery failed: {e}"
            );
            JobError::Retryable(e.to_string())
        })
    }
}
```

### PushNotificationJob (Default Queue, Phase 1 inline → Phase 2 as job)

In Phase 1, WebSocket push is performed inline within `create_notification()` (Redis PUBLISH
is fast and non-blocking). In Phase 2, if push notification delivery to mobile devices is
added, this becomes a proper background job.

| Property | Value |
|----------|-------|
| **Queue** | `default` |
| **Phase 1** | Inline Redis PUBLISH (not a background job) |
| **Phase 2+** | Background job for mobile push delivery (APNs/FCM) |

### CompileDigestJob (Low Queue, Phase 2)

| Property | Value |
|----------|-------|
| **Queue** | `low` (process within 1 hour) `[ARCH §12]` |
| **Schedule** | Daily at 6:00 AM UTC, weekly on Mondays at 6:00 AM UTC `[ARCH §12]` |
| **Payload** | `CompileDigestJob { digest_type: "daily" | "weekly" }` |

```rust
#[derive(Debug, Serialize, Deserialize)]
pub struct CompileDigestJob {
    pub digest_type: String,  // "daily" | "weekly"
}

#[async_trait]
impl Job for CompileDigestJob {
    const QUEUE: &'static str = "low";
    const MAX_RETRIES: u32 = 2;

    async fn perform(&self, ctx: &JobContext) -> Result<(), JobError> {
        // See §15 for compilation algorithm
        todo!("Phase 2 implementation")
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
CompileDigestJob::perform(digest_type):
  1. period = calculate_period(digest_type)
       daily:  (yesterday 6:00 AM UTC, today 6:00 AM UTC)
       weekly: (last Monday 6:00 AM UTC, this Monday 6:00 AM UTC)

  2. families = preference_repo.get_families_with_digest(digest_type)
       → SELECT DISTINCT family_id FROM notify_preferences
         WHERE digest_frequency = $1 AND channel = 'email' AND enabled = true

  3. FOR EACH family_id IN families (batched, 100 at a time):
       a. notifications = notification_repo.list_in_period(family_id, period)
       b. IF notifications.is_empty() → SKIP (empty digest suppression)
       c. content_json = compile_digest_content(notifications)
            → Group by category, sort by created_at DESC within each group
       d. digest_repo.create({ family_id, digest_type, period, content_json })
       e. email = iam_service.get_family_primary_email(family_id)
       f. job_queue.enqueue(SendEmailJob {
              to: email,
              template_alias: format!("{digest_type}-digest"),
              template_model: content_json,
              idempotency_key: format!("digest:{family_id}:{period_start}"),
          })
            → Uses BROADCAST stream (not transactional)
       g. digest_repo.mark_sent(digest_id)
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

All notification errors use `thiserror` and map to HTTP status codes via `AppError`. Internal
details are logged but never exposed in API responses. `[CODING §5.2, S§18]`

```rust
// src/notify/errors.rs

use thiserror::Error;

#[derive(Debug, Error)]
pub enum NotifyError {
    // ─── Notification Errors ─────────────────────────────────────────────
    #[error("Notification not found")]
    NotificationNotFound,

    #[error("Notification not owned by this family")]
    NotificationNotOwned,

    // ─── Preference Errors ───────────────────────────────────────────────
    #[error("Cannot disable system-critical notification type")]
    CannotDisableSystemCritical,

    #[error("Invalid notification type")]
    InvalidNotificationType,

    #[error("Invalid delivery channel")]
    InvalidChannel,

    #[error("Invalid digest frequency")]
    InvalidDigestFrequency,

    // ─── Unsubscribe Errors ──────────────────────────────────────────────
    #[error("Invalid or expired unsubscribe token")]
    InvalidUnsubscribeToken,

    // ─── Adapter Errors ──────────────────────────────────────────────────
    #[error("Email delivery failed")]
    EmailDeliveryFailed,

    // ─── Infrastructure ──────────────────────────────────────────────────
    #[error("Database error")]
    DatabaseError(#[from] sea_orm::DbErr),

    #[error("Redis error")]
    RedisError(String),

    #[error("Email adapter error")]
    EmailAdapterError(String),
}
```

### Error-to-HTTP Mapping

| Error Variant | HTTP Status | Response Code |
|--------------|-------------|---------------|
| `NotificationNotFound` | `404 Not Found` | `notification_not_found` |
| `NotificationNotOwned` | `404 Not Found` | `notification_not_found` (never reveal existence) |
| `CannotDisableSystemCritical` | `422 Unprocessable` | `cannot_disable_system_critical` |
| `InvalidNotificationType` | `422 Unprocessable` | `invalid_notification_type` |
| `InvalidChannel` | `422 Unprocessable` | `invalid_channel` |
| `InvalidDigestFrequency` | `422 Unprocessable` | `invalid_digest_frequency` |
| `InvalidUnsubscribeToken` | `400 Bad Request` | `invalid_unsubscribe_token` |
| `EmailDeliveryFailed` | `502 Bad Gateway` | `email_delivery_failed` |
| `DatabaseError` | `500 Internal` | `internal_error` (no details exposed) |
| `RedisError` | `500 Internal` | `internal_error` (no details exposed) |
| `EmailAdapterError` | `500 Internal` | `internal_error` (no details exposed) |

**Security note**: `NotificationNotOwned` maps to `404` (not `403`) to prevent enumeration.
Callers cannot distinguish "does not exist" from "exists but belongs to another family".
`[S§18]`

---

## §17 Cross-Domain Interactions

### §17.1 notify:: Provides (consumed by other domains)

**Nothing.** `notify::` is a **pure consumer / terminal domain**. It subscribes to events
from other domains but publishes no events and exposes no service trait for other domains
to call. This is by design — notification delivery is a side effect, not a dependency.

No domain should import anything from `notify::`. The dependency graph is strictly one-way.

### §17.2 notify:: Consumes

| Dependency | Source | Purpose |
|-----------|--------|---------|
| `AuthContext` | `iam::` middleware | User identity on every request `[00-core §7.2]` |
| `FamilyScope` | `iam::` middleware | Family-scoped data access `[00-core §8]` |
| `IamService::get_family_primary_email()` | `iam::` | Email address lookup for transactional email personalization |
| `EmailAdapter` | Postmark (self-hosted adapter) | Transactional + broadcast email delivery `[ARCH §2.12]` |
| Redis pub/sub | shared infrastructure | WebSocket notification push `[ARCH §2.16]` |
| Redis counters | shared infrastructure | Streak detection counters |
| sidekiq-rs | shared infrastructure | Background job scheduling `[ARCH §12]` |

### §17.3 Events notify:: Subscribes To

`notify::` subscribes to 20 domain events from 8 source domains. Each event maps to a
handler struct in `src/notify/event_handlers.rs`. `[ARCH §4.6]`

```rust
// src/notify/event_handlers.rs

use crate::social::events::{
    FriendRequestSent, FriendRequestAccepted, MessageSent, EventCancelled,
};
use crate::method::events::FamilyMethodologyChanged;
use crate::onboard::events::OnboardingCompleted;
use crate::learn::events::{
    ActivityLogged, MilestoneAchieved, BookCompleted, DataExportReady,
};
use crate::mkt::events::{PurchaseCompleted, PurchaseRefunded, CreatorOnboarded};
use crate::safety::events::ContentFlagged;
use crate::iam::events::{CoParentAdded, FamilyDeletionScheduled};
use crate::billing::events::{
    SubscriptionCreated, SubscriptionChanged, SubscriptionCancelled, PayoutCompleted,
};

// ─── social:: events ─────────────────────────────────────────────────────

pub struct FriendRequestSentHandler {
    notification_service: Arc<dyn NotificationService>,
}

#[async_trait]
impl DomainEventHandler<FriendRequestSent> for FriendRequestSentHandler {
    async fn handle(&self, event: &FriendRequestSent) -> Result<(), AppError> {
        self.notification_service.handle_friend_request_sent(event).await
    }
}

pub struct FriendRequestAcceptedHandler {
    notification_service: Arc<dyn NotificationService>,
}

#[async_trait]
impl DomainEventHandler<FriendRequestAccepted> for FriendRequestAcceptedHandler {
    async fn handle(&self, event: &FriendRequestAccepted) -> Result<(), AppError> {
        self.notification_service.handle_friend_request_accepted(event).await
    }
}

pub struct MessageSentHandler {
    notification_service: Arc<dyn NotificationService>,
}

#[async_trait]
impl DomainEventHandler<MessageSent> for MessageSentHandler {
    async fn handle(&self, event: &MessageSent) -> Result<(), AppError> {
        self.notification_service.handle_message_sent(event).await
    }
}

pub struct EventCancelledHandler {
    notification_service: Arc<dyn NotificationService>,
}

#[async_trait]
impl DomainEventHandler<EventCancelled> for EventCancelledHandler {
    async fn handle(&self, event: &EventCancelled) -> Result<(), AppError> {
        self.notification_service.handle_event_cancelled(event).await
    }
}

// ─── method:: events ─────────────────────────────────────────────────────

pub struct FamilyMethodologyChangedHandler {
    notification_service: Arc<dyn NotificationService>,
}

#[async_trait]
impl DomainEventHandler<FamilyMethodologyChanged> for FamilyMethodologyChangedHandler {
    async fn handle(&self, event: &FamilyMethodologyChanged) -> Result<(), AppError> {
        self.notification_service.handle_family_methodology_changed(event).await
    }
}

// ─── onboard:: events ────────────────────────────────────────────────────

pub struct OnboardingCompletedHandler {
    notification_service: Arc<dyn NotificationService>,
}

#[async_trait]
impl DomainEventHandler<OnboardingCompleted> for OnboardingCompletedHandler {
    async fn handle(&self, event: &OnboardingCompleted) -> Result<(), AppError> {
        self.notification_service.handle_onboarding_completed(event).await
    }
}

// ─── learn:: events ──────────────────────────────────────────────────────

pub struct ActivityLoggedHandler {
    notification_service: Arc<dyn NotificationService>,
}

#[async_trait]
impl DomainEventHandler<ActivityLogged> for ActivityLoggedHandler {
    async fn handle(&self, event: &ActivityLogged) -> Result<(), AppError> {
        self.notification_service.handle_activity_logged(event).await
    }
}

pub struct MilestoneAchievedHandler {
    notification_service: Arc<dyn NotificationService>,
}

#[async_trait]
impl DomainEventHandler<MilestoneAchieved> for MilestoneAchievedHandler {
    async fn handle(&self, event: &MilestoneAchieved) -> Result<(), AppError> {
        self.notification_service.handle_milestone_achieved(event).await
    }
}

pub struct BookCompletedHandler {
    notification_service: Arc<dyn NotificationService>,
}

#[async_trait]
impl DomainEventHandler<BookCompleted> for BookCompletedHandler {
    async fn handle(&self, event: &BookCompleted) -> Result<(), AppError> {
        self.notification_service.handle_book_completed(event).await
    }
}

pub struct DataExportReadyHandler {
    notification_service: Arc<dyn NotificationService>,
}

#[async_trait]
impl DomainEventHandler<DataExportReady> for DataExportReadyHandler {
    async fn handle(&self, event: &DataExportReady) -> Result<(), AppError> {
        self.notification_service.handle_data_export_ready(event).await
    }
}

// ─── mkt:: events ────────────────────────────────────────────────────────

pub struct PurchaseCompletedHandler {
    notification_service: Arc<dyn NotificationService>,
}

#[async_trait]
impl DomainEventHandler<PurchaseCompleted> for PurchaseCompletedHandler {
    async fn handle(&self, event: &PurchaseCompleted) -> Result<(), AppError> {
        self.notification_service.handle_purchase_completed(event).await
    }
}

pub struct PurchaseRefundedHandler {
    notification_service: Arc<dyn NotificationService>,
}

#[async_trait]
impl DomainEventHandler<PurchaseRefunded> for PurchaseRefundedHandler {
    async fn handle(&self, event: &PurchaseRefunded) -> Result<(), AppError> {
        self.notification_service.handle_purchase_refunded(event).await
    }
}

pub struct CreatorOnboardedHandler {
    notification_service: Arc<dyn NotificationService>,
}

#[async_trait]
impl DomainEventHandler<CreatorOnboarded> for CreatorOnboardedHandler {
    async fn handle(&self, event: &CreatorOnboarded) -> Result<(), AppError> {
        self.notification_service.handle_creator_onboarded(event).await
    }
}

// ─── safety:: events ─────────────────────────────────────────────────────

pub struct ContentFlaggedHandler {
    notification_service: Arc<dyn NotificationService>,
}

#[async_trait]
impl DomainEventHandler<ContentFlagged> for ContentFlaggedHandler {
    async fn handle(&self, event: &ContentFlagged) -> Result<(), AppError> {
        self.notification_service.handle_content_flagged(event).await
    }
}

// ─── iam:: events (Phase 2) ──────────────────────────────────────────────

pub struct CoParentAddedHandler {
    notification_service: Arc<dyn NotificationService>,
}

#[async_trait]
impl DomainEventHandler<CoParentAdded> for CoParentAddedHandler {
    async fn handle(&self, event: &CoParentAdded) -> Result<(), AppError> {
        self.notification_service.handle_co_parent_added(event).await
    }
}

pub struct FamilyDeletionScheduledHandler {
    notification_service: Arc<dyn NotificationService>,
}

#[async_trait]
impl DomainEventHandler<FamilyDeletionScheduled> for FamilyDeletionScheduledHandler {
    async fn handle(&self, event: &FamilyDeletionScheduled) -> Result<(), AppError> {
        self.notification_service.handle_family_deletion_scheduled(event).await
    }
}

// ─── billing:: events (Phase 2) ─────────────────────────────────────────

pub struct SubscriptionCreatedHandler {
    notification_service: Arc<dyn NotificationService>,
}

#[async_trait]
impl DomainEventHandler<SubscriptionCreated> for SubscriptionCreatedHandler {
    async fn handle(&self, event: &SubscriptionCreated) -> Result<(), AppError> {
        self.notification_service.handle_subscription_created(event).await
    }
}

pub struct SubscriptionChangedHandler {
    notification_service: Arc<dyn NotificationService>,
}

#[async_trait]
impl DomainEventHandler<SubscriptionChanged> for SubscriptionChangedHandler {
    async fn handle(&self, event: &SubscriptionChanged) -> Result<(), AppError> {
        self.notification_service.handle_subscription_changed(event).await
    }
}

pub struct SubscriptionCancelledHandler {
    notification_service: Arc<dyn NotificationService>,
}

#[async_trait]
impl DomainEventHandler<SubscriptionCancelled> for SubscriptionCancelledHandler {
    async fn handle(&self, event: &SubscriptionCancelled) -> Result<(), AppError> {
        self.notification_service.handle_subscription_cancelled(event).await
    }
}

pub struct PayoutCompletedHandler {
    notification_service: Arc<dyn NotificationService>,
}

#[async_trait]
impl DomainEventHandler<PayoutCompleted> for PayoutCompletedHandler {
    async fn handle(&self, event: &PayoutCompleted) -> Result<(), AppError> {
        self.notification_service.handle_payout_completed(event).await
    }
}
```

### §17.4 Event Handler Detail

Each event handler follows the same pattern: build a `CreateNotificationCommand` from the
event payload, then call `create_notification()`. Special cases are noted below.

| Event | Target Family | Special Behavior |
|-------|--------------|------------------|
| `FriendRequestSent` | `target_family_id` | — |
| `FriendRequestAccepted` | `requester_family_id` | — |
| `MessageSent` | `recipient_family_id` | Only if recipient has no active WebSocket (offline check via Redis) |
| `EventCancelled` | Each `going_family_ids[i]` | Batch: one notification per RSVP'd family; batch email via `send_batch()` |
| `FamilyMethodologyChanged` | `family_id` | — |
| `OnboardingCompleted` | `family_id` | Skip if `skipped == true` (no welcome email for skipped onboarding) |
| `ActivityLogged` | `family_id` | No in-app notification created directly; only triggers streak check (§9) |
| `MilestoneAchieved` | `family_id` | — |
| `BookCompleted` | `family_id` | Also triggers streak check for reading-specific milestones |
| `DataExportReady` | `family_id` | Email includes signed download URL with expiration |
| `PurchaseCompleted` | `family_id` | Email includes purchase receipt details from `content_metadata` |
| `PurchaseRefunded` | `family_id` | — |
| `CreatorOnboarded` | Looked up via `parent_id` | Uses `iam::IamService` to resolve `parent_id → family_id` |
| `ContentFlagged` | Content owner's `family_id` | System-critical: always delivered regardless of preferences |
| `CoParentAdded` (Phase 2) | `family_id` | System-critical: always delivered |
| `FamilyDeletionScheduled` (Phase 2) | `family_id` | System-critical: always delivered; includes cancellation URL |
| `SubscriptionCreated` (Phase 2) | `family_id` | Email includes tier and billing interval |
| `SubscriptionChanged` (Phase 2) | `family_id` | Email includes `change_type` (interval_change, renewal, reactivation) |
| `SubscriptionCancelled` (Phase 2) | `family_id` | Email includes `effective_at` date; mentions data preservation |
| `PayoutCompleted` (Phase 2) | Looked up via `creator_id` | Uses `iam::IamService` to resolve `creator_id → family_id`; email includes amount and currency |

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
- `SendEmailJob` background job with retry policy
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
- `CompileDigestJob` (daily at 6:00 AM UTC, weekly Monday 6:00 AM UTC)
- Postmark broadcast stream for digest delivery
- `CoParentAdded` and `FamilyDeletionScheduled` event handlers
- `SubscriptionCreated`, `SubscriptionChanged`, `SubscriptionCancelled`, `PayoutCompleted` event handlers (`billing::` events)
- Digest frequency preferences become functional (daily/weekly actually batch)
- Notification retention purge (delete notifications older than 90 days)
- Digest preview endpoint
- **1 additional endpoint, 6 additional event handlers**

### Phase 3+ — Scale `[S§19]`

**In scope**:
- Mobile push notifications (APNs/FCM) via `PushNotificationJob`
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

11. `GET /v1/notifications/preferences` returns all 32 cells (16 types × 2 channels)
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

25. `create_notification()` publishes to `notifications:{family_id}` Redis channel
26. WebSocket frame format matches `{ msg_type: "notification", data: NotificationResponse }`
27. Multiple connected clients for the same family all receive the push

### Email Delivery

28. `SendEmailJob` calls `email_adapter.send_transactional()` with correct template alias and variables
29. Failed email delivery retries 3 times with exponential backoff
30. Email includes `List-Unsubscribe` header with signed token URL
31. `GET /v1/notifications/unsubscribe?token=` with valid token disables the preference
32. `GET /v1/notifications/unsubscribe?token=` with expired or invalid token returns `400`

### Error Handling

33. Internal errors (DB, Redis, email adapter) return `500` with generic message — no details exposed
34. Email delivery failures return `502 Bad Gateway`
35. All `NotifyError` variants map to documented HTTP status codes

---

## §20 Module Structure

```
src/notify/
├── mod.rs                    # Re-exports, domain-level doc comments
├── handlers.rs               # 5 Axum route handlers (thin layer only)
├── service.rs                # NotificationServiceImpl — event handling,
│                             # preference checks, delivery orchestration
├── repository.rs             # PgNotificationRepo, PgPreferenceRepo,
│                             # PgDigestRepo (Phase 2)
├── models.rs                 # Request/response types, notification_types
│                             # constants, WebSocketFrame
├── ports.rs                  # NotificationService trait, all repository
│                             # traits, EmailAdapter trait
├── errors.rs                 # NotifyError thiserror enum
├── event_handlers.rs         # 20 DomainEventHandler structs (one per
│                             # subscribed event type) [ARCH §4.6]
├── jobs.rs                   # SendEmailJob, PushNotificationJob (Phase 2),
│                             # CompileDigestJob (Phase 2) [ARCH §12]
├── adapters/
│   ├── mod.rs
│   └── postmark.rs           # PostmarkEmailAdapter — wraps Postmark HTTP
│                             # API, returns domain types only [ARCH §2.12]
└── entities/                 # SeaORM-generated — never hand-edit [CODING §6.3]
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
├── templates/
│   ├── welcome.html          # Postmark template (HTML + text)
│   ├── purchase-receipt.html
│   └── ...
├── layouts/
│   └── base.html             # Shared header/footer with brand
└── sync.ts                   # Script to sync templates to Postmark API
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

```rust
// Webhook handler for Postmark bounce notifications
pub async fn handle_postmark_bounce(
    &self,
    bounce: PostmarkBounceWebhook,
) -> Result<(), AppError> {
    match bounce.bounce_type.as_str() {
        // Hard bounce: email address is invalid
        "HardBounce" => {
            // Mark email as undeliverable in notify_email_status
            self.repo.mark_email_undeliverable(&bounce.email).await?;
            // After 3 hard bounces, suppress all emails to this address
            let count = self.repo.count_hard_bounces(&bounce.email).await?;
            if count >= 3 {
                self.repo.suppress_email(&bounce.email).await?;
                tracing::warn!(
                    email = %bounce.email,
                    "Email suppressed after 3 hard bounces"
                );
            }
        }
        // Spam complaint: user marked email as spam
        "SpamComplaint" => {
            // Immediately suppress all emails to this address (CAN-SPAM)
            self.repo.suppress_email(&bounce.email).await?;
            tracing::warn!(
                email = %bounce.email,
                "Email suppressed due to spam complaint"
            );
        }
        // Soft bounce: temporary issue (mailbox full, etc.)
        _ => {
            // Log but don't suppress — Postmark handles retries
            tracing::info!(
                email = %bounce.email,
                bounce_type = %bounce.bounce_type,
                "Soft bounce received"
            );
        }
    }
    Ok(())
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

Before sending any email, the `SendEmailJob` checks `notify_email_status` — if the
email is suppressed, the job completes without sending (logged but not delivered).
