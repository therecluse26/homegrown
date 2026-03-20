# Domain Spec 03 — Discovery (discover::)

## §1 Overview

The Discovery domain owns **public-facing educational content** that serves the platform's
mission before a visitor creates an account. It provides the methodology quiz engine,
methodology explorer data for Astro-built pages, state-specific homeschooling legal guides,
and Homeschooling 101 / advocacy content metadata. Discovery content contains zero
user-generated content and zero personal data — it is entirely platform-authored. `[S§5.4]`

| Attribute | Value |
|-----------|-------|
| **Module path** | `src/discover/` |
| **DB prefix** | `disc_` |
| **Complexity class** | Non-complex (no `domain/` subdirectory) `[ARCH §4.5]` |
| **External adapter** | None |
| **Key constraint** | All endpoints are public (no auth middleware); read-heavy with a single write path (quiz submission) |

**What discover:: owns**: Quiz definitions (questions, answer options, scoring weights per
methodology), quiz results (anonymous, stored with URL-safe IDs), state legal guide
structured data, Homeschooling 101 content metadata, and advocacy content metadata. The
quiz scoring algorithm and explanation generation are internal to this domain.

**What discover:: does NOT own**: Methodology definitions (owned by `method::`), the
Astro static site itself (a separate build artifact), onboarding wizard flow (owned by
`onboard::`), state compliance configuration for families (owned by `comply::`), and user
accounts or family data (owned by `iam::`).

**What discover:: delegates**: Methodology data retrieval → `method::MethodologyService`
(for explorer page data and quiz methodology matching). State guide consumption for
compliance auto-configuration → `comply::` pulls from `disc_state_guides` via
`DiscoveryService` trait. Email delivery for guide update notifications → `notify::`
(future phase, via domain events).

---

## §2 Requirements Traceability

Every requirement from SPEC.md §5 that touches Discovery is mapped to the section of this
document that addresses it.

| Requirement | Source | Addressed In |
|-------------|--------|--------------|
| Quiz MUST be accessible without an account | `[S§5.1.1]` | §4 (all endpoints public), §9 |
| Quiz MUST ask about values, preferences, constraints, temperament | `[S§5.1.1]` | §3 (`disc_quiz_definitions`), §9 |
| Quiz MUST produce ranked methodology recommendations with explanations | `[S§5.1.1]` | §9 (scoring engine) |
| Quiz MUST NOT be a lead-capture form | `[S§5.1.1]` | §4 (no auth required, full results returned) |
| Quiz SHOULD be shareable via URL | `[S§5.1.1]` | §3 (`disc_quiz_results.share_id`), §4, §9 |
| Quiz SHOULD support retaking with different answers | `[S§5.1.1]` | §4 (`POST` creates new result each time) |
| Quiz Definition is a versioned set of questions with scoring weights | `[S§5.1.2]` | §3 (`disc_quiz_definitions`) |
| Quiz Result stores methodology scores with unique URL-safe identifier | `[S§5.1.2]` | §3 (`disc_quiz_results`), §9 |
| Anonymous results stored with session/token, not tied to user account | `[S§5.1.2]` | §3 (`session_token`), §9 |
| Pre-to-post-account result transfer MUST be offered | `[S§5.1.3]` | §5, §9 |
| Transfer SHOULD pre-populate methodology selection wizard | `[S§5.1.3]` | §5, §13 |
| Transfer MUST work across browser sessions (via URL or code) | `[S§5.1.3]` | §9 (`share_id` serves as transfer code) |
| Dedicated page for each methodology | `[S§5.2]` | §4, §10 |
| Methodology page includes philosophy, history, typical day, pros/cons, resources, stories | `[S§5.2]` | §10, method:: provides data |
| Methodology explorer pages MUST be publicly accessible and SEO-indexable | `[S§5.2]` | §10 (Astro SSG) |
| Content detailed enough for parent to choose an approach | `[S§5.2]` | §10, method:: `philosophy` JSONB |
| Pages SHOULD link to marketplace content and community groups | `[S§5.2]` | §10 (Astro renders links; account required to access) |
| Legal guides for all 50 states plus DC | `[S§5.3]` | §3 (`disc_state_guides`), §4, §11 |
| Each guide includes notification, subjects, assessment, record-keeping, attendance, umbrella school | `[S§5.3]` | §3, §11 |
| Guides MUST be publicly accessible and SEO-indexable | `[S§5.3]` | §10 (Astro SSG) |
| Guides MUST display "last reviewed" date and legal disclaimer | `[S§5.3]` | §3 (`last_reviewed_at`), §8, §11 |
| Guides MUST follow consistent structured data format for programmatic querying | `[S§5.3]` | §3, §11 |
| Editorial workflow for guide updates | `[S§5.3]` | §3 (`status` column), §11 |
| Homeschooling 101 content addressing common concerns | `[S§5.4]` | §3 (`disc_content_pages`), §10 |
| "Case for homeschooling" content — confident and clear-eyed | `[S§5.4]` | §3 (`disc_content_pages`), §10 |
| All Discovery content MUST contain zero user-generated content and zero personal data | `[S§5.4]` | §1, §3, §4 (no FamilyScope) |
| All public content MUST be server-rendered or pre-rendered for SEO | `[S§5.5]` | §10 (Astro SSG) |
| State guides SHOULD target SEO search queries | `[S§5.5]` | §10 (Astro handles SEO) |
| Quiz SHOULD be designed for social sharing | `[S§5.5]` | §9 (shareable `share_id` URL) |
| Structured data markup (FAQ schema, breadcrumbs) | `[S§5.5]` | §10 (Astro responsibility) |

---

## §3 Database Schema

All tables use the `disc_` prefix. Discovery tables have **no RLS policies** — all data is
public, platform-authored content with no user-generated data. `[S§5.4]`

### §3.1 Tables

```sql
-- =============================================================================
-- Migration: YYYYMMDD_000001_create_disc_tables.rs
-- =============================================================================

-- PostgreSQL extensions (uuid-ossp, pgcrypto) are installed by the bootstrap
-- migration (00-core §9.4). They are available to all domains.

-- Quiz status enum — controls which version is actively served
CREATE TYPE disc_quiz_status_enum AS ENUM (
    'draft',        -- Under development, not publicly visible
    'active',       -- Currently served to visitors (exactly one at a time)
    'archived'      -- Superseded by a newer version, results preserved
);

-- State guide status enum — editorial workflow [S§5.3]
CREATE TYPE disc_guide_status_enum AS ENUM (
    'draft',        -- Under development, not publicly visible
    'published',    -- Publicly visible and SEO-indexed
    'review_due'    -- Published but past scheduled re-review date
);

-- Content page status enum — for 101 and advocacy content
CREATE TYPE disc_content_status_enum AS ENUM (
    'draft',        -- Under development, not publicly visible
    'published'     -- Publicly visible and SEO-indexed
);

-- Quiz definitions: versioned question sets with scoring weights [S§5.1.2]
-- Exactly one row should have status = 'active' at any given time.
CREATE TABLE disc_quiz_definitions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    version         SMALLINT NOT NULL,                  -- monotonically increasing
    title           TEXT NOT NULL,                       -- e.g., "What methodology fits your family?"
    description     TEXT NOT NULL,                       -- introductory text shown before quiz
    status          disc_quiz_status_enum NOT NULL DEFAULT 'draft',
    -- Questions stored as JSONB array. Each element:
    -- {
    --   "id": "q1",
    --   "category": "values" | "learning_style" | "practical" | "temperament",
    --   "text": "How do you feel about textbooks?",
    --   "help_text": "Optional clarification",
    --   "answers": [
    --     {
    --       "id": "q1a1",
    --       "text": "Essential for structured learning",
    --       "weights": { "<methodology_slug>": <float>, ... }
    --     }
    --   ]
    -- }
    -- Weights are NEVER exposed to the client API. [§9]
    questions       JSONB NOT NULL DEFAULT '[]',
    -- Methodology explanations keyed by slug:
    -- { "charlotte-mason": { "match_text": "...", "mismatch_text": "..." }, ... }
    explanations    JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Enforce at most one active quiz at a time
CREATE UNIQUE INDEX idx_disc_quiz_definitions_active
    ON disc_quiz_definitions (status) WHERE status = 'active';

-- Quiz results: anonymous, shareable [S§5.1.2]
CREATE TABLE disc_quiz_results (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    quiz_definition_id UUID NOT NULL REFERENCES disc_quiz_definitions(id),
    -- URL-safe shareable identifier (nanoid, 12 chars, base62)
    -- Used for sharing and pre-to-post-account transfer [S§5.1.3]
    share_id        TEXT NOT NULL UNIQUE,
    -- Optional session token for anonymous result association
    -- Allows the same browser session to retrieve its results [S§5.1.2]
    session_token   TEXT,
    -- Raw answers: { "q1": "q1a1", "q2": "q2a3", ... }
    answers         JSONB NOT NULL,
    -- Computed scores per methodology: { "charlotte-mason": 0.82, "classical": 0.71, ... }
    scores          JSONB NOT NULL,
    -- Ranked methodology recommendations with explanations
    -- [{ "methodology_slug": "charlotte-mason", "score": 0.82, "rank": 1,
    --    "explanation": "..." }, ...]
    recommendations JSONB NOT NULL,
    -- Optional link to family account (set during pre-to-post transfer) [S§5.1.3]
    family_id       UUID,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_disc_quiz_results_share_id ON disc_quiz_results(share_id);
CREATE INDEX idx_disc_quiz_results_session ON disc_quiz_results(session_token)
    WHERE session_token IS NOT NULL;
CREATE INDEX idx_disc_quiz_results_family ON disc_quiz_results(family_id)
    WHERE family_id IS NOT NULL;

-- State legal guides: structured data for all 50 states + DC [S§5.3]
CREATE TABLE disc_state_guides (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    -- Two-letter state code (e.g., 'TX', 'CA', 'DC')
    state_code      CHAR(2) NOT NULL UNIQUE,
    state_name      TEXT NOT NULL,
    status          disc_guide_status_enum NOT NULL DEFAULT 'draft',
    -- Structured legal requirements — consistent format for programmatic access [S§5.3]
    -- See §11 for full schema definition
    requirements    JSONB NOT NULL DEFAULT '{}',
    -- Human-readable guide content (markdown)
    guide_content   TEXT NOT NULL DEFAULT '',
    -- Editorial metadata [S§5.3]
    last_reviewed_at TIMESTAMPTZ,
    next_review_due  TIMESTAMPTZ,
    reviewed_by     TEXT,                               -- editorial attribution
    legal_disclaimer TEXT NOT NULL DEFAULT 'This information is provided for educational purposes only and does not constitute legal advice. Laws change frequently. Verify current requirements with your state education department.',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_disc_state_guides_status ON disc_state_guides(status)
    WHERE status = 'published';

-- Content pages: Homeschooling 101 and advocacy content [S§5.4]
CREATE TABLE disc_content_pages (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug            TEXT NOT NULL UNIQUE,                -- e.g., 'socialization', 'getting-started'
    title           TEXT NOT NULL,
    -- Category for grouping in navigation
    category        TEXT NOT NULL,                       -- '101' or 'advocacy'
    -- Content body (markdown)
    content         TEXT NOT NULL DEFAULT '',
    -- SEO metadata
    meta_title      TEXT,
    meta_description TEXT,
    status          disc_content_status_enum NOT NULL DEFAULT 'draft',
    display_order   SMALLINT NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_disc_content_pages_category ON disc_content_pages(category, display_order)
    WHERE status = 'published';
```

### §3.2 Seed Data

```sql
-- =============================================================================
-- Migration: YYYYMMDD_000002_seed_disc_state_guides.rs
-- =============================================================================

-- Seed all 51 state guide rows (50 states + DC) with draft status.
-- Guide content and requirements JSONB will be populated in a subsequent
-- content migration. This migration establishes the rows so that the API
-- can serve the full state list even before content is written.

INSERT INTO disc_state_guides (state_code, state_name, status) VALUES
    ('AL', 'Alabama', 'draft'),
    ('AK', 'Alaska', 'draft'),
    ('AZ', 'Arizona', 'draft'),
    ('AR', 'Arkansas', 'draft'),
    ('CA', 'California', 'draft'),
    ('CO', 'Colorado', 'draft'),
    ('CT', 'Connecticut', 'draft'),
    ('DE', 'Delaware', 'draft'),
    ('DC', 'District of Columbia', 'draft'),
    ('FL', 'Florida', 'draft'),
    ('GA', 'Georgia', 'draft'),
    ('HI', 'Hawaii', 'draft'),
    ('ID', 'Idaho', 'draft'),
    ('IL', 'Illinois', 'draft'),
    ('IN', 'Indiana', 'draft'),
    ('IA', 'Iowa', 'draft'),
    ('KS', 'Kansas', 'draft'),
    ('KY', 'Kentucky', 'draft'),
    ('LA', 'Louisiana', 'draft'),
    ('ME', 'Maine', 'draft'),
    ('MD', 'Maryland', 'draft'),
    ('MA', 'Massachusetts', 'draft'),
    ('MI', 'Michigan', 'draft'),
    ('MN', 'Minnesota', 'draft'),
    ('MS', 'Mississippi', 'draft'),
    ('MO', 'Missouri', 'draft'),
    ('MT', 'Montana', 'draft'),
    ('NE', 'Nebraska', 'draft'),
    ('NV', 'Nevada', 'draft'),
    ('NH', 'New Hampshire', 'draft'),
    ('NJ', 'New Jersey', 'draft'),
    ('NM', 'New Mexico', 'draft'),
    ('NY', 'New York', 'draft'),
    ('NC', 'North Carolina', 'draft'),
    ('ND', 'North Dakota', 'draft'),
    ('OH', 'Ohio', 'draft'),
    ('OK', 'Oklahoma', 'draft'),
    ('OR', 'Oregon', 'draft'),
    ('PA', 'Pennsylvania', 'draft'),
    ('RI', 'Rhode Island', 'draft'),
    ('SC', 'South Carolina', 'draft'),
    ('SD', 'South Dakota', 'draft'),
    ('TN', 'Tennessee', 'draft'),
    ('TX', 'Texas', 'draft'),
    ('UT', 'Utah', 'draft'),
    ('VT', 'Vermont', 'draft'),
    ('VA', 'Virginia', 'draft'),
    ('WA', 'Washington', 'draft'),
    ('WV', 'West Virginia', 'draft'),
    ('WI', 'Wisconsin', 'draft'),
    ('WY', 'Wyoming', 'draft');
```

```sql
-- =============================================================================
-- Migration: YYYYMMDD_000003_seed_disc_content_pages.rs
-- =============================================================================

-- Seed Homeschooling 101 content page stubs [S§5.4]
INSERT INTO disc_content_pages (slug, title, category, display_order) VALUES
    ('socialization',           'What About Socialization?',                          '101', 1),
    ('cost',                    'How Much Does Homeschooling Cost?',                  '101', 2),
    ('time-commitment',         'How Much Time Does Homeschooling Take?',             '101', 3),
    ('dual-income',             'Can Dual-Income Families Homeschool?',               '101', 4),
    ('special-needs',           'Homeschooling Children with Special Needs',          '101', 5),
    ('transitioning-mid-year',  'Transitioning from Public School Mid-Year',          '101', 6),
    ('skeptical-relatives',     'How to Talk to Skeptical Relatives',                 '101', 7),
    ('getting-started',         'How to Start Homeschooling: A Complete Guide',       '101', 8);

-- Seed advocacy content page stubs [S§5.4]
INSERT INTO disc_content_pages (slug, title, category, display_order) VALUES
    ('case-for-homeschooling',  'The Case for Homeschooling',                        'advocacy', 1),
    ('why-families-choose',     'Why Families Choose Homeschooling',                  'advocacy', 2);
```

---

## §4 API Endpoints

All Discovery endpoints live in `public_routes` — they require **no authentication** and
no `FamilyScope`. Rate limiting still applies. `[S§5.1.1, ARCH §2.3]`

### §4.1 Phase 1 — Foundation

| Method | Path | Handler | Description | Auth |
|--------|------|---------|-------------|------|
| `GET` | `/v1/discovery/quiz` | `get_quiz` | Returns active quiz questions (weights stripped) | None |
| `POST` | `/v1/discovery/quiz/results` | `submit_quiz` | Scores answers and creates a result | None |
| `GET` | `/v1/discovery/quiz/results/:share_id` | `get_quiz_result` | Retrieves a quiz result by shareable ID | None |
| `GET` | `/v1/discovery/state-guides/:state_code` | `get_state_guide` | Returns a state's legal guide | None |
| `GET` | `/v1/discovery/state-guides` | `list_state_guides` | Lists all states with guide status | None |

### §4.2 Phase 2

| Method | Path | Handler | Description | Auth |
|--------|------|---------|-------------|------|
| `POST` | `/v1/discovery/quiz/results/:share_id/claim` | `claim_quiz_result` | Links a quiz result to a family account | Authenticated |
| `GET` | `/v1/discovery/content/:slug` | `get_content_page` | Returns a content page by slug | None |

**Note on `claim_quiz_result`**: This is the only authenticated endpoint in discover::.
It is called during onboarding to transfer an anonymous quiz result to the newly created
family account. `[S§5.1.3]`

### §4.3 Phase 3+ (Admin)

| Method | Path | Handler | Description | Auth |
|--------|------|---------|-------------|------|
| `PUT` | `/v1/admin/discovery/quiz` | `update_quiz` | Create or update quiz definition | Admin |
| `PUT` | `/v1/admin/discovery/state-guides/:state_code` | `update_state_guide` | Update state guide content | Admin |
| `PUT` | `/v1/admin/discovery/content/:slug` | `update_content_page` | Update content page | Admin |

### §4.4 Handler Signatures

```rust
// src/discover/handlers.rs

/// GET /v1/discovery/quiz
/// Returns the active quiz definition with scoring weights stripped.
/// 200 OK with quiz questions; 404 if no active quiz exists.
pub async fn get_quiz(
    State(service): State<Arc<dyn DiscoveryService>>,
) -> Result<Json<QuizResponse>, AppError> {
    let quiz = service.get_active_quiz().await?;
    Ok(Json(quiz))
}

/// POST /v1/discovery/quiz/results
/// Scores the submitted answers and stores the result.
/// 201 Created with result including share_id.
pub async fn submit_quiz(
    State(service): State<Arc<dyn DiscoveryService>>,
    Json(cmd): Json<SubmitQuizCommand>,
) -> Result<(StatusCode, Json<QuizResultResponse>), AppError> {
    let result = service.submit_quiz(cmd).await?;
    Ok((StatusCode::CREATED, Json(result)))
}

/// GET /v1/discovery/quiz/results/:share_id
/// Returns a previously submitted quiz result.
/// 200 OK with result; 404 if share_id not found.
pub async fn get_quiz_result(
    State(service): State<Arc<dyn DiscoveryService>>,
    Path(share_id): Path<String>,
) -> Result<Json<QuizResultResponse>, AppError> {
    let result = service.get_quiz_result(&share_id).await?;
    Ok(Json(result))
}

/// GET /v1/discovery/state-guides
/// Returns a list of all states with guide availability status.
/// 200 OK with state list.
pub async fn list_state_guides(
    State(service): State<Arc<dyn DiscoveryService>>,
) -> Result<Json<Vec<StateGuideSummaryResponse>>, AppError> {
    let guides = service.list_state_guides().await?;
    Ok(Json(guides))
}

/// GET /v1/discovery/state-guides/:state_code
/// Returns the full state guide for a given state code.
/// 200 OK with guide; 404 if state code not found or guide not published.
pub async fn get_state_guide(
    State(service): State<Arc<dyn DiscoveryService>>,
    Path(state_code): Path<String>,
) -> Result<Json<StateGuideResponse>, AppError> {
    let guide = service.get_state_guide(&state_code).await?;
    Ok(Json(guide))
}

/// POST /v1/discovery/quiz/results/:share_id/claim (Phase 2)
/// Links an anonymous quiz result to the authenticated family.
/// Requires auth middleware. 200 OK; 404 if share_id not found; 409 if already claimed.
pub async fn claim_quiz_result(
    auth: AuthContext,
    State(service): State<Arc<dyn DiscoveryService>>,
    Path(share_id): Path<String>,
) -> Result<Json<QuizResultResponse>, AppError> {
    let scope = FamilyScope::from(&auth);
    let result = service.claim_quiz_result(&scope, &share_id).await?;
    Ok(Json(result))
}
```

### §4.5 Route Registration

```rust
// In app.rs — public routes (no auth middleware)

pub fn public_routes() -> Router<AppState> {
    Router::new()
        // Discovery — quiz
        .route("/v1/discovery/quiz", get(discover::handlers::get_quiz))
        .route("/v1/discovery/quiz/results", post(discover::handlers::submit_quiz))
        .route("/v1/discovery/quiz/results/:share_id", get(discover::handlers::get_quiz_result))
        // Discovery — state guides
        .route("/v1/discovery/state-guides", get(discover::handlers::list_state_guides))
        .route("/v1/discovery/state-guides/:state_code", get(discover::handlers::get_state_guide))
}

// In app.rs — authenticated routes (Phase 2, claim endpoint)
pub fn authenticated_routes() -> Router<AppState> {
    Router::new()
        .route(
            "/v1/discovery/quiz/results/:share_id/claim",
            post(discover::handlers::claim_quiz_result),
        )
        // ... other authenticated routes
}
```

---

## §5 Service Interface

The `DiscoveryService` trait defines all use cases exposed to handlers and other domains.
Defined in `src/discover/ports.rs`. `[CODING §8.2]`

```rust
#[async_trait]
pub trait DiscoveryService: Send + Sync {
    // ─── Quiz (Public) ──────────────────────────────────────────────────

    /// Returns the currently active quiz definition with scoring weights stripped.
    /// Used by GET /v1/discovery/quiz. [S§5.1.1]
    async fn get_active_quiz(&self) -> Result<QuizResponse, AppError>;

    /// Scores submitted answers against the active quiz definition and stores
    /// the result with a generated share_id.
    /// Used by POST /v1/discovery/quiz/results. [S§5.1.1]
    async fn submit_quiz(
        &self,
        cmd: SubmitQuizCommand,
    ) -> Result<QuizResultResponse, AppError>;

    /// Retrieves a previously submitted quiz result by its share_id.
    /// Used by GET /v1/discovery/quiz/results/:share_id. [S§5.1.1]
    async fn get_quiz_result(
        &self,
        share_id: &str,
    ) -> Result<QuizResultResponse, AppError>;

    /// Links an anonymous quiz result to a family account.
    /// Used by POST /v1/discovery/quiz/results/:share_id/claim (Phase 2). [S§5.1.3]
    async fn claim_quiz_result(
        &self,
        scope: &FamilyScope,
        share_id: &str,
    ) -> Result<QuizResultResponse, AppError>;

    // ─── State Guides (Public) ──────────────────────────────────────────

    /// Returns a list of all states with guide availability status.
    /// Used by GET /v1/discovery/state-guides. [S§5.3]
    async fn list_state_guides(&self) -> Result<Vec<StateGuideSummaryResponse>, AppError>;

    /// Returns the full state guide for a given state code.
    /// Used by GET /v1/discovery/state-guides/:state_code. [S§5.3]
    async fn get_state_guide(
        &self,
        state_code: &str,
    ) -> Result<StateGuideResponse, AppError>;

    /// Returns structured requirements for a state, used by comply:: for
    /// compliance auto-configuration. Not exposed as an HTTP endpoint.
    /// [S§5.3]
    async fn get_state_requirements(
        &self,
        state_code: &str,
    ) -> Result<StateGuideRequirements, AppError>;

    // ─── Content Pages (Public) ─────────────────────────────────────────

    /// Returns a content page by slug (Phase 2).
    /// Used by GET /v1/discovery/content/:slug. [S§5.4]
    async fn get_content_page(
        &self,
        slug: &str,
    ) -> Result<ContentPageResponse, AppError>;

    /// Returns all published content pages grouped by category (Phase 2).
    /// Used by Astro at build time for navigation and sitemap generation.
    async fn list_content_pages(&self) -> Result<Vec<ContentPageSummaryResponse>, AppError>;
}
```

**Implementation**: `DiscoveryServiceImpl` in `src/discover/service.rs`. Constructor receives:
- `Arc<dyn QuizDefinitionRepository>`
- `Arc<dyn QuizResultRepository>`
- `Arc<dyn StateGuideRepository>`
- `Arc<dyn ContentPageRepository>` (Phase 2)
- `Arc<dyn MethodologyService>` (for methodology slug-to-name mapping in quiz results)

---

## §6 Repository Interfaces

All repository traits defined in `src/discover/ports.rs`. Discovery repositories do NOT
accept `FamilyScope` — all data is public, platform-authored content. `[CODING §2.4]`

### §6.1 QuizDefinitionRepository

```rust
#[async_trait]
pub trait QuizDefinitionRepository: Send + Sync {
    /// Returns the currently active quiz definition (status = 'active').
    /// Returns None if no active quiz exists.
    async fn find_active(&self) -> Result<Option<QuizDefinition>, AppError>;

    /// Returns a quiz definition by ID.
    async fn find_by_id(&self, id: Uuid) -> Result<Option<QuizDefinition>, AppError>;
}
```

**Implementation**: `PgQuizDefinitionRepository` in `src/discover/repository.rs`.

### §6.2 QuizResultRepository

```rust
#[async_trait]
pub trait QuizResultRepository: Send + Sync {
    /// Stores a new quiz result. Returns the created result.
    async fn create(&self, result: CreateQuizResult) -> Result<QuizResult, AppError>;

    /// Finds a quiz result by its URL-safe share_id.
    async fn find_by_share_id(&self, share_id: &str) -> Result<Option<QuizResult>, AppError>;

    /// Links a quiz result to a family account. [S§5.1.3]
    /// Returns AppError if the result is already claimed by a different family.
    async fn claim_for_family(
        &self,
        share_id: &str,
        family_id: Uuid,
    ) -> Result<QuizResult, AppError>;
}
```

**Implementation**: `PgQuizResultRepository` in `src/discover/repository.rs`.

### §6.3 StateGuideRepository

```rust
#[async_trait]
pub trait StateGuideRepository: Send + Sync {
    /// Returns all state guides with summary information (code, name, status).
    async fn list_all(&self) -> Result<Vec<StateGuideSummary>, AppError>;

    /// Returns the full state guide by state code.
    /// Returns None if state code not found.
    async fn find_by_state_code(
        &self,
        state_code: &str,
    ) -> Result<Option<StateGuide>, AppError>;
}
```

**Implementation**: `PgStateGuideRepository` in `src/discover/repository.rs`.

### §6.4 ContentPageRepository (Phase 2)

```rust
#[async_trait]
pub trait ContentPageRepository: Send + Sync {
    /// Returns a published content page by slug.
    async fn find_by_slug(&self, slug: &str) -> Result<Option<ContentPage>, AppError>;

    /// Returns all published content pages ordered by category and display_order.
    async fn list_published(&self) -> Result<Vec<ContentPageSummary>, AppError>;
}
```

**Implementation**: `PgContentPageRepository` in `src/discover/repository.rs`.

---

## §7 Adapter Interfaces

None. The Discovery domain does not call any external services. All data is stored in
PostgreSQL and served directly. Methodology data is consumed via the `method::` service
trait (an internal domain dependency, not an external adapter).

---

## §8 Models (DTOs)

All types defined in `src/discover/models.rs`. API-facing types derive `serde::Serialize`,
`serde::Deserialize`, and `utoipa::ToSchema`. Request types additionally derive
`validator::Validate`. `[CODING §2.3]`

### §8.1 Request Types

```rust
/// POST /v1/discovery/quiz/results [S§5.1.1]
#[derive(Debug, Deserialize, Validate, ToSchema)]
pub struct SubmitQuizCommand {
    /// Map of question ID → selected answer ID: { "q1": "q1a1", "q2": "q2a3" }
    pub answers: HashMap<String, String>,
    /// Optional session token for anonymous result association [S§5.1.2]
    /// Allows the same browser session to retrieve its own results.
    #[validate(length(max = 128))]
    pub session_token: Option<String>,
}
```

### §8.2 Response Types

```rust
/// GET /v1/discovery/quiz — quiz definition with weights stripped [S§5.1.1]
#[derive(Debug, Serialize, ToSchema)]
pub struct QuizResponse {
    pub quiz_id: Uuid,
    pub version: i16,
    pub title: String,
    pub description: String,
    pub questions: Vec<QuizQuestionResponse>,
}

/// Individual quiz question — weights are NOT included [§9]
#[derive(Debug, Serialize, ToSchema)]
pub struct QuizQuestionResponse {
    pub id: String,
    pub category: String,
    pub text: String,
    pub help_text: Option<String>,
    pub answers: Vec<QuizAnswerResponse>,
}

/// Quiz answer option — no scoring weights exposed
#[derive(Debug, Serialize, ToSchema)]
pub struct QuizAnswerResponse {
    pub id: String,
    pub text: String,
}

/// POST /v1/discovery/quiz/results (201) and
/// GET  /v1/discovery/quiz/results/:share_id (200) [S§5.1.1]
#[derive(Debug, Serialize, ToSchema)]
pub struct QuizResultResponse {
    pub share_id: String,
    pub quiz_version: i16,
    pub recommendations: Vec<MethodologyRecommendation>,
    pub created_at: DateTime<Utc>,
    /// Whether this result has been claimed by a family account [S§5.1.3]
    pub is_claimed: bool,
}

/// A single methodology recommendation from quiz results
#[derive(Debug, Serialize, ToSchema)]
pub struct MethodologyRecommendation {
    pub rank: u8,
    pub methodology_slug: String,
    pub methodology_name: String,
    pub score_percentage: u8,           // 0-100, derived from raw float score
    pub explanation: String,            // why this methodology fits [S§5.1.1]
}

/// GET /v1/discovery/state-guides — list item
#[derive(Debug, Serialize, ToSchema)]
pub struct StateGuideSummaryResponse {
    pub state_code: String,
    pub state_name: String,
    pub is_available: bool,             // true if status = 'published'
    pub last_reviewed_at: Option<DateTime<Utc>>,
}

/// GET /v1/discovery/state-guides/:state_code — full guide [S§5.3]
#[derive(Debug, Serialize, ToSchema)]
pub struct StateGuideResponse {
    pub state_code: String,
    pub state_name: String,
    pub requirements: StateGuideRequirements,
    pub guide_content: String,          // markdown
    pub last_reviewed_at: Option<DateTime<Utc>>,
    pub legal_disclaimer: String,
}

/// Structured state requirements — shared with comply:: [S§5.3]
/// This type is both API-facing (in StateGuideResponse) and consumed
/// internally by comply:: for compliance auto-configuration.
#[derive(Debug, Clone, Serialize, Deserialize, ToSchema)]
pub struct StateGuideRequirements {
    /// Must the family notify the state/district? [S§5.3]
    pub notification_required: bool,
    pub notification_details: Option<String>,
    /// Subjects required by law
    pub required_subjects: Vec<String>,
    /// Assessment or testing obligations [S§5.3]
    pub assessment_required: bool,
    pub assessment_details: Option<String>,
    /// Record-keeping requirements [S§5.3]
    pub record_keeping_required: bool,
    pub record_keeping_details: Option<String>,
    /// Attendance requirements [S§5.3]
    pub attendance_required: bool,
    pub attendance_days: Option<u16>,
    pub attendance_details: Option<String>,
    /// Umbrella school options (where applicable) [S§5.3]
    pub umbrella_school_available: bool,
    pub umbrella_school_details: Option<String>,
    /// Overall regulatory classification
    pub regulation_level: String,       // "low", "moderate", "high"
}

/// GET /v1/discovery/content/:slug (Phase 2) [S§5.4]
#[derive(Debug, Serialize, ToSchema)]
pub struct ContentPageResponse {
    pub slug: String,
    pub title: String,
    pub category: String,
    pub content: String,                // markdown
    pub meta_title: Option<String>,
    pub meta_description: Option<String>,
}

/// Content page summary for navigation and Astro build (Phase 2)
#[derive(Debug, Serialize, ToSchema)]
pub struct ContentPageSummaryResponse {
    pub slug: String,
    pub title: String,
    pub category: String,
}
```

### §8.3 Internal Types (not API-facing)

```rust
/// Full quiz definition from database (includes weights — never serialize to API)
pub struct QuizDefinition {
    pub id: Uuid,
    pub version: i16,
    pub title: String,
    pub description: String,
    pub status: String,
    pub questions: serde_json::Value,       // JSONB with scoring weights
    pub explanations: serde_json::Value,    // JSONB methodology explanations
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

/// Quiz result from database
pub struct QuizResult {
    pub id: Uuid,
    pub quiz_definition_id: Uuid,
    pub share_id: String,
    pub session_token: Option<String>,
    pub answers: serde_json::Value,
    pub scores: serde_json::Value,
    pub recommendations: serde_json::Value,
    pub family_id: Option<Uuid>,
    pub created_at: DateTime<Utc>,
}

/// State guide from database
pub struct StateGuide {
    pub id: Uuid,
    pub state_code: String,
    pub state_name: String,
    pub status: String,
    pub requirements: serde_json::Value,
    pub guide_content: String,
    pub last_reviewed_at: Option<DateTime<Utc>>,
    pub next_review_due: Option<DateTime<Utc>>,
    pub reviewed_by: Option<String>,
    pub legal_disclaimer: String,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

/// State guide summary for list endpoint
pub struct StateGuideSummary {
    pub state_code: String,
    pub state_name: String,
    pub status: String,
    pub last_reviewed_at: Option<DateTime<Utc>>,
}

/// Content page from database
pub struct ContentPage {
    pub id: Uuid,
    pub slug: String,
    pub title: String,
    pub category: String,
    pub content: String,
    pub meta_title: Option<String>,
    pub meta_description: Option<String>,
    pub status: String,
    pub display_order: i16,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

/// Content page summary for list endpoint
pub struct ContentPageSummary {
    pub slug: String,
    pub title: String,
    pub category: String,
}

/// Input for creating a quiz result (repository input)
pub struct CreateQuizResult {
    pub quiz_definition_id: Uuid,
    pub share_id: String,
    pub session_token: Option<String>,
    pub answers: serde_json::Value,
    pub scores: serde_json::Value,
    pub recommendations: serde_json::Value,
}
```

---

## §9 Quiz Scoring Engine

The quiz scoring engine is the primary domain-specific logic in discover::. It lives in
`src/discover/service.rs` within `DiscoveryServiceImpl::submit_quiz`. `[S§5.1.1]`

### §9.1 Scoring Algorithm

The quiz uses a **weighted-sum scoring model**. Each answer option carries per-methodology
weights (stored in `disc_quiz_definitions.questions` JSONB). The scoring process:

1. **Retrieve the active quiz definition** (including weights — the internal `QuizDefinition`
   type, not the stripped `QuizResponse`).
2. **Validate submitted answers** — every question ID must exist in the definition, every
   answer ID must be a valid option for its question. Missing answers are treated as zero
   contribution (the quiz tolerates partial completion).
3. **Accumulate raw scores** — for each answered question, look up the selected answer's
   weights and add them to the running total per methodology:
   ```
   for (question_id, answer_id) in submitted_answers:
       weights = quiz.questions[question_id].answers[answer_id].weights
       for (methodology_slug, weight) in weights:
           raw_scores[methodology_slug] += weight
   ```
4. **Normalize scores** — divide each methodology's raw score by the maximum possible score
   for that methodology (sum of its highest weight per question):
   ```
   for methodology in raw_scores:
       max_possible = sum of max(answer.weights[methodology] for answer in question.answers)
                      for each question in quiz
       normalized_scores[methodology] = raw_scores[methodology] / max_possible
   ```
5. **Rank methodologies** — sort by normalized score descending. Convert to 0-100
   percentage for display.
6. **Generate explanations** — look up the explanation text from
   `disc_quiz_definitions.explanations` for each methodology. Select `match_text` for
   methodologies scoring above 50%, `mismatch_text` for those below 50%.

### §9.2 Weight Stripping

The `GET /v1/discovery/quiz` endpoint MUST strip scoring weights from the response.
`[S§5.1.1]` — the quiz must not be gameable.

```rust
// In DiscoveryServiceImpl::get_active_quiz
fn strip_weights(definition: &QuizDefinition) -> QuizResponse {
    // Parse questions JSONB, map each question to QuizQuestionResponse
    // with answers that have id + text only (no weights field)
    // ...
}
```

The `QuizResponse` and `QuizAnswerResponse` types (§8.2) structurally enforce this —
they have no weights field. The stripping happens during the `QuizDefinition` →
`QuizResponse` conversion in the service layer.

### §9.3 URL-Safe Share IDs

Quiz results use a **nanoid** identifier (12 characters, base62 alphabet) as the share ID.
`[S§5.1.1, S§5.1.2]`

- **Format**: 12 characters from `[0-9A-Za-z]` (base62)
- **Collision probability**: With 62^12 ≈ 3.2 × 10^21 possible IDs, collision is negligible
  at any realistic quiz volume
- **URL safety**: No special characters; safe to embed in URLs without encoding
- **Example**: `aB3kLm9xPq2R`

Generation uses the `nanoid` crate:

```rust
use nanoid::nanoid;

const SHARE_ID_ALPHABET: &[char] = &[
    '0', '1', '2', '3', '4', '5', '6', '7', '8', '9',
    'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M',
    'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z',
    'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm',
    'n', 'o', 'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z',
];

fn generate_share_id() -> String {
    nanoid!(12, SHARE_ID_ALPHABET)
}
```

### §9.4 Pre-to-Post-Account Result Transfer

When a user creates an account after taking the quiz, the onboarding flow (`onboard::`)
offers to import their quiz results. `[S§5.1.3]`

**Transfer mechanism**:

1. User takes quiz → receives `share_id` in the result URL (e.g.,
   `https://homegrown.academy/quiz/results/aB3kLm9xPq2R`)
2. User creates account and enters onboarding wizard
3. Onboarding wizard prompts: "Did you already take our methodology quiz?" with an input
   for the result URL or share ID
4. `onboard::` calls `discover::DiscoveryService::claim_quiz_result(scope, share_id)`
5. `claim_quiz_result` sets `disc_quiz_results.family_id` to the family's ID
6. `onboard::` reads the result's top recommendation and pre-populates the methodology
   selection wizard `[S§5.1.3]`

**Cross-session support**: The `share_id` in the URL is the transfer mechanism. Users can
bookmark the result URL, share it, or enter the code on a different device. No cookies or
browser sessions are required for the transfer. `[S§5.1.3]`

**Conflict handling**: If a quiz result is already claimed by a different family,
`claim_quiz_result` returns `DiscoverError::QuizResultAlreadyClaimed`. The result data is
still visible via `get_quiz_result` (it's public content), but the ownership link cannot
be transferred.

---

## §10 Astro/API Content Split

Discovery content is split between the **Astro static site** (which generates SEO-optimized
HTML at build time) and the **Rust API** (which serves JSON data). `[ARCH §2.4, S§5.5]`

### §10.1 What Astro Builds

Astro generates static pages from API data at build time (`astro build`):

| Page Set | Source | Route Pattern |
|----------|--------|---------------|
| Methodology explorer (6 pages) | `method::MethodologyService::list_methodologies()` + `get_methodology()` | `/explore/:slug` |
| State legal guides (51 pages) | `discover::DiscoveryService::list_state_guides()` + `get_state_guide()` | `/states/:state_code` |
| Homeschooling 101 (~8 pages) | `discover::DiscoveryService::list_content_pages()` + `get_content_page()` | `/101/:slug` |
| Advocacy content (~2 pages) | `discover::DiscoveryService::list_content_pages()` + `get_content_page()` | `/why/:slug` |
| Quiz landing page (1 page) | Static template (quiz is client-side interactive) | `/quiz` |

**Build-time data flow**:
```
Astro build step
    │
    ├── Fetch GET /v1/methodologies → list of methodology slugs
    │   └── For each slug: Fetch GET /v1/methodologies/:slug → full detail
    │       (method:: API, not discover::)
    │
    ├── Fetch GET /v1/discovery/state-guides → list of state codes
    │   └── For each state: Fetch GET /v1/discovery/state-guides/:state_code
    │
    └── Fetch GET /v1/discovery/content (Phase 2) → list of content slugs
        └── For each slug: Fetch GET /v1/discovery/content/:slug
```

### §10.2 What the Rust API Serves at Runtime

Only the **quiz** requires runtime API interaction. All other Discovery content is static
HTML served by Cloudflare Pages.

| Endpoint | Runtime Use |
|----------|-------------|
| `GET /v1/discovery/quiz` | Quiz page fetches questions via JavaScript |
| `POST /v1/discovery/quiz/results` | Quiz page submits answers via JavaScript |
| `GET /v1/discovery/quiz/results/:share_id` | Result page fetches results via JavaScript |

**SEO note**: The quiz page itself is an Astro page with an interactive React island.
The surrounding HTML (title, description, structured data) is static. Only the quiz
interaction is client-side JavaScript. Quiz result pages are also Astro pages that
fetch result data client-side (results are dynamic and cannot be pre-rendered). `[S§5.5]`

### §10.3 Astro Rebuild Triggers

Astro pages must be rebuilt when Discovery content changes:

| Change | Trigger |
|--------|---------|
| Methodology definition updated | Astro rebuild on `MethodologyConfigUpdated` event (CI/CD webhook) |
| State guide content updated | Astro rebuild on guide publish (CI/CD webhook) |
| Content page updated | Astro rebuild on content publish (CI/CD webhook) |
| Quiz definition updated | No rebuild needed (quiz is runtime-interactive) |

Phase 1 uses manual Astro rebuilds. Phase 3+ automates rebuilds via CI/CD webhooks
triggered by admin content updates.

---

## §11 State Guide Structure

State guides follow a consistent structured format that serves two purposes: (1) public
display in the Astro-built guide pages, and (2) programmatic access by `comply::` for
compliance auto-configuration. `[S§5.3]`

### §11.1 Requirements Schema

The `disc_state_guides.requirements` JSONB column stores the `StateGuideRequirements`
structure defined in §8.2. This type is shared between discover:: and comply::.

Example for Texas:

```json
{
    "notification_required": false,
    "notification_details": null,
    "required_subjects": [
        "reading", "spelling", "grammar", "mathematics", "citizenship"
    ],
    "assessment_required": false,
    "assessment_details": "No state testing or assessment required for homeschoolers.",
    "record_keeping_required": false,
    "record_keeping_details": "No specific record-keeping required by law, but recommended for your records.",
    "attendance_required": false,
    "attendance_days": null,
    "attendance_details": "No attendance tracking required for homeschoolers in Texas.",
    "umbrella_school_available": false,
    "umbrella_school_details": null,
    "regulation_level": "low"
}
```

Example for New York (high regulation):

```json
{
    "notification_required": true,
    "notification_details": "File an Individualized Home Instruction Plan (IHIP) with the local school district by July 1 or within 14 days of starting homeschooling. Must include: instructional plan, list of textbooks/materials, and planned assessments.",
    "required_subjects": [
        "arithmetic", "reading", "spelling", "writing", "english",
        "geography", "us_history", "science", "health", "music",
        "visual_arts", "physical_education", "bilingual_education",
        "highway_safety", "fire_safety"
    ],
    "assessment_required": true,
    "assessment_details": "Annual assessment required: grades 1-3 use alternative assessment (portfolio, narrative); grades 4-8 require standardized test every other year and portfolio in alternate years; grades 9-12 require standardized testing.",
    "record_keeping_required": true,
    "record_keeping_details": "Must maintain attendance records (substantial equivalent of 180 days), quarterly reports filed with school district, and annual assessment results.",
    "attendance_required": true,
    "attendance_days": 180,
    "attendance_details": "Must provide instruction for a substantial equivalent of 180 days per year. Record daily attendance.",
    "umbrella_school_available": false,
    "umbrella_school_details": null,
    "regulation_level": "high"
}
```

### §11.2 Editorial Workflow

State guides follow a defined editorial lifecycle. `[S§5.3]`

```
    ┌───────────┐
    │   draft   │──── Initial creation (seed migration)
    └─────┬─────┘
          │
    Content written + reviewed
          │
    ┌─────▼──────────┐
    │   published    │──── Publicly visible, SEO-indexed
    └─────┬──────────┘
          │
    next_review_due date reached
          │
    ┌─────▼──────────┐
    │  review_due    │──── Still visible, flagged for editorial review
    └─────┬──────────┘
          │
    Content reviewed + updated
          │
    ┌─────▼──────────┐
    │   published    │──── last_reviewed_at + next_review_due updated
    └────────────────┘
```

**Review cadence**: Every guide MUST be reviewed at minimum annually. `[S§5.3]`
The `next_review_due` column is set to `last_reviewed_at + 1 year` when a guide is
published or re-reviewed. A scheduled job (Phase 3+) transitions published guides to
`review_due` status when the current date exceeds `next_review_due`.

**Phase 1 approach**: Guides are managed via database migrations (content migrations).
No admin UI exists. Status transitions are manual SQL updates.

**Phase 3+ approach**: Admin endpoints (`PUT /v1/admin/discovery/state-guides/:state_code`)
enable editorial management via an admin UI.

---

## §12 Error Types

The `DiscoverError` enum lives in `src/discover/models.rs` (or `src/discover/errors.rs`
if the domain needs a separate error file). Errors map to HTTP status codes via
`From<DiscoverError> for AppError`. `[CODING §2.2]`

```rust
use thiserror::Error;

#[derive(Debug, Error)]
pub enum DiscoverError {
    #[error("no active quiz definition found")]
    NoActiveQuiz,

    #[error("quiz result not found")]
    QuizResultNotFound,

    #[error("invalid quiz answer: question '{question_id}' not found")]
    InvalidQuestionId { question_id: String },

    #[error("invalid quiz answer: answer '{answer_id}' not valid for question '{question_id}'")]
    InvalidAnswerId { question_id: String, answer_id: String },

    #[error("state guide not found for state code '{state_code}'")]
    StateGuideNotFound { state_code: String },

    #[error("state guide not published for state code '{state_code}'")]
    StateGuideNotPublished { state_code: String },

    #[error("content page not found: '{slug}'")]
    ContentPageNotFound { slug: String },

    #[error("quiz result already claimed by another family")]
    QuizResultAlreadyClaimed,

    #[error("invalid state code format")]
    InvalidStateCode,
}
```

### §12.1 HTTP Status Mapping

| Error Variant | HTTP Status | Client Message |
|---------------|-------------|----------------|
| `NoActiveQuiz` | 404 Not Found | "No quiz is currently available." |
| `QuizResultNotFound` | 404 Not Found | "Quiz result not found." |
| `InvalidQuestionId` | 422 Unprocessable Entity | "Invalid quiz submission." |
| `InvalidAnswerId` | 422 Unprocessable Entity | "Invalid quiz submission." |
| `StateGuideNotFound` | 404 Not Found | "State guide not found." |
| `StateGuideNotPublished` | 404 Not Found | "State guide not found." |
| `ContentPageNotFound` | 404 Not Found | "Content not found." |
| `QuizResultAlreadyClaimed` | 409 Conflict | "This quiz result is already linked to an account." |
| `InvalidStateCode` | 422 Unprocessable Entity | "Invalid state code." |

**Note**: `StateGuideNotPublished` maps to 404 (not 403) to avoid revealing that a draft
guide exists. Internal error details are logged but never returned in API responses.
`[CODING §2.2]`

```rust
impl From<DiscoverError> for AppError {
    fn from(err: DiscoverError) -> Self {
        match err {
            DiscoverError::NoActiveQuiz => AppError::not_found("No quiz is currently available."),
            DiscoverError::QuizResultNotFound => AppError::not_found("Quiz result not found."),
            DiscoverError::InvalidQuestionId { .. } => {
                AppError::unprocessable("Invalid quiz submission.")
            }
            DiscoverError::InvalidAnswerId { .. } => {
                AppError::unprocessable("Invalid quiz submission.")
            }
            DiscoverError::StateGuideNotFound { .. } => {
                AppError::not_found("State guide not found.")
            }
            DiscoverError::StateGuideNotPublished { .. } => {
                AppError::not_found("State guide not found.")
            }
            DiscoverError::ContentPageNotFound { .. } => {
                AppError::not_found("Content not found.")
            }
            DiscoverError::QuizResultAlreadyClaimed => {
                AppError::conflict("This quiz result is already linked to an account.")
            }
            DiscoverError::InvalidStateCode => {
                AppError::unprocessable("Invalid state code.")
            }
        }
    }
}
```

---

## §13 Cross-Domain Interactions

### §13.1 discover:: Provides (consumed by other domains)

| Export | Consumers | Mechanism |
|--------|-----------|-----------|
| `DiscoveryService` trait methods | `onboard::`, `comply::` | `Arc<dyn DiscoveryService>` via AppState |
| `get_quiz_result()` | `onboard::` | Service call — pre-populate methodology wizard from quiz result `[S§5.1.3]` |
| `claim_quiz_result()` | `onboard::` | Service call — link anonymous result to new family account `[S§5.1.3]` |
| `get_state_requirements()` | `comply::` | Service call — structured state requirements for compliance auto-configuration `[S§5.3]` |
| `list_state_guides()` | `comply::` | Service call — state list for compliance setup |
| `StateGuideRequirements` type | `comply::` | Shared type — structured requirements data |

### §13.2 discover:: Consumes

| Dependency | Source | Purpose |
|-----------|--------|---------|
| Methodology data (names, slugs) | `method::MethodologyService` | Map methodology slugs to display names in quiz results `[S§5.1.1]` |
| Methodology detail | `method::MethodologyService` | Astro build-time data for methodology explorer pages `[S§5.2]` |

### §13.3 Events discover:: Publishes

None. Discovery is a read-heavy domain with no state changes that other domains need to
react to. The quiz submission is self-contained — the result is stored and retrieved by
the visitor. The only cross-domain interaction is `onboard::` pulling data via service
calls during the onboarding flow.

### §13.4 Events discover:: Subscribes To

| Event | Source | Effect |
|-------|--------|--------|
| `MethodologyConfigUpdated` | `method::` | Invalidate any cached methodology data used by quiz scoring (if cached). Trigger Astro rebuild for methodology explorer pages (Phase 3+). `[ARCH §4.6]` |

---

## §14 Phase Scope

### Phase 1 — Foundation

**In scope**:
- `disc_` tables: `disc_quiz_definitions`, `disc_quiz_results`, `disc_state_guides`, `disc_content_pages`
- Seed migration: 51 state guide rows (draft), 8 Homeschooling 101 stubs, 2 advocacy stubs
- Public endpoints: `GET /v1/discovery/quiz`, `POST /v1/discovery/quiz/results`, `GET /v1/discovery/quiz/results/:share_id`, `GET /v1/discovery/state-guides`, `GET /v1/discovery/state-guides/:state_code`
- `DiscoveryService` trait + `DiscoveryServiceImpl`
- Quiz scoring engine (weighted-sum scoring, weight stripping, nanoid share IDs)
- Repository traits + PostgreSQL implementations for quiz and state guides
- `DiscoverError` enum + HTTP mapping
- All Phase 1 models (request, response, internal)
- OpenAPI spec + TypeScript type generation

**Out of scope (Phase 2)**:
- `POST /v1/discovery/quiz/results/:share_id/claim` (quiz result transfer to family account)
- `GET /v1/discovery/content/:slug` (content page endpoint)
- `ContentPageRepository` implementation
- Quiz content migration (actual quiz questions and scoring weights)
- State guide content migration (actual legal content for published guides)

**Deferred (Phase 3+)**:
- Admin endpoints for quiz, state guide, and content management
- Automated Astro rebuild triggers on content changes
- Automated guide review-due status transitions
- Quiz analytics (completion rates, methodology distribution)
- A/B testing for quiz question ordering and scoring weights
- Quiz versioning UI (admin creates new version, archives old)

---

## §15 Verification Checklist

Numbered assertions that MUST be true when the discover:: implementation is correct. Use
these as acceptance criteria for code review and integration testing.

1. `GET /v1/discovery/quiz` returns quiz questions with NO scoring weights in the response
2. `GET /v1/discovery/quiz` returns 404 when no active quiz definition exists
3. `POST /v1/discovery/quiz/results` with valid answers returns 201 with a `share_id`
4. `POST /v1/discovery/quiz/results` with an invalid question ID returns 422
5. `POST /v1/discovery/quiz/results` with an invalid answer ID for a valid question returns 422
6. `POST /v1/discovery/quiz/results` with partial answers (not all questions answered) succeeds and scores correctly
7. Quiz results include ranked methodology recommendations with explanations
8. Quiz result `score_percentage` values are 0-100 integers, not raw floats
9. `GET /v1/discovery/quiz/results/:share_id` returns the same result that was created
10. `GET /v1/discovery/quiz/results/nonexistent` returns 404
11. Quiz `share_id` is exactly 12 characters, base62 alphabet
12. No quiz endpoint requires authentication
13. `GET /v1/discovery/state-guides` returns exactly 51 entries (50 states + DC)
14. `GET /v1/discovery/state-guides/TX` returns the full state guide when published
15. `GET /v1/discovery/state-guides/XX` returns 404 for invalid state codes
16. `GET /v1/discovery/state-guides/:state_code` returns 404 for draft (unpublished) guides
17. State guide responses include `last_reviewed_at` and `legal_disclaimer`
18. `StateGuideRequirements` is deserializable by comply:: for compliance configuration
19. `claim_quiz_result` links result to family and returns updated response (Phase 2)
20. `claim_quiz_result` returns 409 when result is already claimed by a different family (Phase 2)
21. `claim_quiz_result` allows re-claiming by the same family (idempotent) (Phase 2)
22. No endpoint under `src/discover/` requires `FamilyScope` (except `claim_quiz_result`)
23. All API error responses return generic messages, not SQL or internal details
24. No code under `src/discover/` branches on methodology name/slug
25. `disc_` tables have NO RLS policies
26. Quiz scoring produces deterministic results for the same answers
27. Methodology names in quiz results come from `method::` service, not hardcoded strings

---

## §16 Implementation Checklist

### Phase 1 — Foundation

#### Database
- [ ] Create migration: `disc_quiz_status_enum`, `disc_guide_status_enum`, `disc_content_status_enum` types
- [ ] Create migration: `disc_quiz_definitions` table
- [ ] Create migration: `disc_quiz_results` table
- [ ] Create migration: `disc_state_guides` table
- [ ] Create migration: `disc_content_pages` table
- [ ] Create seed migration: 51 state guide rows (draft status)
- [ ] Create seed migration: 8 Homeschooling 101 + 2 advocacy content page stubs
- [ ] Regenerate SeaORM entities from migrations

#### Ports & Traits
- [ ] Define `DiscoveryService` trait in `src/discover/ports.rs`
- [ ] Define `QuizDefinitionRepository` trait in `src/discover/ports.rs`
- [ ] Define `QuizResultRepository` trait in `src/discover/ports.rs`
- [ ] Define `StateGuideRepository` trait in `src/discover/ports.rs`

#### Error Types
- [ ] Define `DiscoverError` enum
- [ ] Implement `From<DiscoverError> for AppError` conversion

#### Repository Implementations
- [ ] Implement `PgQuizDefinitionRepository`
- [ ] Implement `PgQuizResultRepository`
- [ ] Implement `PgStateGuideRepository`

#### Service Implementation
- [ ] Implement `DiscoveryServiceImpl` with Phase 1 methods
- [ ] Implement quiz scoring engine (weighted-sum, normalization, ranking)
- [ ] Implement weight stripping for `QuizDefinition` → `QuizResponse` conversion
- [ ] Implement nanoid share_id generation
- [ ] Wire `DiscoveryServiceImpl` in `app.rs` with `Arc<dyn DiscoveryService>`

#### API Endpoints
- [ ] `GET  /v1/discovery/quiz` — return active quiz (weights stripped)
- [ ] `POST /v1/discovery/quiz/results` — score and store quiz submission
- [ ] `GET  /v1/discovery/quiz/results/:share_id` — retrieve quiz result
- [ ] `GET  /v1/discovery/state-guides` — list all state guides
- [ ] `GET  /v1/discovery/state-guides/:state_code` — get full state guide
- [ ] Register all endpoints in `public_routes()` (no auth middleware)

#### Models
- [ ] Define `SubmitQuizCommand` request type
- [ ] Define all response types (`QuizResponse`, `QuizResultResponse`, `StateGuideResponse`, etc.)
- [ ] Define all internal types (`QuizDefinition`, `QuizResult`, `StateGuide`, etc.)
- [ ] Add `nanoid` crate to `Cargo.toml` dependencies

#### Code Generation
- [ ] Regenerate OpenAPI spec (`cargo run --bin openapi-gen`)
- [ ] Regenerate TypeScript types (`cd frontend && npm run generate-types`)

#### Testing
- [ ] Unit tests for quiz scoring engine (deterministic scoring, normalization, edge cases)
- [ ] Unit tests for weight stripping (no weights leak to response)
- [ ] Unit tests for share_id generation (length, alphabet)
- [ ] Integration test: quiz happy path (get quiz → submit → retrieve result)
- [ ] Integration test: state guide retrieval (published returns 200, draft returns 404)
- [ ] Integration test: invalid quiz answers return 422
- [ ] Integration test: nonexistent share_id returns 404

### Phase 2

- [ ] Define `ContentPageRepository` trait
- [ ] Implement `PgContentPageRepository`
- [ ] Implement `claim_quiz_result` in service
- [ ] Add `claim_quiz_result` endpoint to authenticated routes
- [ ] Implement `get_content_page` and `list_content_pages`
- [ ] Add content page endpoints
- [ ] Create quiz content migration (actual questions and scoring weights)
- [ ] Create state guide content migrations (legal content for initial set of states)

---

## §17 Module Structure

```
src/discover/
├── mod.rs              # Re-exports, domain-level doc comments
├── handlers.rs         # Axum route handlers (all public, no auth)
├── service.rs          # DiscoveryServiceImpl — quiz scoring, weight stripping
├── repository.rs       # PgQuizDefinitionRepository, PgQuizResultRepository,
│                       # PgStateGuideRepository, PgContentPageRepository
├── models.rs           # Request/response types, DiscoverError enum, internal types
├── ports.rs            # DiscoveryService trait, all repository traits
└── entities/           # SeaORM-generated entity files (do not hand-edit)
```

**Note**: No `domain/` subdirectory, no `adapters/` directory, no `events.rs`, no
`event_handlers.rs`. Discovery is a non-complex domain with no aggregate roots, no
external service adapters, and no emitted events. `[ARCH §4.5]`
