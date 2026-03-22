# Domain Spec 13 — Recommendations & Signals (recs::)

## §1 Overview

The Recommendations & Signals domain is the **recommendation engine and signal pipeline** — it
receives learning signals from other domains via domain events, computes methodology-constrained
recommendations using rule-based heuristics, and serves them to premium families. It is a terminal
consumer domain: it subscribes to events from `learn::`, `mkt::`, `iam::`, and `method::` but
publishes no domain events of its own. `[S§10, V§8, V§10]`

> **Naming note**: This domain is intentionally named "Recommendations & Signals," not "AI."
> Phases 1-2 contain **zero machine learning** — all recommendation logic is rule-based heuristics,
> SQL aggregations, and hand-tuned scoring weights. Calling this "AI" would be misleading. Actual
> AI/ML capabilities (collaborative filtering, semantic search, learning-to-rank) are documented
> in the separate companion spec `AI_FUTURE.md` for Phase 3-4.

| Attribute | Value |
|-----------|-------|
| **Module path** | `internal/recs/` |
| **DB prefix** | `recs_` `[ARCH §5.1]` |
| **Complexity class** | Simple (no `domain/` subdirectory) — recommendation queries, no invariants to enforce `[ARCH §4.5]` |
| **CQRS** | Yes — write (signal recording) / read (recommendation queries) `[ARCH §4.7]` |
| **External adapter** | None — in-house rule-based engine; no external ML service or API |
| **Key constraint** | Premium-only (`RequirePremium` extractor — 402 for free tier) `[S§10.1]`; every user-data query family-scoped via `FamilyScope` `[CODING §2.4, §2.5]`; methodology-constrained output `[S§10.1]`; no PII in aggregated data `[S§10.3, S§17.2]` |

**What recs:: owns**: Signal recording from domain events, pre-computed recommendation
generation (daily batch), recommendation feedback (dismiss/block), family recommendation
preferences, cross-family popularity aggregation (anonymized), anonymized interaction data
collection for future AI/ML use, the scoring algorithm and all scoring weights.

**What recs:: does NOT own**: Starter curriculum recommendations (owned by `onboard::` —
curated Phase 1 content) `[S§6.5, 04-onboard]`, methodology definitions and tool registry
(owned by `method::`), marketplace listings and purchase flow (owned by `mkt::`), learning
activity data (owned by `learn::`), student profiles and family accounts (owned by `iam::`),
tool-adjacent parent education (owned by `learn::` and `method::`) `[S§8.4]`, notification
delivery (owned by `notify::`), search indexing (owned by `search::`).

**What recs:: delegates**: Notification delivery → `notify::` (via domain events — Phase 3).
Family/student identity resolution → `iam::` tables (direct DB read). Methodology config
lookup → `method::` tables (direct DB read). Background task scheduling → asynq
`[ARCH §12]`.

---

## §2 Requirements Traceability

Every SPEC.md §10 requirement maps to a section in this document. Cross-references from
other spec sections are included where the recommendations domain is involved.

| Requirement | SPEC Reference | Domain Spec Section |
|-------------|---------------|---------------------|
| Recommendation engine inputs: methodology, age, history, purchases, community signals | `[S§10.1]` | §10 (algorithm inputs) |
| Recommendations constrained by family's methodology selection | `[S§10.1]` | §10.1 (methodology constraining) |
| Recommendation outputs: marketplace, activity, reading, community suggestions | `[S§10.1]` | §10.2 (recommendation types) |
| Recommendations are a premium feature | `[S§10.1]` | §4 (all endpoints use `RequirePremium`) |
| Community popularity among methodology-matched families | `[S§10.2]` | §10.4 (popularity scoring) |
| Seasonal appropriateness | `[S§10.2]` | §10.5 (seasonal scoring) |
| Progress gaps (e.g., no math in two weeks) | `[S§10.2]` | §10.3 (progress gap detection) |
| Age/grade transitions | `[S§10.2]` | §10.6 (age transition detection) |
| Source signal labeling on every suggestion | `[S§10.2]` | §13.1 (transparency), §8 (`source_label` field) |
| Anonymized interaction data collection for future AI | `[S§10.3]` | §14 (anonymized data collection) |
| Data collection complies with privacy model | `[S§10.3, S§17.2]` | §14 (HMAC anonymization, no PII) |
| No filter bubbles | `[S§10.4]` | §10.7 (filter bubble prevention), §13.4 |
| Transparent recommendation logic | `[S§10.4]` | §13.1 (transparency — `source_label` + `source_signal`) |
| Parent authority: dismiss, block, adjust | `[S§10.4]` | §4 (dismiss/block/undo endpoints), §13.2 (parental control) |
| Content neutrality — no worldview-based favoritism | `[S§10.4]` | §10.8 (content neutrality enforcement), §13.5 |
| AI-generated suggestions clearly labeled | `[S§10.4]` | §13.1 (`is_suggestion: true` on all responses) |
| Methodology constrains recommendation engine outputs | `[S§4.4]` | §10.1 (methodology constraining) |
| Starter curriculum recommendations (curated, age-appropriate) | `[S§6.5]` | N/A — owned by `onboard::` `[04-onboard]` |
| Tool-adjacent parent education | `[S§8.4]` | N/A — owned by `learn::` and `method::` |
| AI recommendations draw from marketplace catalog | `[S§9.7]` | §10.2 (`marketplace_content` type — candidates from `mkt_listings`) |
| Anonymized aggregated usage data flows to AI | `[S§18.5]` | §9 (signal pipeline), §14 (anonymized data) |
| No PII in aggregated datasets | `[S§18.5]` | §14 (privacy invariant: no family_id/student_id) |
| Graceful degradation — recommendations unavailable doesn't break core | `[S§17.5]` | §15 (error handling — all errors return graceful fallback) |
| Phase 1: no AI recommendations | `[S§19 Phase 1]` | §17 (Phase 1: signal recording only, 0 API endpoints) |
| Phase 2: methodology-constrained recommendations | `[S§19 Phase 2]` | §17 (Phase 2: full recommendation engine) |
| Phase 3: advanced AI, data collection for future tutoring | `[S§19 Phase 3]` | §17 (Phase 3), `AI_FUTURE.md` |
| Phase 4: advanced AI tutoring | `[S§19 Phase 4]` | `AI_FUTURE.md` (separate companion spec) |

> **Coverage note on `[S§6.5]` (starter curriculum recommendations)**: SPEC.md §6.5 covers
> curated starter curriculum recommendations during onboarding. These are owned by `onboard::`
> (Phase 1 curated content), not `recs::`. The `recs::` domain provides *personalized*
> recommendations based on accumulated learning signals (Phase 2+).

> **Coverage note on `[S§8.4]` (tool-adjacent parent education)**: SPEC.md §8.4 covers
> contextual methodology-specific guidance within learning tools. This is owned by `learn::`
> (tool integration) and `method::` (philosophy modules, mastery paths), not `recs::`.

---

## §3 Database Schema

The recommendations domain stores learning signals (family-scoped, 90-day retention),
pre-computed recommendations (family-scoped, 14-day TTL), feedback (family-scoped, persistent
for blocks), cross-family popularity scores (NOT family-scoped — aggregated, no PII), anonymized
interaction data (NOT family-scoped — HMAC-anonymized, no PII), and family preferences
(family-scoped). All user-data tables are family-scoped via `family_id` foreign key.
`[ARCH §5.1, ARCH §5.2]`

> **Refinement note**: ARCHITECTURE.md §5.1 sketches `ai_signals` and `ai_recommendations`.
> This spec renames the prefix to `recs_` and adds: (1) `recs_recommendation_feedback` for
> dismiss/block actions, (2) `recs_popularity_scores` for cross-family aggregated popularity,
> (3) `recs_anonymized_interactions` for COPPA-safe data collection `[S§10.3]`,
> (4) `recs_preferences` for family recommendation settings, (5) comprehensive CHECK
> constraints for all status/type columns.

### §3.1 Enums

Implemented as `CHECK` constraints (not PostgreSQL ENUM types) per `[CODING §4.1]`:

```sql
-- Signal type, recommendation type, recommendation status, source signal,
-- feedback action, and exploration frequency are all enforced via CHECK
-- constraints on their respective columns rather than as PostgreSQL enum types.
-- This avoids ALTER TYPE limitations when adding new values in future migrations.
-- [ARCH §5.2]
--
-- Signal type values: activity_logged, book_completed, purchase_completed
-- Recommendation type values: marketplace_content, activity_idea, reading_suggestion, community_group
-- Recommendation status values: active, dismissed, blocked, expired
-- Source signal values: methodology_match, popularity, seasonal, progress_gap,
--                       age_transition, purchase_history, reading_history, exploration
-- Feedback action values: dismiss, block
-- Exploration frequency values: off, occasional, frequent
```

### §3.2 Tables

#### `recs_signals` — Raw Event-Derived Learning Signals

Family-scoped. Records each learning signal derived from domain events. 90-day retention
(purged by `PurgeStaleSignalsTask`).

```sql
CREATE TABLE recs_signals (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id       UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    student_id      UUID REFERENCES iam_students(id) ON DELETE SET NULL,
    signal_type     TEXT NOT NULL CHECK (signal_type IN (
                        'activity_logged', 'book_completed', 'purchase_completed'
                    )),
    -- Denormalized methodology snapshot at signal time. If the family changes
    -- methodology later, old signals retain the old methodology (correct: the
    -- signal reflects context at time of activity). [method:: direct DB read]
    methodology_id  UUID NOT NULL,
    -- Signal-specific payload. Schema varies by signal_type:
    --   activity_logged:    { subject_tags: string[], duration_minutes: int }
    --   book_completed:     { title: string, reading_item_id: uuid }
    --   purchase_completed: { listing_id: uuid, content_type: string }
    payload         JSONB NOT NULL DEFAULT '{}',
    signal_date     DATE NOT NULL DEFAULT CURRENT_DATE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Primary query: "all signals for this family in the last N days"
CREATE INDEX idx_recs_signals_family_date
    ON recs_signals (family_id, signal_date DESC);

-- Purge task: "delete signals older than 90 days"
CREATE INDEX idx_recs_signals_created_at
    ON recs_signals (created_at);

-- Signal type filtering for algorithm queries
CREATE INDEX idx_recs_signals_family_type
    ON recs_signals (family_id, signal_type);

-- FK index: student deletion cascade (ON DELETE SET NULL)
CREATE INDEX idx_recs_signals_student
    ON recs_signals (student_id) WHERE student_id IS NOT NULL;
```

#### `recs_recommendations` — Pre-Computed Recommendations

Family-scoped. Stores the output of the daily recommendation batch task. Each recommendation
has a 14-day TTL and is expired by the batch task on next run.

```sql
CREATE TABLE recs_recommendations (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id           UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    -- Optional: student-specific recommendation (NULL = family-wide)
    student_id          UUID REFERENCES iam_students(id) ON DELETE CASCADE,
    recommendation_type TEXT NOT NULL CHECK (recommendation_type IN (
                            'marketplace_content', 'activity_idea',
                            'reading_suggestion', 'community_group'
                        )),
    -- The recommended entity's ID (listing_id, group_id, etc.)
    target_entity_id    UUID NOT NULL,
    -- Human-readable label explaining the entity (e.g., listing title)
    target_entity_label TEXT NOT NULL,
    -- Which signal produced this recommendation
    source_signal       TEXT NOT NULL CHECK (source_signal IN (
                            'methodology_match', 'popularity', 'seasonal',
                            'progress_gap', 'age_transition', 'purchase_history',
                            'reading_history', 'exploration'
                        )),
    -- User-facing explanation (e.g., "Popular with Charlotte Mason families")
    source_label        TEXT NOT NULL,
    -- Composite score from the scoring algorithm [§10.9]
    score               REAL NOT NULL DEFAULT 0.0,
    status              TEXT NOT NULL DEFAULT 'active' CHECK (status IN (
                            'active', 'dismissed', 'blocked', 'expired'
                        )),
    expires_at          TIMESTAMPTZ NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Primary query: "active recommendations for this family"
CREATE INDEX idx_recs_recommendations_family_status
    ON recs_recommendations (family_id, status) WHERE status = 'active';

-- Student-specific query: "active recommendations for this student"
CREATE INDEX idx_recs_recommendations_student_status
    ON recs_recommendations (student_id, status) WHERE status = 'active';

-- Dedup: prevent duplicate active recommendations for the same entity
CREATE UNIQUE INDEX idx_recs_recommendations_family_entity_active
    ON recs_recommendations (family_id, target_entity_id)
    WHERE status = 'active';

-- Expiry task: "find recommendations past their TTL"
CREATE INDEX idx_recs_recommendations_expires_at
    ON recs_recommendations (expires_at) WHERE status = 'active';
```

#### `recs_recommendation_feedback` — Parent Dismiss/Block Actions

Family-scoped. Records parent feedback on recommendations. Persists after the recommendation
expires — blocked sources stay blocked indefinitely until the parent explicitly unblocks them.

```sql
CREATE TABLE recs_recommendation_feedback (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id           UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    recommendation_id   UUID NOT NULL REFERENCES recs_recommendations(id) ON DELETE CASCADE,
    action              TEXT NOT NULL CHECK (action IN ('dismiss', 'block')),
    -- For blocks: the entity that is blocked (so future recommendations
    -- for this entity are suppressed even after the original expires)
    blocked_entity_id   UUID,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- One feedback per recommendation
CREATE UNIQUE INDEX idx_recs_feedback_recommendation
    ON recs_recommendation_feedback (recommendation_id);

-- Query: "all blocked entities for this family" (used by algorithm to filter candidates)
CREATE INDEX idx_recs_feedback_family_blocks
    ON recs_recommendation_feedback (family_id, blocked_entity_id)
    WHERE action = 'block';

-- Family deletion cascade
CREATE INDEX idx_recs_feedback_family
    ON recs_recommendation_feedback (family_id);
```

#### `recs_popularity_scores` — Cross-Family Aggregated Popularity

**NOT family-scoped** (no PII). Stores aggregated purchase popularity per listing per
methodology. Powers the "Popular with [Methodology] families" signal. Computed by the
`AggregatePopularityTask` using a rolling 90-day window with recency decay.

```sql
CREATE TABLE recs_popularity_scores (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    listing_id      UUID NOT NULL,
    methodology_id  UUID NOT NULL,
    -- Rolling window start (e.g., 90 days ago from computation time)
    period_start    DATE NOT NULL,
    period_end      DATE NOT NULL,
    -- Weighted purchase count (recency-decayed)
    popularity_score REAL NOT NULL DEFAULT 0.0,
    -- Raw purchase count in the window
    purchase_count  INTEGER NOT NULL DEFAULT 0,
    computed_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Unique: one score per listing per methodology per period
CREATE UNIQUE INDEX idx_recs_popularity_listing_method_period
    ON recs_popularity_scores (listing_id, methodology_id, period_start);

-- Query: "top listings for this methodology"
CREATE INDEX idx_recs_popularity_method_score
    ON recs_popularity_scores (methodology_id, popularity_score DESC);

-- Purge: remove old period windows
CREATE INDEX idx_recs_popularity_period
    ON recs_popularity_scores (period_end);
```

#### `recs_anonymized_interactions` — COPPA-Safe Anonymized Data

**NOT family-scoped** (no PII). Contains HMAC-anonymized interaction data for future AI/ML
training `[S§10.3]`. This table MUST NEVER contain `family_id`, `student_id`, or any
personally identifiable information.

```sql
CREATE TABLE recs_anonymized_interactions (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    -- One-way HMAC-SHA256(family_id, server_secret) -> anonymous_id
    -- Cannot be reversed to recover family_id [§14]
    anonymous_id        TEXT NOT NULL,
    interaction_type    TEXT NOT NULL CHECK (interaction_type IN (
                            'activity_logged', 'book_completed', 'purchase_completed'
                        )),
    methodology_slug    TEXT NOT NULL,
    -- Coarsened to 3-year ranges: '4-6', '7-9', '10-12', '13-15', '16-18'
    age_band            TEXT NOT NULL CHECK (age_band IN (
                            '4-6', '7-9', '10-12', '13-15', '16-18'
                        )),
    -- Broad subject category only (not specific tags)
    subject_category    TEXT,
    -- Rounded to nearest 5 minutes
    duration_minutes    INTEGER,
    interaction_date    DATE NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Analytics: query by methodology and interaction type
CREATE INDEX idx_recs_anon_methodology_type
    ON recs_anonymized_interactions (methodology_slug, interaction_type);

-- Analytics: query by age band
CREATE INDEX idx_recs_anon_age_band
    ON recs_anonymized_interactions (age_band);
```

#### `recs_preferences` — Family Recommendation Preferences

Family-scoped. Stores per-family recommendation settings. One row per family (created on
first access with defaults).

```sql
CREATE TABLE recs_preferences (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id               UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    -- Which recommendation types are enabled (all by default)
    enabled_types           TEXT[] NOT NULL DEFAULT ARRAY[
                                'marketplace_content', 'activity_idea',
                                'reading_suggestion', 'community_group'
                            ],
    -- How often to surface content outside typical patterns [S§10.4]
    exploration_frequency   TEXT NOT NULL DEFAULT 'occasional' CHECK (
                                exploration_frequency IN ('off', 'occasional', 'frequent')
                            ),
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- One preference row per family
CREATE UNIQUE INDEX idx_recs_preferences_family
    ON recs_preferences (family_id);
```

### §3.3 Row-Level Security

Application-layer enforcement (not PostgreSQL RLS policies). `[CODING §2.4]`

| Table | Scope | Enforcement |
|-------|-------|-------------|
| `recs_signals` | Family-scoped | `FamilyScope` on all repository methods |
| `recs_recommendations` | Family-scoped | `FamilyScope` on all repository methods |
| `recs_recommendation_feedback` | Family-scoped | `FamilyScope` on all repository methods |
| `recs_popularity_scores` | System (no PII) | No `FamilyScope` — accessed by background tasks and algorithm only |
| `recs_anonymized_interactions` | System (no PII) | No `FamilyScope` — insert-only by background task, no reads in Phase 2 |
| `recs_preferences` | Family-scoped | `FamilyScope` on all repository methods |

---

## §4 API Endpoints

### Phase 1: No API Endpoints

Phase 1 builds the signal recording infrastructure only. The `recs::` module has event
handlers but no HTTP endpoints. `[S§19 Phase 1]`

### Phase 2: Recommendation API (7 endpoints)

All endpoints require `RequirePremium` — returns 402 Payment Required for free-tier
families. `[S§10.1, 00-core §13.3]`

#### `GET /v1/recommendations`

Returns the family's active recommendations, sorted by score descending. Supports filtering
by type and cursor pagination.

```
Auth:       RequirePremium (-> 402 if free tier)
Query:      type?: "marketplace_content" | "activity_idea" | "reading_suggestion" | "community_group"
            cursor?: string
            limit?: integer (default 20, max 50)
Response:   200 -> RecommendationListResponse
            401 -> Unauthorized
            402 -> PremiumRequired
```

```json
// 200 Response
{
  "recommendations": [
    {
      "id": "uuid",
      "recommendation_type": "marketplace_content",
      "target_entity_id": "uuid",
      "target_entity_label": "Charlotte Mason Nature Study Bundle",
      "source_signal": "methodology_match",
      "source_label": "Matches your Charlotte Mason methodology",
      "score": 0.87,
      "is_suggestion": true,
      "student_id": null,
      "created_at": "2026-03-21T04:00:00Z",
      "expires_at": "2026-04-04T04:00:00Z"
    }
  ],
  "next_cursor": "eyJ..."
}
```

#### `GET /v1/recommendations/students/:student_id`

Returns recommendations specific to a student (filtered by age, grade, subjects).

```
Auth:       RequirePremium (-> 402 if free tier)
Path:       student_id: UUID
Query:      type?: "marketplace_content" | "activity_idea" | "reading_suggestion" | "community_group"
            cursor?: string
            limit?: integer (default 20, max 50)
Validate:   student_id must belong to the family (FamilyScope check)
Response:   200 -> RecommendationListResponse
            401 -> Unauthorized
            402 -> PremiumRequired
            404 -> StudentNotFound (student doesn't exist or doesn't belong to family)
```

#### `POST /v1/recommendations/:id/dismiss`

Dismisses a recommendation. The recommendation becomes hidden but can be restored via
`DELETE /v1/recommendations/:id/feedback`.

```
Auth:       RequirePremium (-> 402 if free tier)
Path:       id: UUID (recommendation_id)
Validate:   recommendation must belong to the family (FamilyScope check)
Response:   200 -> { "status": "dismissed" }
            401 -> Unauthorized
            402 -> PremiumRequired
            404 -> RecommendationNotFound
            409 -> AlreadyDismissedOrBlocked
```

#### `POST /v1/recommendations/:id/block`

Blocks the recommendation's source entity. Future recommendations for this entity will be
suppressed. Can be restored via `DELETE /v1/recommendations/:id/feedback`.

```
Auth:       RequirePremium (-> 402 if free tier)
Path:       id: UUID (recommendation_id)
Validate:   recommendation must belong to the family (FamilyScope check)
Response:   200 -> { "status": "blocked", "blocked_entity_id": "uuid" }
            401 -> Unauthorized
            402 -> PremiumRequired
            404 -> RecommendationNotFound
            409 -> AlreadyBlocked
```

#### `DELETE /v1/recommendations/:id/feedback`

Undoes a dismiss or block action. Restores the recommendation to active status (if not
expired). For blocks, removes the blocked entity from the suppress list.

```
Auth:       RequirePremium (-> 402 if free tier)
Path:       id: UUID (recommendation_id)
Validate:   recommendation must belong to the family (FamilyScope check)
Response:   200 -> { "status": "active" }
            401 -> Unauthorized
            402 -> PremiumRequired
            404 -> FeedbackNotFound
```

#### `GET /v1/recommendations/preferences`

Returns the family's recommendation preferences.

```
Auth:       RequirePremium (-> 402 if free tier)
Response:   200 -> RecommendationPreferencesResponse
            401 -> Unauthorized
            402 -> PremiumRequired
```

```json
// 200 Response
{
  "enabled_types": ["marketplace_content", "activity_idea", "reading_suggestion", "community_group"],
  "exploration_frequency": "occasional"
}
```

#### `PATCH /v1/recommendations/preferences`

Updates the family's recommendation preferences. Partial update — only provided fields are
changed.

```
Auth:       RequirePremium (-> 402 if free tier)
Body:       UpdatePreferencesCommand (partial)
Response:   200 -> RecommendationPreferencesResponse
            401 -> Unauthorized
            402 -> PremiumRequired
            422 -> ValidationError (invalid type or frequency value)
```

```json
// Request body
{
  "enabled_types": ["marketplace_content", "reading_suggestion"],
  "exploration_frequency": "frequent"
}
```

---

## §5 Service Interface

Defined in `internal/recs/ports.go`. CQRS separation: command methods (write) and query methods
(read) are grouped separately. `[ARCH §4.7]`

```go
// internal/recs/ports.go

package recs

import (
    "context"
    "time"

    "github.com/google/uuid"
    "github.com/homegrown-academy/homegrown-academy/internal/shared/types"
)

// RecsService is the primary service interface for the Recommendations & Signals domain.
// Injected into handlers via RecsService interface in AppState.
type RecsService interface {
    // ── Commands (write side) ──────────────────────────────────

    // RecordSignal records a learning signal from a domain event.
    // Called by event handlers, not by HTTP handlers.
    RecordSignal(ctx context.Context, command RecordSignalCommand) error

    // RegisterListing registers a newly published listing in the popularity catalog.
    // Called by ListingPublishedHandler.
    RegisterListing(ctx context.Context, command RegisterListingCommand) error

    // DismissRecommendation dismisses a recommendation (marks as dismissed, creates feedback record).
    DismissRecommendation(ctx context.Context, scope *types.FamilyScope, recommendationID uuid.UUID) error

    // BlockRecommendation blocks a recommendation's source entity (marks as blocked, creates feedback
    // with blocked_entity_id for future suppression).
    BlockRecommendation(ctx context.Context, scope *types.FamilyScope, recommendationID uuid.UUID) error

    // UndoFeedback undoes a dismiss or block action.
    UndoFeedback(ctx context.Context, scope *types.FamilyScope, recommendationID uuid.UUID) error

    // UpdatePreferences updates family recommendation preferences.
    UpdatePreferences(ctx context.Context, scope *types.FamilyScope, command UpdatePreferencesCommand) (*RecommendationPreferencesResponse, error)

    // ── Queries (read side) ────────────────────────────────────

    // GetRecommendations returns active recommendations for the family, filterable by type.
    GetRecommendations(ctx context.Context, scope *types.FamilyScope, params RecommendationListParams) (*RecommendationListResponse, error)

    // GetStudentRecommendations returns active recommendations for a specific student.
    GetStudentRecommendations(ctx context.Context, scope *types.FamilyScope, params StudentRecommendationParams) (*RecommendationListResponse, error)

    // GetPreferences returns the family's recommendation preferences (or defaults if none set).
    GetPreferences(ctx context.Context, scope *types.FamilyScope) (*RecommendationPreferencesResponse, error)

    // ── Lifecycle event handlers ───────────────────────────────

    // HandleFamilyDeletion deletes all recs data for the family.
    HandleFamilyDeletion(ctx context.Context, familyID types.FamilyID) error

    // InvalidateMethodologyCache invalidates cached methodology config (e.g., methodology definitions cache).
    InvalidateMethodologyCache(ctx context.Context) error
}
```

**Implementation**: `RecsServiceImpl` in `internal/recs/service.go` with:
- Injected repositories: `SignalRepository`, `RecommendationRepository`, `FeedbackRepository`,
  `PopularityRepository`, `PreferenceRepository`, `AnonymizedInteractionRepository`
- Direct DB connection (via `*gorm.DB`) for cross-domain reads from `iam_families`,
  `iam_students`, `method_definitions`, `mkt_listings`, `soc_groups`

---

## §6 Repository Interfaces

Defined in `internal/recs/ports.go` alongside the service interface. Each repository handles one
table or closely related table group. FamilyScope is required on all methods that access
user data. `[CODING §2.4, §2.5]`

```go
// internal/recs/ports.go (continued)

// SignalRepository is the repository for recs_signals table.
type SignalRepository interface {
    // Create creates a new signal record.
    Create(ctx context.Context, signal NewSignal) error

    // FindByFamily finds signals for a family within a date range.
    FindByFamily(ctx context.Context, scope *types.FamilyScope, since time.Time) ([]Signal, error)

    // DeleteByFamily deletes all signals for a family (family deletion cascade).
    DeleteByFamily(ctx context.Context, familyID types.FamilyID) (int64, error)

    // DeleteStale deletes signals older than the retention period (purge task).
    DeleteStale(ctx context.Context, before time.Time) (int64, error)
}

// RecommendationRepository is the repository for recs_recommendations table.
type RecommendationRepository interface {
    // CreateBatch creates a batch of recommendations (daily task output).
    CreateBatch(ctx context.Context, recommendations []NewRecommendation) (int64, error)

    // FindActiveByFamily finds active recommendations for a family, with optional type filter.
    FindActiveByFamily(ctx context.Context, scope *types.FamilyScope, recommendationType *string, cursor *string, limit int64) ([]Recommendation, *string, error)

    // FindActiveByStudent finds active recommendations for a specific student.
    FindActiveByStudent(ctx context.Context, scope *types.FamilyScope, studentID uuid.UUID, recommendationType *string, cursor *string, limit int64) ([]Recommendation, *string, error)

    // UpdateStatus updates recommendation status (dismiss, block, expire, restore).
    UpdateStatus(ctx context.Context, scope *types.FamilyScope, recommendationID uuid.UUID, status string) error

    // ExpireStale expires recommendations past their TTL.
    ExpireStale(ctx context.Context) (int64, error)

    // DeleteByFamily deletes all recommendations for a family (family deletion cascade).
    DeleteByFamily(ctx context.Context, familyID types.FamilyID) (int64, error)
}

// FeedbackRepository is the repository for recs_recommendation_feedback table.
type FeedbackRepository interface {
    // Create creates a feedback record (dismiss or block).
    Create(ctx context.Context, feedback NewFeedback) error

    // FindByRecommendation finds feedback for a specific recommendation.
    FindByRecommendation(ctx context.Context, scope *types.FamilyScope, recommendationID uuid.UUID) (*Feedback, error)

    // FindBlockedByFamily finds all blocked entity IDs for a family (used by algorithm to exclude).
    FindBlockedByFamily(ctx context.Context, scope *types.FamilyScope) ([]uuid.UUID, error)

    // Delete deletes a feedback record (undo dismiss/block).
    Delete(ctx context.Context, scope *types.FamilyScope, recommendationID uuid.UUID) error
}

// PopularityRepository is the repository for recs_popularity_scores table.
// NOT family-scoped — operates on cross-family aggregated data (no PII).
type PopularityRepository interface {
    // Upsert upserts a popularity score for a listing-methodology-period combination.
    Upsert(ctx context.Context, score NewPopularityScore) error

    // FindByMethodology finds top listings by popularity for a methodology.
    FindByMethodology(ctx context.Context, methodologyID uuid.UUID, limit int64) ([]PopularityScore, error)

    // DeleteStale deletes popularity scores for expired periods.
    DeleteStale(ctx context.Context, before time.Time) (int64, error)
}

// PreferenceRepository is the repository for recs_preferences table.
type PreferenceRepository interface {
    // FindOrDefault finds preferences for a family, or returns defaults if none exist.
    FindOrDefault(ctx context.Context, scope *types.FamilyScope) (*Preferences, error)

    // Upsert creates or updates preferences for a family.
    Upsert(ctx context.Context, scope *types.FamilyScope, preferences UpdatePreferences) (*Preferences, error)
}

// AnonymizedInteractionRepository is the repository for recs_anonymized_interactions table.
// NOT family-scoped — insert-only, anonymized data (no PII).
type AnonymizedInteractionRepository interface {
    // CreateBatch batch-inserts anonymized interaction records (weekly task output).
    CreateBatch(ctx context.Context, interactions []NewAnonymizedInteraction) (int64, error)
}
```

---

## §7 Adapter Interface

N/A — the Recommendations & Signals domain has no external adapter. All recommendation logic
is an in-house rule-based engine implemented in `internal/recs/algorithm.go`. There is no external
ML service, embedding API, or third-party recommendation provider in Phases 1-2.

See `AI_FUTURE.md` for Phase 3-4 infrastructure requirements when actual ML capabilities
are introduced.

---

## §8 Models (DTOs)

Defined in `internal/recs/models.go`. All types use struct tags (`json:"field"`) and appropriate
swaggo annotations. `[CODING §3.2]`

### Request Types

```go
// internal/recs/models.go

package recs

import (
    "time"

    "github.com/google/uuid"
    "github.com/homegrown-academy/homegrown-academy/internal/shared/types"
)

// RecommendationListParams holds query parameters for GET /v1/recommendations.
type RecommendationListParams struct {
    Type   *string `query:"type"`   // e.g., "marketplace_content"
    Cursor *string `query:"cursor"`
    Limit  *int64  `query:"limit"`  // Default 20, max 50
}

// StudentRecommendationParams holds query parameters for GET /v1/recommendations/students/:student_id.
type StudentRecommendationParams struct {
    StudentID uuid.UUID `param:"student_id"`
    Type      *string   `query:"type"`   // e.g., "marketplace_content"
    Cursor    *string   `query:"cursor"`
    Limit     *int64    `query:"limit"`  // Default 20, max 50
}

// UpdatePreferencesCommand is the request body for PATCH /v1/recommendations/preferences.
type UpdatePreferencesCommand struct {
    EnabledTypes          []string `json:"enabled_types,omitempty"`
    ExplorationFrequency  *string  `json:"exploration_frequency,omitempty"`
}
```

### Response Types

```go
// RecommendationListResponse is the response for recommendation list endpoints.
type RecommendationListResponse struct {
    Recommendations []RecommendationResponse `json:"recommendations"`
    NextCursor      *string                  `json:"next_cursor,omitempty"`
}

// RecommendationResponse is a single recommendation in a list response.
type RecommendationResponse struct {
    ID                 uuid.UUID  `json:"id"`
    RecommendationType string     `json:"recommendation_type"`
    TargetEntityID     uuid.UUID  `json:"target_entity_id"`
    TargetEntityLabel  string     `json:"target_entity_label"`
    SourceSignal       string     `json:"source_signal"`
    // SourceLabel is a user-facing explanation (e.g., "Popular with Charlotte Mason families").
    SourceLabel        string     `json:"source_label"`
    Score              float32    `json:"score"`
    // IsSuggestion is always true — all recommendations are automated suggestions [S§10.4]
    IsSuggestion       bool       `json:"is_suggestion"`
    StudentID          *uuid.UUID `json:"student_id,omitempty"`
    CreatedAt          time.Time  `json:"created_at"`
    ExpiresAt          time.Time  `json:"expires_at"`
}

// RecommendationPreferencesResponse is the response for preferences endpoints.
type RecommendationPreferencesResponse struct {
    EnabledTypes         []string `json:"enabled_types"`
    ExplorationFrequency string   `json:"exploration_frequency"`
}
```

### Internal Types (not exposed via API)

```go
// RecordSignalCommand is a command to record a signal from a domain event.
type RecordSignalCommand struct {
    FamilyID      types.FamilyID
    StudentID     *uuid.UUID
    SignalType    SignalType
    MethodologyID uuid.UUID
    Payload       map[string]any
    SignalDate    time.Time
}

// RegisterListingCommand is a command to register a listing in the popularity catalog.
type RegisterListingCommand struct {
    ListingID   uuid.UUID
    PublisherID uuid.UUID
    ContentType string
    SubjectTags []string
}

// SignalType represents signal types derived from domain events.
type SignalType string

const (
    SignalActivityLogged    SignalType = "activity_logged"
    SignalBookCompleted     SignalType = "book_completed"
    SignalPurchaseCompleted SignalType = "purchase_completed"
)

// String returns the string representation of the signal type.
func (s SignalType) String() string {
    return string(s)
}

// RecommendationType represents recommendation types.
type RecommendationType string

const (
    RecommendationMarketplaceContent RecommendationType = "marketplace_content"
    RecommendationActivityIdea       RecommendationType = "activity_idea"
    RecommendationReadingSuggestion  RecommendationType = "reading_suggestion"
    RecommendationCommunityGroup     RecommendationType = "community_group"
)

// String returns the string representation of the recommendation type.
func (r RecommendationType) String() string {
    return string(r)
}

// SourceSignalType represents source signals that explain why a recommendation was made.
type SourceSignalType string

const (
    SourceMethodologyMatch SourceSignalType = "methodology_match"
    SourcePopularity       SourceSignalType = "popularity"
    SourceSeasonal         SourceSignalType = "seasonal"
    SourceProgressGap      SourceSignalType = "progress_gap"
    SourceAgeTransition    SourceSignalType = "age_transition"
    SourcePurchaseHistory  SourceSignalType = "purchase_history"
    SourceReadingHistory   SourceSignalType = "reading_history"
    SourceExploration      SourceSignalType = "exploration"
)

// String returns the string representation of the source signal.
func (s SourceSignalType) String() string {
    return string(s)
}

// ExplorationFrequency represents exploration frequency preference.
type ExplorationFrequency string

const (
    ExplorationOff        ExplorationFrequency = "off"
    ExplorationOccasional ExplorationFrequency = "occasional"
    ExplorationFrequent   ExplorationFrequency = "frequent"
)

// String returns the string representation of the exploration frequency.
func (e ExplorationFrequency) String() string {
    return string(e)
}

// ExplorationRatio returns the percentage of recommendation slots reserved for exploration.
func (e ExplorationFrequency) ExplorationRatio() float32 {
    switch e {
    case ExplorationOff:
        return 0.0
    case ExplorationOccasional:
        return 0.10
    case ExplorationFrequent:
        return 0.25
    default:
        return 0.10
    }
}
```

---

## §9 Signal Processing Pipeline (Deep Dive 1)

### §9.1 Event Flow

```
Domain Event -> EventBus -> recs::event_handlers -> RecsService.RecordSignal -> SignalRepository.Create
```

Each domain event is mapped to a signal type and recorded with the family's current
methodology snapshot. The event handlers are thin — they extract the relevant fields from
the event and construct a `RecordSignalCommand`.

### §9.2 Signal Recording Per Event Type

| Source Event | Source Domain | Signal Type | Family-Scoped? | Additional Action |
|-------------|-------------|-------------|----------------|-------------------|
| `ActivityLogged` | `learn::` | `activity_logged` | Yes | None |
| `BookCompleted` | `learn::` | `book_completed` | Yes | None |
| `PurchaseCompleted` | `mkt::` | `purchase_completed` | Yes | None |
| `ListingPublished` | `mkt::` | N/A (no signal) | No | Updates `recs_popularity_scores` catalog |

### §9.3 Methodology Resolution

When recording a signal, the handler resolves the family's `primary_methodology_id` from
`iam_families` (direct DB read) and denormalizes it onto the signal record:

```go
// In event handler (simplified)
family, err := db.First(&IamFamily{}, familyID)
if err != nil {
    return fmt.Errorf("looking up family: %w", err)
}
command := RecordSignalCommand{
    FamilyID:      event.FamilyID,
    StudentID:     &event.StudentID,
    SignalType:    SignalActivityLogged,
    MethodologyID: family.PrimaryMethodologyID,
    Payload: map[string]any{
        "subject_tags":     event.SubjectTags,
        "duration_minutes": event.DurationMinutes,
    },
    SignalDate: event.ActivityDate,
}
return service.RecordSignal(ctx, command)
```

**Why snapshot methodology?** If a family switches methodologies, old signals retain the
methodology context in which they were generated. This is correct — an activity done under
Charlotte Mason methodology should not retroactively become a Classical methodology signal.

### §9.4 ListingPublished Handling

`ListingPublished` does not create a `recs_signals` row because it is a catalog-level event,
not a family-scoped learning signal. Instead, it ensures the listing exists in the popularity
catalog so it can receive popularity scores:

```go
// ListingPublishedHandler
func (h *ListingPublishedHandler) Handle(ctx context.Context, event *mktevents.ListingPublished) error {
    return h.recsService.RegisterListing(ctx, RegisterListingCommand{
        ListingID:   event.ListingID,
        PublisherID: event.PublisherID,
        ContentType: event.ContentType,
        SubjectTags: event.SubjectTags,
    })
}
```

The actual popularity score computation happens in the `AggregatePopularityTask` (§11.2),
not in this event handler.

---

## §10 Recommendation Algorithm (Deep Dive 2)

The Phase 2 algorithm is entirely rule-based — hand-tuned scoring weights, SQL aggregations,
and deterministic heuristics. There is no machine learning, no gradient descent, no model
training. The algorithm runs as a daily batch task (`ComputeRecommendationsTask`).

### §10.1 Methodology Constraining

**Every recommendation candidate MUST pass through a methodology filter first.** A family
with Charlotte Mason as their primary methodology MUST NOT receive recommendations for
content tagged with incompatible methodologies (e.g., Traditional textbook packages) unless
the recommendation is an exploration slot (§10.7). `[S§10.1]`

```
Candidate passes if:
  candidate.methodology_ids INTERSECT family.active_methodology_ids != empty
  OR candidate is in an exploration slot (§10.7)
```

**Active methodology IDs** = the family's `primary_methodology_id` plus any
`secondary_methodology_ids` from `iam_families`. Resolved via direct DB read.

The algorithm uses `methodology_id` (UUID), never methodology name strings. There is no
branching on methodology name anywhere in the algorithm. `[CODING §6.2]`

### §10.2 Recommendation Types and Candidate Selection

| Type | `recommendation_type` | Candidate Source | Selection Logic |
|------|----------------------|------------------|-----------------|
| **Marketplace content** | `marketplace_content` | `mkt_listings` (status = 'published') | Methodology-matched listings not already purchased by the family. Subject overlap with family's recent activity tags. |
| **Activity ideas** | `activity_idea` | `method_definitions` → activity templates | Activities from the family's active methodologies that the family hasn't logged recently. Subject gaps detected by §10.3. |
| **Reading suggestions** | `reading_suggestion` | `mkt_listings` (content_type = 'book') + community reading lists | Books methodology-matched, age-appropriate, not already on the family's reading list. |
| **Community groups** | `community_group` | `soc_groups` (visibility = 'public' or 'request_to_join') | Groups tagged with the family's methodology that the family hasn't joined. Location-matched if location available (Phase 3). |

### §10.3 Progress Gap Detection

Identifies subjects where a student's engagement has dropped below expected levels for their
methodology. Uses signal data to detect gaps:

```
For each student in the family:
  For each subject in methodology's expected_subjects:
    recent_activity_count = COUNT(signals WHERE signal_type = 'activity_logged'
                                  AND payload->>'subject_tags' @> [subject]
                                  AND signal_date > now() - interval '14 days')
    IF recent_activity_count < methodology_minimum_frequency:
      -> Generate progress_gap recommendation for this subject
```

**Methodology minimum frequency** is read from `method_definitions` configuration (not
hardcoded). Each methodology defines expected subject engagement cadence.

### §10.4 Popularity Scoring

Pre-computed by `AggregatePopularityTask` (§11.2). The score uses purchase count with
recency decay over a 90-day rolling window:

```
popularity_score = SUM(purchase_weight * recency_decay)

Where:
  purchase_weight = 1.0  (each purchase counts equally)
  recency_decay   = e^(-lambda * days_since_purchase)
  lambda          = 0.03  (half-life ~ 23 days)
```

Popularity is computed **per-methodology** — Charlotte Mason families' purchases affect
Charlotte Mason popularity scores only. This prevents majority-methodology popularity from
drowning out minority methodologies (content neutrality — §10.8).

### §10.5 Seasonal Appropriateness

Maps the current month to a season, then adjusts subject emphasis:

| Season | Months | Emphasized Subjects |
|--------|--------|-------------------|
| Spring | Mar-May | Nature study, gardening, biology, outdoor activities |
| Summer | Jun-Aug | Nature study, art, physical education, field trips |
| Autumn | Sep-Nov | History, literature, harvest themes, back-to-school |
| Winter | Dec-Feb | Indoor crafts, music, reading, science experiments |

**Implementation**: A static lookup table in `algorithm.go`. The seasonal signal provides a
small score boost (not a filter) to recommendations that align with the current season's
emphasized subjects.

> **Phase 3 enhancement**: Location-aware seasons (Southern Hemisphere families have inverted
> seasons). Documented in `AI_FUTURE.md`.

### §10.6 Age/Grade Transition Detection

Detects when a student is approaching a methodology-specific stage transition and surfaces
relevant content:

| Methodology | Transitions |
|-------------|------------|
| Classical | Grammar → Logic (~age 10-11), Logic → Rhetoric (~age 14-15) |
| Charlotte Mason | Form I → Form II (~age 9), Form II → Form III (~age 12), Form III → Form IV (~age 15) |
| Montessori | First Plane → Second Plane (~age 6), Second Plane → Third Plane (~age 12) |
| Waldorf | Early Childhood → Grade School (~age 7), Grade School → High School (~age 14) |

**Implementation**: For each student, compare `iam_students.date_of_birth` against the
methodology's transition ages from `method_definitions`. If a student is within 6 months
of a transition, generate `age_transition` recommendations for content tagged with the
upcoming stage.

Transition ages are read from `method_definitions` configuration, not hardcoded in the
algorithm. `[CODING §6.2]`

### §10.7 Filter Bubble Prevention

SPEC.md §10.4 requires that recommendations do NOT create filter bubbles. The algorithm
reserves a percentage of recommendation slots for **exploration** — content outside the
family's typical patterns. `[S§10.4]`

| `exploration_frequency` | Exploration Slot % | Effect |
|------------------------|-------------------|--------|
| `off` | 0% | No exploration recommendations |
| `occasional` (default) | 10% | 1 in 10 slots is from outside typical patterns |
| `frequent` | 25% | 1 in 4 slots is from outside typical patterns |

**Exploration candidate selection**:
1. Select listings/groups from methodologies the family does NOT currently use
2. Filter to high-popularity items only (popularity_score > 75th percentile) — exploration
   should surface quality content, not random noise
3. Label clearly: `source_signal = "exploration"`, `source_label = "Something different -- popular with [Other Methodology] families"`

### §10.8 Content Neutrality Enforcement

The recommendation engine MUST NOT favor or suppress content based on worldview, religious
affiliation, or methodology preference beyond the user's own selections. `[S§10.4, V§12]`

**Implementation rules**:

1. **worldview_tags excluded from scoring**: The `mkt_listings` table has a `worldview_tags`
   column. The recommendation algorithm MUST NOT read this column. It is used for user-facing
   marketplace browse filters only (owned by `mkt::`).

2. **Per-methodology popularity isolation**: Popularity scores are computed per-methodology
   (`recs_popularity_scores.methodology_id`). A listing's popularity among Charlotte Mason
   families does not influence its score for Classical families. This prevents majority
   methodology preferences from biasing minority methodology recommendations.

3. **No boosting/penalizing by publisher identity**: The algorithm does not consider
   `publisher_id` as a scoring factor. No publisher gets preferential recommendation
   placement.

4. **Audit invariant**: The recommendation algorithm code (`internal/recs/algorithm.go`) MUST
   NOT import or reference `worldview_tags` from any table or struct. This is a testable
   assertion (§18, item 25).

### §10.9 Scoring Formula

Each candidate recommendation receives a composite score:

```
score = (methodology_match * 0.35)
      + (popularity        * 0.25)
      + (relevance         * 0.25)
      + (freshness         * 0.10)
      + (exploration       * 0.05)
```

| Factor | Weight | Computation |
|--------|--------|------------|
| `methodology_match` | 0.35 | 1.0 if candidate matches primary methodology, 0.7 if secondary, 0.0 otherwise |
| `popularity` | 0.25 | Normalized popularity_score (0.0-1.0 range, per-methodology percentile) |
| `relevance` | 0.25 | Subject overlap with family's recent signals (Jaccard similarity on subject tags) |
| `freshness` | 0.10 | Recency of the listing/group (newer = higher, exponential decay over 90 days) |
| `exploration` | 0.05 | 1.0 for exploration slots, 0.0 otherwise (ensures exploration items aren't always last) |

**Top-N selection**: After scoring, the algorithm selects the top 50 recommendations per
family (configurable), deduplicating by `target_entity_id`, and inserts them into
`recs_recommendations` with a 14-day TTL.

---

## §11 Background Tasks (Deep Dive 3)

All background tasks run via asynq `[ARCH §12]`. Phase 2 introduces 4 tasks.

### §11.1 ComputeRecommendationsTask

**Schedule**: Daily at 4:00 AM UTC (after `AggregatePopularityTask` completes).

**Process**:
1. Query all families with `subscription_tier = 'premium'` from `iam_families`
2. For each premium family:
   a. Load family's active methodology IDs
   b. Load family's signals from last 90 days
   c. Load family's blocked entity IDs (from `recs_recommendation_feedback`)
   d. Load family's preferences (enabled types, exploration frequency)
   e. Expire existing recommendations past their TTL (set `status = 'expired'`)
   f. Generate candidate recommendations per type (§10.2)
   g. Filter candidates: remove already-purchased, already-blocked, already-active
   h. Score candidates using the formula (§10.9)
   i. Select top 50, insert into `recs_recommendations` with `expires_at = now() + 14 days`
3. Log total recommendations generated, total families processed, duration

**Concurrency**: Processes families sequentially within a single worker. At launch scale
(< 10K families), this completes well within the daily window. Parallelization (multiple
workers with family ID sharding) is a Phase 3 optimization.

**Idempotency**: The task does NOT delete existing active recommendations before inserting.
The unique index `idx_recs_recommendations_family_entity_active` prevents duplicates. If the
task is re-run on the same day, duplicate inserts are silently skipped (ON CONFLICT DO NOTHING).

### §11.2 AggregatePopularityTask

**Schedule**: Daily at 3:00 AM UTC (runs before `ComputeRecommendationsTask`).

**Process**:
1. For each methodology in `method_definitions`:
   a. Count purchases per listing in the last 90 days (from `recs_signals` where
      `signal_type = 'purchase_completed'` and `methodology_id = ?`)
   b. Apply recency decay: `SUM(e^(-0.03 * days_since_purchase))`
   c. Upsert into `recs_popularity_scores` (listing_id, methodology_id, period_start)
2. Delete popularity scores where `period_end < now() - interval '90 days'`
3. Log total scores computed, total methodologies processed, duration

### §11.3 PurgeStaleSignalsTask

**Schedule**: Weekly (Sunday 2:00 AM UTC).

**Process**:
1. Delete from `recs_signals` where `created_at < now() - interval '90 days'`
2. Log total signals purged

### §11.4 AnonymizeInteractionsTask

**Schedule**: Weekly (Sunday 3:00 AM UTC, after `PurgeStaleSignalsTask`).

**Process**:
1. Query `recs_signals` from the last 7 days
2. For each signal, produce an anonymized record:
   a. `anonymous_id` = HMAC-SHA256(signal.family_id, server_secret)
   b. `methodology_slug` = lookup from `method_definitions` by signal.methodology_id
   c. `age_band` = compute from `iam_students.date_of_birth` (coarsen to 3-year range)
   d. `subject_category` = first tag from signal payload (broad category only)
   e. `duration_minutes` = round to nearest 5 minutes
3. Batch-insert into `recs_anonymized_interactions`
4. Log total interactions anonymized

**Privacy invariant**: The anonymization step MUST NOT write `family_id` or `student_id` to
the output table. The HMAC is a one-way function — the server secret is not stored alongside
the anonymized data and is rotated periodically.

---

## §12 Event Handlers

Defined in `internal/recs/event_handlers.go`. All handlers implement the event handler interface
and delegate to `RecsService`. `[ARCH §4.6]`

### Phase 1 Handlers (Signal Recording)

```go
// internal/recs/event_handlers.go

package recs

import (
    "context"
    "time"

    "github.com/google/uuid"
    learnevents "github.com/homegrown-academy/homegrown-academy/internal/learn/events"
    mktevents "github.com/homegrown-academy/homegrown-academy/internal/mkt/events"
    iamevents "github.com/homegrown-academy/homegrown-academy/internal/iam/events"
    methodevents "github.com/homegrown-academy/homegrown-academy/internal/method/events"
)

// ActivityLoggedHandler records an activity signal when a student logs a learning activity.
// Source: learn::events::ActivityLogged [06-learn §18.3]
type ActivityLoggedHandler struct {
    RecsService RecsService
}

func (h *ActivityLoggedHandler) Handle(ctx context.Context, event *learnevents.ActivityLogged) error {
    return h.RecsService.RecordSignal(ctx, RecordSignalCommand{
        FamilyID:      event.FamilyID,
        StudentID:     &event.StudentID,
        SignalType:    SignalActivityLogged,
        MethodologyID: uuid.Nil, // resolved by service from iam_families
        Payload: map[string]any{
            "subject_tags":     event.SubjectTags,
            "duration_minutes": event.DurationMinutes,
        },
        SignalDate: event.ActivityDate,
    })
}

// BookCompletedHandler records a book completion signal.
// Source: learn::events::BookCompleted [06-learn §18.3]
type BookCompletedHandler struct {
    RecsService RecsService
}

func (h *BookCompletedHandler) Handle(ctx context.Context, event *learnevents.BookCompleted) error {
    return h.RecsService.RecordSignal(ctx, RecordSignalCommand{
        FamilyID:      event.FamilyID,
        StudentID:     &event.StudentID,
        SignalType:    SignalBookCompleted,
        MethodologyID: uuid.Nil, // resolved by service from iam_families
        Payload: map[string]any{
            "title":           event.ReadingItemTitle,
            "reading_item_id": event.ReadingItemID,
        },
        SignalDate: time.Now().UTC(),
    })
}

// PurchaseCompletedHandler records a purchase signal when a family buys marketplace content.
// Source: mkt::events::PurchaseCompleted [07-mkt §18.3]
type PurchaseCompletedHandler struct {
    RecsService RecsService
}

func (h *PurchaseCompletedHandler) Handle(ctx context.Context, event *mktevents.PurchaseCompleted) error {
    return h.RecsService.RecordSignal(ctx, RecordSignalCommand{
        FamilyID:      event.FamilyID,
        StudentID:     nil, // purchases are family-level, not student-specific
        SignalType:    SignalPurchaseCompleted,
        MethodologyID: uuid.Nil, // resolved by service from iam_families
        Payload: map[string]any{
            "listing_id":   event.ListingID,
            "content_type": event.ContentMetadata.ContentType,
        },
        SignalDate: time.Now().UTC(),
    })
}

// ListingPublishedHandler registers a newly published listing in the popularity catalog.
// Source: mkt::events::ListingPublished [07-mkt §18.3]
// NOTE: Does NOT create a recs_signals row — this is a catalog-level event.
type ListingPublishedHandler struct {
    RecsService RecsService
}

func (h *ListingPublishedHandler) Handle(ctx context.Context, event *mktevents.ListingPublished) error {
    return h.RecsService.RegisterListing(ctx, RegisterListingCommand{
        ListingID:   event.ListingID,
        PublisherID: event.PublisherID,
        ContentType: event.ContentType,
        SubjectTags: event.SubjectTags,
    })
}
```

### Lifecycle Handlers (Family Deletion, Config Invalidation)

```go
// FamilyDeletionScheduledHandler deletes all recs data for a family when deletion is scheduled.
// Source: iam::events::FamilyDeletionScheduled [01-iam §13.3]
type FamilyDeletionScheduledHandler struct {
    RecsService RecsService
}

func (h *FamilyDeletionScheduledHandler) Handle(ctx context.Context, event *iamevents.FamilyDeletionScheduled) error {
    return h.RecsService.HandleFamilyDeletion(ctx, event.FamilyID)
}

// MethodologyConfigUpdatedHandler invalidates cached methodology configuration when definitions change.
// Source: method::events::MethodologyConfigUpdated [02-method §12]
type MethodologyConfigUpdatedHandler struct {
    RecsService RecsService
}

func (h *MethodologyConfigUpdatedHandler) Handle(ctx context.Context, event *methodevents.MethodologyConfigUpdated) error {
    return h.RecsService.InvalidateMethodologyCache(ctx)
}
```

### Event Handler Summary

| Handler | Event | Source | Phase | Effect |
|---------|-------|--------|-------|--------|
| `ActivityLoggedHandler` | `ActivityLogged` | `learn::` | 1 | Records `activity_logged` signal |
| `BookCompletedHandler` | `BookCompleted` | `learn::` | 1 | Records `book_completed` signal |
| `PurchaseCompletedHandler` | `PurchaseCompleted` | `mkt::` | 1 | Records `purchase_completed` signal |
| `ListingPublishedHandler` | `ListingPublished` | `mkt::` | 1 | Updates popularity catalog |
| `FamilyDeletionScheduledHandler` | `FamilyDeletionScheduled` | `iam::` | 1 | Deletes all family recs data |
| `MethodologyConfigUpdatedHandler` | `MethodologyConfigUpdated` | `method::` | 1 | Invalidates methodology cache |

---

## §13 Ethical Safeguards (Deep Dive 4)

This section consolidates the ethical requirements from `[S§10.4]` and `[V§12]` and maps
them to concrete implementation mechanisms.

### §13.1 Transparency

Every recommendation response includes two explanation fields:

- **`source_signal`**: Machine-readable signal type (e.g., `"methodology_match"`,
  `"popularity"`, `"progress_gap"`)
- **`source_label`**: Human-readable explanation (e.g., "Popular with Charlotte Mason
  families", "Based on your reading history", "Your student hasn't had math activities in
  two weeks")

Additionally, **every recommendation response includes `is_suggestion: true`**. This field
is always true (there are no non-suggestion recommendations) and serves as a clear label
that the content is an automated suggestion, not a human curation. `[S§10.4]`

### §13.2 Parental Control

Parents have full authority over recommendations. `[S§10.4, V§12]`

| Action | Endpoint | Effect | Reversible? |
|--------|----------|--------|-------------|
| **Dismiss** | `POST /v1/recommendations/:id/dismiss` | Hides the recommendation | Yes — `DELETE /v1/recommendations/:id/feedback` |
| **Block** | `POST /v1/recommendations/:id/block` | Hides the recommendation AND suppresses future recommendations for the same entity | Yes — `DELETE /v1/recommendations/:id/feedback` |
| **Adjust types** | `PATCH /v1/recommendations/preferences` | Enables/disables recommendation types | Yes |
| **Adjust exploration** | `PATCH /v1/recommendations/preferences` | Controls filter bubble prevention intensity | Yes |

### §13.3 Recommendation Labeling

All recommendation responses include `is_suggestion: true` as required by `[S§10.4]`
("AI-generated suggestions MUST be clearly labeled as AI-generated").

> **Note on "AI-generated" language**: Although `[S§10.4]` uses the term "AI-generated,"
> the Phase 2 implementation labels these as "suggestions" rather than "AI-generated" because
> there is no AI involved — the recommendations are rule-based heuristics. The `is_suggestion`
> field satisfies the labeling requirement honestly. If actual ML models are introduced in
> Phase 3+, the label should be updated to "AI-generated" at that time.

### §13.4 Filter Bubble Prevention

Implemented via exploration slots (§10.7). The default `exploration_frequency` is
`occasional` (10% of recommendation slots), ensuring families are periodically exposed to
quality content outside their typical patterns. Parents can adjust this to `off` (no
exploration) or `frequent` (25% exploration). `[S§10.4]`

### §13.5 Content Neutrality

Implemented via four mechanisms (§10.8):
1. `worldview_tags` excluded from scoring algorithm
2. Per-methodology popularity isolation
3. No publisher identity boosting
4. Testable audit invariant (no `worldview_tags` reference in `algorithm.go`)

### §13.6 Content Neutrality Audit

The following assertion MUST hold and is included in the verification checklist (§18):

> `internal/recs/algorithm.go` MUST NOT import, reference, or query `worldview_tags` from any
> table, struct, or function. A `grep -r "worldview_tags" internal/recs/` MUST return zero results.

---

## §14 Anonymized Data Collection (Deep Dive 5)

Implements `[S§10.3]` — collecting anonymized learning interaction data for future AI/ML
capabilities.

### §14.1 What Is Collected

| Field | Source | Anonymization |
|-------|--------|---------------|
| `anonymous_id` | `HMAC-SHA256(family_id, server_secret)` | One-way hash — cannot be reversed |
| `interaction_type` | `recs_signals.signal_type` | Unchanged (non-PII) |
| `methodology_slug` | `method_definitions.slug` via `signal.methodology_id` | Unchanged (non-PII) |
| `age_band` | `iam_students.date_of_birth` | Coarsened to 3-year ranges: `4-6`, `7-9`, `10-12`, `13-15`, `16-18` |
| `subject_category` | `signal.payload.subject_tags[0]` | Broad category only (e.g., "math", "science"), not specific tags |
| `duration_minutes` | `signal.payload.duration_minutes` | Rounded to nearest 5 minutes |
| `interaction_date` | `signal.signal_date` | Unchanged (date only, no timestamp) |

### §14.2 What Is NOT Collected

The `recs_anonymized_interactions` table MUST NOT contain:

- `family_id` — replaced by one-way HMAC
- `student_id` — not present at all
- Student names, email addresses, or any PII
- Specific book titles or listing names
- Exact timestamps (date only, no time)
- Exact ages (3-year bands only)
- Specific subject tags (broad categories only)

### §14.3 HMAC Anonymization

```
anonymous_id = HMAC-SHA256(family_id_bytes, server_secret)
```

- **Server secret**: Stored in environment variable `RECS_ANONYMIZATION_SECRET`, not in the
  database or alongside the anonymized data
- **One-way**: Given an `anonymous_id`, it is computationally infeasible to recover the
  `family_id`
- **Consistent**: The same family always produces the same `anonymous_id` (enabling
  longitudinal analysis without re-identification)
- **Rotation**: When the server secret is rotated, old anonymized data becomes unlinkable
  to new data (acceptable: old cohorts are treated as separate anonymous entities)

### §14.4 Retention

Anonymized interaction data is retained **indefinitely**. Because it contains no PII, there
is no privacy concern with long-term retention. This data will be used for ML model training
when Phase 3 capabilities are implemented (see `AI_FUTURE.md`).

### §14.5 Privacy Invariant

> **The `recs_anonymized_interactions` table MUST NOT contain `family_id`, `student_id`, or
> any column that can identify a specific family or student.** This invariant MUST be
> enforced by:
> 1. Schema design — the table has no FK to `iam_families` or `iam_students`
> 2. Code review — the `AnonymizeInteractionsTask` MUST NOT write these values
> 3. Verification checklist — §18, items 28-31

---

## §15 Error Types

Defined in `internal/recs/errors.go` using custom error types. `[CODING §3.1]`

```go
// internal/recs/errors.go

package recs

import "errors"

var (
    ErrRecommendationNotFound     = errors.New("recommendation not found")
    ErrStudentNotFound            = errors.New("student not found or does not belong to family")
    ErrFeedbackNotFound           = errors.New("feedback not found for this recommendation")
    ErrAlreadyHasFeedback         = errors.New("recommendation already has feedback")
    ErrInvalidRecommendationType  = errors.New("invalid recommendation type")
    ErrInvalidExplorationFrequency = errors.New("invalid exploration frequency")
    ErrPremiumRequired            = errors.New("premium subscription required")
    ErrSignalRecordingFailed      = errors.New("signal recording failed")
    ErrDatabaseError              = errors.New("database error")
    ErrInternalError              = errors.New("internal error")
)
```

### Error-to-HTTP Mapping

| Error Variant | HTTP Status | Response Body | Notes |
|--------------|-------------|---------------|-------|
| `ErrRecommendationNotFound` | 404 | `{ "error": "Recommendation not found" }` | |
| `ErrStudentNotFound` | 404 | `{ "error": "Student not found" }` | Does not reveal whether student exists in another family |
| `ErrFeedbackNotFound` | 404 | `{ "error": "No feedback found for this recommendation" }` | |
| `ErrAlreadyHasFeedback` | 409 | `{ "error": "Recommendation already dismissed or blocked" }` | |
| `ErrInvalidRecommendationType` | 422 | `{ "error": "Invalid recommendation type" }` | |
| `ErrInvalidExplorationFrequency` | 422 | `{ "error": "Invalid exploration frequency" }` | |
| `ErrPremiumRequired` | 402 | `{ "error": "Premium subscription required" }` | Handled by `RequirePremium` extractor |
| `ErrSignalRecordingFailed` | 500 | `{ "error": "Internal server error" }` | Source error logged internally, not exposed `[CODING §3.1]` |
| `ErrDatabaseError` | 500 | `{ "error": "Internal server error" }` | Source error logged internally, not exposed |
| `ErrInternalError` | 500 | `{ "error": "Internal server error" }` | Generic internal error |

---

## §16 Cross-Domain Interactions

### §16.1 recs:: Provides (consumed by other domains)

| Export | Consumers | Mechanism |
|--------|-----------|-----------|
| `RecsService` interface | None currently | `RecsService` interface via AppState (available for future consumers) |

**recs:: publishes no domain events.** It is a terminal consumer — it receives events from
other domains, processes them into signals and recommendations, and serves them via API.
Recommendations are consumed by the frontend directly, not by other backend domains.

### §16.2 recs:: Consumes

| Dependency | Source | Purpose |
|-----------|--------|---------|
| `AuthContext` | `iam::` middleware | User identity on every request `[00-core §7.2]` |
| `FamilyScope` | `iam::` middleware | Family-scoped data access `[00-core §8]` |
| `RequirePremium` | `iam::` extractor | Premium tier gating `[00-core §13.3]` |
| `iam_families` table | `iam::` (direct DB read) | Family methodology IDs, subscription tier |
| `iam_students` table | `iam::` (direct DB read) | Student age (date_of_birth) for age-band computation |
| `method_definitions` table | `method::` (direct DB read) | Methodology config: expected subjects, transition ages, slugs |
| `mkt_listings` table | `mkt::` (direct DB read) | Listing metadata: subject tags, content type, methodology tags |
| `soc_groups` table | `social::` (direct DB read) | Group metadata: methodology tags, visibility |
| `ActivityLogged` event | `learn::` | Signal: student activity `[06-learn §18.3]` |
| `BookCompleted` event | `learn::` | Signal: book completion `[06-learn §18.3]` |
| `PurchaseCompleted` event | `mkt::` | Signal: marketplace purchase `[07-mkt §18.3]` |
| `ListingPublished` event | `mkt::` | Popularity catalog update `[07-mkt §18.3]` |
| `FamilyDeletionScheduled` event | `iam::` | Family data cascade deletion `[01-iam §13.3]` |
| `MethodologyConfigUpdated` event | `method::` | Cache invalidation `[02-method §12]` |

### §16.3 Direct DB Reads (Cross-Domain)

recs:: reads from tables owned by other domains. These are read-only queries executed by
the recommendation algorithm and event handlers. recs:: MUST NOT write to these tables.
`[CODING §2.4]`

| Table | Owner | What recs:: Reads | Used By |
|-------|-------|-------------------|---------|
| `iam_families` | `iam::` | `primary_methodology_id`, `secondary_methodology_ids`, `subscription_tier` | Signal recording (methodology snapshot), algorithm (methodology filter) |
| `iam_students` | `iam::` | `date_of_birth`, `family_id` | Age transition detection, age-band anonymization |
| `method_definitions` | `method::` | `slug`, `expected_subjects`, `transition_ages`, `config` | Algorithm: gap detection, transition detection, seasonal mapping |
| `mkt_listings` | `mkt::` | `id`, `subject_tags`, `content_type`, `methodology_ids`, `title`, `status` | Algorithm: marketplace content candidates |
| `soc_groups` | `social::` | `id`, `methodology_tags`, `visibility`, `name` | Algorithm: community group candidates |

### §16.4 Event Struct Cross-References

The events consumed by recs:: are defined authoritatively in their source domains:

| Event | Authoritative Definition | Struct Fields Used by recs:: |
|-------|-------------------------|------------------------------|
| `ActivityLogged` | `internal/learn/events.go` `[06-learn §18.3]` | `FamilyID`, `StudentID`, `SubjectTags`, `DurationMinutes`, `ActivityDate` |
| `BookCompleted` | `internal/learn/events.go` `[06-learn §18.3]` | `FamilyID`, `StudentID`, `ReadingItemID`, `ReadingItemTitle` |
| `PurchaseCompleted` | `internal/mkt/events.go` `[07-mkt §18.3]` | `FamilyID`, `ListingID`, `ContentMetadata.ContentType` |
| `ListingPublished` | `internal/mkt/events.go` `[07-mkt §18.3]` | `ListingID`, `PublisherID`, `ContentType`, `SubjectTags` |
| `FamilyDeletionScheduled` | `internal/iam/events.go` `[01-iam §13.3]` | `FamilyID` |
| `MethodologyConfigUpdated` | `internal/method/events.go` `[02-method §12]` | (empty struct — no fields) |

---

## §17 Phase Scope

### Phase 1 — Signal Recording Infrastructure

| Component | Included |
|-----------|----------|
| **Database** | All 6 tables created (schema ready for Phase 2) |
| **Event handlers** | All 6 handlers active (signals accumulate before recommendations exist) |
| **API endpoints** | 0 (no HTTP routes registered) |
| **Background tasks** | 0 (signals accumulate; no purging yet) |
| **Service** | `RecordSignal()`, `RegisterListing()`, `HandleFamilyDeletion()`, `InvalidateMethodologyCache()` |
| **Algorithm** | Not implemented |

**Rationale**: Phase 1 starts signal collection so that when Phase 2 recommendations launch,
there is already historical data to work with. Without Phase 1 signal accumulation, Phase 2
recommendations would start cold with no learning history.

### Phase 2 — Full Recommendation Engine

| Component | Included |
|-----------|----------|
| **API endpoints** | All 7 endpoints (all `RequirePremium`) |
| **Background tasks** | All 4 tasks (`ComputeRecommendations`, `AggregatePopularity`, `PurgeStaleSignals`, `AnonymizeInteractions`) |
| **Algorithm** | Full scoring formula (§10.9), all recommendation types (§10.2) |
| **Feedback** | Dismiss, block, undo, preferences CRUD |
| **Service** | Full interface implementation |

### Phase 3 — Advanced Algorithms (see `AI_FUTURE.md`)

| Enhancement | Details |
|-------------|---------|
| View/click tracking | Record recommendation views and clicks for feedback signal |
| Location-aware seasons | Southern Hemisphere season inversion |
| Cross-methodology recommendations | "Families who switched from X to Y found Z helpful" |
| Enhanced anonymization | Differential privacy noise injection |
| Recommendation analytics dashboard | Admin view of recommendation performance |
| Collaborative filtering | ML-based "families like yours" (see `AI_FUTURE.md`) |
| Learning-to-rank | ML replaces hand-tuned scoring weights (see `AI_FUTURE.md`) |
| Content-based filtering | Semantic similarity on listing descriptions (see `AI_FUTURE.md`) |

### Phase 4 — AI Tutoring & Adaptive Learning (see `AI_FUTURE.md`)

| Enhancement | Details |
|-------------|---------|
| Semantic search | Natural language curriculum discovery |
| AI tutoring | LLM-powered methodology-aware tutoring |
| Adaptive learning paths | Sequence models for optimal next activity |
| Curriculum gap analysis | Compliance integration |

---

## §18 Verification Checklist

Testable assertions that MUST hold before the `recs::` domain is considered complete for
each phase.

### Signal Recording (Phase 1)

1. `ActivityLogged` event creates a `recs_signals` row with `signal_type = 'activity_logged'`
2. `BookCompleted` event creates a `recs_signals` row with `signal_type = 'book_completed'`
3. `PurchaseCompleted` event creates a `recs_signals` row with `signal_type = 'purchase_completed'`
4. `ListingPublished` event does NOT create a `recs_signals` row
5. `ListingPublished` event updates `recs_popularity_scores` catalog
6. Signal records include denormalized `methodology_id` from `iam_families`
7. `FamilyDeletionScheduled` event deletes all `recs_signals`, `recs_recommendations`, `recs_recommendation_feedback`, and `recs_preferences` for the family
8. `MethodologyConfigUpdated` event invalidates any cached methodology data

### Recommendations (Phase 2)

9. `GET /v1/recommendations` returns 402 for free-tier families
10. `GET /v1/recommendations` returns only `active` recommendations for the authenticated family
11. Recommendations are constrained to the family's active methodologies (primary + secondary)
12. Each recommendation has a non-empty `source_signal` and `source_label`
13. Each recommendation response includes `is_suggestion: true`
14. `ComputeRecommendationsTask` generates at most 50 recommendations per family
15. Duplicate recommendations (same family + entity) are prevented by unique index

### Feedback (Phase 2)

16. `POST /v1/recommendations/:id/dismiss` sets status to `dismissed` and creates feedback
17. `POST /v1/recommendations/:id/block` sets status to `blocked` and records `blocked_entity_id`
18. `DELETE /v1/recommendations/:id/feedback` restores status to `active` (if not expired)
19. Blocked entities are excluded from future recommendation generation
20. Feedback persists after recommendation expires (blocked sources stay blocked)

### Ethical Requirements

21. Default `exploration_frequency` is `occasional` (10% exploration slots)
22. Exploration recommendations are clearly labeled with `source_signal = "exploration"`
23. Algorithm does NOT read `worldview_tags` from any table
24. Popularity scores are computed per-methodology (not globally)

### Content Neutrality Audit

25. `grep -r "worldview_tags" internal/recs/` returns zero results

### Privacy & Anonymization

26. `recs_anonymized_interactions` table has no `family_id` column
27. `recs_anonymized_interactions` table has no `student_id` column
28. `anonymous_id` is computed via HMAC-SHA256 (one-way)
29. `age_band` uses 3-year ranges only (no exact ages)
30. `duration_minutes` is rounded to nearest 5 minutes
31. `subject_category` stores broad categories only (not specific tags)

### Error Handling

32. All 500 errors log the source error internally but return generic message to client
33. Student not found returns 404 (does not reveal existence in another family)
34. Invalid recommendation ID for wrong family returns 404 (not 403)

---

## §19 Module Structure

```
internal/recs/
+-- handlers.go          # HTTP handlers (Phase 2): thin extractors -> service -> response
+-- service.go           # RecsServiceImpl: business logic, CQRS command/query methods
+-- repository.go        # GORM repository implementations (6 repos)
+-- models.go            # Request/response types, internal types, enums,
|                        # GORM models
+-- ports.go             # RecsService interface, repository interfaces
+-- errors.go            # Sentinel error variables
+-- event_handlers.go    # Domain event handlers (6 handlers)
+-- algorithm.go         # Recommendation scoring algorithm (Phase 2)
+-- tasks.go             # Background tasks (Phase 2): compute, aggregate, purge, anonymize
```

---

## §20 Cross-Reference Migration

The following existing files reference `ai::` or `ai_` and will need updating to use
`recs::` / `recs_`. These updates are a **separate follow-up task** — not performed as
part of this spec creation.

| File | Location | Current Reference | Needed Update |
|------|----------|-------------------|---------------|
| `ARCHITECTURE.md` | §3.2 (~line 459) | `ai::` in Domain-to-Module Mapping table | → `recs::` |
| `ARCHITECTURE.md` | ASCII diagram (~line 425) | `ai::` in monolith modules | → `recs::` |
| `ARCHITECTURE.md` | §4.5 (~line 796) | `ai/` in domain complexity table | → `recs/` |
| `ARCHITECTURE.md` | §4.6 (~line 931) | `ai::` in event subscribers table | → `recs::` |
| `ARCHITECTURE.md` | §4.7 (~line 995) | `ai/` in CQRS table | → `recs/` |
| `ARCHITECTURE.md` | §5.1 (~line 1078) | `ai_` table prefix, `ai_signals`, `ai_recommendations` | → `recs_`, `recs_signals`, `recs_recommendations` |
| `06-learn.md` | §1 (~line 30) | `ai::` in ownership boundary | → `recs::` |
| `06-learn.md` | §18.1 (~line 3094) | `ai::` as `ActivityLogged` event consumer | → `recs::` |
| `07-mkt.md` | §1 (~line 30) | `ai::` in ownership boundary | → `recs::` |
| `07-mkt.md` | §2 (~line 80) | `ai::` in traceability table | → `recs::` |
| `07-mkt.md` | §18.1 (~line 3126) | `ai::` as `ListingPublished` event consumer | → `recs::` |
| `07-mkt.md` | §18.3 (~line 3180) | `ai::` in `ListingPublished` struct doc comment | → `recs::` |
| `00-core.md` | §13.3 (~line 1584) | `ai::` as `RequirePremium` consumer | → `recs::` |
| `SPEC.md` | §10 heading (~line 644) | "AI & Recommendations" | Consider noting rename to "Recommendations & Signals" |
| `SPEC.md` | §18.5 (~line 1099) | "All -> AI" | Consider noting rename |
| `SPEC.md` | §19 Phase 2 (~line 1147) | "AI recommendations" | Consider noting rename |
| `SPEC.md` | §19 Phase 3 (~line 1164) | "Advanced AI" | Consider noting rename |

> **Note**: SPEC.md references use the original "AI & Recommendations" terminology. Since
> SPEC.md is a requirements document (not implementation), the terminology rename is optional
> there. The important rename is in ARCHITECTURE.md and domain specs where `ai::` refers to
> actual module/table names that will exist in code.
