# AI/ML Future Capabilities — Phase 3 & 4

This document is the authoritative specification for the AI/ML capabilities planned for
Phases 3 and 4 of the Recommendations & Signals domain (`13-recs`). Phases 1–2 are
rule-based heuristics; this spec introduces machine-learning-backed recommendations while
preserving the platform's core privacy and COPPA compliance constraints.

---

## Governing Constraints (Non-Negotiable)

1. **COPPA compliance**: No student-identifiable data (name, DOB, exact age, exact location)
   may be used for ML training or sent to any external API.
2. **Privacy-first**: All ML training data is sourced exclusively from
   `recs_anonymized_interactions` (HMAC-anonymized, no `family_id`, no `student_id`).
3. **Opt-in for external APIs**: Features requiring external API calls (e.g., embedding
   generation) MUST be disabled by default and require explicit operator configuration.
4. **Content neutrality**: Collaborative filtering MUST NOT amplify worldview-correlated
   recommendations. The content neutrality invariant from `13-recs §10.8` applies equally
   to ML-derived scores.
5. **Methodology constraint**: ML scores are inputs to, not replacements for, the
   methodology-constrained scoring pipeline. CF scores are blended with the existing
   `ScoringFactors`, not applied independently.
6. **No breaking changes to Phase 1–2**: Phase 3 enhancements are additive. The existing
   rule-based engine remains the fallback when ML data is insufficient (cold-start).

---

## Phase 3 — Advanced Algorithms

### Phase 3.1: Item–Item Collaborative Filtering

**Approach**: Item–item CF on the existing `recs_anonymized_interactions` table.
No external ML service. Pure PostgreSQL + Go matrix math.

**Design rationale**: We use item–item (not user–user) CF because:
- Our anonymization deliberately strips family identity — user-user CF is impossible.
- Item–item CF is computed on interaction co-occurrence patterns in `recs_anonymized_interactions`.
- Cold-start is handled gracefully: items with < `MIN_CF_INTERACTIONS` (default: 10)
  interactions fall back to the rule-based Jaccard relevance score.

**Database schema additions** (`migration 25`):

```sql
-- Precomputed item-item similarity pairs.
-- Populated by AggregateItemSimilarityTask (weekly, Saturday 2 AM UTC).
CREATE TABLE recs_item_similarity (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    listing_id_a        UUID NOT NULL REFERENCES mkt_listings(id) ON DELETE CASCADE,
    listing_id_b        UUID NOT NULL REFERENCES mkt_listings(id) ON DELETE CASCADE,
    similarity_score    FLOAT4 NOT NULL CHECK (similarity_score BETWEEN 0 AND 1),
    methodology_slug    TEXT NOT NULL,
    interaction_count   INT NOT NULL,  -- min interactions used in computation
    computed_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT recs_item_similarity_pair_unique
        UNIQUE (listing_id_a, listing_id_b, methodology_slug)
);
CREATE INDEX idx_recs_item_similarity_a ON recs_item_similarity (listing_id_a, methodology_slug);
CREATE INDEX idx_recs_item_similarity_b ON recs_item_similarity (listing_id_b, methodology_slug);
```

**Algorithm** (`AggregateItemSimilarityTask`):

1. For each `methodology_slug` in `recs_anonymized_interactions`:
   - Build a co-occurrence matrix: `cooccur[item_a][item_b]` = # of `anonymous_id`s that
     interacted with both items (within the same age band + methodology bucket).
   - Compute cosine similarity: `sim(a,b) = cooccur(a,b) / sqrt(count(a) * count(b))`
   - Only persist pairs where `similarity_score >= 0.1` and `interaction_count >= MIN_CF_INTERACTIONS`.

2. Truncate and repopulate `recs_item_similarity` weekly (full recompute is safe at current
   data volumes; switch to incremental when interaction rows exceed 1M).

**Score blending**: CF score is injected into `ScoringFactors.Relevance` when a
family's purchase history contains `listing_id_a` and we have similarity pairs for it.
CF score replaces the Jaccard relevance when CF data is available; otherwise Jaccard is used.

**New task constant**: `recs:aggregate_item_similarity` — weekly, Saturday 2:00 AM UTC.

---

### Phase 3.2: A/B Testing Framework for Recommendation Quality

**Purpose**: Allow operators to test algorithm variants (e.g., CF-enabled vs. rule-only)
without code deploys and to measure engagement lift.

**Database schema additions** (`migration 25`, continued):

```sql
-- Experiment definitions. Created/activated by admins via admin API.
CREATE TABLE recs_experiments (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            TEXT NOT NULL UNIQUE,
    description     TEXT,
    status          TEXT NOT NULL DEFAULT 'draft'
                        CHECK (status IN ('draft', 'active', 'paused', 'completed')),
    variant_names   TEXT[] NOT NULL DEFAULT ARRAY['control', 'treatment'],
    traffic_pct     FLOAT4 NOT NULL DEFAULT 0.5
                        CHECK (traffic_pct BETWEEN 0.0 AND 1.0),
    start_at        TIMESTAMPTZ,
    end_at          TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Family-level assignment. Deterministic: hash(family_id + experiment_id) % 100.
-- family_id is present here because this table governs which algorithm a family sees,
-- not what they did. RLS ensures a family can only read their own row.
CREATE TABLE recs_experiment_assignments (
    experiment_id   UUID NOT NULL REFERENCES recs_experiments(id) ON DELETE CASCADE,
    family_id       UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    variant         TEXT NOT NULL,
    assigned_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (experiment_id, family_id)
);

-- Engagement events for metric collection.
-- No student_id, no PII. family_id is present for aggregation only.
CREATE TABLE recs_experiment_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    experiment_id   UUID NOT NULL REFERENCES recs_experiments(id) ON DELETE CASCADE,
    family_id       UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    variant         TEXT NOT NULL,
    event_type      TEXT NOT NULL  -- 'impression', 'click', 'purchase', 'dismiss'
                        CHECK (event_type IN ('impression', 'click', 'purchase', 'dismiss')),
    recommendation_id UUID REFERENCES recs_recommendations(id) ON DELETE SET NULL,
    occurred_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_recs_exp_events_experiment ON recs_experiment_events (experiment_id, variant, event_type);
```

**Assignment rule**: When generating recommendations for a family, check active experiments.
Assign variant deterministically: `fnv32a(family_id_bytes XOR experiment_id_bytes) % 100 < traffic_pct * 100 → treatment; else → control`.

**Metric**: Primary metric is **click-through rate** (CTR) = `click` events / `impression`
events per variant, measured over 14-day windows. Secondary: dismiss rate, purchase rate.

**Admin API** (Phase 3.2): `GET/POST /v1/admin/recs/experiments`, `PATCH /v1/admin/recs/experiments/:id`
(managed by `16-admin` domain, with cross-domain adapter to `recs::`).

---

### Phase 3.3: TF-IDF Semantic Content Search

**Approach**: PostgreSQL native full-text search (`tsvector`/`tsquery`) with `GiST` index
on listing titles and descriptions. No external API. No pgvector required.

**Purpose**: Enable semantic content search for the recommendations engine — replace the
current `subject_tags` Jaccard similarity for listing relevance with richer text-based
similarity.

**Database schema additions** (`migration 26`):

```sql
-- Precomputed tsvector for listings. Populated by trigger on INSERT/UPDATE of mkt_listings.
ALTER TABLE mkt_listings
    ADD COLUMN IF NOT EXISTS search_vector TSVECTOR
        GENERATED ALWAYS AS (
            setweight(to_tsvector('english', coalesce(title, '')), 'A') ||
            setweight(to_tsvector('english', coalesce(description, '')), 'B') ||
            setweight(to_tsvector('english', array_to_string(coalesce(subject_tags, '{}'), ' ')), 'C')
        ) STORED;

CREATE INDEX IF NOT EXISTS idx_mkt_listings_search_vector
    ON mkt_listings USING GIN (search_vector);
```

**Usage in recommendation engine**: In `generateMarketplaceCandidates`, when a family has
recent subject-tag signals, build a `tsquery` from those tags and use `ts_rank` as the
`ScoringFactors.Relevance` score. This replaces Jaccard for families with sufficient signal.

**Cold-start fallback**: When `ts_rank` returns 0 (no query terms match), fall back to
Jaccard similarity as before.

---

### Phase 3.4: Enhanced Anonymization with Differential Privacy

**Purpose**: Add Laplace noise injection to aggregate counts before they leave the
`recs_anonymized_interactions` table via the item-similarity computation. This provides
ε-differential privacy protection against inference attacks.

**Laplace mechanism**: For each co-occurrence count `c`, add noise sampled from `Laplace(0, 1/ε)`
where `ε = 1.0` (moderate privacy). Counts are rounded to integers after noise injection.
Pairs where the noisy count falls below `MIN_CF_INTERACTIONS` are suppressed.

**Implementation**: Noise injection happens inside `AggregateItemSimilarityTask` before
writing to `recs_item_similarity`. No DB schema changes required.

**New config constant**: `RECS_DP_EPSILON` (float, default `1.0`). Setting to `0.0`
disables differential privacy (not recommended for production).

---

### Phase 3.5: pgvector + Embedding-Based Semantic Similarity (Opt-In)

**Status**: Optional enhancement. Requires `RECS_EMBEDDING_API_KEY` env var and explicit
opt-in via admin config. Disabled by default.

**Approach**: Use the Anthropic Claude API (`claude-3-haiku-20240307`) to generate
512-dimensional text embeddings for listing titles + descriptions. Store in PostgreSQL via
the `pgvector` extension. Use cosine similarity for candidate ranking.

**Privacy invariant**: Only listing metadata (title, description, subject tags) is sent to
the Anthropic API — never any family or student data.

**Database schema additions** (when enabled, `migration 27`):

```sql
CREATE EXTENSION IF NOT EXISTS vector;

ALTER TABLE mkt_listings
    ADD COLUMN IF NOT EXISTS embedding VECTOR(512);

CREATE INDEX IF NOT EXISTS idx_mkt_listings_embedding
    ON mkt_listings USING ivfflat (embedding vector_cosine_ops)
    WITH (lists = 100);

-- Tracks embedding generation jobs to avoid redundant API calls.
CREATE TABLE recs_embedding_jobs (
    listing_id      UUID PRIMARY KEY REFERENCES mkt_listings(id) ON DELETE CASCADE,
    model_version   TEXT NOT NULL,
    generated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    status          TEXT NOT NULL DEFAULT 'pending'
                        CHECK (status IN ('pending', 'completed', 'failed'))
);
```

**New task**: `recs:generate_embeddings` — triggered on listing publish (event handler on
`mkt::ListingPublished`) and on operator demand. Batches up to 20 listings per API call.

**Score blending**: When `embedding` is non-null, cosine similarity replaces `ts_rank` in
`ScoringFactors.Relevance`. Graceful degradation: falls back to TF-IDF → Jaccard in order.

**Board approval required**: Adding Anthropic API costs requires board approval before
this phase is activated. Estimated cost: ~$0.002 / 1,000 listings (haiku tier).

---

## Phase 4 — AI Tutoring & Adaptive Learning

> Phase 4 capabilities are further in the future and depend on Phase 3 data maturity.
> This section captures intent; detailed specs will be written as Phase 3 matures.

| Enhancement | Dependency | Notes |
|-------------|-----------|-------|
| Natural language curriculum search | Phase 3.5 embeddings | Families describe what they want in plain language |
| LLM-powered methodology-aware tutoring | External Claude API | Scoped to `method::` config; no student PII |
| Sequence model for optimal next activity | 6+ months of interaction data | Requires training data maturity |
| Curriculum gap analysis + compliance integration | Phase 3.1 CF + `17-plan` domain | Maps CF gaps to state-specific requirements |

---

## Migration Sequence

| Migration | Contents |
|-----------|---------|
| 25 | `recs_item_similarity`, `recs_experiments`, `recs_experiment_assignments`, `recs_experiment_events` |
| 26 | `mkt_listings.search_vector` (generated column + GIN index) |
| 27 | `vector` extension, `mkt_listings.embedding`, `recs_embedding_jobs` (opt-in only) |

---

## New Task Constants

| Constant | Schedule | Phase |
|----------|---------|-------|
| `recs:aggregate_item_similarity` | Weekly, Saturday 2:00 AM UTC | 3.1 |
| `recs:generate_embeddings` | Event-triggered + on-demand | 3.5 |

---

## Verification Checklist (Phase 3)

### Collaborative Filtering (Phase 3.1)
- [ ] `recs_item_similarity` has no `family_id`, `student_id` columns
- [ ] Pairs with `interaction_count < MIN_CF_INTERACTIONS` are not persisted
- [ ] CF score is blended via `ScoringFactors.Relevance`, not applied directly
- [ ] Cold-start families (< 3 purchases) use Jaccard fallback
- [ ] Differential privacy noise applied before writing pairs (Phase 3.4)

### A/B Testing (Phase 3.2)
- [ ] Assignment is deterministic for same (family, experiment) pair
- [ ] Assignment hash uses no PII beyond family UUID
- [ ] `recs_experiment_events` has no `student_id` column
- [ ] Only active experiments are evaluated at recommendation compute time
- [ ] Admin can pause experiments without code deploy

### Semantic Search (Phase 3.3)
- [ ] `search_vector` column is `GENERATED ALWAYS AS ... STORED`
- [ ] `ts_rank` used as relevance score when signal tags match
- [ ] Falls back to Jaccard when ts_rank returns 0

### Embeddings (Phase 3.5 — opt-in)
- [ ] `RECS_EMBEDDING_API_KEY` absent → embedding features disabled entirely
- [ ] Only listing metadata (title, description, tags) sent to external API
- [ ] `recs_embedding_jobs` tracks all API calls for cost auditing
- [ ] Board approval documented before activation in production
