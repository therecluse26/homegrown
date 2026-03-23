-- +goose Up
-- Migration: Create onboarding domain tables
-- Domain: onboard:: [04-onboard.md §3]

-- ─── Enums ───────────────────────────────────────────────────────────────────

CREATE TYPE onb_wizard_step_enum AS ENUM (
    'family_profile',      -- step 1: family name, state, children (required)
    'children',            -- step 2: add student profiles (optional, can skip)
    'methodology',         -- step 3: select methodology (required)
    'roadmap_review'       -- step 4: review roadmap + recommendations (optional)
);

CREATE TYPE onb_wizard_status_enum AS ENUM (
    'in_progress',         -- wizard is active
    'completed',           -- wizard finished (all required steps done)
    'skipped'              -- user explicitly skipped remaining steps
);

CREATE TYPE onb_roadmap_item_type_enum AS ENUM (
    'task',                -- actionable to-do
    'resource',            -- link to educational content
    'feature_tour'         -- guided tour of a platform feature
);

-- ─── Tables ──────────────────────────────────────────────────────────────────

-- Wizard progress tracking [S§6]
-- One row per family. Created when FamilyCreated event is received.
CREATE TABLE onb_wizard_progress (
    id                    UUID PRIMARY KEY DEFAULT uuidv7(),
    family_id             UUID NOT NULL UNIQUE REFERENCES iam_families(id) ON DELETE CASCADE,
    status                onb_wizard_status_enum NOT NULL DEFAULT 'in_progress',
    current_step          onb_wizard_step_enum NOT NULL DEFAULT 'family_profile',
    completed_steps       onb_wizard_step_enum[] NOT NULL DEFAULT '{}',
    quiz_share_id         TEXT,
    methodology_path      TEXT CHECK (methodology_path IN (
                            'quiz_informed', 'exploration', 'skip'
                          )),
    completed_at          TIMESTAMPTZ,
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
    methodology_slug      TEXT NOT NULL,
    item_type             onb_roadmap_item_type_enum NOT NULL,
    title                 TEXT NOT NULL,
    description           TEXT,
    link_url              TEXT,
    age_group             TEXT,
    sort_order            SMALLINT NOT NULL DEFAULT 0,
    is_completed          BOOLEAN NOT NULL DEFAULT false,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_onb_roadmap_family ON onb_roadmap_items(family_id);
CREATE INDEX idx_onb_roadmap_wizard ON onb_roadmap_items(wizard_id);

-- Starter curriculum recommendations [S§6.5]
CREATE TABLE onb_starter_recommendations (
    id                    UUID PRIMARY KEY DEFAULT uuidv7(),
    family_id             UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    wizard_id             UUID NOT NULL REFERENCES onb_wizard_progress(id) ON DELETE CASCADE,
    methodology_slug      TEXT NOT NULL,
    title                 TEXT NOT NULL,
    description           TEXT,
    link_url              TEXT,
    price_type            TEXT NOT NULL CHECK (price_type IN ('free', 'paid')),
    age_group             TEXT,
    sort_order            SMALLINT NOT NULL DEFAULT 0,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_onb_recs_family ON onb_starter_recommendations(family_id);
CREATE INDEX idx_onb_recs_wizard ON onb_starter_recommendations(wizard_id);

-- Community connection suggestions [S§6.6]
CREATE TABLE onb_community_suggestions (
    id                    UUID PRIMARY KEY DEFAULT uuidv7(),
    family_id             UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    wizard_id             UUID NOT NULL REFERENCES onb_wizard_progress(id) ON DELETE CASCADE,
    methodology_slug      TEXT NOT NULL,
    suggestion_type       TEXT NOT NULL CHECK (suggestion_type IN (
                            'methodology_group',
                            'local_group',
                            'mentor'
                          )),
    title                 TEXT NOT NULL,
    description           TEXT,
    reference_id          UUID,
    sort_order            SMALLINT NOT NULL DEFAULT 0,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_onb_community_family ON onb_community_suggestions(family_id);
CREATE INDEX idx_onb_community_wizard ON onb_community_suggestions(wizard_id);

-- ─── RLS Policies ────────────────────────────────────────────────────────────
-- All onb_ tables contain user data and MUST have RLS. [CODING §3.2, 04-onboard §3.3]

ALTER TABLE onb_wizard_progress ENABLE ROW LEVEL SECURITY;
ALTER TABLE onb_roadmap_items ENABLE ROW LEVEL SECURITY;
ALTER TABLE onb_starter_recommendations ENABLE ROW LEVEL SECURITY;
ALTER TABLE onb_community_suggestions ENABLE ROW LEVEL SECURITY;

CREATE POLICY onb_wizard_progress_family ON onb_wizard_progress
    USING (family_id = current_setting('app.current_family_id')::uuid);
CREATE POLICY onb_roadmap_items_family ON onb_roadmap_items
    USING (family_id = current_setting('app.current_family_id')::uuid);
CREATE POLICY onb_starter_recommendations_family ON onb_starter_recommendations
    USING (family_id = current_setting('app.current_family_id')::uuid);
CREATE POLICY onb_community_suggestions_family ON onb_community_suggestions
    USING (family_id = current_setting('app.current_family_id')::uuid);

-- ─── Seed Methodology Onboarding & Community Config ──────────────────────────
-- Populate onboarding_config and community_config JSONB on method_definitions
-- for all 6 methodologies. These are read by onboard:: materialization. [04-onboard §10]

UPDATE method_definitions SET onboarding_config = '{
  "roadmap_steps": [
    {"type": "task", "title": "Create your first Book of Centuries", "description": "A timeline notebook that grows with your child — the heart of a Charlotte Mason education.", "link_url": "/app/tools/book-of-centuries", "age_group": "9-12", "sort_order": 1},
    {"type": "resource", "title": "Read: Charlotte Mason Philosophy Introduction", "description": "Understand the principles of living books, narration, and nature study.", "link_url": "/discover/content/cm-philosophy", "sort_order": 2},
    {"type": "task", "title": "Set up your first nature journal", "description": "Outdoor observation is central to CM — start with 15 minutes of nature study.", "link_url": "/app/tools/nature-journal", "sort_order": 3},
    {"type": "task", "title": "Choose your first living book", "description": "Select an engaging, literary-quality book for your child'\''s reading level.", "link_url": "/app/marketplace?method=charlotte-mason&type=books", "sort_order": 4},
    {"type": "feature_tour", "title": "Explore narration tools", "description": "Learn how to use the platform'\''s narration recording and tracking features.", "link_url": "/app/tour/narration", "sort_order": 5}
  ],
  "starter_recs": [
    {"title": "Ambleside Online Year 1", "description": "Free Charlotte Mason curriculum with carefully chosen living books.", "link_url": "/marketplace/ambleside-y1", "price_type": "free", "age_group": "6-8", "sort_order": 1},
    {"title": "Simply Charlotte Mason Enrichment Studies", "description": "Nature study, picture study, and composer study guides.", "link_url": "/marketplace/scm-enrichment", "price_type": "paid", "age_group": "6-8", "sort_order": 2},
    {"title": "A Gentle Feast Year 2", "description": "Literature-rich curriculum following CM principles for older students.", "link_url": "/marketplace/gentle-feast-y2", "price_type": "paid", "age_group": "9-12", "sort_order": 3},
    {"title": "Handbook of Nature Study", "description": "The classic nature study guide for all ages.", "link_url": "/marketplace/nature-study-handbook", "price_type": "paid", "sort_order": 4}
  ]
}'::jsonb, community_config = '{
  "default_groups": [
    {"type": "methodology_group", "title": "Charlotte Mason Community", "description": "Connect with families using the Charlotte Mason method. Share living book finds, nature study ideas, and narration tips."}
  ]
}'::jsonb WHERE slug = 'charlotte-mason';

UPDATE method_definitions SET onboarding_config = '{
  "roadmap_steps": [
    {"type": "task", "title": "Set up your daily schedule", "description": "Traditional homeschooling thrives on consistent routines — map out your school day.", "link_url": "/app/tools/scheduler", "sort_order": 1},
    {"type": "resource", "title": "Read: Getting Started with Traditional Homeschooling", "description": "An overview of textbook-based learning at home.", "link_url": "/discover/content/traditional-intro", "sort_order": 2},
    {"type": "task", "title": "Choose your core curriculum", "description": "Select textbooks and workbooks for your main subjects.", "link_url": "/app/marketplace?method=traditional&type=curriculum", "sort_order": 3},
    {"type": "feature_tour", "title": "Explore grade tracking", "description": "Learn to use the platform'\''s assignment and grade tracking tools.", "link_url": "/app/tour/grades", "sort_order": 4}
  ],
  "starter_recs": [
    {"title": "Saxon Math", "description": "Structured, incremental math curriculum used in many traditional programs.", "link_url": "/marketplace/saxon-math", "price_type": "paid", "age_group": "6-8", "sort_order": 1},
    {"title": "Abeka Language Arts", "description": "Comprehensive phonics and grammar program.", "link_url": "/marketplace/abeka-la", "price_type": "paid", "age_group": "6-8", "sort_order": 2},
    {"title": "Teaching Textbooks Algebra", "description": "Self-paced math with video lessons for older students.", "link_url": "/marketplace/tt-algebra", "price_type": "paid", "age_group": "13-15", "sort_order": 3}
  ]
}'::jsonb, community_config = '{
  "default_groups": [
    {"type": "methodology_group", "title": "Traditional Homeschool Community", "description": "Connect with families using traditional textbook-based approaches. Share curriculum reviews and scheduling tips."}
  ]
}'::jsonb WHERE slug = 'traditional';

UPDATE method_definitions SET onboarding_config = '{
  "roadmap_steps": [
    {"type": "resource", "title": "Read: The Trivium Explained", "description": "Understand the Grammar, Logic, and Rhetoric stages of classical education.", "link_url": "/discover/content/trivium-explained", "sort_order": 1},
    {"type": "task", "title": "Identify your child'\''s Trivium stage", "description": "Determine whether your child is in the Grammar, Logic, or Rhetoric stage based on age and development.", "link_url": "/app/tools/trivium-assessment", "sort_order": 2},
    {"type": "task", "title": "Start memory work", "description": "Classical education emphasizes memorization — begin with timeline facts or Latin vocabulary.", "link_url": "/app/tools/memory-work", "age_group": "6-8", "sort_order": 3},
    {"type": "feature_tour", "title": "Explore Socratic discussion tools", "description": "Learn how to use guided discussion prompts for Logic-stage students.", "link_url": "/app/tour/socratic", "age_group": "13-15", "sort_order": 4}
  ],
  "starter_recs": [
    {"title": "Classical Conversations Foundations", "description": "Memory work program covering history, science, math, and Latin.", "link_url": "/marketplace/cc-foundations", "price_type": "paid", "age_group": "6-8", "sort_order": 1},
    {"title": "Well-Trained Mind Guide", "description": "Free planning guide for classical homeschooling by Susan Wise Bauer.", "link_url": "/marketplace/wtm-guide", "price_type": "free", "sort_order": 2},
    {"title": "Institute for Excellence in Writing", "description": "Structured writing program aligned with classical methods.", "link_url": "/marketplace/iew", "price_type": "paid", "age_group": "9-12", "sort_order": 3}
  ]
}'::jsonb, community_config = '{
  "default_groups": [
    {"type": "methodology_group", "title": "Classical Education Community", "description": "Connect with families following classical education. Discuss Great Books, Latin study, and Trivium stages."}
  ]
}'::jsonb WHERE slug = 'classical';

UPDATE method_definitions SET onboarding_config = '{
  "roadmap_steps": [
    {"type": "resource", "title": "Read: Waldorf Education at Home", "description": "Understand rhythm, imagination, and handwork in Waldorf-inspired homeschooling.", "link_url": "/discover/content/waldorf-home", "sort_order": 1},
    {"type": "task", "title": "Create a daily rhythm chart", "description": "Waldorf emphasizes rhythm over rigid schedules — map your family'\''s natural flow.", "link_url": "/app/tools/rhythm-chart", "sort_order": 2},
    {"type": "task", "title": "Set up a handwork station", "description": "Knitting, painting, and crafts are integral to Waldorf — prepare a creative space.", "link_url": "/app/tools/handwork", "sort_order": 3},
    {"type": "task", "title": "Plan your first main lesson block", "description": "Waldorf uses 3-4 week focused blocks on one subject.", "link_url": "/app/tools/main-lesson", "age_group": "6-8", "sort_order": 4}
  ],
  "starter_recs": [
    {"title": "Live Education Waldorf Curriculum", "description": "Comprehensive Waldorf curriculum for grades 1-8.", "link_url": "/marketplace/live-education", "price_type": "paid", "age_group": "6-8", "sort_order": 1},
    {"title": "Painting and Drawing in Waldorf Schools", "description": "Free guide to wet-on-wet watercolor and form drawing.", "link_url": "/marketplace/waldorf-art-guide", "price_type": "free", "sort_order": 2},
    {"title": "Christopherus Homeschool Resources", "description": "Waldorf-inspired lesson plans and handwork projects.", "link_url": "/marketplace/christopherus", "price_type": "paid", "age_group": "9-12", "sort_order": 3}
  ]
}'::jsonb, community_config = '{
  "default_groups": [
    {"type": "methodology_group", "title": "Waldorf Homeschool Community", "description": "Connect with Waldorf-inspired families. Share main lesson ideas, handwork projects, and rhythm tips."}
  ]
}'::jsonb WHERE slug = 'waldorf';

UPDATE method_definitions SET onboarding_config = '{
  "roadmap_steps": [
    {"type": "resource", "title": "Read: Montessori at Home Guide", "description": "Understand the prepared environment, sensitive periods, and self-directed learning.", "link_url": "/discover/content/montessori-home", "sort_order": 1},
    {"type": "task", "title": "Prepare your learning environment", "description": "Montessori requires accessible, organized materials at child height.", "link_url": "/app/tools/environment-setup", "sort_order": 2},
    {"type": "task", "title": "Observe your child'\''s interests", "description": "Spend a week observing what draws your child'\''s attention — this guides material selection.", "link_url": "/app/tools/observation-log", "sort_order": 3},
    {"type": "task", "title": "Choose your first practical life activities", "description": "Start with pouring, sorting, or food preparation appropriate to your child'\''s age.", "link_url": "/app/marketplace?method=montessori&type=practical-life", "age_group": "3-5", "sort_order": 4}
  ],
  "starter_recs": [
    {"title": "Montessori Primary Guide", "description": "Free album pages covering practical life, sensorial, and math.", "link_url": "/marketplace/montessori-primary", "price_type": "free", "age_group": "3-5", "sort_order": 1},
    {"title": "Keys of the Universe", "description": "Upper elementary cosmic education curriculum.", "link_url": "/marketplace/keys-universe", "price_type": "paid", "age_group": "9-12", "sort_order": 2},
    {"title": "Montessori R&D Geometry Set", "description": "Hands-on geometry materials for elementary.", "link_url": "/marketplace/montessori-geometry", "price_type": "paid", "age_group": "6-8", "sort_order": 3}
  ]
}'::jsonb, community_config = '{
  "default_groups": [
    {"type": "methodology_group", "title": "Montessori Homeschool Community", "description": "Connect with Montessori homeschooling families. Share environment setups, material recommendations, and observation tips."}
  ]
}'::jsonb WHERE slug = 'montessori';

UPDATE method_definitions SET onboarding_config = '{
  "roadmap_steps": [
    {"type": "resource", "title": "Read: What is Unschooling?", "description": "Understand child-led learning, deschooling, and trusting the process.", "link_url": "/discover/content/unschooling-intro", "sort_order": 1},
    {"type": "task", "title": "Start a deschooling journal", "description": "If transitioning from school, allow 1 month per year of schooling to decompress.", "link_url": "/app/tools/journal", "sort_order": 2},
    {"type": "task", "title": "Create a resource-rich environment", "description": "Fill your home with books, art supplies, games, and tools — then step back.", "link_url": "/app/tools/strewing", "sort_order": 3},
    {"type": "task", "title": "Document learning for compliance", "description": "Unschooling still requires record-keeping in most states — learn to document naturally.", "link_url": "/app/tools/documentation", "sort_order": 4}
  ],
  "starter_recs": [
    {"title": "Free to Learn by Peter Gray", "description": "Essential reading on self-directed education and play.", "link_url": "/marketplace/free-to-learn", "price_type": "paid", "sort_order": 1},
    {"title": "Khan Academy", "description": "Free, self-paced learning for when kids choose to explore academics.", "link_url": "/marketplace/khan-academy", "price_type": "free", "sort_order": 2},
    {"title": "Project-Based Homeschooling Guide", "description": "Free guide to supporting child-led project work.", "link_url": "/marketplace/project-based", "price_type": "free", "sort_order": 3}
  ]
}'::jsonb, community_config = '{
  "default_groups": [
    {"type": "methodology_group", "title": "Unschooling Community", "description": "Connect with unschooling families. Share project ideas, strewing inspiration, and documentation strategies."}
  ]
}'::jsonb WHERE slug = 'unschooling';

-- +goose Down

DROP POLICY IF EXISTS onb_community_suggestions_family ON onb_community_suggestions;
DROP POLICY IF EXISTS onb_starter_recommendations_family ON onb_starter_recommendations;
DROP POLICY IF EXISTS onb_roadmap_items_family ON onb_roadmap_items;
DROP POLICY IF EXISTS onb_wizard_progress_family ON onb_wizard_progress;

DROP TABLE IF EXISTS onb_community_suggestions;
DROP TABLE IF EXISTS onb_starter_recommendations;
DROP TABLE IF EXISTS onb_roadmap_items;
DROP TABLE IF EXISTS onb_wizard_progress;

-- Revert methodology configs to empty JSONB
UPDATE method_definitions SET onboarding_config = '{}'::jsonb, community_config = '{}'::jsonb;

DROP TYPE IF EXISTS onb_roadmap_item_type_enum;
DROP TYPE IF EXISTS onb_wizard_status_enum;
DROP TYPE IF EXISTS onb_wizard_step_enum;
