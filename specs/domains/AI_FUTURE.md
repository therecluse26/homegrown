# AI & Machine Learning Roadmap

> **Status**: Directional roadmap — not an implementable domain spec.
> **Prerequisite**: `13-recs.md` (Recommendations & Signals domain) must be implemented
> through Phase 2 before any capability in this document is actionable.

---

## §1 Purpose & Scope

This document covers capabilities that require **actual machine learning or AI models** —
collaborative filtering, semantic search, learning-to-rank, and LLM-powered tutoring. These
are planned for Phase 3-4 of the platform roadmap. `[S§19]`

**What this document is NOT**:
- Not a numbered domain spec (no database schemas, no Go interfaces, no API contracts)
- Not implementable today — each capability has explicit data prerequisites and decision
  triggers (§6)
- Not a commitment — these are directional capabilities, subject to product validation

**What this document IS**:
- A roadmap for when the `recs::` module evolves beyond rule-based heuristics
- A record of architectural decisions made now to support future ML (e.g., anonymized data
  collection in `recs_anonymized_interactions`)
- A reference for infrastructure planning

**Relationship to `13-recs.md`**: The `recs::` domain (Phase 1-2) builds the signal pipeline,
recommendation infrastructure, and anonymized data collection that all Phase 3-4 ML
capabilities depend on. The `recs_` database prefix, `internal/recs/` package path, and
`RecsService` interface will be extended (not replaced) when ML capabilities are added.

---

## §2 Phase 3 Capabilities

| Capability | Model Type | What It Does | Data Source | Prerequisites |
|-----------|-----------|-------------|------------|---------------|
| **Collaborative filtering** | Matrix factorization (ALS) or item-based k-NN | "Families like yours liked X" — finds similarity patterns across families with same methodology, similar student ages, overlapping purchase history | `recs_anonymized_interactions`, `recs_popularity_scores` | ~1,000+ active premium families with purchase history from Phase 2 |
| **Learning-to-rank** | Gradient boosted trees (LightGBM / XGBoost) | Replaces hand-tuned scoring weights (0.35 / 0.25 / 0.25 / 0.10 / 0.05) in `internal/recs/algorithm.go` with weights learned from dismiss/block/click feedback data | `recs_recommendation_feedback`, recommendation view/click tracking (Phase 3 addition) | 10,000+ feedback records in `recs_recommendation_feedback` |
| **Content-based filtering** | TF-IDF or sentence embeddings (e.g., all-MiniLM-L6-v2) on listing descriptions | "This listing is semantically similar to ones you've engaged with" — finds content similarity beyond keyword/tag matching | `mkt_listings.description`, `recs_signals` (purchase/activity signals) | Marketplace listings with descriptions; can bootstrap from Phase 1 data |
| **Semantic search** | Embedding model + pgvector | Natural language curriculum discovery: "find me gentle math for a wiggly 7-year-old" — vector similarity search on listing embeddings | `mkt_listings` content → embedding vectors | pgvector PostgreSQL extension, embedding generation pipeline, 10,000+ marketplace listings |

### Collaborative Filtering — Detail

The Phase 2 popularity signal (`recs_popularity_scores`) uses raw purchase counts. Collaborative
filtering replaces this with latent factor models that capture *why* families with similar
profiles make similar purchases.

**Approach**: Implicit feedback matrix factorization (Alternating Least Squares). The
interaction matrix is `(anonymous_family × listing)` with implicit signals (purchase = 1.0,
activity = 0.5, view = 0.1). Factorization produces family and listing embedding vectors;
recommendations are the top-k dot products.

**Privacy**: Training uses `recs_anonymized_interactions` (no PII). The `anonymous_id` groups
interactions by family without revealing identity. Model outputs (listing embeddings) are
public-safe.

### Learning-to-Rank — Detail

The Phase 2 scoring formula uses hand-tuned weights. Learning-to-rank trains a model to
predict which recommendations a family will engage with (click, purchase) vs. dismiss/block.

**Approach**: Gradient boosted decision trees (LightGBM) with features:
- Methodology match score (primary vs. secondary)
- Popularity percentile (per-methodology)
- Subject overlap (Jaccard similarity)
- Listing freshness (days since publication)
- Family engagement history (signals per week)
- Student age proximity to content target age

**Training data**: Pairs of (recommendation, outcome) where outcome ∈ {clicked, purchased,
dismissed, blocked, expired_unseen}. Requires Phase 2 feedback data + Phase 3 view/click
tracking.

### Semantic Search — Detail

Phase 2 search (`search::`) uses PostgreSQL full-text search (keyword matching). Semantic
search adds vector similarity for natural language queries that don't match exact keywords.

**Approach**: Generate embeddings for all marketplace listing descriptions using a sentence
transformer model (e.g., all-MiniLM-L6-v2, 384-dimensional vectors). Store vectors in
pgvector columns. At query time, embed the user's query and find nearest neighbors via
cosine similarity.

**Hybrid**: Combine keyword FTS score with vector similarity score (reciprocal rank fusion)
for best results.

---

## §3 Phase 4 Capabilities

| Capability | Model Type | What It Does | Data Source | Prerequisites |
|-----------|-----------|-------------|------------|---------------|
| **AI tutoring** | LLM (Claude API or similar) with RAG over curriculum content | Interactive, methodology-aware tutoring for students — answers questions, explains concepts, suggests activities, all constrained by the family's methodology | Curriculum content embeddings, `method_definitions` (methodology philosophy), `recs_anonymized_interactions` | Curriculum content embeddings, significant COPPA/safety review, separate product decision |
| **Adaptive learning paths** | Sequence models (RNN or transformer on activity sequences) | Predicts optimal next learning activity based on student's history and methodology progression | Per-student activity sequences from `recs_signals` (1+ year of data) | Substantial per-student activity data (1+ year), methodology progression models |
| **Curriculum gap analysis** | Classification model on subject coverage vs. state requirements | "Your student is missing 20 hours of science for [State] compliance" — automated gap detection | `learn_activities`, `comply_state_configs`, `recs_signals` | `comply::` domain implemented, learning signals covering 1+ academic year |

### AI Tutoring — Detail

AI tutoring is the highest-impact and highest-risk capability. It requires:

1. **Methodology awareness**: The tutoring model must understand and respect each methodology's
   philosophy. A Charlotte Mason tutor should use narration-based techniques; a Classical tutor
   should use Socratic questioning. This is achieved via methodology-specific system prompts
   derived from `method_definitions.philosophy_modules`.

2. **Curriculum RAG**: Retrieval-Augmented Generation over the family's purchased curriculum
   content, ensuring tutoring is grounded in material the family owns.

3. **COPPA compliance**: AI tutoring involves an AI system interacting with children under 13.
   This requires a fresh COPPA compliance review, likely with legal counsel, before
   implementation. The anonymized data collection in Phase 2 (`recs_anonymized_interactions`)
   is designed to support training without PII, but the tutoring interaction itself generates
   new PII (student questions/responses) that must be handled carefully.

4. **Safety guardrails**: Content filtering, topic boundaries, mandatory parent visibility
   into tutoring sessions, session logging, and the ability for parents to disable tutoring
   per-student.

> **This is a separate product decision** — not an automatic Phase 4 trigger. The team must
> decide whether to build AI tutoring based on market demand, competitive landscape, safety
> review outcomes, and available resources.

---

## §4 Infrastructure Requirements

### pgvector (Phase 3)

- **What**: PostgreSQL extension for vector similarity search
- **Why**: Stores and queries embedding vectors for semantic search and content-based filtering
- **Migration**: `CREATE EXTENSION IF NOT EXISTS vector;` + add `embedding vector(384)` columns
  to relevant tables
- **Operational impact**: Increases PostgreSQL memory usage; vector indexes (IVFFlat or HNSW)
  require tuning

### Embedding Pipeline (Phase 3)

- **What**: Background job to generate embeddings for marketplace listings
- **Model**: all-MiniLM-L6-v2 (384-dim, ~80MB, runs on CPU) or similar
- **Approach**: Batch processing — new/updated listings are embedded by a background job,
  not real-time. Embeddings stored in pgvector columns.
- **Infrastructure**: The model runs as a sidecar service (Python with ONNX Runtime or
  similar) called from Go via gRPC/HTTP. No external API dependency.

### Model Training Pipeline (Phase 3-4)

- **What**: Offline training on anonymized data, model artifact storage, versioned deployment
- **Where**: Training runs offline (not on the production server). Model artifacts (LightGBM
  models, matrix factorization vectors) are stored in S3 and loaded by the application at
  startup.
- **Frequency**: Retrained weekly or monthly depending on data volume
- **Versioning**: Each model artifact is versioned; the application loads a specific version
  via configuration (not automatic deployment)

### Feature Store (Phase 3)

- **What**: Materialized feature vectors per family/student from `recs_signals`
- **Why**: The Phase 2 algorithm queries raw signals on every run. At scale, pre-computed
  feature vectors (e.g., "subject engagement histogram over last 30 days") are more efficient.
- **Implementation**: Materialized views or a dedicated `recs_features` table updated by
  background job

### No External LLM Dependency Until Phase 4

Phases 1-3 run entirely on in-house models and PostgreSQL. There is no external API
dependency (no OpenAI, no Claude API) until Phase 4 AI tutoring. This is intentional:
- Reduces operational complexity and cost
- Avoids vendor lock-in for core recommendation functionality
- Ensures recommendations work offline / during API outages
- Keeps latency predictable (no external network calls in the recommendation path)

---

## §5 Data Requirements & Privacy

### Training Data Sources

| Capability | Training Data | PII? | Source |
|-----------|--------------|------|--------|
| Collaborative filtering | Anonymized interaction matrix (anonymous_family × listing) | No | `recs_anonymized_interactions` |
| Learning-to-rank | (Recommendation, outcome) pairs with feature vectors | No | `recs_recommendation_feedback` + recommendation features |
| Content-based filtering | Listing descriptions + subject tags | No | `mkt_listings` (public content) |
| Semantic search | Listing descriptions → embedding vectors | No | `mkt_listings` (public content) |
| AI tutoring | Methodology philosophy modules + curriculum content | No (philosophy is public) | `method_definitions`, purchased curriculum |
| Adaptive learning paths | Anonymized activity sequences | No | `recs_anonymized_interactions` |

### Privacy Invariants

1. **All model training uses anonymized data** — no `family_id` or `student_id` in training
   datasets. `[S§10.3, S§17.2]`
2. **Collaborative filtering uses aggregated patterns** — the interaction matrix uses
   `anonymous_id` (HMAC of family_id), not actual family identifiers.
3. **Sentence embeddings are computed on public content** — listing descriptions are
   publisher-provided public content, not family data.
4. **Model outputs are not PII** — embedding vectors, feature importance weights, and
   factorization matrices do not contain PII.
5. **Phase 4 AI tutoring requires fresh COPPA review** — tutoring sessions generate new
   student interaction data (questions, responses) that IS PII and requires careful handling
   under COPPA. `[S§17.2]`

### Anonymized Data Design Decisions

The `recs_anonymized_interactions` table (defined in `13-recs.md §3.2`) was designed
specifically to support Phase 3-4 ML:

- **HMAC anonymous_id**: Enables longitudinal analysis (same family = same anonymous_id)
  without re-identification
- **3-year age bands**: Sufficient granularity for age-based recommendations while preventing
  identification of specific children
- **Broad subject categories**: Enough signal for collaborative filtering without revealing
  specific curriculum choices
- **Methodology slug**: Key dimension for per-methodology model training
- **Duration (rounded)**: Engagement signal for learning-to-rank features

---

## §6 Decision Triggers

Each capability has specific, measurable triggers that indicate when investment is warranted.
These are NOT automatic deployment triggers — they are signals that the capability is worth
evaluating.

| Capability | Metric Trigger | Data Trigger | Business Trigger |
|-----------|---------------|-------------|-----------------|
| **Collaborative filtering** | Recommendation dismiss rate > 30% (hand-tuned scoring not improving) | 1,000+ active premium families with 6+ months of purchase history | Product team identifies "families like yours" as a top-requested feature |
| **Learning-to-rank** | Hand-tuned scoring weights plateau (A/B test shows no improvement from weight adjustments) | 10,000+ records in `recs_recommendation_feedback` | Data science capacity available for model training pipeline |
| **Content-based filtering** | Search click-through on exact keyword matches < 15% (users can't find what they want with keywords) | 5,000+ marketplace listings with descriptions | Marketplace team reports content discovery as a growth bottleneck |
| **Semantic search** | Keyword search returns zero results for > 10% of queries | 10,000+ marketplace listings | User research shows "I don't know the right keywords" as a pain point |
| **AI tutoring** | N/A — this is a strategic product decision, not a metric-triggered one | `recs_anonymized_interactions` has 1+ year of data | Board/leadership decision + COPPA legal review completed + safety review completed |
| **Adaptive learning paths** | N/A — depends on AI tutoring infrastructure | Per-student activity sequences spanning 1+ academic year | AI tutoring successfully launched and stable |

### What "Not Yet" Looks Like

If the decision triggers are not met, the platform continues with the Phase 2 rule-based
recommendation engine indefinitely. The rule-based engine is a complete, shippable product —
not a placeholder. ML capabilities enhance it but are not required for the product to deliver
value.
