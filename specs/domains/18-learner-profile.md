# 18 — Learner Profile (`internal/learner_profile`)

> Approved by board: HOM-93 (child of HOM-89). Implementation migration: 30.

---

## § 1. Purpose

Provides a per-student "observable engagement preferences" profile built from a 14-question quiz (12 scored + 2 interest). Solves two problems:
1. Gives parents a readable summary of *how* their child engages with learning.
2. Provides a cold-start prior for the Recommendations engine (`internal/recs`) before behavioral signals accumulate.

**v0 scope (this spec):** Declared data only. No behavioral feedback loop. Schema is forward-compatible for v1.

---

## § 2. Data Model

### 2.1 `learner_profiles` (migration 30)

Family-scoped via RLS (standard pattern).

| Column | Type | Notes |
|--------|------|-------|
| `id` | UUID PK | `gen_random_uuid()` |
| `family_id` | UUID FK → `iam_families` | CASCADE DELETE |
| `student_id` | UUID FK → `iam_students` | CASCADE DELETE; UNIQUE |
| `activity_format` | NUMERIC(5,4) | 0=text/listen, 1=hands-on; NULL=unanswered |
| `session_length` | NUMERIC(5,4) | 0=short-bursts, 1=long-deep-dives |
| `motivation` | NUMERIC(5,4) | 0=mastery, 1=discovery |
| `solo_collaborative` | NUMERIC(5,4) | 0=solo, 1=collaborative |
| `structure` | NUMERIC(5,4) | 0=step-by-step, 1=open-ended |
| `outdoor_kinesthetic` | NUMERIC(5,4) | 0=not-important, 1=think-better-moving |
| `interests` | TEXT[] | Controlled vocabulary = `mkt_listings.subject_tags` vocab |
| `answered_count` | SMALLINT | Count of the 12 scored questions answered |
| `confidence` | NUMERIC(4,3) | = answered_count / 12; stored for query efficiency |
| `source` | TEXT | `'declared'` in v0; `'blended'` in v1 |
| `respondent` | TEXT | `'parent'` \| `'child'` |
| `created_at` | TIMESTAMPTZ | |
| `updated_at` | TIMESTAMPTZ | |

**Constraints:**
- `UNIQUE (student_id)` — one profile per student; retake overwrites.
- `CHECK (confidence BETWEEN 0 AND 1)`
- `CHECK (source IN ('declared', 'blended'))`
- NULL means unanswered; do NOT use 0.0 as "unanswered" sentinel.

### 2.2 `mkt_listings.preference_tags` (migration 30)

Nullable JSONB column added to existing table. Keys match `learner_profiles` dimension column names. NULL = untagged; badge suppressed for untagged content (fail-safe).

```sql
ALTER TABLE mkt_listings ADD COLUMN preference_tags JSONB;
```

### 2.3 `recs_recommendations.fit_score` / `fit_why` (migration 30)

```sql
ALTER TABLE recs_recommendations
    ADD COLUMN fit_score NUMERIC(4,3),
    ADD COLUMN fit_why TEXT;
```

### 2.4 `onb_wizard_step_enum` (migration 30)

Add `learner_profile` to the PostgreSQL enum for the optional 5th wizard step.

---

## § 3. Package Structure

```
internal/learner_profile/
  models.go          — GORM model, request/response types
  ports.go           — IamServiceForLearnerProfile (consumer-defined)
                       ProfileRepository, LearnerProfileService interfaces
  repository.go      — PgProfileRepository (gorm impl)
  service.go         — learnerProfileServiceImpl
  handlers.go        — Echo handlers
  quiz.go            — Quiz questions, dimension→vector computation, fit scoring
  event_handlers.go  — HandleStudentDeletion, HandleFamilyDeletion
```

---

## § 4. API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/v1/students/:student_id/learner-profile/submissions` | Submit quiz (upserts profile) |
| `GET`  | `/v1/students/:student_id/learner-profile` | Get existing profile for student |

### 4.1 POST /v1/students/:student_id/learner-profile/submissions

Request body:
```json
{
  "answers": [{"question_id": 1, "value": 0.75}],
  "respondent": "parent",
  "interests": ["art", "nature_study"]
}
```

Response: `LearnerProfileResponse` (the created/updated profile).

**Upsert semantics:** Submitting overwrites the existing profile. Retake = same endpoint.

### 4.2 GET /v1/students/:student_id/learner-profile

Returns `LearnerProfileResponse` or 404 if no profile exists yet.

---

## § 5. Quiz Engine

### 5.1 Question List (14 questions)

12 scored questions map to the 6 dimensions (2 questions per dimension). 2 interest questions (multi-select, mapped to `subject_tags` vocabulary).

**Dimension → Question mapping:**
- `activity_format` (Q1, Q2): How does the student prefer to engage with material?
- `session_length` (Q3, Q4): How long do they prefer to work on one thing?
- `motivation` (Q5, Q6): Mastery-driven or discovery-driven?
- `solo_collaborative` (Q7, Q8): Solo work vs. working with others?
- `structure` (Q9, Q10): Step-by-step vs. open-ended exploration?
- `outdoor_kinesthetic` (Q11, Q12): Physical movement / outdoor learning importance?
- `interests` (Q13–14): Interest chip multi-select (parent + child variants)

Per-dimension value = mean of the 2 questions covering that dimension. Computed server-side.

### 5.2 Respondent Variants

- **Parent variant** (respondent=`parent`): "How does [name] prefer..." — observational questions
- **Child variant** (respondent=`child`): "How do YOU like to..." — first-person, emoji options
- **Age floor for child self-report:** 8 years old. Under 8 → parent-only mode.

### 5.3 Computation

```
answered_count = # of the 12 scored questions with a non-skip answer
confidence = answered_count / 12
per-dimension value = mean(answer values for that dimension)
```

---

## § 6. Fit Score

Computed during `handleComputeRecommendationsTask` in `internal/recs/tasks.go`.

### 6.1 Formula

```
fit_score = mean(1 - |P_d - C_d|) for each tagged dimension d
```

Where `P_d` = profile value, `C_d` = content `preference_tags` value (both 0.0–1.0).
Only dimensions where both P and C are non-NULL contribute.

**Interest boost:** +0.10 if content `subject_tags` ∩ student `interests` ≠ ∅ (capped at 1.0).

### 6.2 Badge Gate

Both conditions must pass:
- `fit_score >= 0.60`
- `confidence >= 0.60`

### 6.3 Why Text

Generated from the highest-contributing dimension using server-side templates:
```
"activity_format"     → "{name} loves hands-on, build-it learning."
"session_length"      → "{name} gets absorbed — long, deep-dive content is their sweet spot."
"motivation"          → "{name} is driven by discovery over mastery drills."
"solo_collaborative"  → "{name} learns well with others."
"structure"           → "{name} thrives with step-by-step structure."
"outdoor_kinesthetic" → "{name} thinks better when moving."
```

Student name substituted server-side (keeps PII out of client template logic).

### 6.4 Exploration Exclusion

`SourceExploration` candidates never receive a fit badge.

---

## § 7. Recs Integration (Cross-Domain)

### 7.1 Consumer-Defined Interface

`internal/recs` MUST NOT import `internal/learner_profile`. Port defined in recs:

```go
// internal/recs/ports.go
type LearnerProfilePort interface {
    GetStudentInterestsByFamily(ctx context.Context, familyID shared.FamilyID) (map[uuid.UUID][]string, error)
}
```

Wired in `main.go` via function adapter (same pattern as `IamServiceForRecs`).

### 7.2 Cold-Start Prior

When a family has < 3 behavioral signals in 90 days, seed `recentSubjectTags` from declared interests:

```go
if len(signals) < 3 {
    for _, interests := range studentInterests { // from LearnerProfilePort
        recentSubjectTags = append(recentSubjectTags, interests...)
        recentSubjectTags = append(recentSubjectTags, interests...) // 2x weight
    }
}
```

Does not alter scoring weights. Raises Jaccard relevance for matching content.

### 7.3 Model Changes

`NewRecommendation`, `Recommendation`, `RecommendationResponse` gain:
- `FitScore *float32` (omit from JSON when nil)
- `FitWhy *string` (omit from JSON when nil)

---

## § 8. Onboarding

Optional 5th step `learner_profile` in wizard after `roadmap_review`.

Changes to `internal/onboard/models.go`:
- Add `StepLearnerProfile WizardStep = "learner_profile"` constant
- Append to `wizardStepOrder` (NOT to `requiredSteps` — it is optional)

The step key per child is `learner_profile_{studentId}` on the frontend side.

---

## § 9. Privacy

- Family-scoped RLS — same guarantees as all other family data.
- No GPS coordinates, no free text, no sensitive categories.
- `interests TEXT[]` uses controlled vocabulary (no open input).
- Declared profile data is **excluded** from `recs_anonymized_interactions` pipeline. Hard constraint.
- Deletion paths:
  1. Student deletion → CASCADE + explicit `profileRepo.DeleteByStudent()` in `HandleStudentDeletion`
  2. Family deletion → CASCADE + `HandleFamilyDeletion`
  3. Retake → upsert overwrites; no history

---

## § 10. Frontend (FitBadge Token)

The `FitBadge` component (`src/components/ui/fit-badge.tsx`) uses:
- `bg-tertiary-container text-on-tertiary-container` ✓ (corrected; original was `primary-container`)

---

## § 11. Feature Flag (Content Tagging Rollout)

Badge surfacing in discovery UI gated behind:
```go
const FitBadgeSurfacingEnabled = false // enable when ≥ 100 content items are tagged
```

In v0, badges appear only on recommendation cards where `fit_score` is pre-computed.

---

## § 12. App Wiring

Add to `internal/app/app.go` AppState:
```go
LearnerProfile learner_profile.LearnerProfileService
```

Register routes in `cmd/server/main.go`:
```go
learner_profile.RegisterHandlers(e, &app, authMiddleware)
```

---

## § 13. Frontend — ProfileSummary Display Contract

Route: `/students/:studentId/learner-profile`  
Component tree: `LearnerProfilePage` → `ProfileSummary`

### 13.1 Dimension Display

All 6 dimension scores (`activity_format`, `session_length`, `motivation`, `solo_collaborative`, `structure`, `outdoor_kinesthetic`) are passed from `LearnerProfileResponse` to `ProfileSummary` as a `dimensions` prop. Dimensions with a `null`/`undefined` value are silently omitted (partially-completed quizzes are valid).

Each dimension renders:
- **Parent-friendly label** (see table below) — not the raw DB column name
- **Readable description** derived from the 0.0–1.0 value using three threshold buckets (< 0.4 / 0.4–0.65 / ≥ 0.65)
- **Visual bar** — track `bg-surface-container-high`, fill `bg-primary`, width = `value × 100%`
- **Pole labels** at each end of the bar

| Column | Label | Low pole | High pole |
|--------|-------|----------|-----------|
| `activity_format` | Learning Style | Listens & Reads | Hands-On |
| `session_length` | Focus Stamina | Short Bursts | Deep Dives |
| `motivation` | What Drives Learning | Mastery | Discovery |
| `solo_collaborative` | Learning Together | Works Alone | With Others |
| `structure` | Guidance Preference | Step-by-Step | Open-Ended |
| `outdoor_kinesthetic` | Movement & Space | Desk-Based | On the Move |

### 13.2 Interest Formatting

Raw `interests` slugs (e.g., `nature_study`) are formatted client-side to Title Case with underscores replaced by spaces. No server-side mapping is required.

### 13.3 Recommendations Explanation Panel

Below the dimensions, `ProfileSummary` renders a fixed "How this shapes recommendations" panel explaining:
1. Fit badges on matched content
2. Ranking boost for profile-aligned recommendations
3. Immediate effect of retaking the quiz

The panel includes a link to `/recommendations`.

### 13.4 Props Contract

```typescript
type ProfileSummaryProps = {
  studentName: string;
  summaryText: string;           // pre-composed server-side summary
  interests: string[];            // controlled-vocab slugs
  dimensions?: Partial<Record<string, number>>;  // 0.0–1.0 per dimension
  onRetake: () => void;
  onEditInterests: () => void;
};
```
