# Domain Spec 04 — Onboarding & Family Setup (onboard::)

## §1 Overview

The Onboarding domain owns the **post-account-creation wizard** that guides new families from
a bare account to a configured homeschooling workspace. It orchestrates interactions between
IAM (family/student data), methodology (config lookup), and discovery (quiz results) to produce
a personalized first experience. Onboarding owns no core business entities — it owns workflow
state and materialized guidance artifacts. `[S§6]`

| Attribute | Value |
|-----------|-------|
| **Module path** | `src/onboard/` |
| **DB prefix** | `onb_` |
| **Complexity class** | Non-complex (no `domain/` subdirectory) `[ARCH §4.5]` |
| **External adapter** | None |
| **Key constraint** | Workflow orchestration only; all core data lives in other domains |

**What onboard:: owns**: Wizard progress tracking, getting-started roadmaps (materialized from
methodology config), starter curriculum recommendations (materialized from methodology config),
community connection suggestions (materialized from methodology config), and the onboarding
completion event.

**What onboard:: does NOT own**: Family accounts or student profiles (owned by `iam::`),
methodology definitions or tool activation mappings (owned by `method::`), quiz definitions
or quiz results (owned by `discover::`), social profiles or group memberships (owned by
`social::`), marketplace listings or search (owned by `mkt::` / `search::`).

**What onboard:: delegates**: Family profile updates → `iam::IamService`. Student CRUD →
`iam::IamService`. Methodology selection and validation → `method::MethodologyService`.
Quiz result retrieval and claiming → `discover::DiscoveryService`. Email delivery →
`notify::` (via domain events).

---

## §2 Requirements Traceability

Every SPEC.md §6 requirement maps to a section in this document.

| Requirement | SPEC Reference | Domain Spec Section |
|-------------|---------------|---------------------|
| Account creation via email/password and OAuth | `[S§6.1]` | §1 (delegated to `iam::` — Kratos handles auth, IAM creates family) |
| COPPA compliance notice during account creation | `[S§6.1]` | §1 (delegated to `iam::` — IAM handles COPPA, not onboard) |
| Atomic family + parent creation | `[S§6.1]` | §1 (delegated to `iam::`) |
| Email verification before activation | `[S§6.1]` | §1 (delegated to `iam::` — Kratos handles verification) |
| Collect parent names, child info, family location | `[S§6.2]` | §4 (`PATCH /v1/onboarding/family-profile`), §9 (wizard step: `family_profile`) |
| Allow adding multiple children during setup | `[S§6.2]` | §4 (`POST /v1/onboarding/children`, `DELETE /v1/onboarding/children/:id`), §9 |
| Allow adding co-parent during or after setup | `[S§6.2]` | §13 (deferred to Phase 2 — co-parent invite is IAM Phase 2) |
| Location collection with explanation | `[S§6.2]` | §4 (`PATCH /v1/onboarding/family-profile`), §8 |
| Quiz result import as starting point | `[S§6.3]` | §4 (`POST /v1/onboarding/methodology/import-quiz`), §9.4 |
| Three methodology selection paths | `[S§6.3]` | §4 (`PATCH /v1/onboarding/methodology`), §8, §9.3 |
| Primary + secondary methodology selection | `[S§6.3]` | §4 (`PATCH /v1/onboarding/methodology`), §8 |
| Multi-methodology model explanation | `[S§6.3]` | §8 (`MethodologySelectionRequest.explanation_acknowledged`) |
| Methodology-specific getting-started roadmap | `[S§6.4]` | §4 (`GET /v1/onboarding/roadmap`), §10 |
| Concrete, actionable first-week checklist | `[S§6.4]` | §10.2 (item types: `task`, `resource`, `feature_tour`) |
| Age-adapted roadmaps | `[S§6.4]` | §10.3 (5 age brackets from student birth_year) |
| Roadmap items link to platform features and content | `[S§6.4]` | §10.2 (`link_url` field on roadmap items) |
| Curated starter curriculum recommendations | `[S§6.5]` | §4 (`GET /v1/onboarding/recommendations`), §8 |
| Specific, limited recommendations (not catalog dump) | `[S§6.5]` | §10.4 (max 6 per age group, from `onboarding_config.starter_recs`) |
| Methodology-specific, age-appropriate recommendations | `[S§6.5]` | §10.3, §10.4 |
| Both free and paid options in recommendations | `[S§6.5]` | §10.4 (`price_type` enum: `free`, `paid`) |
| Methodology-matched group suggestions | `[S§6.6]` | §4 (`GET /v1/onboarding/community`), §10.5 |
| Nearby homeschool family suggestions | `[S§6.6]` | §13 (deferred to Phase 2 — requires `social::` location queries) |
| Mentor matching suggestions | `[S§6.6]` | §13 (deferred to Phase 2 — requires `social::` mentor system) |
| Privacy-respecting suggestions (opt-in visibility) | `[S§6.6]` | §10.5 (suggestions are static references, not live user queries) |

---

## §3 Database Schema

### §3.1 Enums

```sql
-- Wizard step identifiers
CREATE TYPE onb_wizard_step_enum AS ENUM (
    'family_profile',      -- step 1: family name, state, children (required)
    'children',            -- step 2: add student profiles (optional, can skip)
    'methodology',         -- step 3: select methodology (required)
    'roadmap_review'       -- step 4: review roadmap + recommendations (optional)
);

-- Overall wizard status
CREATE TYPE onb_wizard_status_enum AS ENUM (
    'in_progress',         -- wizard is active
    'completed',           -- wizard finished (all required steps done)
    'skipped'              -- user explicitly skipped remaining steps
);

-- Roadmap item type
CREATE TYPE onb_roadmap_item_type_enum AS ENUM (
    'task',                -- actionable to-do (e.g., "Set up your first reading list")
    'resource',            -- link to educational content (e.g., "Read: CM philosophy intro")
    'feature_tour'         -- guided tour of a platform feature
);
```

### §3.2 Tables

```sql
-- Wizard progress tracking [S§6]
-- One row per family. Created when FamilyCreated event is received.
CREATE TABLE onb_wizard_progress (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id             UUID NOT NULL UNIQUE REFERENCES iam_families(id) ON DELETE CASCADE,
    status                onb_wizard_status_enum NOT NULL DEFAULT 'in_progress',
    current_step          onb_wizard_step_enum NOT NULL DEFAULT 'family_profile',
    completed_steps       onb_wizard_step_enum[] NOT NULL DEFAULT '{}',  -- tracks which steps are done
    quiz_share_id         TEXT,                                -- share_id if quiz was imported [S§6.3]
    methodology_path      TEXT CHECK (methodology_path IN (
                            'quiz_informed', 'exploration', 'skip'
                          )),                                  -- which path user took [S§6.3]
    completed_at          TIMESTAMPTZ,                         -- set when status → completed
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_onb_wizard_family ON onb_wizard_progress(family_id);
CREATE INDEX idx_onb_wizard_status ON onb_wizard_progress(status)
    WHERE status = 'in_progress';

-- Getting-started roadmap items [S§6.4]
-- Materialized from method_definitions.onboarding_config when methodology step completes.
CREATE TABLE onb_roadmap_items (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id             UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    wizard_id             UUID NOT NULL REFERENCES onb_wizard_progress(id) ON DELETE CASCADE,
    methodology_id        UUID NOT NULL,                       -- source methodology (for display);
                                                              -- app-level ref only, no DB FK (cross-domain boundary)
    item_type             onb_roadmap_item_type_enum NOT NULL, -- task, resource, feature_tour
    title                 TEXT NOT NULL,                        -- e.g., "Set up your first reading list"
    description           TEXT,                                 -- longer explanation
    link_url              TEXT,                                 -- deep link into platform feature or content
    age_group             TEXT,                                 -- e.g., "6-8" — NULL means all ages
    sort_order            SMALLINT NOT NULL DEFAULT 0,
    is_completed          BOOLEAN NOT NULL DEFAULT false,       -- user can check items off
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_onb_roadmap_family ON onb_roadmap_items(family_id);
CREATE INDEX idx_onb_roadmap_wizard ON onb_roadmap_items(wizard_id);

-- Starter curriculum recommendations [S§6.5]
-- Materialized from method_definitions.onboarding_config.starter_recs.
-- Phase 1: static data from config. Phase 2: live marketplace queries.
CREATE TABLE onb_starter_recommendations (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id             UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    wizard_id             UUID NOT NULL REFERENCES onb_wizard_progress(id) ON DELETE CASCADE,
    methodology_id        UUID NOT NULL,                       -- source methodology;
                                                              -- app-level ref only, no DB FK (cross-domain boundary)
    title                 TEXT NOT NULL,                        -- e.g., "Ambleside Online Year 1"
    description           TEXT,
    link_url              TEXT,                                 -- marketplace link or external URL
    price_type            TEXT NOT NULL CHECK (price_type IN ('free', 'paid')),  -- [S§6.5]
    age_group             TEXT,                                 -- e.g., "6-8"
    sort_order            SMALLINT NOT NULL DEFAULT 0,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_onb_recs_family ON onb_starter_recommendations(family_id);
CREATE INDEX idx_onb_recs_wizard ON onb_starter_recommendations(wizard_id);

-- Community connection suggestions [S§6.6]
-- Materialized from method_definitions.community_config.
-- Phase 1: static group references. Phase 2: live social queries for nearby families/mentors.
CREATE TABLE onb_community_suggestions (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id             UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    wizard_id             UUID NOT NULL REFERENCES onb_wizard_progress(id) ON DELETE CASCADE,
    methodology_id        UUID NOT NULL,                       -- source methodology;
                                                              -- app-level ref only, no DB FK (cross-domain boundary)
    suggestion_type       TEXT NOT NULL CHECK (suggestion_type IN (
                            'methodology_group',               -- platform-managed methodology group
                            'local_group',                     -- nearby community group (Phase 2)
                            'mentor'                           -- mentor match (Phase 2)
                          )),
    title                 TEXT NOT NULL,                        -- e.g., "Charlotte Mason Community"
    description           TEXT,
    reference_id          UUID,                                 -- FK to social group/user (app-level, not DB FK)
    sort_order            SMALLINT NOT NULL DEFAULT 0,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_onb_community_family ON onb_community_suggestions(family_id);
CREATE INDEX idx_onb_community_wizard ON onb_community_suggestions(wizard_id);
```

### §3.3 RLS Policies

All `onb_` tables contain user data and MUST have RLS policies for defense-in-depth.
The application enforces family-scoping via `FamilyScope`, but RLS provides a database-level
safety net. `[CODING §3.2]`

```sql
ALTER TABLE onb_wizard_progress ENABLE ROW LEVEL SECURITY;
ALTER TABLE onb_roadmap_items ENABLE ROW LEVEL SECURITY;
ALTER TABLE onb_starter_recommendations ENABLE ROW LEVEL SECURITY;
ALTER TABLE onb_community_suggestions ENABLE ROW LEVEL SECURITY;

-- App role can only access rows matching current family_id
CREATE POLICY onb_wizard_progress_family_policy ON onb_wizard_progress
    USING (family_id = current_setting('app.current_family_id')::uuid);

CREATE POLICY onb_roadmap_items_family_policy ON onb_roadmap_items
    USING (family_id = current_setting('app.current_family_id')::uuid);

CREATE POLICY onb_starter_recommendations_family_policy ON onb_starter_recommendations
    USING (family_id = current_setting('app.current_family_id')::uuid);

CREATE POLICY onb_community_suggestions_family_policy ON onb_community_suggestions
    USING (family_id = current_setting('app.current_family_id')::uuid);
```

---

## §4 API Endpoints

All onboarding endpoints require authentication. The onboarding wizard operates within the
authenticated family context via `FamilyScope`. `[ARCH §3.3]`

### §4.1 Phase 1 Endpoints

#### `GET /v1/onboarding/progress`

Returns the current wizard progress for the authenticated family.

- **Auth**: Required (`FamilyScope`)
- **Response**: `WizardProgressResponse` (200 OK)
- **404**: If no wizard exists (family created before onboarding feature existed)

#### `PATCH /v1/onboarding/family-profile`

Updates family profile fields as part of the wizard. This is the wizard-specific path for
setting display_name, state_code, and location_region. Delegates to `iam::IamService`.

- **Auth**: Required (`FamilyScope`)
- **Body**: `UpdateFamilyProfileRequest` (`display_name`, `state_code`, `location_region`)
- **Validation**: `display_name` required (non-empty); `state_code` must be valid 2-letter code
- **Response**: `WizardProgressResponse` (200 OK) — wizard advances if this was the current step
- **Side effect**: Calls `iam::IamService::update_family_profile`

#### `POST /v1/onboarding/children`

Adds a student profile during the onboarding wizard. Delegates to `iam::IamService`.

- **Auth**: Required (`FamilyScope`, `RequireCoppaConsent`)
- **Body**: `CreateStudentRequest` (`display_name`, `birth_year?`, `grade_level?`)
- **Response**: `StudentResponse` (201 Created) — student data from IAM
- **Side effect**: Calls `iam::IamService::create_student`

#### `DELETE /v1/onboarding/children/:student_id`

Removes a student profile during onboarding. Delegates to `iam::IamService`.

- **Auth**: Required (`FamilyScope`)
- **Response**: 204 No Content
- **Side effect**: Calls `iam::IamService::delete_student`

#### `PATCH /v1/onboarding/methodology`

Selects the family's methodology during the wizard. Delegates validation and persistence
to `method::MethodologyService`.

- **Auth**: Required (`FamilyScope`)
- **Body**: `MethodologySelectionRequest` (`primary_methodology_id`, `secondary_methodology_ids?`, `methodology_path`, `explanation_acknowledged`)
- **Validation**: `methodology_path` must be one of `quiz_informed`, `exploration`, `skip`. `explanation_acknowledged` must be `true` if secondary methodologies are selected. All methodology IDs must be valid and active.
- **Response**: `WizardProgressResponse` (200 OK) — advances wizard to `roadmap_review`
- **Side effect**: Calls `method::MethodologyService::update_family_methodology`. Triggers roadmap, recommendations, and community suggestion materialization.

#### `POST /v1/onboarding/methodology/import-quiz`

Imports a previously taken quiz result to pre-populate methodology selection. Calls
`discover::DiscoveryService::claim_quiz_result` and `discover::DiscoveryService::get_quiz_result`.

- **Auth**: Required (`FamilyScope`)
- **Body**: `ImportQuizRequest` (`share_id`)
- **Response**: `QuizImportResponse` (200 OK) — quiz result data with methodology recommendations
- **404**: Quiz result not found
- **409**: Quiz result already claimed by a different family
- **Dependency note**: `discover::claim_quiz_result` is Phase 2 in 03-discover.md. This endpoint will return 501 Not Implemented until discover implements claiming. The `get_quiz_result` portion (viewing results) works immediately.

#### `GET /v1/onboarding/roadmap`

Returns the materialized getting-started roadmap for the family.

- **Auth**: Required (`FamilyScope`)
- **Response**: `RoadmapResponse` (200 OK) — list of roadmap items, grouped by age group
- **404**: If roadmap has not been materialized yet (methodology step not completed)

#### `GET /v1/onboarding/recommendations`

Returns starter curriculum recommendations for the family.

- **Auth**: Required (`FamilyScope`)
- **Response**: `RecommendationsResponse` (200 OK) — list of recommendations, grouped by age group
- **404**: If recommendations have not been materialized yet

#### `GET /v1/onboarding/community`

Returns community connection suggestions for the family.

- **Auth**: Required (`FamilyScope`)
- **Response**: `CommunitySuggestionsResponse` (200 OK) — list of group/mentor suggestions
- **404**: If suggestions have not been materialized yet

#### `POST /v1/onboarding/complete`

Marks the onboarding wizard as completed. Publishes `OnboardingCompleted` event.

- **Auth**: Required (`FamilyScope`)
- **Precondition**: `family_profile` and `methodology` steps must be in `completed_steps`
- **Response**: `WizardProgressResponse` (200 OK) — status set to `completed`
- **422**: If required steps are not completed
- **Events**: `OnboardingCompleted`

#### `POST /v1/onboarding/skip`

Allows the user to skip the remaining wizard steps. Sets status to `skipped`.

- **Auth**: Required (`FamilyScope`)
- **Precondition**: Wizard must be `in_progress`
- **Response**: `WizardProgressResponse` (200 OK) — status set to `skipped`
- **Events**: `OnboardingCompleted` (with `skipped: true`)

### §4.2 Phase 2 Endpoints

#### `PATCH /v1/onboarding/roadmap/:item_id/complete`

Marks a roadmap item as completed.

- **Auth**: Required (`FamilyScope`)
- **Response**: `RoadmapItemResponse` (200 OK)

#### `POST /v1/onboarding/restart`

Restarts the onboarding wizard for a family that has already completed it (e.g., after
methodology change). Resets wizard to `in_progress`, clears materialized data.

- **Auth**: Required (`FamilyScope`)
- **Response**: `WizardProgressResponse` (200 OK)

---

## §5 Service Interface

```rust
// src/onboard/ports.rs

#[async_trait]
pub trait OnboardingService: Send + Sync {
    // ─── Wizard Progress ──────────────────────────────────────────────

    /// Returns current wizard progress for the family.
    /// Used by GET /v1/onboarding/progress. [S§6]
    async fn get_progress(
        &self,
        scope: &FamilyScope,
    ) -> Result<WizardProgressResponse, AppError>;

    /// Initializes wizard progress for a new family.
    /// Called by FamilyCreated event handler. [S§6]
    async fn initialize_wizard(
        &self,
        family_id: FamilyId,
    ) -> Result<(), AppError>;

    // ─── Family Profile Step ──────────────────────────────────────────

    /// Updates family profile as part of onboarding wizard.
    /// Delegates to iam::IamService::update_family_profile.
    /// Advances wizard if family_profile is the current step. [S§6.2]
    async fn update_family_profile(
        &self,
        scope: &FamilyScope,
        cmd: UpdateFamilyProfileRequest,
    ) -> Result<WizardProgressResponse, AppError>;

    // ─── Children Step ────────────────────────────────────────────────

    /// Adds a student during onboarding. Delegates to iam::IamService. [S§6.2]
    async fn add_child(
        &self,
        scope: &FamilyScope,
        cmd: CreateStudentRequest,
    ) -> Result<StudentResponse, AppError>;

    /// Removes a student during onboarding. Delegates to iam::IamService. [S§6.2]
    async fn remove_child(
        &self,
        scope: &FamilyScope,
        student_id: Uuid,
    ) -> Result<(), AppError>;

    // ─── Methodology Step ─────────────────────────────────────────────

    /// Selects family methodology and materializes roadmap, recommendations,
    /// and community suggestions.
    /// Delegates methodology persistence to method::MethodologyService. [S§6.3]
    async fn select_methodology(
        &self,
        scope: &FamilyScope,
        cmd: MethodologySelectionRequest,
    ) -> Result<WizardProgressResponse, AppError>;

    /// Imports quiz results to pre-populate methodology selection.
    /// Calls discover::DiscoveryService::claim_quiz_result + get_quiz_result. [S§6.3]
    async fn import_quiz(
        &self,
        scope: &FamilyScope,
        cmd: ImportQuizRequest,
    ) -> Result<QuizImportResponse, AppError>;

    // ─── Roadmap & Recommendations ────────────────────────────────────

    /// Returns materialized roadmap items, grouped by age group. [S§6.4]
    async fn get_roadmap(
        &self,
        scope: &FamilyScope,
    ) -> Result<RoadmapResponse, AppError>;

    /// Returns materialized starter recommendations, grouped by age group. [S§6.5]
    async fn get_recommendations(
        &self,
        scope: &FamilyScope,
    ) -> Result<RecommendationsResponse, AppError>;

    /// Returns materialized community suggestions. [S§6.6]
    async fn get_community_suggestions(
        &self,
        scope: &FamilyScope,
    ) -> Result<CommunitySuggestionsResponse, AppError>;

    // ─── Wizard Completion ────────────────────────────────────────────

    /// Completes the onboarding wizard. Validates required steps are done.
    /// Publishes OnboardingCompleted event. [S§6]
    async fn complete_wizard(
        &self,
        scope: &FamilyScope,
    ) -> Result<WizardProgressResponse, AppError>;

    /// Skips remaining wizard steps. Publishes OnboardingCompleted (skipped=true). [S§6]
    async fn skip_wizard(
        &self,
        scope: &FamilyScope,
    ) -> Result<WizardProgressResponse, AppError>;

    // ─── Event Handlers ───────────────────────────────────────────────

    /// Handles FamilyMethodologyChanged event.
    /// Re-materializes roadmap, recommendations, and community suggestions
    /// if wizard is still in progress. [S§6.4]
    async fn handle_methodology_changed(
        &self,
        family_id: FamilyId,
        primary_methodology_id: Uuid,
        secondary_methodology_ids: Vec<Uuid>,
    ) -> Result<(), AppError>;

    // ─── Phase 2 ──────────────────────────────────────────────────────

    /// Marks a roadmap item as completed. (Phase 2) [S§6.4]
    async fn complete_roadmap_item(
        &self,
        scope: &FamilyScope,
        item_id: Uuid,
    ) -> Result<RoadmapItemResponse, AppError>;

    /// Restarts onboarding wizard. Resets progress, clears materialized data. (Phase 2)
    async fn restart_wizard(
        &self,
        scope: &FamilyScope,
    ) -> Result<WizardProgressResponse, AppError>;
}
```

---

## §6 Repository Interfaces

```rust
// src/onboard/ports.rs (continued)

#[async_trait]
pub trait WizardProgressRepository: Send + Sync {
    /// Creates a new wizard progress record for a family.
    async fn create(
        &self,
        family_id: FamilyId,
    ) -> Result<WizardProgress, AppError>;

    /// Finds wizard progress by family_id. Family-scoped.
    async fn find_by_family(
        &self,
        scope: &FamilyScope,
    ) -> Result<Option<WizardProgress>, AppError>;

    /// Updates wizard progress (current_step, completed_steps, status, etc.).
    async fn update(
        &self,
        scope: &FamilyScope,
        progress: &WizardProgress,
    ) -> Result<WizardProgress, AppError>;
}

#[async_trait]
pub trait RoadmapRepository: Send + Sync {
    /// Bulk-inserts roadmap items for a family wizard.
    async fn create_batch(
        &self,
        items: Vec<CreateRoadmapItem>,
    ) -> Result<Vec<RoadmapItem>, AppError>;

    /// Returns all roadmap items for a family, ordered by sort_order.
    async fn find_by_family(
        &self,
        scope: &FamilyScope,
    ) -> Result<Vec<RoadmapItem>, AppError>;

    /// Deletes all roadmap items for a wizard (used during re-materialization).
    async fn delete_by_wizard(
        &self,
        scope: &FamilyScope,
        wizard_id: Uuid,
    ) -> Result<(), AppError>;

    /// Marks a roadmap item as completed. (Phase 2)
    async fn mark_completed(
        &self,
        scope: &FamilyScope,
        item_id: Uuid,
    ) -> Result<RoadmapItem, AppError>;
}

#[async_trait]
pub trait RecommendationRepository: Send + Sync {
    /// Bulk-inserts starter recommendations for a family wizard.
    async fn create_batch(
        &self,
        items: Vec<CreateRecommendation>,
    ) -> Result<Vec<StarterRecommendation>, AppError>;

    /// Returns all recommendations for a family, ordered by sort_order.
    async fn find_by_family(
        &self,
        scope: &FamilyScope,
    ) -> Result<Vec<StarterRecommendation>, AppError>;

    /// Deletes all recommendations for a wizard (used during re-materialization).
    async fn delete_by_wizard(
        &self,
        scope: &FamilyScope,
        wizard_id: Uuid,
    ) -> Result<(), AppError>;
}

#[async_trait]
pub trait CommunitySuggestionRepository: Send + Sync {
    /// Bulk-inserts community suggestions for a family wizard.
    async fn create_batch(
        &self,
        items: Vec<CreateCommunitySuggestion>,
    ) -> Result<Vec<CommunitySuggestion>, AppError>;

    /// Returns all community suggestions for a family, ordered by sort_order.
    async fn find_by_family(
        &self,
        scope: &FamilyScope,
    ) -> Result<Vec<CommunitySuggestion>, AppError>;

    /// Deletes all community suggestions for a wizard (used during re-materialization).
    async fn delete_by_wizard(
        &self,
        scope: &FamilyScope,
        wizard_id: Uuid,
    ) -> Result<(), AppError>;
}
```

---

## §7 Adapter Interfaces

None. The onboarding domain has no external service dependencies. All cross-domain
communication goes through service traits (`iam::IamService`, `method::MethodologyService`,
`discover::DiscoveryService`) injected via `AppState`.

---

## §8 Models (DTOs)

### §8.1 Request Types

```rust
// src/onboard/models.rs

/// Family profile update during onboarding wizard. [S§6.2]
#[derive(Debug, Deserialize, Validate, ToSchema)]
pub struct UpdateFamilyProfileRequest {
    /// Family display name (required, non-empty)
    #[validate(length(min = 1, max = 200))]
    pub display_name: String,

    /// Two-letter state code for compliance. [S§6.2]
    /// Explained to user as needed for state-specific legal info.
    #[validate(length(equal = 2))]
    pub state_code: Option<String>,

    /// Coarse location region (city or region). [ARCH §1.5]
    #[validate(length(max = 200))]
    pub location_region: Option<String>,
}

/// Student creation during onboarding wizard. [S§6.2]
/// Mirrors iam::CreateStudentRequest — the handler delegates to IAM.
#[derive(Debug, Deserialize, Validate, ToSchema)]
pub struct CreateStudentRequest {
    #[validate(length(min = 1, max = 200))]
    pub display_name: String,

    /// Birth year for age-adapted roadmaps. [S§6.4]
    pub birth_year: Option<i16>,

    /// Optional grade level. [S§6.2]
    #[validate(length(max = 50))]
    pub grade_level: Option<String>,
}

/// Methodology selection during onboarding wizard. [S§6.3]
#[derive(Debug, Deserialize, Validate, ToSchema)]
pub struct MethodologySelectionRequest {
    /// Primary methodology ID (required). [S§6.3]
    pub primary_methodology_id: Uuid,

    /// Optional secondary methodology IDs. [S§6.3]
    #[serde(default)]
    pub secondary_methodology_ids: Vec<Uuid>,

    /// Which selection path the user took. [S§6.3]
    /// One of: "quiz_informed", "exploration", "skip"
    pub methodology_path: String,

    /// User acknowledged the multi-methodology model explanation.
    /// Required to be true when secondary methodologies are selected. [S§6.3]
    #[serde(default)]
    pub explanation_acknowledged: bool,
}

/// Quiz result import request. [S§6.3]
#[derive(Debug, Deserialize, ToSchema)]
pub struct ImportQuizRequest {
    /// The share_id from a previously taken quiz. [S§5.1.3]
    pub share_id: String,
}
```

### §8.2 Response Types

```rust
/// Wizard progress response. [S§6]
#[derive(Debug, Serialize, ToSchema)]
pub struct WizardProgressResponse {
    pub id: Uuid,
    pub family_id: Uuid,
    pub status: String,                    // "in_progress", "completed", "skipped"
    pub current_step: String,              // enum value as string
    pub completed_steps: Vec<String>,      // list of completed step names
    pub quiz_share_id: Option<String>,     // if quiz was imported
    pub methodology_path: Option<String>,  // which path was taken
    pub completed_at: Option<DateTime<Utc>>,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

/// Roadmap response, grouped by age group. [S§6.4]
#[derive(Debug, Serialize, ToSchema)]
pub struct RoadmapResponse {
    pub family_id: Uuid,
    pub groups: Vec<RoadmapAgeGroup>,
}

#[derive(Debug, Serialize, ToSchema)]
pub struct RoadmapAgeGroup {
    pub age_group: Option<String>,         // NULL means "all ages"
    pub items: Vec<RoadmapItemResponse>,
}

#[derive(Debug, Serialize, ToSchema)]
pub struct RoadmapItemResponse {
    pub id: Uuid,
    pub item_type: String,                 // "task", "resource", "feature_tour"
    pub title: String,
    pub description: Option<String>,
    pub link_url: Option<String>,
    pub age_group: Option<String>,
    pub sort_order: i16,
    pub is_completed: bool,
}

/// Starter recommendations response, grouped by age group. [S§6.5]
#[derive(Debug, Serialize, ToSchema)]
pub struct RecommendationsResponse {
    pub family_id: Uuid,
    pub groups: Vec<RecommendationAgeGroup>,
}

#[derive(Debug, Serialize, ToSchema)]
pub struct RecommendationAgeGroup {
    pub age_group: Option<String>,
    pub items: Vec<RecommendationItemResponse>,
}

#[derive(Debug, Serialize, ToSchema)]
pub struct RecommendationItemResponse {
    pub id: Uuid,
    pub title: String,
    pub description: Option<String>,
    pub link_url: Option<String>,
    pub price_type: String,                // "free" or "paid"
    pub age_group: Option<String>,
    pub sort_order: i16,
}

/// Community suggestions response. [S§6.6]
#[derive(Debug, Serialize, ToSchema)]
pub struct CommunitySuggestionsResponse {
    pub family_id: Uuid,
    pub suggestions: Vec<CommunitySuggestionResponse>,
}

#[derive(Debug, Serialize, ToSchema)]
pub struct CommunitySuggestionResponse {
    pub id: Uuid,
    pub suggestion_type: String,           // "methodology_group", "local_group", "mentor"
    pub title: String,
    pub description: Option<String>,
    pub reference_id: Option<Uuid>,        // ID of the social group or user
    pub sort_order: i16,
}

/// Quiz import response. [S§6.3]
#[derive(Debug, Serialize, ToSchema)]
pub struct QuizImportResponse {
    pub share_id: String,
    pub methodology_recommendations: Vec<QuizMethodologyRecommendation>,
    pub suggested_primary_id: Uuid,        // top recommendation
}

#[derive(Debug, Serialize, ToSchema)]
pub struct QuizMethodologyRecommendation {
    pub methodology_id: Uuid,
    pub methodology_name: String,
    pub score_percentage: i32,             // 0-100
    pub explanation: String,
}
```

### §8.3 Internal Types

```rust
/// Internal wizard progress model (maps to onb_wizard_progress row).
#[derive(Debug, Clone)]
pub struct WizardProgress {
    pub id: Uuid,
    pub family_id: Uuid,
    pub status: WizardStatus,
    pub current_step: WizardStep,
    pub completed_steps: Vec<WizardStep>,
    pub quiz_share_id: Option<String>,
    pub methodology_path: Option<MethodologyPath>,
    pub completed_at: Option<DateTime<Utc>>,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

#[derive(Debug, Clone, PartialEq)]
pub enum WizardStatus {
    InProgress,
    Completed,
    Skipped,
}

#[derive(Debug, Clone, PartialEq)]
pub enum WizardStep {
    FamilyProfile,
    Children,
    Methodology,
    RoadmapReview,
}

#[derive(Debug, Clone, PartialEq)]
pub enum MethodologyPath {
    QuizInformed,
    Exploration,
    Skip,
}

/// Internal roadmap item model (maps to onb_roadmap_items row).
#[derive(Debug, Clone)]
pub struct RoadmapItem {
    pub id: Uuid,
    pub family_id: Uuid,
    pub wizard_id: Uuid,
    pub methodology_id: Uuid,
    pub item_type: RoadmapItemType,
    pub title: String,
    pub description: Option<String>,
    pub link_url: Option<String>,
    pub age_group: Option<String>,
    pub sort_order: i16,
    pub is_completed: bool,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

#[derive(Debug, Clone, PartialEq)]
pub enum RoadmapItemType {
    Task,
    Resource,
    FeatureTour,
}

/// Input type for batch-creating roadmap items during materialization.
#[derive(Debug)]
pub struct CreateRoadmapItem {
    pub family_id: Uuid,
    pub wizard_id: Uuid,
    pub methodology_id: Uuid,
    pub item_type: RoadmapItemType,
    pub title: String,
    pub description: Option<String>,
    pub link_url: Option<String>,
    pub age_group: Option<String>,
    pub sort_order: i16,
}

/// Internal recommendation model (maps to onb_starter_recommendations row).
#[derive(Debug, Clone)]
pub struct StarterRecommendation {
    pub id: Uuid,
    pub family_id: Uuid,
    pub wizard_id: Uuid,
    pub methodology_id: Uuid,
    pub title: String,
    pub description: Option<String>,
    pub link_url: Option<String>,
    pub price_type: String,
    pub age_group: Option<String>,
    pub sort_order: i16,
    pub created_at: DateTime<Utc>,
}

/// Input type for batch-creating recommendations during materialization.
#[derive(Debug)]
pub struct CreateRecommendation {
    pub family_id: Uuid,
    pub wizard_id: Uuid,
    pub methodology_id: Uuid,
    pub title: String,
    pub description: Option<String>,
    pub link_url: Option<String>,
    pub price_type: String,
    pub age_group: Option<String>,
    pub sort_order: i16,
}

/// Internal community suggestion model (maps to onb_community_suggestions row).
#[derive(Debug, Clone)]
pub struct CommunitySuggestion {
    pub id: Uuid,
    pub family_id: Uuid,
    pub wizard_id: Uuid,
    pub methodology_id: Uuid,
    pub suggestion_type: String,
    pub title: String,
    pub description: Option<String>,
    pub reference_id: Option<Uuid>,
    pub sort_order: i16,
    pub created_at: DateTime<Utc>,
}

/// Input type for batch-creating community suggestions during materialization.
#[derive(Debug)]
pub struct CreateCommunitySuggestion {
    pub family_id: Uuid,
    pub wizard_id: Uuid,
    pub methodology_id: Uuid,
    pub suggestion_type: String,
    pub title: String,
    pub description: Option<String>,
    pub reference_id: Option<Uuid>,
    pub sort_order: i16,
}

/// Age bracket for roadmap/recommendation filtering. [S§6.4]
#[derive(Debug, Clone, PartialEq)]
pub enum AgeGroup {
    EarlyChildhood,   // 3-5
    EarlyElementary,  // 6-8
    UpperElementary,  // 9-11
    MiddleSchool,     // 12-14
    HighSchool,       // 15-18
}
```

---

## §9 Wizard State Machine

### §9.1 Steps

The onboarding wizard has 4 ordered steps:

| Step | Name | Required? | Purpose |
|------|------|-----------|---------|
| 1 | `family_profile` | **Yes** | Collect display name, state, location `[S§6.2]` |
| 2 | `children` | No | Add student profiles (can skip, add later) `[S§6.2]` |
| 3 | `methodology` | **Yes** | Select methodology via one of three paths `[S§6.3]` |
| 4 | `roadmap_review` | No | Review roadmap, recommendations, community `[S§6.4-6.6]` |

### §9.2 Navigation Rules

1. Steps proceed in order: `family_profile` → `children` → `methodology` → `roadmap_review`.
2. The user **may skip** optional steps (`children`, `roadmap_review`) and proceed to the next step.
3. The user **may revisit** any previously completed step (the wizard is not locked forward-only).
4. The user **may complete the wizard** at any point after completing all required steps
   (`family_profile` and `methodology`), regardless of whether optional steps are done.
5. The user **may skip the entire wizard** at any time via `POST /v1/onboarding/skip`.
6. Completing a step adds it to `completed_steps` and advances `current_step` to the next step.

### §9.3 Methodology Path Logic

The `methodology` step supports three paths per `[S§6.3]`:

| Path | Behavior |
|------|----------|
| `quiz_informed` | User imported quiz results. Primary methodology pre-populated from top quiz recommendation. User can adjust. |
| `exploration` | User browsed methodology summaries via `GET /v1/methodologies` (served by `method::`, not `onboard::`) and selected directly. The frontend calls `method::` endpoints directly for methodology listing and detail — `onboard::` does not proxy these. |
| `skip` | User selected "I don't know yet". The family's primary methodology is set to the default (via `method::get_default_methodology_id()`). A prompt to revisit is recorded for later notification. |

### §9.4 Quiz Import Flow

1. User provides `share_id` from a previously taken quiz.
2. `OnboardingService::import_quiz` calls `discover::DiscoveryService::get_quiz_result(share_id)`.
3. If the result exists, the service calls `discover::DiscoveryService::claim_quiz_result(scope, share_id)` to link it to the family.
4. The quiz result's methodology recommendations are returned as `QuizImportResponse`.
5. The top recommendation's methodology ID is provided as `suggested_primary_id`.
6. The wizard records `quiz_share_id` on `onb_wizard_progress`.
7. The user proceeds to methodology selection with pre-populated values (path = `quiz_informed`).

**Dependency**: `claim_quiz_result` is Phase 2 in 03-discover.md. Until implemented:
- `get_quiz_result` works (the quiz result data can be viewed)
- `claim_quiz_result` will fail — the onboard service should handle this gracefully by
  logging a warning and proceeding without claiming (the quiz data is still usable for
  pre-populating methodology selection)

### §9.5 State Persistence

Wizard state is persisted to `onb_wizard_progress` after every step action. This ensures:
- Users can close their browser and resume the wizard later.
- The wizard state survives server restarts.
- Multiple browser tabs see consistent state (last-write-wins).

### §9.6 Age Group Resolution

Student `birth_year` values from `iam::IamService::list_students` are converted to age
groups for roadmap and recommendation filtering:

```rust
impl AgeGroup {
    pub fn from_birth_year(birth_year: i16, current_year: i16) -> Self {
        let age = current_year - birth_year;
        match age {
            ..=5 => AgeGroup::EarlyChildhood,
            6..=8 => AgeGroup::EarlyElementary,
            9..=11 => AgeGroup::UpperElementary,
            12..=14 => AgeGroup::MiddleSchool,
            15.. => AgeGroup::HighSchool,
        }
    }

    pub fn as_str(&self) -> &'static str {
        match self {
            AgeGroup::EarlyChildhood => "3-5",
            AgeGroup::EarlyElementary => "6-8",
            AgeGroup::UpperElementary => "9-11",
            AgeGroup::MiddleSchool => "12-14",
            AgeGroup::HighSchool => "15-18",
        }
    }
}
```

If a family has no students (the `children` step was skipped), roadmaps and recommendations
are materialized without age filtering — all age groups are included.

---

## §10 Materialization

When the methodology step completes, the service materializes three types of data from
methodology configuration into `onb_` tables. This converts JSONB config into structured
rows that the frontend can render efficiently.

### §10.1 Materialization Trigger

Materialization occurs when:
1. The `methodology` wizard step is completed (user selects methodology)
2. A `FamilyMethodologyChanged` event is received while the wizard is in progress

Both triggers call the same internal materialization logic.

### §10.2 Data Source

All materialized data comes from `method_definitions` columns:
- **Roadmap items** ← `method_definitions.onboarding_config` → `{ "roadmap_steps": [...] }`
- **Starter recommendations** ← `method_definitions.onboarding_config` → `{ "starter_recs": [...] }`
- **Community suggestions** ← `method_definitions.community_config` → `{ "default_group_ids": [...] }`

The service calls `method::MethodologyService::get_methodology(slug)` for each of the family's
selected methodologies (primary + secondary) and extracts these JSONB values.

### §10.3 Age-Adapted Filtering

Roadmap items and recommendations in the JSONB config include an `age_group` field. During
materialization:

1. The service fetches the family's students via `iam::IamService::list_students(scope)`.
2. Each student's `birth_year` is converted to an `AgeGroup` (§9.6).
3. Config items are filtered to include only those matching the family's age groups, plus
   items with `age_group: null` (applicable to all ages).
4. If the family has no students, all items are included (no filtering).

**Age brackets** (5 groups):

| Age Group | Age Range | Label |
|-----------|-----------|-------|
| Early Childhood | 3-5 | `"3-5"` |
| Early Elementary | 6-8 | `"6-8"` |
| Upper Elementary | 9-11 | `"9-11"` |
| Middle School | 12-14 | `"12-14"` |
| High School | 15-18 | `"15-18"` |

### §10.4 Roadmap and Recommendation Materialization

**Roadmap items**: Each methodology's `onboarding_config.roadmap_steps` array produces rows
in `onb_roadmap_items`. Expected structure of each roadmap step in the JSONB:

```json
{
  "title": "Set up your first reading list",
  "description": "Create a reading list using the Reading Lists tool...",
  "item_type": "task",
  "link_url": "/app/tools/reading-lists/new",
  "age_group": "6-8",
  "sort_order": 1
}
```

**Starter recommendations**: Each methodology's `onboarding_config.starter_recs` array produces
rows in `onb_starter_recommendations`. Maximum 6 recommendations per age group (excess are
dropped by sort_order). Expected structure:

```json
{
  "title": "Ambleside Online Year 1",
  "description": "Free Charlotte Mason curriculum for grades 1-3",
  "link_url": "/app/marketplace/listings/ambleside-online-y1",
  "price_type": "free",
  "age_group": "6-8",
  "sort_order": 1
}
```

### §10.5 Community Suggestion Materialization

Community suggestions are materialized from `method_definitions.community_config`:

```json
{
  "default_group_ids": ["uuid-of-charlotte-mason-group"],
  "mentor_criteria": { "min_experience_years": 2 }
}
```

Phase 1: Each `default_group_id` becomes a `methodology_group` row in
`onb_community_suggestions`. The group title and description are sourced from the
`community_config` JSONB (not from a live `social::` query — the data is static
in methodology config).

Phase 2: Live queries to `social::` for nearby families and mentor matches will
supplement the static group suggestions.

### §10.6 Re-Materialization

When methodology changes during onboarding (via `FamilyMethodologyChanged` event):
1. Delete all existing rows in `onb_roadmap_items`, `onb_starter_recommendations`, and
   `onb_community_suggestions` for the wizard.
2. Re-materialize from the new methodology config.
3. Any user-completed roadmap items are lost (acceptable — methodology change is a reset).

---

## §11 Error Types

```rust
// src/onboard/models.rs (error section)

#[derive(Debug, thiserror::Error)]
pub enum OnboardError {
    #[error("wizard not found for this family")]
    WizardNotFound,

    #[error("wizard is not in progress")]
    WizardNotInProgress,

    #[error("wizard is already completed")]
    WizardAlreadyCompleted,

    #[error("required steps not completed: {0:?}")]
    RequiredStepsIncomplete(Vec<String>),

    #[error("invalid wizard step transition")]
    InvalidStepTransition,

    #[error("methodology explanation not acknowledged")]
    ExplanationNotAcknowledged,

    #[error("quiz result not found")]
    QuizResultNotFound,

    #[error("quiz result already claimed by another family")]
    QuizAlreadyClaimed,

    #[error("roadmap not yet materialized")]
    RoadmapNotMaterialized,

    #[error("recommendations not yet materialized")]
    RecommendationsNotMaterialized,

    #[error("community suggestions not yet materialized")]
    CommunitySuggestionsNotMaterialized,

    #[error("invalid methodology path: {0}")]
    InvalidMethodologyPath(String),
}

impl From<OnboardError> for AppError {
    fn from(err: OnboardError) -> Self {
        match err {
            OnboardError::WizardNotFound => AppError::NotFound,
            OnboardError::WizardNotInProgress => AppError::Conflict("wizard is not in progress".into()),
            OnboardError::WizardAlreadyCompleted => AppError::Conflict("wizard is already completed".into()),
            OnboardError::RequiredStepsIncomplete(_) => {
                AppError::Validation("required onboarding steps not completed".into())
            }
            OnboardError::InvalidStepTransition => {
                AppError::Validation("invalid wizard step transition".into())
            }
            OnboardError::ExplanationNotAcknowledged => {
                AppError::Validation("multi-methodology explanation must be acknowledged".into())
            }
            OnboardError::QuizResultNotFound => AppError::NotFound,
            OnboardError::QuizAlreadyClaimed => AppError::Conflict("quiz result already claimed".into()),
            OnboardError::RoadmapNotMaterialized => AppError::NotFound,
            OnboardError::RecommendationsNotMaterialized => AppError::NotFound,
            OnboardError::CommunitySuggestionsNotMaterialized => AppError::NotFound,
            OnboardError::InvalidMethodologyPath(_) => {
                AppError::Validation("invalid methodology selection path".into())
            }
        }
    }
}
```

### §11.1 HTTP Status Mapping

| OnboardError Variant | AppError Variant | HTTP Status |
|---------------------|------------------|-------------|
| `WizardNotFound` | `NotFound` | 404 |
| `WizardNotInProgress` | `Conflict` | 409 |
| `WizardAlreadyCompleted` | `Conflict` | 409 |
| `RequiredStepsIncomplete` | `Validation` | 422 |
| `InvalidStepTransition` | `Validation` | 422 |
| `ExplanationNotAcknowledged` | `Validation` | 422 |
| `QuizResultNotFound` | `NotFound` | 404 |
| `QuizAlreadyClaimed` | `Conflict` | 409 |
| `RoadmapNotMaterialized` | `NotFound` | 404 |
| `RecommendationsNotMaterialized` | `NotFound` | 404 |
| `CommunitySuggestionsNotMaterialized` | `NotFound` | 404 |
| `InvalidMethodologyPath` | `Validation` | 422 |

---

## §12 Cross-Domain Interactions

### §12.1 onboard:: Provides (consumed by other domains)

| Export | Consumers | Mechanism |
|--------|-----------|-----------|
| `OnboardingService` trait methods | None currently | `Arc<dyn OnboardingService>` via AppState |
| `OnboardingCompleted` event | `notify::` | Domain event (send welcome email, dismiss onboarding prompts) |

Onboarding is primarily a **consumer** of other domains' services. No other domain depends
on onboard::'s service for business logic.

### §12.2 onboard:: Consumes

| Dependency | Source | Purpose |
|-----------|--------|---------|
| Family profile CRUD | `iam::IamService` | Update family display_name, state_code, location during wizard `[S§6.2]` |
| Student CRUD | `iam::IamService` | Add/remove students during wizard `[S§6.2]` |
| Student listing | `iam::IamService` | List students for age-group resolution `[S§6.4]` |
| Methodology listing | `method::MethodologyService` | List methodologies for selection wizard `[S§6.3]` |
| Methodology detail | `method::MethodologyService` | Get onboarding_config and community_config for materialization `[S§6.4]` |
| Methodology validation + persistence | `method::MethodologyService` | Validate and set family methodology `[S§6.3]` |
| Default methodology ID | `method::MethodologyService` | Set methodology when user picks "skip" path `[S§6.3]` |
| Quiz result retrieval | `discover::DiscoveryService` | Get quiz results for methodology pre-population `[S§6.3]` |
| Quiz result claiming | `discover::DiscoveryService` | Link quiz result to family account `[S§5.1.3]` (Phase 2 dependency) |

### §12.3 Events onboard:: Publishes

Defined in `src/onboard/events.rs`. `[CODING §8.4]`

| Event | Subscribers | Effect |
|-------|------------|--------|
| `OnboardingCompleted { family_id, skipped: bool }` | `notify::` | Send welcome email with roadmap summary (if not skipped) |

```rust
// src/onboard/events.rs

#[derive(Clone, Debug)]
pub struct OnboardingCompleted {
    pub family_id: FamilyId,
    /// True if the user skipped the wizard rather than completing all steps.
    pub skipped: bool,
}
impl DomainEvent for OnboardingCompleted {}
```

### §12.4 Events onboard:: Subscribes To

| Event | Source | Effect |
|-------|--------|--------|
| `FamilyCreated { family_id, parent_id }` | `iam::` | Create `onb_wizard_progress` row for the new family. Sets wizard to `in_progress` at step `family_profile`. `[ARCH §4.6]` |
| `FamilyMethodologyChanged { family_id, ... }` | `method::` | If wizard is `in_progress`, re-materialize roadmap, recommendations, and community suggestions using the new methodology. `[S§6.4]` |

```rust
// src/onboard/event_handlers.rs
use crate::iam::events::FamilyCreated;           // defined in src/iam/events.rs (01-iam §13.3)
use crate::method::events::FamilyMethodologyChanged; // defined in src/method/events.rs (02-method §11.3)

pub struct FamilyCreatedHandler {
    onboarding_service: Arc<dyn OnboardingService>,
}

#[async_trait]
impl DomainEventHandler<FamilyCreated> for FamilyCreatedHandler {
    async fn handle(&self, event: &FamilyCreated) -> Result<(), AppError> {
        self.onboarding_service.initialize_wizard(event.family_id).await
    }
}

pub struct FamilyMethodologyChangedHandler {
    onboarding_service: Arc<dyn OnboardingService>,
}

#[async_trait]
impl DomainEventHandler<FamilyMethodologyChanged> for FamilyMethodologyChangedHandler {
    async fn handle(&self, event: &FamilyMethodologyChanged) -> Result<(), AppError> {
        self.onboarding_service.handle_methodology_changed(
            event.family_id,
            event.primary_methodology_id,
            event.secondary_methodology_ids.clone(),
        ).await
    }
}
```

---

## §13 Phase Scope

### Phase 1 — Foundation

**In scope**:
- `onb_` tables: `onb_wizard_progress`, `onb_roadmap_items`, `onb_starter_recommendations`,
  `onb_community_suggestions`
- Enum types: `onb_wizard_step_enum`, `onb_wizard_status_enum`, `onb_roadmap_item_type_enum`
- RLS policies for all `onb_` tables
- Authenticated endpoints: `GET /v1/onboarding/progress`, `PATCH /v1/onboarding/family-profile`,
  `POST /v1/onboarding/children`, `DELETE /v1/onboarding/children/:id`,
  `PATCH /v1/onboarding/methodology`, `POST /v1/onboarding/methodology/import-quiz`,
  `GET /v1/onboarding/roadmap`, `GET /v1/onboarding/recommendations`,
  `GET /v1/onboarding/community`, `POST /v1/onboarding/complete`, `POST /v1/onboarding/skip`
- `OnboardingService` trait + `OnboardingServiceImpl`
- 4 repository traits + PostgreSQL implementations
- Wizard state machine (4 steps, navigation rules)
- Materialization logic (roadmap, recommendations, community from methodology config)
- Age-group resolution from student birth_year
- `OnboardError` enum + HTTP mapping
- Domain events: `OnboardingCompleted`
- Event handlers: `FamilyCreatedHandler`, `FamilyMethodologyChangedHandler`
- All Phase 1 models (request, response, internal)
- OpenAPI spec + TypeScript type generation

**Note**: `POST /v1/onboarding/methodology/import-quiz` will partially work in Phase 1
(quiz viewing via `get_quiz_result`) but quiz claiming requires `discover::claim_quiz_result`
which is Phase 2 in 03-discover.md. The onboard handler will gracefully degrade (log warning,
skip claiming, still return quiz data for pre-population).

### Phase 2 — Depth

**In scope**:
- `PATCH /v1/onboarding/roadmap/:item_id/complete` (mark roadmap items done)
- `POST /v1/onboarding/restart` (restart wizard after methodology change)
- Live marketplace queries for recommendations (replace static config data)
- Live social queries for nearby families and mentor suggestions `[S§6.6]`
- Co-parent invite suggestion during wizard (once `iam::` co-parent flow is implemented)
- Full quiz claiming flow (once `discover::claim_quiz_result` is implemented)

### Phase 3+ — Expansion

**Deferred**:
- Onboarding analytics (step completion rates, drop-off analysis, time-to-complete)
- A/B testing for wizard step ordering and content
- Admin dashboard for managing roadmap/recommendation content
- Personalized recommendations based on user behavior (AI-driven)
- Onboarding re-engagement campaigns (email reminders for incomplete wizards)

---

## §14 Verification Checklist

Numbered assertions that MUST be true when the onboard:: implementation is correct. Use these
as acceptance criteria for code review and integration testing.

### Wizard Lifecycle

1. `FamilyCreated` event creates an `onb_wizard_progress` row with status `in_progress` and `current_step = family_profile`
2. `GET /v1/onboarding/progress` returns 200 with wizard state after family creation
3. `GET /v1/onboarding/progress` returns 404 for a family with no wizard record
4. Wizard steps proceed in order: `family_profile` → `children` → `methodology` → `roadmap_review`
5. Completing a step adds it to `completed_steps` and advances `current_step`
6. Optional steps (`children`, `roadmap_review`) can be skipped without error
7. Required steps (`family_profile`, `methodology`) must be completed before `POST /v1/onboarding/complete` succeeds
8. `POST /v1/onboarding/complete` returns 422 if required steps are missing
9. `POST /v1/onboarding/complete` sets status to `completed` and publishes `OnboardingCompleted`
10. `POST /v1/onboarding/skip` sets status to `skipped` and publishes `OnboardingCompleted { skipped: true }`
11. Wizard actions on a completed/skipped wizard return 409 Conflict

### Family Profile Step

12. `PATCH /v1/onboarding/family-profile` delegates to `iam::IamService::update_family_profile`
13. `PATCH /v1/onboarding/family-profile` with empty `display_name` returns 422
14. Completing the family profile step advances the wizard to the `children` step

### Children Step

15. `POST /v1/onboarding/children` delegates to `iam::IamService::create_student`
16. `DELETE /v1/onboarding/children/:id` delegates to `iam::IamService::delete_student`
17. `POST /v1/onboarding/children` requires COPPA consent (`RequireCoppaConsent` extractor)

### Methodology Step

18. `PATCH /v1/onboarding/methodology` validates all methodology IDs via `method::MethodologyService`
19. `PATCH /v1/onboarding/methodology` with secondary IDs but `explanation_acknowledged: false` returns 422
20. `PATCH /v1/onboarding/methodology` with `methodology_path: "skip"` uses `method::get_default_methodology_id()`
21. Completing the methodology step triggers materialization of roadmap, recommendations, and community suggestions
22. `methodology_path` is recorded on `onb_wizard_progress`

### Quiz Import

23. `POST /v1/onboarding/methodology/import-quiz` returns quiz result data with methodology recommendations
24. `POST /v1/onboarding/methodology/import-quiz` with nonexistent `share_id` returns 404
25. `POST /v1/onboarding/methodology/import-quiz` records `quiz_share_id` on wizard progress
26. Quiz import gracefully degrades when `discover::claim_quiz_result` is not yet implemented

### Materialization

27. Roadmap items are filtered by the family's student age groups
28. Families with no students receive all roadmap items (no age filtering)
29. Starter recommendations include both `free` and `paid` items with max 6 per age group
30. Community suggestions include methodology group references from `community_config`
31. `FamilyMethodologyChanged` event triggers re-materialization (old items deleted, new items created)

### Data Access & Security

32. All `onb_` queries use `FamilyScope` — cross-family access is impossible
33. RLS policies on all `onb_` tables enforce family_id matching
34. No code under `src/onboard/` branches on methodology name/slug
35. All API error responses return generic messages, not SQL or internal details
36. No PII appears in application logs

---

## §15 Implementation Checklist

### Phase 1 — Foundation

#### Database
- [ ] Create migration: `onb_wizard_step_enum`, `onb_wizard_status_enum`, `onb_roadmap_item_type_enum` types
- [ ] Create migration: `onb_wizard_progress` table
- [ ] Create migration: `onb_roadmap_items` table
- [ ] Create migration: `onb_starter_recommendations` table
- [ ] Create migration: `onb_community_suggestions` table
- [ ] Create migration: RLS policies for all `onb_` tables
- [ ] Regenerate SeaORM entities from migrations

#### Ports & Traits
- [ ] Define `OnboardingService` trait in `src/onboard/ports.rs`
- [ ] Define `WizardProgressRepository` trait in `src/onboard/ports.rs`
- [ ] Define `RoadmapRepository` trait in `src/onboard/ports.rs`
- [ ] Define `RecommendationRepository` trait in `src/onboard/ports.rs`
- [ ] Define `CommunitySuggestionRepository` trait in `src/onboard/ports.rs`

#### Error Types
- [ ] Define `OnboardError` enum
- [ ] Implement `From<OnboardError> for AppError` conversion

#### Repository Implementations
- [ ] Implement `PgWizardProgressRepository`
- [ ] Implement `PgRoadmapRepository`
- [ ] Implement `PgRecommendationRepository`
- [ ] Implement `PgCommunitySuggestionRepository`

#### Service Implementation
- [ ] Implement `OnboardingServiceImpl` with all Phase 1 methods
- [ ] Implement wizard state machine (step advancement, completion validation)
- [ ] Implement materialization logic (roadmap, recommendations, community from methodology config)
- [ ] Implement age-group resolution from student birth_year
- [ ] Implement quiz import with graceful degradation for missing `claim_quiz_result`
- [ ] Wire `OnboardingServiceImpl` in `app.rs` with `Arc<dyn OnboardingService>`

#### Event Handlers
- [ ] Implement `FamilyCreatedHandler` → `initialize_wizard`
- [ ] Implement `FamilyMethodologyChangedHandler` → `handle_methodology_changed`
- [ ] Register event handlers in `app.rs` EventBus wiring

#### Domain Events
- [ ] Define `OnboardingCompleted` event in `src/onboard/events.rs`
- [ ] Publish `OnboardingCompleted` from `complete_wizard` and `skip_wizard`

#### API Endpoints
- [ ] `GET    /v1/onboarding/progress` — wizard progress
- [ ] `PATCH  /v1/onboarding/family-profile` — update family profile
- [ ] `POST   /v1/onboarding/children` — add student
- [ ] `DELETE /v1/onboarding/children/:id` — remove student
- [ ] `PATCH  /v1/onboarding/methodology` — select methodology
- [ ] `POST   /v1/onboarding/methodology/import-quiz` — import quiz result
- [ ] `GET    /v1/onboarding/roadmap` — get roadmap
- [ ] `GET    /v1/onboarding/recommendations` — get recommendations
- [ ] `GET    /v1/onboarding/community` — get community suggestions
- [ ] `POST   /v1/onboarding/complete` — complete wizard
- [ ] `POST   /v1/onboarding/skip` — skip wizard
- [ ] Register all endpoints in `authenticated_routes()` (all require auth)

#### Models
- [ ] Define `UpdateFamilyProfileRequest` request type
- [ ] Define `CreateStudentRequest` request type
- [ ] Define `MethodologySelectionRequest` request type
- [ ] Define `ImportQuizRequest` request type
- [ ] Define all response types (`WizardProgressResponse`, `RoadmapResponse`, etc.)
- [ ] Define all internal types (`WizardProgress`, `WizardStep`, `AgeGroup`, etc.)

#### Code Generation
- [ ] Regenerate OpenAPI spec (`cargo run --bin openapi-gen`)
- [ ] Regenerate TypeScript types (`cd frontend && npm run generate-types`)

#### Testing
- [ ] Unit tests for wizard state machine (step advancement, completion validation, skip)
- [ ] Unit tests for age-group resolution (birth_year → AgeGroup for all 5 brackets)
- [ ] Unit tests for materialization logic (JSONB config → structured items)
- [ ] Unit tests for age-adapted filtering (items filtered by family's student ages)
- [ ] Integration test: wizard happy path (create family → complete all steps → complete wizard)
- [ ] Integration test: wizard skip path (create family → skip wizard)
- [ ] Integration test: methodology selection triggers materialization
- [ ] Integration test: required step validation on complete
- [ ] Integration test: quiz import with valid share_id
- [ ] Integration test: cross-family access prevention (FamilyScope enforcement)

### Phase 2

- [ ] Implement `complete_roadmap_item` in service
- [ ] Add `PATCH /v1/onboarding/roadmap/:item_id/complete` endpoint
- [ ] Implement `restart_wizard` in service
- [ ] Add `POST /v1/onboarding/restart` endpoint
- [ ] Integrate live marketplace queries for recommendations (replace static config data)
- [ ] Integrate live social queries for nearby families and mentor suggestions
- [ ] Full quiz claiming integration (once `discover::claim_quiz_result` is available)

---

## §16 Module Structure

```
src/onboard/
├── mod.rs              # Re-exports, domain-level doc comments
├── handlers.rs         # Axum route handlers (all authenticated)
├── service.rs          # OnboardingServiceImpl — wizard state machine,
│                       # materialization, cross-domain orchestration
├── repository.rs       # PgWizardProgressRepository, PgRoadmapRepository,
│                       # PgRecommendationRepository, PgCommunitySuggestionRepository
├── models.rs           # Request/response types, OnboardError enum, internal types
├── ports.rs            # OnboardingService trait, all repository traits
├── events.rs           # OnboardingCompleted event definition
├── event_handlers.rs   # FamilyCreatedHandler, FamilyMethodologyChangedHandler
└── entities/           # SeaORM-generated entity files (do not hand-edit)
```

**Note**: No `domain/` subdirectory (non-complex domain — workflow orchestration, no aggregate
roots). No `adapters/` directory (no external service integrations). Event handlers are in a
separate file because onboard:: subscribes to events from two other domains. `[ARCH §4.5]`
