# Domain Spec 04 — Onboarding & Family Setup (onboard::)

## §1 Overview

The Onboarding domain owns the **post-account-creation wizard** that guides new families from
a bare account to a configured homeschooling workspace. It orchestrates interactions between
IAM (family/student data), methodology (config lookup), and discovery (quiz results) to produce
a personalized first experience. Onboarding owns no core business entities — it owns workflow
state and materialized guidance artifacts. `[S§6]`

| Attribute | Value |
|-----------|-------|
| **Module path** | `internal/onboard/` |
| **DB prefix** | `onb_` |
| **Complexity class** | Non-complex `[ARCH §4.5]` |
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
| Multi-methodology model explanation | `[S§6.3]` | §8 (`MethodologySelectionRequest.ExplanationAcknowledged`) |
| Methodology-specific getting-started roadmap | `[S§6.4]` | §4 (`GET /v1/onboarding/roadmap`), §10 |
| Concrete, actionable first-week checklist | `[S§6.4]` | §10.2 (item types: `task`, `resource`, `feature_tour`) |
| Age-adapted roadmaps | `[S§6.4]` | §10.3 (5 age brackets from student birth_year) |
| Roadmap items link to platform features and content | `[S§6.4]` | §10.2 (`LinkURL` field on roadmap items) |
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
    id                    UUID PRIMARY KEY DEFAULT uuidv7(),
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
    id                    UUID PRIMARY KEY DEFAULT uuidv7(),
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
    id                    UUID PRIMARY KEY DEFAULT uuidv7(),
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
    id                    UUID PRIMARY KEY DEFAULT uuidv7(),
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
- **Side effect**: Calls `iam::IamService::UpdateFamilyProfile`

#### `POST /v1/onboarding/children`

Adds a student profile during the onboarding wizard. Delegates to `iam::IamService`.

- **Auth**: Required (`FamilyScope`, `RequireCoppaConsent`)
- **Body**: `CreateStudentRequest` (`display_name`, `birth_year?`, `grade_level?`)
- **Response**: `StudentResponse` (201 Created) — student data from IAM
- **Side effect**: Calls `iam::IamService::CreateStudent`

#### `DELETE /v1/onboarding/children/:student_id`

Removes a student profile during onboarding. Delegates to `iam::IamService`.

- **Auth**: Required (`FamilyScope`)
- **Response**: 204 No Content
- **Side effect**: Calls `iam::IamService::DeleteStudent`

#### `PATCH /v1/onboarding/methodology`

Selects the family's methodology during the wizard. Delegates validation and persistence
to `method::MethodologyService`.

- **Auth**: Required (`FamilyScope`)
- **Body**: `MethodologySelectionRequest` (`primary_methodology_id`, `secondary_methodology_ids?`, `methodology_path`, `explanation_acknowledged`)
- **Validation**: `methodology_path` must be one of `quiz_informed`, `exploration`, `skip`. `explanation_acknowledged` must be `true` if secondary methodologies are selected. All methodology IDs must be valid and active.
- **Response**: `WizardProgressResponse` (200 OK) — advances wizard to `roadmap_review`
- **Side effect**: Calls `method::MethodologyService::UpdateFamilyMethodology`. Triggers roadmap, recommendations, and community suggestion materialization.

#### `POST /v1/onboarding/methodology/import-quiz`

Imports a previously taken quiz result to pre-populate methodology selection. Calls
`discover::DiscoveryService::ClaimQuizResult` and `discover::DiscoveryService::GetQuizResult`.

- **Auth**: Required (`FamilyScope`)
- **Body**: `ImportQuizRequest` (`share_id`)
- **Response**: `QuizImportResponse` (200 OK) — quiz result data with methodology recommendations
- **404**: Quiz result not found
- **409**: Quiz result already claimed by a different family
- **Dependency note**: `discover::ClaimQuizResult` is Phase 2 in 03-discover.md. This endpoint will return 501 Not Implemented until discover implements claiming. The `GetQuizResult` portion (viewing results) works immediately.

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
- **Events**: `OnboardingCompleted` (with `Skipped: true`)

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

```go
// internal/onboard/ports.go

// OnboardingService defines the onboarding domain's service interface.
type OnboardingService interface {
    // ─── Wizard Progress ──────────────────────────────────────────────

    // GetProgress returns current wizard progress for the family.
    // Used by GET /v1/onboarding/progress. [S§6]
    GetProgress(ctx context.Context, scope *FamilyScope) (*WizardProgressResponse, error)

    // InitializeWizard initializes wizard progress for a new family.
    // Called by FamilyCreated event handler. [S§6]
    InitializeWizard(ctx context.Context, familyID uuid.UUID) error

    // ─── Family Profile Step ──────────────────────────────────────────

    // UpdateFamilyProfile updates family profile as part of onboarding wizard.
    // Delegates to iam.IamService.UpdateFamilyProfile.
    // Advances wizard if family_profile is the current step. [S§6.2]
    UpdateFamilyProfile(ctx context.Context, scope *FamilyScope, cmd *UpdateFamilyProfileRequest) (*WizardProgressResponse, error)

    // ─── Children Step ────────────────────────────────────────────────

    // AddChild adds a student during onboarding. Delegates to iam.IamService. [S§6.2]
    AddChild(ctx context.Context, scope *FamilyScope, cmd *CreateStudentRequest) (*StudentResponse, error)

    // RemoveChild removes a student during onboarding. Delegates to iam.IamService. [S§6.2]
    RemoveChild(ctx context.Context, scope *FamilyScope, studentID uuid.UUID) error

    // ─── Methodology Step ─────────────────────────────────────────────

    // SelectMethodology selects family methodology and materializes roadmap,
    // recommendations, and community suggestions.
    // Delegates methodology persistence to method.MethodologyService. [S§6.3]
    SelectMethodology(ctx context.Context, scope *FamilyScope, cmd *MethodologySelectionRequest) (*WizardProgressResponse, error)

    // ImportQuiz imports quiz results to pre-populate methodology selection.
    // Calls discover.DiscoveryService.ClaimQuizResult + GetQuizResult. [S§6.3]
    ImportQuiz(ctx context.Context, scope *FamilyScope, cmd *ImportQuizRequest) (*QuizImportResponse, error)

    // ─── Roadmap & Recommendations ────────────────────────────────────

    // GetRoadmap returns materialized roadmap items, grouped by age group. [S§6.4]
    GetRoadmap(ctx context.Context, scope *FamilyScope) (*RoadmapResponse, error)

    // GetRecommendations returns materialized starter recommendations, grouped by age group. [S§6.5]
    GetRecommendations(ctx context.Context, scope *FamilyScope) (*RecommendationsResponse, error)

    // GetCommunitySuggestions returns materialized community suggestions. [S§6.6]
    GetCommunitySuggestions(ctx context.Context, scope *FamilyScope) (*CommunitySuggestionsResponse, error)

    // ─── Wizard Completion ────────────────────────────────────────────

    // CompleteWizard completes the onboarding wizard. Validates required steps are done.
    // Publishes OnboardingCompleted event. [S§6]
    CompleteWizard(ctx context.Context, scope *FamilyScope) (*WizardProgressResponse, error)

    // SkipWizard skips remaining wizard steps. Publishes OnboardingCompleted (skipped=true). [S§6]
    SkipWizard(ctx context.Context, scope *FamilyScope) (*WizardProgressResponse, error)

    // ─── Event Handlers ───────────────────────────────────────────────

    // HandleMethodologyChanged handles FamilyMethodologyChanged event.
    // Re-materializes roadmap, recommendations, and community suggestions
    // if wizard is still in progress. [S§6.4]
    HandleMethodologyChanged(ctx context.Context, familyID uuid.UUID, primaryMethodologyID uuid.UUID, secondaryMethodologyIDs []uuid.UUID) error

    // ─── Phase 2 ──────────────────────────────────────────────────────

    // CompleteRoadmapItem marks a roadmap item as completed. (Phase 2) [S§6.4]
    CompleteRoadmapItem(ctx context.Context, scope *FamilyScope, itemID uuid.UUID) (*RoadmapItemResponse, error)

    // RestartWizard restarts onboarding wizard. Resets progress, clears materialized data. (Phase 2)
    RestartWizard(ctx context.Context, scope *FamilyScope) (*WizardProgressResponse, error)
}
```

---

## §6 Repository Interfaces

```go
// internal/onboard/ports.go (continued)

// WizardProgressRepository defines persistence operations for onb_wizard_progress.
type WizardProgressRepository interface {
    // Create creates a new wizard progress record for a family.
    Create(ctx context.Context, familyID uuid.UUID) (*WizardProgress, error)

    // FindByFamily finds wizard progress by family_id. Family-scoped.
    FindByFamily(ctx context.Context, scope *FamilyScope) (*WizardProgress, error)

    // Update updates wizard progress (current_step, completed_steps, status, etc.).
    Update(ctx context.Context, scope *FamilyScope, progress *WizardProgress) (*WizardProgress, error)
}

// RoadmapRepository defines persistence operations for onb_roadmap_items.
type RoadmapRepository interface {
    // CreateBatch bulk-inserts roadmap items for a family wizard.
    CreateBatch(ctx context.Context, items []CreateRoadmapItem) ([]RoadmapItem, error)

    // FindByFamily returns all roadmap items for a family, ordered by sort_order.
    FindByFamily(ctx context.Context, scope *FamilyScope) ([]RoadmapItem, error)

    // DeleteByWizard deletes all roadmap items for a wizard (used during re-materialization).
    DeleteByWizard(ctx context.Context, scope *FamilyScope, wizardID uuid.UUID) error

    // MarkCompleted marks a roadmap item as completed. (Phase 2)
    MarkCompleted(ctx context.Context, scope *FamilyScope, itemID uuid.UUID) (*RoadmapItem, error)
}

// RecommendationRepository defines persistence operations for onb_starter_recommendations.
type RecommendationRepository interface {
    // CreateBatch bulk-inserts starter recommendations for a family wizard.
    CreateBatch(ctx context.Context, items []CreateRecommendation) ([]StarterRecommendation, error)

    // FindByFamily returns all recommendations for a family, ordered by sort_order.
    FindByFamily(ctx context.Context, scope *FamilyScope) ([]StarterRecommendation, error)

    // DeleteByWizard deletes all recommendations for a wizard (used during re-materialization).
    DeleteByWizard(ctx context.Context, scope *FamilyScope, wizardID uuid.UUID) error
}

// CommunitySuggestionRepository defines persistence operations for onb_community_suggestions.
type CommunitySuggestionRepository interface {
    // CreateBatch bulk-inserts community suggestions for a family wizard.
    CreateBatch(ctx context.Context, items []CreateCommunitySuggestion) ([]CommunitySuggestion, error)

    // FindByFamily returns all community suggestions for a family, ordered by sort_order.
    FindByFamily(ctx context.Context, scope *FamilyScope) ([]CommunitySuggestion, error)

    // DeleteByWizard deletes all community suggestions for a wizard (used during re-materialization).
    DeleteByWizard(ctx context.Context, scope *FamilyScope, wizardID uuid.UUID) error
}
```

---

## §7 Adapter Interfaces

None. The onboarding domain has no external service dependencies. All cross-domain
communication goes through service interfaces (`iam.IamService`, `method.MethodologyService`,
`discover.DiscoveryService`) injected via dependency injection.

---

## §8 Models (DTOs)

### §8.1 Request Types

```go
// internal/onboard/models.go

// UpdateFamilyProfileRequest represents the family profile update during onboarding wizard. [S§6.2]
type UpdateFamilyProfileRequest struct {
    // Family display name (required, non-empty)
    DisplayName string `json:"display_name" validate:"required,min=1,max=200"`

    // Two-letter state code for compliance. [S§6.2]
    // Explained to user as needed for state-specific legal info.
    StateCode *string `json:"state_code,omitempty" validate:"omitempty,len=2"`

    // Coarse location region (city or region). [ARCH §1.5]
    LocationRegion *string `json:"location_region,omitempty" validate:"omitempty,max=200"`
}

// CreateStudentRequest represents student creation during onboarding wizard. [S§6.2]
// Mirrors iam.CreateStudentRequest — the handler delegates to IAM.
type CreateStudentRequest struct {
    DisplayName string `json:"display_name" validate:"required,min=1,max=200"`

    // Birth year for age-adapted roadmaps. [S§6.4]
    BirthYear *int16 `json:"birth_year,omitempty"`

    // Optional grade level. [S§6.2]
    GradeLevel *string `json:"grade_level,omitempty" validate:"omitempty,max=50"`
}

// MethodologySelectionRequest represents methodology selection during onboarding wizard. [S§6.3]
type MethodologySelectionRequest struct {
    // Primary methodology ID (required). [S§6.3]
    PrimaryMethodologyID uuid.UUID `json:"primary_methodology_id" validate:"required"`

    // Optional secondary methodology IDs. [S§6.3]
    SecondaryMethodologyIDs []uuid.UUID `json:"secondary_methodology_ids"`

    // Which selection path the user took. [S§6.3]
    // One of: "quiz_informed", "exploration", "skip"
    MethodologyPath string `json:"methodology_path" validate:"required"`

    // User acknowledged the multi-methodology model explanation.
    // Required to be true when secondary methodologies are selected. [S§6.3]
    ExplanationAcknowledged bool `json:"explanation_acknowledged"`
}

// ImportQuizRequest represents quiz result import request. [S§6.3]
type ImportQuizRequest struct {
    // The share_id from a previously taken quiz. [S§5.1.3]
    ShareID string `json:"share_id" validate:"required"`
}
```

### §8.2 Response Types

```go
// WizardProgressResponse represents wizard progress response. [S§6]
type WizardProgressResponse struct {
    ID               uuid.UUID  `json:"id"`
    FamilyID         uuid.UUID  `json:"family_id"`
    Status           string     `json:"status"`             // "in_progress", "completed", "skipped"
    CurrentStep      string     `json:"current_step"`       // enum value as string
    CompletedSteps   []string   `json:"completed_steps"`    // list of completed step names
    QuizShareID      *string    `json:"quiz_share_id"`      // if quiz was imported
    MethodologyPath  *string    `json:"methodology_path"`   // which path was taken
    CompletedAt      *time.Time `json:"completed_at"`
    CreatedAt        time.Time  `json:"created_at"`
    UpdatedAt        time.Time  `json:"updated_at"`
}

// RoadmapResponse represents roadmap response, grouped by age group. [S§6.4]
type RoadmapResponse struct {
    FamilyID uuid.UUID        `json:"family_id"`
    Groups   []RoadmapAgeGroup `json:"groups"`
}

// RoadmapAgeGroup represents a group of roadmap items for a specific age range.
type RoadmapAgeGroup struct {
    AgeGroup *string              `json:"age_group"` // nil means "all ages"
    Items    []RoadmapItemResponse `json:"items"`
}

// RoadmapItemResponse represents a single roadmap item.
type RoadmapItemResponse struct {
    ID          uuid.UUID `json:"id"`
    ItemType    string    `json:"item_type"` // "task", "resource", "feature_tour"
    Title       string    `json:"title"`
    Description *string   `json:"description"`
    LinkURL     *string   `json:"link_url"`
    AgeGroup    *string   `json:"age_group"`
    SortOrder   int16     `json:"sort_order"`
    IsCompleted bool      `json:"is_completed"`
}

// RecommendationsResponse represents starter recommendations response, grouped by age group. [S§6.5]
type RecommendationsResponse struct {
    FamilyID uuid.UUID                  `json:"family_id"`
    Groups   []RecommendationAgeGroup   `json:"groups"`
}

// RecommendationAgeGroup represents a group of recommendations for a specific age range.
type RecommendationAgeGroup struct {
    AgeGroup *string                      `json:"age_group"`
    Items    []RecommendationItemResponse `json:"items"`
}

// RecommendationItemResponse represents a single starter recommendation.
type RecommendationItemResponse struct {
    ID          uuid.UUID `json:"id"`
    Title       string    `json:"title"`
    Description *string   `json:"description"`
    LinkURL     *string   `json:"link_url"`
    PriceType   string    `json:"price_type"` // "free" or "paid"
    AgeGroup    *string   `json:"age_group"`
    SortOrder   int16     `json:"sort_order"`
}

// CommunitySuggestionsResponse represents community suggestions response. [S§6.6]
type CommunitySuggestionsResponse struct {
    FamilyID    uuid.UUID                     `json:"family_id"`
    Suggestions []CommunitySuggestionResponse `json:"suggestions"`
}

// CommunitySuggestionResponse represents a single community suggestion.
type CommunitySuggestionResponse struct {
    ID             uuid.UUID  `json:"id"`
    SuggestionType string     `json:"suggestion_type"` // "methodology_group", "local_group", "mentor"
    Title          string     `json:"title"`
    Description    *string    `json:"description"`
    ReferenceID    *uuid.UUID `json:"reference_id"` // ID of the social group or user
    SortOrder      int16      `json:"sort_order"`
}

// QuizImportResponse represents quiz import response. [S§6.3]
type QuizImportResponse struct {
    ShareID                     string                          `json:"share_id"`
    MethodologyRecommendations []QuizMethodologyRecommendation `json:"methodology_recommendations"`
    SuggestedPrimaryID         uuid.UUID                       `json:"suggested_primary_id"` // top recommendation
}

// QuizMethodologyRecommendation represents a quiz-derived methodology recommendation.
type QuizMethodologyRecommendation struct {
    MethodologyID   uuid.UUID `json:"methodology_id"`
    MethodologyName string    `json:"methodology_name"`
    ScorePercentage int32     `json:"score_percentage"` // 0-100
    Explanation     string    `json:"explanation"`
}
```

### §8.3 Internal Types

```go
// WizardProgress represents the internal wizard progress model (maps to onb_wizard_progress row).
type WizardProgress struct {
    ID              uuid.UUID
    FamilyID        uuid.UUID
    Status          WizardStatus
    CurrentStep     WizardStep
    CompletedSteps  []WizardStep
    QuizShareID     *string
    MethodologyPath *MethodologyPath
    CompletedAt     *time.Time
    CreatedAt       time.Time
    UpdatedAt       time.Time
}

// WizardStatus represents the overall wizard status.
type WizardStatus string

const (
    WizardStatusInProgress WizardStatus = "in_progress"
    WizardStatusCompleted  WizardStatus = "completed"
    WizardStatusSkipped    WizardStatus = "skipped"
)

// WizardStep represents a wizard step identifier.
type WizardStep string

const (
    WizardStepFamilyProfile WizardStep = "family_profile"
    WizardStepChildren      WizardStep = "children"
    WizardStepMethodology   WizardStep = "methodology"
    WizardStepRoadmapReview WizardStep = "roadmap_review"
)

// MethodologyPath represents the methodology selection path.
type MethodologyPath string

const (
    MethodologyPathQuizInformed MethodologyPath = "quiz_informed"
    MethodologyPathExploration  MethodologyPath = "exploration"
    MethodologyPathSkip         MethodologyPath = "skip"
)

// RoadmapItem represents the internal roadmap item model (maps to onb_roadmap_items row).
type RoadmapItem struct {
    ID             uuid.UUID
    FamilyID       uuid.UUID
    WizardID       uuid.UUID
    MethodologyID  uuid.UUID
    ItemType       RoadmapItemType
    Title          string
    Description    *string
    LinkURL        *string
    AgeGroup       *string
    SortOrder      int16
    IsCompleted    bool
    CreatedAt      time.Time
    UpdatedAt      time.Time
}

// RoadmapItemType represents the type of roadmap item.
type RoadmapItemType string

const (
    RoadmapItemTypeTask        RoadmapItemType = "task"
    RoadmapItemTypeResource    RoadmapItemType = "resource"
    RoadmapItemTypeFeatureTour RoadmapItemType = "feature_tour"
)

// CreateRoadmapItem is the input type for batch-creating roadmap items during materialization.
type CreateRoadmapItem struct {
    FamilyID      uuid.UUID
    WizardID      uuid.UUID
    MethodologyID uuid.UUID
    ItemType      RoadmapItemType
    Title         string
    Description   *string
    LinkURL       *string
    AgeGroup      *string
    SortOrder     int16
}

// StarterRecommendation represents the internal recommendation model (maps to onb_starter_recommendations row).
type StarterRecommendation struct {
    ID            uuid.UUID
    FamilyID      uuid.UUID
    WizardID      uuid.UUID
    MethodologyID uuid.UUID
    Title         string
    Description   *string
    LinkURL       *string
    PriceType     string
    AgeGroup      *string
    SortOrder     int16
    CreatedAt     time.Time
}

// CreateRecommendation is the input type for batch-creating recommendations during materialization.
type CreateRecommendation struct {
    FamilyID      uuid.UUID
    WizardID      uuid.UUID
    MethodologyID uuid.UUID
    Title         string
    Description   *string
    LinkURL       *string
    PriceType     string
    AgeGroup      *string
    SortOrder     int16
}

// CommunitySuggestion represents the internal community suggestion model (maps to onb_community_suggestions row).
type CommunitySuggestion struct {
    ID             uuid.UUID
    FamilyID       uuid.UUID
    WizardID       uuid.UUID
    MethodologyID  uuid.UUID
    SuggestionType string
    Title          string
    Description    *string
    ReferenceID    *uuid.UUID
    SortOrder      int16
    CreatedAt      time.Time
}

// CreateCommunitySuggestion is the input type for batch-creating community suggestions during materialization.
type CreateCommunitySuggestion struct {
    FamilyID       uuid.UUID
    WizardID       uuid.UUID
    MethodologyID  uuid.UUID
    SuggestionType string
    Title          string
    Description    *string
    ReferenceID    *uuid.UUID
    SortOrder      int16
}

// AgeGroup represents an age bracket for roadmap/recommendation filtering. [S§6.4]
type AgeGroup string

const (
    AgeGroupEarlyChildhood  AgeGroup = "3-5"
    AgeGroupEarlyElementary AgeGroup = "6-8"
    AgeGroupUpperElementary AgeGroup = "9-11"
    AgeGroupMiddleSchool    AgeGroup = "12-14"
    AgeGroupHighSchool      AgeGroup = "15-18"
)
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
| `skip` | User selected "I don't know yet". The family's primary methodology is set to the default (via `method.GetDefaultMethodologyID()`). A prompt to revisit is recorded for later notification. |

### §9.4 Quiz Import Flow

1. User provides `share_id` from a previously taken quiz.
2. `OnboardingService.ImportQuiz` calls `discover.DiscoveryService.GetQuizResult(shareID)`.
3. If the result exists, the service calls `discover.DiscoveryService.ClaimQuizResult(scope, shareID)` to link it to the family.
4. The quiz result's methodology recommendations are returned as `QuizImportResponse`.
5. The top recommendation's methodology ID is provided as `SuggestedPrimaryID`.
6. The wizard records `QuizShareID` on `onb_wizard_progress`.
7. The user proceeds to methodology selection with pre-populated values (path = `quiz_informed`).

**Dependency**: `ClaimQuizResult` is Phase 2 in 03-discover.md. Until implemented:
- `GetQuizResult` works (the quiz result data can be viewed)
- `ClaimQuizResult` will fail — the onboard service should handle this gracefully by
  logging a warning and proceeding without claiming (the quiz data is still usable for
  pre-populating methodology selection)

### §9.5 State Persistence

Wizard state is persisted to `onb_wizard_progress` after every step action. This ensures:
- Users can close their browser and resume the wizard later.
- The wizard state survives server restarts.
- Multiple browser tabs see consistent state (last-write-wins).

### §9.6 Age Group Resolution

Student `birth_year` values from `iam.IamService.ListStudents` are converted to age
groups for roadmap and recommendation filtering:

```go
// AgeGroupFromBirthYear converts a birth year to an AgeGroup.
func AgeGroupFromBirthYear(birthYear int16, currentYear int16) AgeGroup {
    age := currentYear - birthYear
    switch {
    case age <= 5:
        return AgeGroupEarlyChildhood
    case age <= 8:
        return AgeGroupEarlyElementary
    case age <= 11:
        return AgeGroupUpperElementary
    case age <= 14:
        return AgeGroupMiddleSchool
    default:
        return AgeGroupHighSchool
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
- **Roadmap items** <- `method_definitions.onboarding_config` -> `{ "roadmap_steps": [...] }`
- **Starter recommendations** <- `method_definitions.onboarding_config` -> `{ "starter_recs": [...] }`
- **Community suggestions** <- `method_definitions.community_config` -> `{ "default_group_ids": [...] }`

The service calls `method.MethodologyService.GetMethodology(slug)` for each of the family's
selected methodologies (primary + secondary) and extracts these JSONB values.

### §10.3 Age-Adapted Filtering

Roadmap items and recommendations in the JSONB config include an `age_group` field. During
materialization:

1. The service fetches the family's students via `iam.IamService.ListStudents(scope)`.
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

```go
// internal/onboard/errors.go

import "errors"

var (
    ErrWizardNotFound                    = errors.New("wizard not found for this family")
    ErrWizardNotInProgress               = errors.New("wizard is not in progress")
    ErrWizardAlreadyCompleted            = errors.New("wizard is already completed")
    ErrRequiredStepsIncomplete           = errors.New("required steps not completed")
    ErrInvalidStepTransition             = errors.New("invalid wizard step transition")
    ErrExplanationNotAcknowledged        = errors.New("methodology explanation not acknowledged")
    ErrQuizResultNotFound                = errors.New("quiz result not found")
    ErrQuizAlreadyClaimed                = errors.New("quiz result already claimed by another family")
    ErrRoadmapNotMaterialized            = errors.New("roadmap not yet materialized")
    ErrRecommendationsNotMaterialized    = errors.New("recommendations not yet materialized")
    ErrCommunitySuggestionsNotMaterialized = errors.New("community suggestions not yet materialized")
    ErrInvalidMethodologyPath            = errors.New("invalid methodology path")
)
```

### §11.1 HTTP Status Mapping

| Error | HTTP Status |
|-------|-------------|
| `ErrWizardNotFound` | 404 |
| `ErrWizardNotInProgress` | 409 |
| `ErrWizardAlreadyCompleted` | 409 |
| `ErrRequiredStepsIncomplete` | 422 |
| `ErrInvalidStepTransition` | 422 |
| `ErrExplanationNotAcknowledged` | 422 |
| `ErrQuizResultNotFound` | 404 |
| `ErrQuizAlreadyClaimed` | 409 |
| `ErrRoadmapNotMaterialized` | 404 |
| `ErrRecommendationsNotMaterialized` | 404 |
| `ErrCommunitySuggestionsNotMaterialized` | 404 |
| `ErrInvalidMethodologyPath` | 422 |

---

## §12 Cross-Domain Interactions

### §12.1 onboard:: Provides (consumed by other domains)

| Export | Consumers | Mechanism |
|--------|-----------|-----------|
| `OnboardingService` interface methods | None currently | Interface value via DI |
| `OnboardingCompleted` event | `notify::` | Domain event (send welcome email, dismiss onboarding prompts) |

Onboarding is primarily a **consumer** of other domains' services. No other domain depends
on onboard::'s service for business logic.

### §12.2 onboard:: Consumes

| Dependency | Source | Purpose |
|-----------|--------|---------|
| Family profile CRUD | `iam.IamService` | Update family display_name, state_code, location during wizard `[S§6.2]` |
| Student CRUD | `iam.IamService` | Add/remove students during wizard `[S§6.2]` |
| Student listing | `iam.IamService` | List students for age-group resolution `[S§6.4]` |
| Methodology listing | `method.MethodologyService` | List methodologies for selection wizard `[S§6.3]` |
| Methodology detail | `method.MethodologyService` | Get onboarding_config and community_config for materialization `[S§6.4]` |
| Methodology validation + persistence | `method.MethodologyService` | Validate and set family methodology `[S§6.3]` |
| Default methodology ID | `method.MethodologyService` | Set methodology when user picks "skip" path `[S§6.3]` |
| Quiz result retrieval | `discover.DiscoveryService` | Get quiz results for methodology pre-population `[S§6.3]` |
| Quiz result claiming | `discover.DiscoveryService` | Link quiz result to family account `[S§5.1.3]` (Phase 2 dependency) |

### §12.3 Events onboard:: Publishes

Defined in `internal/onboard/events.go`. `[CODING §8.4]`

| Event | Subscribers | Effect |
|-------|------------|--------|
| `OnboardingCompleted { FamilyID, Skipped bool }` | `notify::` | Send welcome email with roadmap summary (if not skipped) |

```go
// internal/onboard/events.go

// OnboardingCompleted is published when a family completes or skips the onboarding wizard.
type OnboardingCompleted struct {
    FamilyID uuid.UUID
    // Skipped is true if the user skipped the wizard rather than completing all steps.
    Skipped bool
}
```

### §12.4 Events onboard:: Subscribes To

| Event | Source | Effect |
|-------|--------|--------|
| `FamilyCreated { FamilyID, ParentID }` | `iam::` | Create `onb_wizard_progress` row for the new family. Sets wizard to `in_progress` at step `family_profile`. `[ARCH §4.6]` |
| `FamilyMethodologyChanged { FamilyID, ... }` | `method::` | If wizard is `in_progress`, re-materialize roadmap, recommendations, and community suggestions using the new methodology. `[S§6.4]` |

```go
// internal/onboard/event_handlers.go

// FamilyCreatedHandler handles the FamilyCreated event from iam::.
type FamilyCreatedHandler struct {
    onboardingService OnboardingService
}

func (h *FamilyCreatedHandler) Handle(ctx context.Context, event *iam.FamilyCreated) error {
    return h.onboardingService.InitializeWizard(ctx, event.FamilyID)
}

// FamilyMethodologyChangedHandler handles the FamilyMethodologyChanged event from method::.
type FamilyMethodologyChangedHandler struct {
    onboardingService OnboardingService
}

func (h *FamilyMethodologyChangedHandler) Handle(ctx context.Context, event *method.FamilyMethodologyChanged) error {
    return h.onboardingService.HandleMethodologyChanged(
        ctx,
        event.FamilyID,
        event.PrimaryMethodologyID,
        event.SecondaryMethodologyIDs,
    )
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
- `OnboardingService` interface + `OnboardingServiceImpl`
- 4 repository interfaces + GORM implementations
- Wizard state machine (4 steps, navigation rules)
- Materialization logic (roadmap, recommendations, community from methodology config)
- Age-group resolution from student birth_year
- `OnboardError` sentinel errors + HTTP mapping
- Domain events: `OnboardingCompleted`
- Event handlers: `FamilyCreatedHandler`, `FamilyMethodologyChangedHandler`
- All Phase 1 models (request, response, internal)
- OpenAPI spec + TypeScript type generation

**Note**: `POST /v1/onboarding/methodology/import-quiz` will partially work in Phase 1
(quiz viewing via `GetQuizResult`) but quiz claiming requires `discover.ClaimQuizResult`
which is Phase 2 in 03-discover.md. The onboard handler will gracefully degrade (log warning,
skip claiming, still return quiz data for pre-population).

### Phase 2 — Depth

**In scope**:
- `PATCH /v1/onboarding/roadmap/:item_id/complete` (mark roadmap items done)
- `POST /v1/onboarding/restart` (restart wizard after methodology change)
- Live marketplace queries for recommendations (replace static config data)
- Live social queries for nearby families and mentor suggestions `[S§6.6]`
- Co-parent invite suggestion during wizard (once `iam::` co-parent flow is implemented)
- Full quiz claiming flow (once `discover.ClaimQuizResult` is implemented)

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
4. Wizard steps proceed in order: `family_profile` -> `children` -> `methodology` -> `roadmap_review`
5. Completing a step adds it to `completed_steps` and advances `current_step`
6. Optional steps (`children`, `roadmap_review`) can be skipped without error
7. Required steps (`family_profile`, `methodology`) must be completed before `POST /v1/onboarding/complete` succeeds
8. `POST /v1/onboarding/complete` returns 422 if required steps are missing
9. `POST /v1/onboarding/complete` sets status to `completed` and publishes `OnboardingCompleted`
10. `POST /v1/onboarding/skip` sets status to `skipped` and publishes `OnboardingCompleted { Skipped: true }`
11. Wizard actions on a completed/skipped wizard return 409 Conflict

### Family Profile Step

12. `PATCH /v1/onboarding/family-profile` delegates to `iam.IamService.UpdateFamilyProfile`
13. `PATCH /v1/onboarding/family-profile` with empty `display_name` returns 422
14. Completing the family profile step advances the wizard to the `children` step

### Children Step

15. `POST /v1/onboarding/children` delegates to `iam.IamService.CreateStudent`
16. `DELETE /v1/onboarding/children/:id` delegates to `iam.IamService.DeleteStudent`
17. `POST /v1/onboarding/children` requires COPPA consent (`RequireCoppaConsent` middleware)

### Methodology Step

18. `PATCH /v1/onboarding/methodology` validates all methodology IDs via `method.MethodologyService`
19. `PATCH /v1/onboarding/methodology` with secondary IDs but `explanation_acknowledged: false` returns 422
20. `PATCH /v1/onboarding/methodology` with `methodology_path: "skip"` uses `method.GetDefaultMethodologyID()`
21. Completing the methodology step triggers materialization of roadmap, recommendations, and community suggestions
22. `methodology_path` is recorded on `onb_wizard_progress`

### Quiz Import

23. `POST /v1/onboarding/methodology/import-quiz` returns quiz result data with methodology recommendations
24. `POST /v1/onboarding/methodology/import-quiz` with nonexistent `share_id` returns 404
25. `POST /v1/onboarding/methodology/import-quiz` records `quiz_share_id` on wizard progress
26. Quiz import gracefully degrades when `discover.ClaimQuizResult` is not yet implemented

### Materialization

27. Roadmap items are filtered by the family's student age groups
28. Families with no students receive all roadmap items (no age filtering)
29. Starter recommendations include both `free` and `paid` items with max 6 per age group
30. Community suggestions include methodology group references from `community_config`
31. `FamilyMethodologyChanged` event triggers re-materialization (old items deleted, new items created)

### Data Access & Security

32. All `onb_` queries use `FamilyScope` — cross-family access is impossible
33. RLS policies on all `onb_` tables enforce family_id matching
34. No code under `internal/onboard/` branches on methodology name/slug
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
- [ ] Define GORM models in `internal/onboard/models.go`

#### Ports & Interfaces
- [ ] Define `OnboardingService` interface in `internal/onboard/ports.go`
- [ ] Define `WizardProgressRepository` interface in `internal/onboard/ports.go`
- [ ] Define `RoadmapRepository` interface in `internal/onboard/ports.go`
- [ ] Define `RecommendationRepository` interface in `internal/onboard/ports.go`
- [ ] Define `CommunitySuggestionRepository` interface in `internal/onboard/ports.go`

#### Error Types
- [ ] Define `OnboardError` sentinel errors
- [ ] Implement HTTP status mapping in error handler

#### Repository Implementations
- [ ] Implement `GormWizardProgressRepository`
- [ ] Implement `GormRoadmapRepository`
- [ ] Implement `GormRecommendationRepository`
- [ ] Implement `GormCommunitySuggestionRepository`

#### Service Implementation
- [ ] Implement `OnboardingServiceImpl` with all Phase 1 methods
- [ ] Implement wizard state machine (step advancement, completion validation)
- [ ] Implement materialization logic (roadmap, recommendations, community from methodology config)
- [ ] Implement age-group resolution from student birth_year
- [ ] Implement quiz import with graceful degradation for missing `ClaimQuizResult`
- [ ] Wire `OnboardingServiceImpl` in app setup with interface injection

#### Event Handlers
- [ ] Implement `FamilyCreatedHandler` -> `InitializeWizard`
- [ ] Implement `FamilyMethodologyChangedHandler` -> `HandleMethodologyChanged`
- [ ] Register event handlers in app setup EventBus wiring

#### Domain Events
- [ ] Define `OnboardingCompleted` event in `internal/onboard/events.go`
- [ ] Publish `OnboardingCompleted` from `CompleteWizard` and `SkipWizard`

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
- [ ] Register all endpoints in authenticated route group (all require auth)

#### Models
- [ ] Define `UpdateFamilyProfileRequest` request type
- [ ] Define `CreateStudentRequest` request type
- [ ] Define `MethodologySelectionRequest` request type
- [ ] Define `ImportQuizRequest` request type
- [ ] Define all response types (`WizardProgressResponse`, `RoadmapResponse`, etc.)
- [ ] Define all internal types (`WizardProgress`, `WizardStep`, `AgeGroup`, etc.)

#### Code Generation
- [ ] Regenerate OpenAPI spec (`swag init`)
- [ ] Regenerate TypeScript types (`cd frontend && npm run generate-types`)

#### Testing
- [ ] Unit tests for wizard state machine (step advancement, completion validation, skip)
- [ ] Unit tests for age-group resolution (birth_year -> AgeGroup for all 5 brackets)
- [ ] Unit tests for materialization logic (JSONB config -> structured items)
- [ ] Unit tests for age-adapted filtering (items filtered by family's student ages)
- [ ] Integration test: wizard happy path (create family -> complete all steps -> complete wizard)
- [ ] Integration test: wizard skip path (create family -> skip wizard)
- [ ] Integration test: methodology selection triggers materialization
- [ ] Integration test: required step validation on complete
- [ ] Integration test: quiz import with valid share_id
- [ ] Integration test: cross-family access prevention (FamilyScope enforcement)

### Phase 2

- [ ] Implement `CompleteRoadmapItem` in service
- [ ] Add `PATCH /v1/onboarding/roadmap/:item_id/complete` endpoint
- [ ] Implement `RestartWizard` in service
- [ ] Add `POST /v1/onboarding/restart` endpoint
- [ ] Integrate live marketplace queries for recommendations (replace static config data)
- [ ] Integrate live social queries for nearby families and mentor suggestions
- [ ] Full quiz claiming integration (once `discover.ClaimQuizResult` is available)

---

## §16 Module Structure

```
internal/onboard/
├── handler.go          # Echo route handlers (all authenticated)
├── service.go          # OnboardingServiceImpl — wizard state machine,
│                       # materialization, cross-domain orchestration
├── repository.go       # GormWizardProgressRepository, GormRoadmapRepository,
│                       # GormRecommendationRepository, GormCommunitySuggestionRepository
├── models.go           # GORM models, request/response types, internal types
├── ports.go            # OnboardingService interface, all repository interfaces
├── errors.go           # OnboardError sentinel errors
├── events.go           # OnboardingCompleted event definition
└── event_handlers.go   # FamilyCreatedHandler, FamilyMethodologyChangedHandler
```

**Note**: No `domain/` subdirectory (non-complex domain — workflow orchestration, no aggregate
roots). No `adapters/` directory (no external service integrations). Event handlers are in a
separate file because onboard:: subscribes to events from two other domains. `[ARCH §4.5]`
