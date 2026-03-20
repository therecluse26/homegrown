# Domain Spec 02 — Methodology Configuration (method::)

## §1 Overview

The Methodology domain owns **methodology definitions**, the **master tool catalog**, and the
**tool activation mappings** that determine which learning tools are available to each family.
It is the platform's **cross-cutting configuration domain** — nearly every other domain consumes
its output, but it owns no user data. Methodology definitions are platform-managed global
configuration: they are created via database inserts, not user actions. `[S§4.1, ARCH §1.6]`

| Attribute | Value |
|-----------|-------|
| **Module path** | `src/method/` |
| **DB prefix** | `method_` |
| **Complexity class** | Complex (has `domain/` subdirectory) `[ARCH §4.5]` |
| **External adapter** | None |
| **Key constraint** | Methodology-dependent behavior resolved by config lookup, never code branching `[CODING §5.1]` |

**What method:: owns**: Methodology definitions (identity, philosophy, onboarding config,
mastery paths, community config), master tool catalog, per-methodology tool activations,
tool resolution logic (family-level and student-level), methodology validation, and the
`PATCH /v1/families/methodology` endpoint.

**What method:: does NOT own**: Family and student records (owned by `iam::`), learning tool
*implementations* (owned by `learn::`), onboarding wizard flow (owned by `onboard::`),
discovery/quiz content rendering (owned by `discover::`), marketplace content tagging
(owned by `mkt::`).

**What method:: delegates**: Family methodology persistence → `iam::FamilyRepository::set_methodology`.
Student methodology override persistence → `iam::StudentRepository::update`. Email notifications
→ `notify::` (via domain events).

---

## §2 Requirements Traceability

Every requirement from SPEC.md §4 that touches methodology is mapped to the section of this
document that addresses it.

| Requirement | Source | Addressed In |
|-------------|--------|--------------|
| Methodology is a first-class configuration entity | `[S§4.1]` | §3, §8, §9 |
| Philosophy module (history, principles, typical day, pros/cons, resources) | `[S§4.1]` | §3.1 (`philosophy` JSONB column), §8 |
| Tool Registry pattern (master catalog + per-methodology activation) | `[S§4.2]` | §3.1, §10 |
| Family's active tool set = union of selected methodologies' tools | `[S§4.2]` | §10.1 |
| Multi-methodology (eclectic) support — primary + secondary | `[S§4.3]` | §4, §5, §10 |
| No artificial "Eclectic" methodology category | `[S§4.3]` | §9 (not in seed data) |
| Families can change methodology at any time | `[S§4.3]` | §4.3, §5 |
| Changing methodology MUST NOT delete historical data | `[S§4.3]` | §4.3 (note) |
| Per-domain methodology impact | `[S§4.4]` | §11 |
| Methodology definitions are platform-managed, not user-editable | `[S§4.5]` | §3.2 (no RLS), §9 |
| Adding methodology via configuration, not code | `[S§4.5]` | §9.2 |
| Initial set: Charlotte Mason, Traditional, Classical, Waldorf, Montessori, Unschooling | `[S§4.5]` | §3.3 |
| All methodologies receive equal platform investment | `[S§4.5]` | §9 (no default/preferred flag) |
| Per-student methodology overrides | `[S§4.6]` | §4.4, §5, §10.2 |
| Tool catalog (all tools from §8.1) | `[S§8.1]` | §3.3 (seed data) |
| Methodology-specific tools activated per methodology | `[S§8.1.8]` | §3.3 (activation mappings) |
| Per-student tool assignment based on methodology override | `[S§8.2]` | §10.2 |
| No methodology name branching in code | `[CODING §5.1]` | §9, §10 |
| Content tags associated with methodology | `[S§4.1]` | §3.1 (`onboarding_config` JSONB) |
| Mastery paths (beginner/intermediate/advanced) | `[S§4.1]` | §3.1 (`mastery_paths` JSONB) |
| Community config (group IDs, mentor criteria) | `[S§4.1]` | §3.1 (`community_config` JSONB) |

---

## §3 Database Schema

All tables use the `method_` prefix. These tables store **platform-managed global configuration**,
not user data. There are no RLS policies on method_ tables. `[ARCH §5.1, ARCH §5.2]`

### §3.1 Tables

```sql
-- =============================================================================
-- Migration: YYYYMMDD_000001_create_method_tables.rs
-- =============================================================================

-- Platform-defined methodologies [S§4.1, S§4.5]
-- These are global config rows, NOT user data — no RLS needed.
CREATE TABLE method_definitions (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug              TEXT NOT NULL UNIQUE,              -- e.g., 'charlotte-mason'
    display_name      TEXT NOT NULL,                     -- e.g., 'Charlotte Mason'
    short_desc        TEXT NOT NULL,                     -- one-line summary for lists
    icon_url          TEXT,                              -- CDN URL for methodology icon
    philosophy        JSONB NOT NULL DEFAULT '{}',       -- philosophy module [S§4.1]
                      -- { "history": "...", "principles": [...], "typical_day": "...",
                      --   "pros_cons": {...}, "starting_resources": [...] }
    onboarding_config JSONB NOT NULL DEFAULT '{}',       -- roadmaps, starter recs [S§6.4]
                      -- { "roadmap_steps": [...], "starter_recs": [...],
                      --   "content_tags": [...] }
    community_config  JSONB NOT NULL DEFAULT '{}',       -- group IDs, mentor criteria [S§6.6]
                      -- { "default_group_ids": [...], "mentor_criteria": {...} }
    mastery_paths     JSONB NOT NULL DEFAULT '{}',       -- beginner/intermediate/advanced [S§4.1]
                      -- { "levels": [{ "name": "Beginner", "content": [...] }, ...] }
    terminology       JSONB NOT NULL DEFAULT '{}',       -- methodology-specific labels [S§4.4]
                      -- { "activity": "Lesson", "journal": "Narration", ... }
    display_order     SMALLINT NOT NULL DEFAULT 0,       -- controls listing order
    is_active         BOOLEAN NOT NULL DEFAULT true,     -- soft-disable without deletion
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_method_definitions_active ON method_definitions(is_active, display_order);
CREATE INDEX idx_method_definitions_slug ON method_definitions(slug);

-- Master tool catalog [S§4.2, S§8.1]
-- Every learning tool on the platform. Tools exist independently of methodologies.
CREATE TABLE method_tools (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug              TEXT NOT NULL UNIQUE,              -- e.g., 'reading-lists'
    display_name      TEXT NOT NULL,                     -- e.g., 'Reading Lists'
    description       TEXT,                              -- tool description for UI
    config_schema     JSONB NOT NULL DEFAULT '{}',       -- JSON Schema for tool config
    tier              TEXT NOT NULL DEFAULT 'free'
                      CHECK (tier IN ('free', 'premium')),  -- subscription gating [S§15.2]
    is_active         BOOLEAN NOT NULL DEFAULT true,     -- soft-disable tool globally
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_method_tools_active ON method_tools(is_active);
CREATE INDEX idx_method_tools_slug ON method_tools(slug);

-- Per-methodology tool activation [S§4.2]
-- Many-to-many: which tools are active for which methodology, with config overrides.
CREATE TABLE method_tool_activations (
    methodology_id    UUID NOT NULL REFERENCES method_definitions(id) ON DELETE CASCADE,
    tool_id           UUID NOT NULL REFERENCES method_tools(id) ON DELETE CASCADE,
    config_overrides  JSONB NOT NULL DEFAULT '{}',       -- methodology-specific labels, guidance
                      -- { "label": "Nature Journal", "guidance": "...",
                      --   "entry_types": ["observation", "sketch", "specimen"] }
    sort_order        SMALLINT NOT NULL DEFAULT 0,       -- tool display order within methodology
    PRIMARY KEY (methodology_id, tool_id)
);

CREATE INDEX idx_method_activations_tool ON method_tool_activations(tool_id);
```

### §3.2 No Row-Level Security

Unlike IAM tables, method_ tables contain **global platform configuration**, not user data.
Every authenticated user reads the same methodology definitions and tool catalog. There is
no family-scoped data to isolate.

- `method_definitions` — platform-managed, read by all users
- `method_tools` — platform-managed, read by all users
- `method_tool_activations` — platform-managed, read by all users

No RLS policies are created for these tables. The application role has SELECT access;
INSERT/UPDATE/DELETE is restricted to migrations and admin operations (Phase 3+).

### §3.3 Seed Data Migration

```sql
-- =============================================================================
-- Migration: YYYYMMDD_000002_seed_method_data.rs
-- =============================================================================

-- 6 initial methodologies [S§4.5]
INSERT INTO method_definitions (slug, display_name, short_desc, philosophy, terminology, display_order)
VALUES
    ('charlotte-mason', 'Charlotte Mason', 'Living books, nature study, and habit formation',
     '{"history": "Developed by Charlotte Mason (1842-1923), a British educator who believed children are born persons deserving respect and rich ideas.", "principles": ["Living books over textbooks", "Nature study and outdoor time", "Short lessons with full attention", "Habit training as foundation", "Narration as the primary method of retention"], "typical_day": "Short focused lessons in morning (15-45 min depending on age), nature walks, handicrafts, and free reading in afternoon.", "pros_cons": {"pros": ["Gentle, literature-rich approach", "Develops love of learning", "Emphasizes character formation", "Works well across age ranges"], "cons": ["Requires significant read-aloud time", "Fewer structured assessments", "Can be hard to document for compliance"]}, "starting_resources": ["A Philosophy of Education by Charlotte Mason", "Consider This by Karen Glass", "AmblesideOnline.org curriculum guide"]}',
     '{"activity": "Lesson", "journal": "Narration", "project": "Handicraft"}',
     1),
    ('traditional', 'Traditional', 'Structured, textbook-based learning with clear grade levels',
     '{"history": "Mirrors conventional school structure adapted for home. Uses published curricula with textbooks, workbooks, and standardized assessments.", "principles": ["Structured daily schedule", "Grade-level progression", "Textbook and workbook based", "Regular testing and grading", "Teacher-directed instruction"], "typical_day": "Structured school day with subject blocks, textbook reading, workbook exercises, and regular quizzes or tests.", "pros_cons": {"pros": ["Easy to plan and follow", "Clear progress metrics", "Smooth transition to/from public school", "Widely available curricula"], "cons": ["Can feel rigid", "May not suit all learning styles", "Less room for child-led exploration"]}, "starting_resources": ["Abeka curriculum packages", "Saxon Math", "Bob Jones University Press"]}',
     '{"activity": "Lesson", "journal": "Journal Entry", "project": "Assignment"}',
     2),
    ('classical', 'Classical', 'Trivium-based education: grammar, logic, and rhetoric stages',
     '{"history": "Rooted in medieval Western education, popularized for homeschoolers by Dorothy Sayers essay \"The Lost Tools of Learning\" (1947) and Susan Wise Bauer''s \"The Well-Trained Mind.\"", "principles": ["Three stages of the Trivium", "Grammar stage (K-4): memorization and facts", "Logic stage (5-8): analytical thinking", "Rhetoric stage (9-12): persuasive expression", "Great Books and primary sources", "Latin and classical languages"], "typical_day": "Morning time with memory work, structured lessons following Trivium stage, Latin study, and Great Books reading/discussion.", "pros_cons": {"pros": ["Rigorous academic foundation", "Develops strong critical thinking", "Rich literary and historical content", "Well-documented curriculum options"], "cons": ["Can be academically intense", "Requires parent study of method", "Less emphasis on arts and hands-on learning"]}, "starting_resources": ["The Well-Trained Mind by Susan Wise Bauer", "Classical Conversations community", "Memoria Press curriculum"]}',
     '{"activity": "Lesson", "journal": "Composition", "project": "Project"}',
     3),
    ('waldorf', 'Waldorf', 'Arts-integrated, rhythm-based education honoring developmental stages',
     '{"history": "Founded by Rudolf Steiner in 1919 in Stuttgart, Germany. Emphasizes imagination, creativity, and holistic development across head, heart, and hands.", "principles": ["Education follows developmental stages", "Rhythm and routine as foundation", "Arts integration across all subjects", "Delayed academics (formal reading at age 7)", "Handwork and practical skills", "Limited screen time", "Main lesson blocks (3-4 week deep dives)"], "typical_day": "Morning circle with verses and movement, main lesson block (2 hours on one subject), artistic activities, handwork, outdoor play, and storytelling.", "pros_cons": {"pros": ["Holistic development emphasis", "Strong arts integration", "Respects childhood developmental stages", "Builds creativity and imagination"], "cons": ["Delayed academics may concern some parents", "Requires artistic confidence from parent", "Screen-free philosophy can be challenging"]}, "starting_resources": ["Simplicity Parenting by Kim John Payne", "Waldorf Essentials curriculum", "Live Education curriculum"]}',
     '{"activity": "Main Lesson", "journal": "Main Lesson Book", "project": "Handwork"}',
     4),
    ('montessori', 'Montessori', 'Child-led, hands-on learning in prepared environments',
     '{"history": "Developed by Dr. Maria Montessori (1870-1952), an Italian physician who observed that children learn best through self-directed activity in carefully prepared environments.", "principles": ["Follow the child", "Prepared environment", "Hands-on manipulatives", "Mixed-age learning", "Uninterrupted work periods (3-hour work cycles)", "Observation-based assessment", "Practical life skills as foundation"], "typical_day": "3-hour uninterrupted morning work cycle with self-chosen activities from prepared shelves, practical life work, outdoor time, and afternoon enrichment.", "pros_cons": {"pros": ["Develops independence and self-motivation", "Hands-on, concrete learning", "Respects individual pace", "Strong practical life skills"], "cons": ["Materials can be expensive", "Requires significant environment preparation", "Less structured than some parents prefer"]}, "starting_resources": ["The Absorbent Mind by Maria Montessori", "Montessori from the Start by Paula Polk Lillard", "Keys of the Universe curriculum (elementary)"]}',
     '{"activity": "Work", "journal": "Observation Log", "project": "Investigation"}',
     5),
    ('unschooling', 'Unschooling', 'Child-directed, interest-led natural learning',
     '{"history": "Coined by educator John Holt in the 1970s. Based on the belief that children are natural learners and learn best when free to pursue their own interests without imposed curricula.", "principles": ["Trust the child''s natural curiosity", "No imposed curriculum or schedule", "Life is the classroom", "Parent as facilitator, not teacher", "Learning happens through living", "Document rather than direct"], "typical_day": "No typical day — each day follows the child''s interests. Might include cooking, building, reading, nature exploration, community activities, or deep dives into passion projects.", "pros_cons": {"pros": ["Honors individual interests and passions", "Reduces school-related stress", "Develops intrinsic motivation", "Flexible and adaptable"], "cons": ["Can be hard to document for compliance", "Requires trust in the process", "May face skepticism from others", "Parent must observe and document actively"]}, "starting_resources": ["How Children Learn by John Holt", "Free to Learn by Peter Gray", "Sandra Dodd''s Big Book of Unschooling"]}',
     '{"activity": "Exploration", "journal": "Reflection", "project": "Interest Project"}',
     6);

-- FK migration: add foreign key from iam_families to method_definitions
-- (Deferred from IAM migration — method_definitions must exist first)
-- =============================================================================
-- Migration: YYYYMMDD_000003_add_method_fk_to_iam.rs
-- =============================================================================

ALTER TABLE iam_families
    ADD CONSTRAINT fk_iam_families_primary_methodology
    FOREIGN KEY (primary_methodology_id) REFERENCES method_definitions(id);

ALTER TABLE iam_students
    ADD CONSTRAINT fk_iam_students_methodology_override
    FOREIGN KEY (methodology_override_id) REFERENCES method_definitions(id);

-- =============================================================================
-- Master tool catalog [S§8.1]
-- =============================================================================

-- Core tools (available across most/all methodologies) [S§8.1.1-8.1.7]
INSERT INTO method_tools (slug, display_name, description, tier)
VALUES
    ('activities',          'Activities',              'Log daily learning activities with title, description, subjects, and attachments', 'free'),
    ('tests-grades',        'Tests & Grades',          'Record assessments with scores, grading scales, and running averages',             'free'),
    ('reading-lists',       'Reading Lists',           'Create and manage book lists with status tracking and ISBN lookup',                'free'),
    ('journaling',          'Journaling & Narration',  'Create journal entries with text, images, and typed categorization',               'free'),
    ('projects',            'Projects',                'Multi-step projects with milestones, status tracking, and portfolio export',        'free'),
    ('video-lessons',       'Video Lessons',           'Access purchased marketplace video content with progress tracking',                'free'),
    ('progress-tracking',   'Progress Tracking',       'Activity counts, subject hours, and trend visualization',                          'free');

-- Methodology-specific tools [S§8.1.8]
INSERT INTO method_tools (slug, display_name, description, tier)
VALUES
    ('nature-journals',     'Nature Journals',         'Nature observations with drawing/photo support and species identification',          'free'),
    ('trivium-tracker',     'Trivium Tracker',         'Track student progress through grammar, logic, and rhetoric stages per subject',     'free'),
    ('rhythm-planner',      'Rhythm Planner',          'Weekly/daily rhythm planning with blocks for lessons, arts, practical work, play',   'free'),
    ('observation-logs',    'Observation Logs',        'Structured observation records: work chosen, duration, concentration, follow-up',     'free'),
    ('habit-tracking',      'Habit Tracking',          'Track daily/weekly habit formation goals with streaks and parent notes',              'free'),
    ('interest-led-logs',   'Interest-Led Activity Log','Document child-initiated learning with auto-tagging to subjects/skills',            'free'),
    ('handwork-tracker',    'Handwork Project Tracker','Track handwork and craft projects with materials, techniques, and photos',            'free'),
    ('practical-life',      'Practical Life Activities','Log and track practical life skill development with age-appropriate suggestions',    'free');

-- Additional cross-methodology tools
INSERT INTO method_tools (slug, display_name, description, tier)
VALUES
    ('unit-studies',        'Unit Studies',            'Thematic unit studies integrating multiple subjects around a central topic',           'free'),
    ('field-trip-log',      'Field Trip Log',          'Document field trips with location, subjects covered, photos, and reflections',       'free'),
    ('lesson-planner',      'Lesson Planner',          'Weekly/daily lesson planning with subject scheduling and resource linking',           'free');

-- =============================================================================
-- Tool activation mappings [S§4.2]
-- config_overrides carry methodology-specific labels, guidance text, and entry types
-- =============================================================================

-- Charlotte Mason activations
INSERT INTO method_tool_activations (methodology_id, tool_id, config_overrides, sort_order)
VALUES
    ((SELECT id FROM method_definitions WHERE slug = 'charlotte-mason'),
     (SELECT id FROM method_tools WHERE slug = 'activities'),
     '{"label": "Lessons", "guidance": "Charlotte Mason recommended short, focused lessons — 15 minutes for young children, up to 45 minutes for older students. Full attention is expected for the duration."}',
     1),
    ((SELECT id FROM method_definitions WHERE slug = 'charlotte-mason'),
     (SELECT id FROM method_tools WHERE slug = 'reading-lists'),
     '{"label": "Living Books", "guidance": "Choose ''living books'' — books written by a single author passionate about the subject, with literary quality and narrative power. Avoid twaddle (dumbed-down or overly simplified texts)."}',
     2),
    ((SELECT id FROM method_definitions WHERE slug = 'charlotte-mason'),
     (SELECT id FROM method_tools WHERE slug = 'journaling'),
     '{"label": "Narration", "guidance": "After reading a passage, ask the child to tell back what they heard in their own words. Written narration begins around age 10. This is the primary method of retention.", "entry_types": ["narration", "written_narration", "free_journal"]}',
     3),
    ((SELECT id FROM method_definitions WHERE slug = 'charlotte-mason'),
     (SELECT id FROM method_tools WHERE slug = 'nature-journals'),
     '{"label": "Nature Journal", "guidance": "Spend regular time outdoors observing nature. Children draw what they see with careful attention to detail, adding notes about species, weather, and season."}',
     4),
    ((SELECT id FROM method_definitions WHERE slug = 'charlotte-mason'),
     (SELECT id FROM method_tools WHERE slug = 'habit-tracking'),
     '{"label": "Habit Training", "guidance": "Charlotte Mason considered habit training the foundation of education. Focus on one habit at a time for 4-6 weeks before adding another."}',
     5),
    ((SELECT id FROM method_definitions WHERE slug = 'charlotte-mason'),
     (SELECT id FROM method_tools WHERE slug = 'progress-tracking'),
     '{"label": "Progress"}',
     6),
    ((SELECT id FROM method_definitions WHERE slug = 'charlotte-mason'),
     (SELECT id FROM method_tools WHERE slug = 'unit-studies'),
     '{"label": "Unit Studies", "guidance": "While not strictly CM, thematic studies complement living books by connecting subjects around historical periods or nature topics."}',
     7),
    ((SELECT id FROM method_definitions WHERE slug = 'charlotte-mason'),
     (SELECT id FROM method_tools WHERE slug = 'field-trip-log'),
     '{"label": "Field Trip Log"}',
     8),
    ((SELECT id FROM method_definitions WHERE slug = 'charlotte-mason'),
     (SELECT id FROM method_tools WHERE slug = 'lesson-planner'),
     '{"label": "Lesson Planner", "guidance": "Plan short, varied lessons across subjects. CM schedules typically alternate between demanding and lighter subjects."}',
     9);

-- Traditional activations
INSERT INTO method_tool_activations (methodology_id, tool_id, config_overrides, sort_order)
VALUES
    ((SELECT id FROM method_definitions WHERE slug = 'traditional'),
     (SELECT id FROM method_tools WHERE slug = 'activities'),
     '{"label": "Lessons", "guidance": "Follow your curriculum''s lesson plans. Log each subject session with the textbook/workbook pages covered."}',
     1),
    ((SELECT id FROM method_definitions WHERE slug = 'traditional'),
     (SELECT id FROM method_tools WHERE slug = 'tests-grades'),
     '{"label": "Tests & Grades", "guidance": "Record quiz and test scores regularly. Use the grading scale that matches your curriculum or state requirements."}',
     2),
    ((SELECT id FROM method_definitions WHERE slug = 'traditional'),
     (SELECT id FROM method_tools WHERE slug = 'reading-lists'),
     '{"label": "Reading Lists"}',
     3),
    ((SELECT id FROM method_definitions WHERE slug = 'traditional'),
     (SELECT id FROM method_tools WHERE slug = 'progress-tracking'),
     '{"label": "Progress & Grades"}',
     4),
    ((SELECT id FROM method_definitions WHERE slug = 'traditional'),
     (SELECT id FROM method_tools WHERE slug = 'unit-studies'),
     '{"label": "Unit Studies", "guidance": "Supplement textbook work with thematic unit studies to bring subjects to life."}',
     5),
    ((SELECT id FROM method_definitions WHERE slug = 'traditional'),
     (SELECT id FROM method_tools WHERE slug = 'field-trip-log'),
     '{"label": "Field Trip Log"}',
     6),
    ((SELECT id FROM method_definitions WHERE slug = 'traditional'),
     (SELECT id FROM method_tools WHERE slug = 'lesson-planner'),
     '{"label": "Lesson Planner", "guidance": "Plan your week by subject and textbook chapter. Track pages assigned vs. completed."}',
     7);

-- Classical activations
INSERT INTO method_tool_activations (methodology_id, tool_id, config_overrides, sort_order)
VALUES
    ((SELECT id FROM method_definitions WHERE slug = 'classical'),
     (SELECT id FROM method_tools WHERE slug = 'activities'),
     '{"label": "Lessons", "guidance": "Adapt lesson structure to the student''s Trivium stage: memory-heavy for Grammar, discussion-based for Logic, presentation-focused for Rhetoric."}',
     1),
    ((SELECT id FROM method_definitions WHERE slug = 'classical'),
     (SELECT id FROM method_tools WHERE slug = 'tests-grades'),
     '{"label": "Assessments", "guidance": "Grammar stage: fact recall and recitation. Logic stage: analysis and essay. Rhetoric stage: thesis defense and presentation."}',
     2),
    ((SELECT id FROM method_definitions WHERE slug = 'classical'),
     (SELECT id FROM method_tools WHERE slug = 'reading-lists'),
     '{"label": "Great Books", "guidance": "Build lists from the Great Books canon appropriate to the student''s stage. Include primary sources alongside secondary readings."}',
     3),
    ((SELECT id FROM method_definitions WHERE slug = 'classical'),
     (SELECT id FROM method_tools WHERE slug = 'journaling'),
     '{"label": "Composition", "guidance": "Grammar stage: copywork and dictation. Logic stage: outlines and essays. Rhetoric stage: research papers and persuasive writing.", "entry_types": ["copywork", "dictation", "essay", "research_paper"]}',
     4),
    ((SELECT id FROM method_definitions WHERE slug = 'classical'),
     (SELECT id FROM method_tools WHERE slug = 'trivium-tracker'),
     '{"label": "Trivium Progress", "guidance": "Track each subject through the three stages. Students may be at different stages for different subjects."}',
     5),
    ((SELECT id FROM method_definitions WHERE slug = 'classical'),
     (SELECT id FROM method_tools WHERE slug = 'progress-tracking'),
     '{"label": "Academic Progress"}',
     6),
    ((SELECT id FROM method_definitions WHERE slug = 'classical'),
     (SELECT id FROM method_tools WHERE slug = 'unit-studies'),
     '{"label": "Unit Studies", "guidance": "Use unit studies to integrate history cycles with literature, writing, and geography."}',
     7),
    ((SELECT id FROM method_definitions WHERE slug = 'classical'),
     (SELECT id FROM method_tools WHERE slug = 'field-trip-log'),
     '{"label": "Field Trip Log"}',
     8),
    ((SELECT id FROM method_definitions WHERE slug = 'classical'),
     (SELECT id FROM method_tools WHERE slug = 'lesson-planner'),
     '{"label": "Lesson Planner", "guidance": "Plan lessons around the 4-year history cycle and Trivium stage. Include memory work, reading assignments, and writing exercises."}',
     9);

-- Waldorf activations
INSERT INTO method_tool_activations (methodology_id, tool_id, config_overrides, sort_order)
VALUES
    ((SELECT id FROM method_definitions WHERE slug = 'waldorf'),
     (SELECT id FROM method_tools WHERE slug = 'activities'),
     '{"label": "Main Lessons", "guidance": "Main lesson blocks last 3-4 weeks, diving deep into one subject each morning for ~2 hours. Integrate artistic elements into every lesson."}',
     1),
    ((SELECT id FROM method_definitions WHERE slug = 'waldorf'),
     (SELECT id FROM method_tools WHERE slug = 'journaling'),
     '{"label": "Main Lesson Book", "guidance": "The main lesson book is the child''s own textbook — created through beautiful illustrations, summaries, and artistic renderings of the material.", "entry_types": ["main_lesson_page", "illustration", "summary"]}',
     2),
    ((SELECT id FROM method_definitions WHERE slug = 'waldorf'),
     (SELECT id FROM method_tools WHERE slug = 'projects'),
     '{"label": "Handwork & Projects", "guidance": "Waldorf handwork progresses through developmental stages: finger knitting → knitting → crocheting → sewing → woodworking."}',
     3),
    ((SELECT id FROM method_definitions WHERE slug = 'waldorf'),
     (SELECT id FROM method_tools WHERE slug = 'rhythm-planner'),
     '{"label": "Rhythm Planner", "guidance": "Waldorf education is built on rhythm — daily, weekly, and seasonal. Plan blocks for main lessons, artistic activities, practical work, and free play."}',
     4),
    ((SELECT id FROM method_definitions WHERE slug = 'waldorf'),
     (SELECT id FROM method_tools WHERE slug = 'handwork-tracker'),
     '{"label": "Handwork Projects", "guidance": "Track handwork projects with materials, techniques, and photos. Each project builds on skills from previous work."}',
     5),
    ((SELECT id FROM method_definitions WHERE slug = 'waldorf'),
     (SELECT id FROM method_tools WHERE slug = 'progress-tracking'),
     '{"label": "Development Progress"}',
     6),
    ((SELECT id FROM method_definitions WHERE slug = 'waldorf'),
     (SELECT id FROM method_tools WHERE slug = 'unit-studies'),
     '{"label": "Block Studies", "guidance": "Waldorf main lesson blocks are similar to unit studies — deep, immersive explorations of a single subject area."}',
     7),
    ((SELECT id FROM method_definitions WHERE slug = 'waldorf'),
     (SELECT id FROM method_tools WHERE slug = 'field-trip-log'),
     '{"label": "Field Trip Log"}',
     8),
    ((SELECT id FROM method_definitions WHERE slug = 'waldorf'),
     (SELECT id FROM method_tools WHERE slug = 'lesson-planner'),
     '{"label": "Block Planner", "guidance": "Plan 3-4 week main lesson blocks with daily rhythm. Include morning circle, main lesson, practice period, and artistic activities."}',
     9);

-- Montessori activations
INSERT INTO method_tool_activations (methodology_id, tool_id, config_overrides, sort_order)
VALUES
    ((SELECT id FROM method_definitions WHERE slug = 'montessori'),
     (SELECT id FROM method_tools WHERE slug = 'activities'),
     '{"label": "Work", "guidance": "In Montessori, activities are called ''work.'' Children choose their work from the prepared environment. Log the work chosen, materials used, and duration."}',
     1),
    ((SELECT id FROM method_definitions WHERE slug = 'montessori'),
     (SELECT id FROM method_tools WHERE slug = 'projects'),
     '{"label": "Investigations", "guidance": "Child-initiated deep dives into topics of interest. Follow the child''s questions and provide resources and materials to explore further."}',
     2),
    ((SELECT id FROM method_definitions WHERE slug = 'montessori'),
     (SELECT id FROM method_tools WHERE slug = 'observation-logs'),
     '{"label": "Observation Logs", "guidance": "Observe the child at work: What did they choose? How long did they concentrate? Did they repeat the activity? What social interactions occurred? Use these observations to guide environment preparation."}',
     3),
    ((SELECT id FROM method_definitions WHERE slug = 'montessori'),
     (SELECT id FROM method_tools WHERE slug = 'practical-life'),
     '{"label": "Practical Life", "guidance": "Practical life activities are the foundation of Montessori education. Include care of self, care of environment, grace and courtesy, and control of movement."}',
     4),
    ((SELECT id FROM method_definitions WHERE slug = 'montessori'),
     (SELECT id FROM method_tools WHERE slug = 'progress-tracking'),
     '{"label": "Development Progress"}',
     5),
    ((SELECT id FROM method_definitions WHERE slug = 'montessori'),
     (SELECT id FROM method_tools WHERE slug = 'unit-studies'),
     '{"label": "Cosmic Education", "guidance": "Montessori cosmic education connects all subjects through the Great Lessons, showing children how everything in the universe is interconnected."}',
     6),
    ((SELECT id FROM method_definitions WHERE slug = 'montessori'),
     (SELECT id FROM method_tools WHERE slug = 'field-trip-log'),
     '{"label": "Field Trip Log"}',
     7);

-- Unschooling activations
INSERT INTO method_tool_activations (methodology_id, tool_id, config_overrides, sort_order)
VALUES
    ((SELECT id FROM method_definitions WHERE slug = 'unschooling'),
     (SELECT id FROM method_tools WHERE slug = 'activities'),
     '{"label": "Explorations", "guidance": "Document what your child naturally gravitates toward. Everything counts — cooking, building, gaming, conversations, outings. Learning is living."}',
     1),
    ((SELECT id FROM method_definitions WHERE slug = 'unschooling'),
     (SELECT id FROM method_tools WHERE slug = 'journaling'),
     '{"label": "Reflections", "guidance": "Use journaling to reflect on and document your child''s learning journey. This is primarily a parent tool for recognizing learning as it happens.", "entry_types": ["observation", "reflection", "documentation"]}',
     2),
    ((SELECT id FROM method_definitions WHERE slug = 'unschooling'),
     (SELECT id FROM method_tools WHERE slug = 'projects'),
     '{"label": "Interest Projects", "guidance": "When a child dives deep into a passion, help them structure it into a project. These emerge naturally from sustained interest, not from assignment."}',
     3),
    ((SELECT id FROM method_definitions WHERE slug = 'unschooling'),
     (SELECT id FROM method_tools WHERE slug = 'interest-led-logs'),
     '{"label": "Interest-Led Learning", "guidance": "Document child-initiated activities and the learning within them. The auto-tagging helps map natural learning to academic subjects for compliance purposes."}',
     4),
    ((SELECT id FROM method_definitions WHERE slug = 'unschooling'),
     (SELECT id FROM method_tools WHERE slug = 'progress-tracking'),
     '{"label": "Learning Documentation"}',
     5),
    ((SELECT id FROM method_definitions WHERE slug = 'unschooling'),
     (SELECT id FROM method_tools WHERE slug = 'field-trip-log'),
     '{"label": "Adventures", "guidance": "Document outings, museum visits, nature explorations, community events — all the ways your family engages with the world."}',
     6);

-- Video Lessons — activated for all methodologies [S§8.1.6]
-- This tool surfaces purchased marketplace video content through the learning interface.
-- It appears in every methodology's tool set but only has content when a family has
-- marketplace purchases.
INSERT INTO method_tool_activations (methodology_id, tool_id, config_overrides, sort_order)
SELECT md.id, mt.id, '{"label": "Video Lessons"}', 99
FROM method_definitions md
CROSS JOIN method_tools mt
WHERE mt.slug = 'video-lessons'
  AND md.is_active = true;
```

### §3.4 Seed Data Summary

| Methodology | Activated Tools |
|-------------|----------------|
| **Charlotte Mason** | Activities, Reading Lists, Journaling, Nature Journals, Habit Tracking, Progress Tracking, Unit Studies, Field Trip Log, Lesson Planner, Video Lessons |
| **Traditional** | Activities, Tests & Grades, Reading Lists, Progress Tracking, Unit Studies, Field Trip Log, Lesson Planner, Video Lessons |
| **Classical** | Activities, Tests & Grades, Reading Lists, Journaling, Trivium Tracker, Progress Tracking, Unit Studies, Field Trip Log, Lesson Planner, Video Lessons |
| **Waldorf** | Activities, Journaling, Projects, Rhythm Planner, Handwork Tracker, Progress Tracking, Unit Studies, Field Trip Log, Lesson Planner, Video Lessons |
| **Montessori** | Activities, Projects, Observation Logs, Practical Life, Progress Tracking, Unit Studies, Field Trip Log, Video Lessons |
| **Unschooling** | Activities, Journaling, Projects, Interest-Led Logs, Progress Tracking, Field Trip Log, Video Lessons |

**Tool coverage**: 18 tools in master catalog. Activities, Progress Tracking, Field Trip Log,
and Video Lessons are activated for all 6 methodologies. Video Lessons is activated universally
because it surfaces marketplace-purchased content regardless of methodology `[S§8.1.6]`.
Every methodology-specific tool (`[S§8.1.8]`) is activated for exactly the methodology it
was designed for. Lesson Planner is intentionally omitted from Montessori (child-directed
3-hour work cycles are not pre-planned) and Unschooling (no imposed schedule).

---

## §4 API Endpoints

All endpoints return JSON. Error responses follow `AppError` → HTTP status mapping (§10.4).

### §4.1 Phase 1 Endpoints

| # | Method | Path | Auth | Description | Status Codes |
|---|--------|------|------|-------------|-------------|
| 1 | `GET` | `/v1/methodologies` | None | List all active methodologies | 200 |
| 2 | `GET` | `/v1/methodologies/:slug` | None | Get methodology detail by slug | 200, 404 |
| 3 | `GET` | `/v1/methodologies/:slug/tools` | None | List tools activated for a methodology | 200, 404 |
| 4 | `GET` | `/v1/families/tools` | Required | Get family's resolved active tool set | 200, 401 |
| 5 | `GET` | `/v1/families/students/:id/tools` | Required | Get student's resolved tool set | 200, 401, 404 |
| 6 | `PATCH` | `/v1/families/methodology` | Required | Update family methodology selection | 200, 401, 422 |

**Phase 1 total**: 3 public + 3 authenticated = 6 routes.

### §4.2 Phase 2 Endpoints

| # | Method | Path | Auth | Description | Status Codes |
|---|--------|------|------|-------------|-------------|
| 7 | `GET` | `/v1/families/methodology-context` | Required | Full methodology context for dashboard | 200, 401 |
| 8 | `PATCH` | `/v1/families/students/:id/methodology` | Required | Set student methodology override | 200, 401, 404, 422 |

### §4.3 Phase 3+ Endpoints (Admin)

| # | Method | Path | Auth | Description | Status Codes |
|---|--------|------|------|-------------|-------------|
| 9 | `POST` | `/admin/v1/methodologies` | Admin | Create methodology | 201, 401, 403, 422 |
| 10 | `PATCH` | `/admin/v1/methodologies/:slug` | Admin | Update methodology | 200, 401, 403, 404, 422 |
| 11 | `POST` | `/admin/v1/tools` | Admin | Create tool | 201, 401, 403, 422 |
| 12 | `POST` | `/admin/v1/methodologies/:slug/tools` | Admin | Activate tool for methodology | 201, 401, 403, 404, 422 |

### §4.4 Endpoint Details

#### `GET /v1/methodologies`

Returns all active methodologies, ordered by `display_order`. Used by the onboarding wizard,
methodology explorer, and methodology selection UI. Public — no authentication required. `[S§5.2]`

- **Extractors**: None
- **FamilyScope**: No (public data)
- **Response**: `Vec<MethodologySummaryResponse>`

#### `GET /v1/methodologies/:slug`

Returns full methodology detail including philosophy module content. Used by methodology
explorer pages and the selection wizard detail view. `[S§5.2]`

- **Extractors**: `Path<String>` (slug)
- **FamilyScope**: No (public data)
- **Response**: `MethodologyDetailResponse`

#### `GET /v1/methodologies/:slug/tools`

Returns all tools activated for a specific methodology with their config overrides.
Used by the methodology explorer to show "what tools come with this methodology."

- **Extractors**: `Path<String>` (slug)
- **FamilyScope**: No (public data)
- **Response**: `Vec<ActiveToolResponse>`

#### `GET /v1/families/tools`

Returns the family's resolved active tool set: the union of all tools across primary +
secondary methodologies, deduplicated. `[S§4.2]`

- **Extractors**: `AuthContext`, `FamilyScope`
- **Response**: `Vec<ActiveToolResponse>`

#### `GET /v1/families/students/:id/tools`

Returns a specific student's resolved tool set. If the student has a methodology override,
returns tools for that override methodology. Otherwise, returns the family-level tool set.
`[S§4.6]`

- **Extractors**: `AuthContext`, `FamilyScope`, `Path<Uuid>` (student_id)
- **Response**: `Vec<ActiveToolResponse>`

#### `PATCH /v1/families/methodology`

Updates the family's primary and/or secondary methodology selections. Validates that all
methodology IDs exist and are active. Delegates persistence to `iam::FamilyRepository::set_methodology`.
Publishes `FamilyMethodologyChanged` event. `[S§4.3]`

- **Extractors**: `AuthContext`, `FamilyScope`, `Json<UpdateMethodologyCommand>`
- **Validation**: All methodology IDs must reference active `method_definitions` rows
- **Response**: `MethodologySelectionResponse`
- **Events**: `FamilyMethodologyChanged`
- **Note**: Changing methodology does NOT delete historical learning data `[S§4.3]`

#### `GET /v1/families/methodology-context` (Phase 2)

Returns full methodology context for the dashboard: primary and secondary methodology
summaries, resolved terminology overrides, and mastery level. `[ARCH §7.2]`

- **Extractors**: `AuthContext`, `FamilyScope`
- **Response**: `MethodologyContext`

#### `PATCH /v1/families/students/:id/methodology` (Phase 2)

Sets or clears the methodology override for a specific student. Validates the methodology
ID exists if provided. Delegates persistence to `iam::StudentRepository::update`. `[S§4.6]`

- **Extractors**: `AuthContext`, `FamilyScope`, `Path<Uuid>`, `Json<UpdateStudentMethodologyCommand>`
- **Response**: `MethodologySelectionResponse`

---

## §5 Service Interface

The `MethodologyService` trait defines all use cases exposed to handlers and other domains.
Defined in `src/method/ports.rs`. `[CODING §8.2]`

```rust
#[async_trait]
pub trait MethodologyService: Send + Sync {
    // ─── Public Queries (no auth required) ──────────────────────────────

    /// Lists all active methodologies, ordered by display_order.
    /// Used by GET /v1/methodologies and consumed by onboard:: and discover::.
    async fn list_methodologies(&self) -> Result<Vec<MethodologySummaryResponse>, AppError>;

    /// Returns full methodology detail by slug, including philosophy module.
    /// Used by GET /v1/methodologies/:slug.
    async fn get_methodology(
        &self,
        slug: &str,
    ) -> Result<MethodologyDetailResponse, AppError>;

    /// Returns tools activated for a specific methodology.
    /// Used by GET /v1/methodologies/:slug/tools.
    async fn get_methodology_tools(
        &self,
        slug: &str,
    ) -> Result<Vec<ActiveToolResponse>, AppError>;

    /// Validates that a methodology ID exists and is active.
    /// Used by iam:: during registration to validate default methodology.
    async fn validate_methodology_id(&self, id: Uuid) -> Result<bool, AppError>;

    /// Returns the default methodology ID (first active by display_order).
    /// Used by iam:: during registration when no methodology is specified.
    async fn get_default_methodology_id(&self) -> Result<Uuid, AppError>;

    // ─── Family-Scoped Queries (auth required) ─────────────────────────

    /// Resolves the family's active tool set (union of all methodology tools,
    /// deduplicated). Used by GET /v1/families/tools and consumed by learn::.
    /// [S§4.2]
    async fn resolve_family_tools(
        &self,
        scope: &FamilyScope,
    ) -> Result<Vec<ActiveToolResponse>, AppError>;

    /// Resolves a student's active tool set, considering methodology overrides.
    /// Used by GET /v1/families/students/:id/tools. [S§4.6]
    async fn resolve_student_tools(
        &self,
        scope: &FamilyScope,
        student_id: Uuid,
    ) -> Result<Vec<ActiveToolResponse>, AppError>;

    /// Returns full methodology context for the family dashboard.
    /// Used by GET /v1/families/methodology-context (Phase 2). [ARCH §7.2]
    async fn get_methodology_context(
        &self,
        scope: &FamilyScope,
    ) -> Result<MethodologyContext, AppError>;

    // ─── Commands ───────────────────────────────────────────────────────

    /// Updates the family's methodology selection (primary + secondary).
    /// Validates all IDs, delegates persistence to iam::FamilyRepository.
    /// Publishes FamilyMethodologyChanged. [S§4.3]
    async fn update_family_methodology(
        &self,
        scope: &FamilyScope,
        cmd: UpdateMethodologyCommand,
    ) -> Result<MethodologySelectionResponse, AppError>;

    /// Sets or clears a student's methodology override.
    /// Delegates persistence to iam::StudentRepository. [S§4.6] (Phase 2)
    async fn update_student_methodology(
        &self,
        scope: &FamilyScope,
        student_id: Uuid,
        cmd: UpdateStudentMethodologyCommand,
    ) -> Result<MethodologySelectionResponse, AppError>;
}
```

**Implementation**: `MethodologyServiceImpl` in `src/method/service.rs`. Constructor receives:
- `Arc<dyn MethodologyDefinitionRepository>`
- `Arc<dyn ToolRepository>`
- `Arc<dyn ToolActivationRepository>`
- `Arc<dyn IamService>` (for family/student data and methodology persistence)
- `Arc<EventBus>`
- `Arc<RedisPool>` (for caching)

---

## §6 Repository Interfaces

Defined in `src/method/ports.rs`. All repositories are **NOT family-scoped** — method_ tables
contain global platform config, not user data. `[CODING §2.4, CODING §8.2]`

```rust
#[async_trait]
pub trait MethodologyDefinitionRepository: Send + Sync {
    /// Lists all active methodologies, ordered by display_order.
    /// NOT family-scoped — global config.
    async fn list_active(&self) -> Result<Vec<MethodologyDefinition>, AppError>;

    /// Finds a methodology by slug. NOT family-scoped.
    async fn find_by_slug(
        &self,
        slug: &str,
    ) -> Result<Option<MethodologyDefinition>, AppError>;

    /// Finds a methodology by ID. NOT family-scoped.
    async fn find_by_id(
        &self,
        id: Uuid,
    ) -> Result<Option<MethodologyDefinition>, AppError>;

    /// Finds multiple methodologies by IDs. NOT family-scoped.
    /// Used by tool resolution to batch-load primary + secondary methodologies.
    async fn find_by_ids(
        &self,
        ids: &[Uuid],
    ) -> Result<Vec<MethodologyDefinition>, AppError>;

    /// Checks that all provided IDs reference active methodology rows.
    /// Returns true only if ALL IDs are valid and active.
    async fn all_active(
        &self,
        ids: &[Uuid],
    ) -> Result<bool, AppError>;
}

#[async_trait]
pub trait ToolRepository: Send + Sync {
    /// Lists all active tools. NOT family-scoped — global catalog.
    async fn list_active(&self) -> Result<Vec<Tool>, AppError>;

    /// Finds a tool by slug. NOT family-scoped.
    async fn find_by_slug(
        &self,
        slug: &str,
    ) -> Result<Option<Tool>, AppError>;

    /// Finds a tool by ID. NOT family-scoped.
    async fn find_by_id(
        &self,
        id: Uuid,
    ) -> Result<Option<Tool>, AppError>;
}

#[async_trait]
pub trait ToolActivationRepository: Send + Sync {
    /// Lists all tool activations for a methodology, ordered by sort_order.
    /// Joins with method_tools to include tool metadata. NOT family-scoped.
    async fn list_by_methodology(
        &self,
        methodology_id: Uuid,
    ) -> Result<Vec<ToolActivationWithTool>, AppError>;

    /// Lists all tool activations for multiple methodologies.
    /// Used by tool resolution (union across primary + secondary). NOT family-scoped.
    async fn list_by_methodologies(
        &self,
        methodology_ids: &[Uuid],
    ) -> Result<Vec<ToolActivationWithTool>, AppError>;
}
```

**FamilyScope exception documentation**: None of these repositories accept `FamilyScope`
because all method_ data is global platform configuration. This is documented here and
in repository trait doc comments. When tool resolution needs family data (which methodologies
a family has selected), it retrieves that data through `IamService`, not by querying
method_ tables with a family filter.

**Implementations**:
- `PgMethodologyDefinitionRepository` in `src/method/repository.rs`
- `PgToolRepository` in `src/method/repository.rs`
- `PgToolActivationRepository` in `src/method/repository.rs`

---

## §7 Adapter Interfaces

None. The methodology domain has no external third-party service dependencies. All data is
stored in PostgreSQL and cached in Redis using shared infrastructure
(`src/shared/redis.rs`). `[CODING §8.1]`

---

## §8 Models (DTOs)

All types defined in `src/method/models.rs`. API-facing types derive `serde::Serialize`,
`serde::Deserialize`, and `utoipa::ToSchema`. Request types additionally derive
`validator::Validate`. `[CODING §2.3]`

### §8.1 Request Types

```rust
/// PATCH /v1/families/methodology [S§4.3]
#[derive(Debug, Deserialize, Validate, ToSchema)]
pub struct UpdateMethodologyCommand {
    /// Primary methodology ID (required). Must reference an active method_definitions row.
    pub primary_methodology_id: Uuid,
    /// Secondary methodology IDs (optional). Each must reference an active row. [S§4.3]
    /// An empty vec means "no secondary methodologies."
    #[validate(length(max = 5))]
    pub secondary_methodology_ids: Vec<Uuid>,
}

/// PATCH /v1/families/students/:id/methodology (Phase 2) [S§4.6]
#[derive(Debug, Deserialize, Validate, ToSchema)]
pub struct UpdateStudentMethodologyCommand {
    /// Set to a methodology ID to override, or null/absent to clear the override
    /// and inherit family methodology. [S§4.6]
    pub methodology_override_id: Option<Uuid>,
}
```

### §8.2 Response Types

```rust
/// GET /v1/methodologies — list item
#[derive(Debug, Serialize, ToSchema)]
pub struct MethodologySummaryResponse {
    pub id: Uuid,
    pub slug: String,
    pub display_name: String,
    pub short_desc: String,
    pub icon_url: Option<String>,
}

/// GET /v1/methodologies/:slug — full detail
#[derive(Debug, Serialize, ToSchema)]
pub struct MethodologyDetailResponse {
    pub id: Uuid,
    pub slug: String,
    pub display_name: String,
    pub short_desc: String,
    pub icon_url: Option<String>,
    pub philosophy: serde_json::Value,         // [S§4.1] full philosophy module
    pub onboarding_config: serde_json::Value,  // [S§6.4] consumed by onboard:: for materialization
    pub community_config: serde_json::Value,   // [S§6.6] consumed by onboard:: for community suggestions
    pub mastery_paths: serde_json::Value,      // [S§4.1]
    pub terminology: serde_json::Value,        // [S§4.4]
}

/// Tool as resolved for a family or methodology
/// Used by GET /v1/families/tools, GET /v1/methodologies/:slug/tools
#[derive(Debug, Clone, Serialize, ToSchema)]
pub struct ActiveToolResponse {
    pub tool_id: Uuid,
    pub slug: String,
    pub display_name: String,          // base tool name
    pub label: Option<String>,         // methodology-specific override label
    pub description: Option<String>,
    pub tier: String,                  // "free" or "premium"
    pub guidance: Option<String>,      // methodology-specific guidance text
    pub config_overrides: serde_json::Value,  // full override payload
    pub sort_order: i16,
}

/// PATCH /v1/families/methodology response
#[derive(Debug, Serialize, ToSchema)]
pub struct MethodologySelectionResponse {
    pub primary: MethodologySummaryResponse,
    pub secondary: Vec<MethodologySummaryResponse>,
    pub active_tool_count: usize,
}

/// GET /v1/families/methodology-context (Phase 2) [ARCH §7.2]
#[derive(Debug, Serialize, ToSchema)]
pub struct MethodologyContext {
    pub primary: MethodologySummaryResponse,
    pub secondary: Vec<MethodologySummaryResponse>,
    /// Merged terminology overrides from primary methodology.
    /// e.g., {"activity": "Lesson", "journal": "Narration"}
    pub terminology: serde_json::Value,
    /// Current mastery path level, if set [S§4.1]
    pub mastery_level: Option<String>,
}
```

### §8.3 Internal Types (not API-facing)

```rust
/// Full methodology record from database
pub struct MethodologyDefinition {
    pub id: Uuid,
    pub slug: String,
    pub display_name: String,
    pub short_desc: String,
    pub icon_url: Option<String>,
    pub philosophy: serde_json::Value,
    pub onboarding_config: serde_json::Value,
    pub community_config: serde_json::Value,
    pub mastery_paths: serde_json::Value,
    pub terminology: serde_json::Value,
    pub display_order: i16,
    pub is_active: bool,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

/// Tool record from database
pub struct Tool {
    pub id: Uuid,
    pub slug: String,
    pub display_name: String,
    pub description: Option<String>,
    pub config_schema: serde_json::Value,
    pub tier: String,
    pub is_active: bool,
    pub created_at: DateTime<Utc>,
}

/// Tool activation joined with tool metadata — repository return type
pub struct ToolActivationWithTool {
    pub methodology_id: Uuid,
    pub tool_id: Uuid,
    pub tool_slug: String,
    pub tool_display_name: String,
    pub tool_description: Option<String>,
    pub tool_tier: String,
    pub tool_is_active: bool,
    pub config_overrides: serde_json::Value,
    pub sort_order: i16,
}
```

---

## §9 Config-Driven Architecture

This section documents the enforcement of `[CODING §5.1]` (Methodology-as-Configuration)
within the method:: domain itself.

### §9.1 No Methodology Name Branching

The following patterns are **forbidden** in any file under `src/method/` and in any other
domain that consumes methodology data:

```rust
// FORBIDDEN — branching on methodology name [CODING §5.1]
if methodology.slug == "charlotte-mason" { ... }
match methodology.slug.as_str() { "classical" => ..., _ => ... }
```

All methodology-dependent behavior is resolved by reading the methodology's configuration
columns (`philosophy`, `terminology`, `onboarding_config`, `mastery_paths`, `community_config`)
or the tool activation's `config_overrides` column.

### §9.2 Adding a New Methodology

Adding a methodology (e.g., Reggio Emilia) requires only database operations, not code
changes. `[S§4.5, ARCH §7.3]`

1. **INSERT** into `method_definitions` — the new methodology appears in `GET /v1/methodologies`
2. **INSERT** into `method_tool_activations` — tools are activated with appropriate config overrides
3. **INSERT** into `soc_groups` — platform-managed community group (optional, owned by `social::`) `[S§6.6]`
4. No deployment required. No code changes. No cache warming needed (cache TTL handles it).

### §9.3 Redis Caching Strategy

Methodology definitions and tool activations are read frequently and change rarely. Redis
caching reduces database load for these hot paths.

| Cache Key | TTL | Content | Invalidation |
|-----------|-----|---------|-------------|
| `method:definitions:active` | 1 hour | All active methodology summaries (JSON) | `MethodologyConfigUpdated` event |
| `method:definition:{slug}` | 1 hour | Single methodology detail (JSON) | `MethodologyConfigUpdated` event |
| `method:tools:{methodology_id}` | 1 hour | Tools for a methodology (JSON) | `MethodologyConfigUpdated` event |
| `method:family_tools:{family_id}` | 10 min | Family's resolved tool set (JSON) | `FamilyMethodologyChanged` event |
| `method:student_tools:{student_id}` | 10 min | Student's resolved tool set (JSON) | `FamilyMethodologyChanged` event |

**Cache implementation**: Uses `src/shared/redis.rs` helpers. Cache-aside pattern:
1. Check Redis for cached value
2. On miss: query PostgreSQL, serialize to JSON, store in Redis with TTL
3. On `MethodologyConfigUpdated`: delete all `method:definitions:*` and `method:tools:*` keys
4. On `FamilyMethodologyChanged`: delete `method:family_tools:{family_id}` and
   `method:student_tools:{student_id}` keys for all students in the family

**Phase 1 note**: Caching is optional in Phase 1 if the database query is fast enough (< 5ms
for methodology lookups). Add caching when measured latency justifies it, following the
progressive optimization ladder `[ARCH §4.7]`.

---

## §10 Domain Layer

`method::` is classified as a **complex domain** `[ARCH §4.5]` because tool resolution
involves multi-methodology union logic and deduplication invariants that benefit from
structural enforcement.

### §10.1 ToolResolver Aggregate Root

The `ToolResolver` is a **stateless computation aggregate**. Unlike stateful aggregates
(e.g., `MarketplaceListing` with a lifecycle), `ToolResolver` loads data, enforces
invariants, and returns a computed result. It does not persist its own state.

Defined in `src/method/domain/tool_resolver.rs`:

```rust
/// Resolves the active tool set for a given set of methodology selections.
/// Enforces: deduplication, config precedence, inactive tool filtering. [S§4.2]
pub struct ToolResolver {
    /// All tool activations for the selected methodologies, loaded from DB
    activations: Vec<ToolActivationWithTool>,
    /// The primary methodology ID — used for config precedence
    primary_methodology_id: Uuid,
}

impl ToolResolver {
    /// Creates a new ToolResolver with the given activations and primary methodology.
    pub fn new(
        activations: Vec<ToolActivationWithTool>,
        primary_methodology_id: Uuid,
    ) -> Self {
        Self {
            activations,
            primary_methodology_id,
        }
    }

    /// Resolves the active tool set by applying the tool resolution algorithm:
    ///
    /// 1. Filter out inactive tools (tool.is_active == false)
    /// 2. Union all tools across selected methodologies
    /// 3. Deduplicate: if a tool appears in multiple methodologies, keep the
    ///    activation from the PRIMARY methodology. If the tool is not activated
    ///    by the primary, keep the first secondary activation encountered.
    /// 4. Sort by the winning activation's sort_order
    ///
    /// Returns Ok(resolved_tools) or Err(MethodError) if invariants are violated.
    pub fn resolve(&self) -> Result<Vec<ResolvedTool>, MethodError> {
        let mut seen: HashMap<Uuid, ResolvedTool> = HashMap::new();

        // First pass: insert all primary methodology activations
        for activation in &self.activations {
            if !activation.tool_is_active {
                continue;
            }
            if activation.methodology_id == self.primary_methodology_id {
                seen.insert(activation.tool_id, ResolvedTool::from(activation));
            }
        }

        // Second pass: insert secondary activations only if tool not already present
        for activation in &self.activations {
            if !activation.tool_is_active {
                continue;
            }
            if activation.methodology_id != self.primary_methodology_id {
                seen.entry(activation.tool_id)
                    .or_insert_with(|| ResolvedTool::from(activation));
            }
        }

        let mut tools: Vec<ResolvedTool> = seen.into_values().collect();
        tools.sort_by_key(|t| t.sort_order);
        Ok(tools)
    }
}

/// A tool with its resolved configuration (after dedup and precedence).
#[derive(Debug, Clone)]
pub struct ResolvedTool {
    pub tool_id: Uuid,
    pub slug: String,
    pub display_name: String,
    pub description: Option<String>,
    pub tier: String,
    pub config_overrides: serde_json::Value,
    pub sort_order: i16,
    /// Which methodology's activation was selected (for debugging/display)
    pub source_methodology_id: Uuid,
}

impl ResolvedTool {
    fn from(activation: &ToolActivationWithTool) -> Self {
        Self {
            tool_id: activation.tool_id,
            slug: activation.tool_slug.clone(),
            display_name: activation.tool_display_name.clone(),
            description: activation.tool_description.clone(),
            tier: activation.tool_tier.clone(),
            config_overrides: activation.config_overrides.clone(),
            sort_order: activation.sort_order,
            source_methodology_id: activation.methodology_id,
        }
    }
}
```

### §10.2 Tool Resolution Flows

**Family tool resolution** `[S§4.2]`:
1. Load family from `iam::` → get `primary_methodology_id` + `secondary_methodology_ids`
2. Collect all methodology IDs: `[primary, ...secondary]`
3. Load all tool activations for those IDs from `ToolActivationRepository::list_by_methodologies`
4. Construct `ToolResolver` with activations and primary ID
5. Call `resolver.resolve()` → returns deduplicated, precedence-resolved tool set
6. Map to `Vec<ActiveToolResponse>`

**Student tool resolution** `[S§4.6]`:
1. Load student from `iam::` → check `methodology_override_id`
2. If override exists: load tool activations for override methodology only (single methodology,
   no union needed)
3. If no override: fall through to family tool resolution (step 1-6 above)

### §10.3 MethodError Enum

Defined in `src/method/domain/errors.rs`. `[CODING §8.3]`

```rust
#[derive(Debug, thiserror::Error)]
pub enum MethodError {
    #[error("methodology not found: {slug}")]
    MethodologyNotFound { slug: String },

    #[error("methodology not found by id: {id}")]
    MethodologyNotFoundById { id: Uuid },

    #[error("methodology is not active: {slug}")]
    MethodologyNotActive { slug: String },

    #[error("invalid methodology IDs in selection: {ids:?}")]
    InvalidMethodologyIds { ids: Vec<Uuid> },

    #[error("primary methodology cannot also be a secondary")]
    PrimaryInSecondary,

    #[error("duplicate secondary methodology IDs")]
    DuplicateSecondary,

    #[error("student not found: {id}")]
    StudentNotFound { id: Uuid },

    #[error("tool not found: {slug}")]
    ToolNotFound { slug: String },

    #[error("database error")]
    DatabaseError(#[from] sea_orm::DbErr),
}
```

### §10.4 Error-to-HTTP Mapping

`MethodError` maps to `AppError` via `From<MethodError> for AppError` in the service layer
(see 00-core §6.4 for the conversion pattern). `[CODING §2.2]`

| MethodError Variant | HTTP Status | Error Code |
|--------------------|-------------|------------|
| `MethodologyNotFound` | 404 Not Found | `methodology_not_found` |
| `MethodologyNotFoundById` | 404 Not Found | `methodology_not_found` |
| `MethodologyNotActive` | 422 Unprocessable Entity | `methodology_not_active` |
| `InvalidMethodologyIds` | 422 Unprocessable Entity | `invalid_methodology_ids` |
| `PrimaryInSecondary` | 422 Unprocessable Entity | `primary_in_secondary` |
| `DuplicateSecondary` | 422 Unprocessable Entity | `duplicate_secondary` |
| `StudentNotFound` | 404 Not Found | `student_not_found` |
| `ToolNotFound` | 404 Not Found | `tool_not_found` |
| `DatabaseError` | 500 Internal Server Error | `internal_error` |

**API error responses** MUST NOT expose internal details. The error codes above are returned
as `{"error": "<code>", "message": "<user-friendly message>"}`. Internal details are logged
server-side only. `[CODING §2.2, §5.2]`

### §10.5 Module Structure

```
src/method/
├── mod.rs
├── handlers.rs       # thin: extractors → service call → response
├── service.rs        # MethodologyServiceImpl — orchestration + caching
├── repository.rs     # PgMethodologyDefinitionRepository, PgToolRepository,
│                     # PgToolActivationRepository
├── models.rs         # request/response types, internal types
├── ports.rs          # MethodologyService trait, repository traits
├── events.rs         # MethodologyConfigUpdated, FamilyMethodologyChanged
├── domain/
│   ├── mod.rs
│   ├── tool_resolver.rs   # ToolResolver aggregate root
│   └── errors.rs          # MethodError enum
└── entities/              # SeaORM-generated — never hand-edit [CODING §6.3]
```

---

## §11 Cross-Domain Interactions

### §11.1 method:: Provides (consumed by other domains)

| Export | Consumers | Mechanism |
|--------|-----------|-----------|
| `MethodologyService` trait methods | All domains | `Arc<dyn MethodologyService>` via AppState |
| `list_methodologies()` | `onboard::`, `discover::` | Service call — methodology selection wizard, quiz |
| `get_methodology()` | `discover::`, `onboard::` | Service call — methodology explorer pages, onboarding materialization (onboarding_config + community_config) `[S§6.4, S§6.6]` |
| `resolve_family_tools()` | `learn::` | Service call — determines which tools to show |
| `resolve_student_tools()` | `learn::` | Service call — student-specific tool set |
| `validate_methodology_id()` | `iam::` | Service call — validates during registration |
| `get_default_methodology_id()` | `iam::` | Service call — default for new families |
| `get_methodology_context()` | Dashboard (any domain) | Service call — terminology, mastery level |

### §11.2 method:: Consumes

| Dependency | Source | Purpose |
|-----------|--------|---------|
| Family data (methodology IDs) | `iam::IamService` | Retrieve family's selected methodology IDs for tool resolution |
| Student data (override ID) | `iam::IamService` | Retrieve student's methodology override for tool resolution |
| Family methodology persistence | `iam::FamilyRepository::set_methodology` | Persist methodology selection changes (method:: validates, iam:: persists) |
| Student methodology persistence | `iam::StudentRepository::update` | Persist student methodology override changes |

**Boundary note**: method:: *validates* methodology IDs and *orchestrates* the update flow,
but the actual `primary_methodology_id` and `secondary_methodology_ids` columns live on
`iam_families` and `iam_students` — those are IAM-owned tables. See Appendix A for the
full boundary documentation.

### §11.3 Events method:: Publishes

Defined in `src/method/events.rs`. `[CODING §8.4]`

| Event | Subscribers | Effect |
|-------|------------|--------|
| `MethodologyConfigUpdated` | All domains | Invalidate methodology config caches. Published when admin changes methodology definitions or tool activations (Phase 3+). `[ARCH §4.6]` |
| `FamilyMethodologyChanged { family_id, primary_methodology_id, secondary_methodology_ids }` | `learn::` | Recalculate family's active tool set; archive tools no longer active |
| | `social::` | Update family profile methodology display |
| | `notify::` | Send "methodology updated" notification |
| | `onboard::` | Update getting-started roadmap if in progress |

```rust
// src/method/events.rs

#[derive(Clone, Debug)]
pub struct MethodologyConfigUpdated;
impl DomainEvent for MethodologyConfigUpdated {}

#[derive(Clone, Debug)]
pub struct FamilyMethodologyChanged {
    pub family_id: FamilyId,
    /// Raw Uuid — no MethodologyId newtype exists yet. When one is introduced
    /// (likely in this domain's implementation), update these fields.
    pub primary_methodology_id: Uuid,
    pub secondary_methodology_ids: Vec<Uuid>,
}
impl DomainEvent for FamilyMethodologyChanged {}
```

### §11.4 Events method:: Subscribes To

None. The methodology domain is a provider of configuration, not a reactor to external
events. When methodology definitions change, the change originates from admin operations
within method:: itself.

---

## §12 Phase Scope

### Phase 1 — Foundation

**In scope**:
- method_ tables: `method_definitions`, `method_tools`, `method_tool_activations`
- Seed migration: 6 methodologies, 18 tools, all activation mappings
- FK migration: `iam_families.primary_methodology_id` → `method_definitions`, `iam_students.methodology_override_id` → `method_definitions`
- Public endpoints: `GET /v1/methodologies`, `GET /v1/methodologies/:slug`, `GET /v1/methodologies/:slug/tools`
- Authenticated endpoints: `GET /v1/families/tools`, `GET /v1/families/students/:id/tools`, `PATCH /v1/families/methodology`
- `MethodologyService` trait + `MethodologyServiceImpl`
- `ToolResolver` aggregate root
- `MethodError` enum + HTTP mapping
- Repository traits + PostgreSQL implementations
- Domain events: `FamilyMethodologyChanged`
- All models (request, response, internal)
- OpenAPI spec + TypeScript type generation

**Out of scope (Phase 2)**:
- `GET /v1/families/methodology-context` (dashboard context)
- `PATCH /v1/families/students/:id/methodology` (student override endpoint)
- Redis caching (add when measured latency justifies it)

**Deferred (Phase 3+)**:
- Admin CRUD endpoints for methodologies and tools
- `MethodologyConfigUpdated` event (only relevant when admin changes exist)
- Advanced philosophy module content (video, interactive elements)
- Methodology comparison features
- A/B testing for methodology recommendation weights

---

## §13 Verification Checklist

Numbered assertions that MUST be true when the method:: implementation is correct. Use these
as acceptance criteria for code review and integration testing.

1. `GET /v1/methodologies` returns exactly 6 methodologies in display_order (after seed migration)
2. `GET /v1/methodologies/charlotte-mason` returns full philosophy module as structured JSON
3. `GET /v1/methodologies/nonexistent` returns 404
4. `GET /v1/methodologies/charlotte-mason/tools` returns 10 tools with methodology-specific config overrides
5. Tool resolution for a single-methodology family returns exactly the tools activated for that methodology
6. Tool resolution for a multi-methodology family returns the **union** (no duplicates) of all activated tools
7. When a tool is activated by both primary and secondary methodologies, the **primary methodology's config overrides** are used
8. When a tool is activated by secondary but not primary, the secondary's config overrides are used
9. `PATCH /v1/families/methodology` with invalid methodology ID returns 422
10. `PATCH /v1/families/methodology` where primary ID is also in secondary list returns 422
11. `PATCH /v1/families/methodology` publishes `FamilyMethodologyChanged` event
12. Changing methodology does NOT delete any learning data (read-only archive)
13. Student tool resolution with override returns tools for the override methodology only
14. Student tool resolution without override returns the family-level tool set
15. No code under `src/method/` or any consumer branches on methodology name/slug
16. method_ tables have NO RLS policies
17. All API error responses return generic messages, not SQL or internal details
18. `validate_methodology_id` returns false for inactive methodologies
19. `get_default_methodology_id` returns the first active methodology by display_order

---

## §14 Implementation Checklist

### Phase 1 — Foundation

#### Database
- [ ] Create migration: `method_definitions` table
- [ ] Create migration: `method_tools` table
- [ ] Create migration: `method_tool_activations` table
- [ ] Create seed migration: 6 methodologies with philosophy content
- [ ] Create seed migration: 18 tools in master catalog
- [ ] Create seed migration: all tool activation mappings with config overrides
- [ ] Create FK migration: `iam_families.primary_methodology_id` → `method_definitions(id)`
- [ ] Create FK migration: `iam_students.methodology_override_id` → `method_definitions(id)`
- [ ] Regenerate SeaORM entities from migrations

#### Ports & Traits
- [ ] Define `MethodologyService` trait in `src/method/ports.rs`
- [ ] Define `MethodologyDefinitionRepository` trait in `src/method/ports.rs`
- [ ] Define `ToolRepository` trait in `src/method/ports.rs`
- [ ] Define `ToolActivationRepository` trait in `src/method/ports.rs`

#### Domain Layer
- [ ] Create `src/method/domain/mod.rs`
- [ ] Implement `ToolResolver` aggregate in `src/method/domain/tool_resolver.rs`
- [ ] Define `MethodError` enum in `src/method/domain/errors.rs`
- [ ] Implement `From<MethodError> for AppError` conversion

#### Repository Implementations
- [ ] Implement `PgMethodologyDefinitionRepository`
- [ ] Implement `PgToolRepository`
- [ ] Implement `PgToolActivationRepository`

#### Service Implementation
- [ ] Implement `MethodologyServiceImpl` with all Phase 1 methods
- [ ] Wire `MethodologyServiceImpl` in `app.rs` with `Arc<dyn MethodologyService>`

#### API Endpoints
- [ ] `GET  /v1/methodologies` — list active methodologies
- [ ] `GET  /v1/methodologies/:slug` — get methodology detail
- [ ] `GET  /v1/methodologies/:slug/tools` — list tools for methodology
- [ ] `GET  /v1/families/tools` — resolve family's active tool set
- [ ] `GET  /v1/families/students/:id/tools` — resolve student's tool set
- [ ] `PATCH /v1/families/methodology` — update family methodology selection

#### Models (DTOs)
- [ ] `UpdateMethodologyCommand` with validator derives
- [ ] `MethodologySummaryResponse` with serde + utoipa derives
- [ ] `MethodologyDetailResponse` with serde + utoipa derives
- [ ] `ActiveToolResponse` with serde + utoipa derives
- [ ] `MethodologySelectionResponse` with serde + utoipa derives
- [ ] All internal types (`MethodologyDefinition`, `Tool`, `ToolActivationWithTool`)

#### Domain Events
- [ ] Define `FamilyMethodologyChanged` event in `src/method/events.rs`
- [ ] Register event subscriptions in `app.rs`

#### Tests
- [ ] Integration test: seed migration produces 6 methodologies and 18 tools
- [ ] Integration test: `GET /v1/methodologies` returns all 6 in order
- [ ] Integration test: `GET /v1/methodologies/:slug` returns full detail
- [ ] Integration test: `GET /v1/methodologies/:slug/tools` returns correct tool set
- [ ] Unit test: `ToolResolver` dedup — same tool in primary + secondary → primary config wins
- [ ] Unit test: `ToolResolver` union — tools from multiple methodologies are combined
- [ ] Unit test: `ToolResolver` filters inactive tools
- [ ] Integration test: `PATCH /v1/families/methodology` validates IDs and publishes event
- [ ] Integration test: `PATCH /v1/families/methodology` with invalid IDs returns 422
- [ ] Integration test: student tool resolution with/without override
- [ ] Verify: no methodology name branching in `src/method/`
- [ ] Verify: `cargo clippy -- -D warnings` passes
- [ ] Verify: `cargo test` passes

#### Code Generation
- [ ] Generate OpenAPI spec from Rust types
- [ ] Generate TypeScript types from OpenAPI spec

### Phase 2 — Dashboard Context & Student Overrides

#### Endpoints
- [ ] `GET  /v1/families/methodology-context` — full methodology context
- [ ] `PATCH /v1/families/students/:id/methodology` — student override

#### Supporting Work
- [ ] Add `MethodologyContext` response type
- [ ] Add `UpdateStudentMethodologyCommand` request type
- [ ] Add Redis caching layer (if measured latency justifies it)
- [ ] Implement cache invalidation via event handlers

### Phase 3+ — Admin

#### Endpoints
- [ ] `POST  /admin/v1/methodologies` — create methodology
- [ ] `PATCH /admin/v1/methodologies/:slug` — update methodology
- [ ] `POST  /admin/v1/tools` — create tool
- [ ] `POST  /admin/v1/methodologies/:slug/tools` — activate tool for methodology

#### Supporting Work
- [ ] Define `MethodologyConfigUpdated` event
- [ ] Implement admin authentication/authorization
- [ ] Add audit logging for methodology config changes

---

## Appendix A — IAM / method:: Boundary

### What lives where

| Data | Owner | Table | Reason |
|------|-------|-------|--------|
| Methodology definitions | `method::` | `method_definitions` | Platform config |
| Tool catalog | `method::` | `method_tools` | Platform config |
| Tool activations | `method::` | `method_tool_activations` | Platform config |
| Family's selected methodology IDs | `iam::` | `iam_families.primary_methodology_id`, `iam_families.secondary_methodology_ids` | Family identity data |
| Student's methodology override | `iam::` | `iam_students.methodology_override_id` | Student identity data |

### Update flows

**Family methodology change** (`PATCH /v1/families/methodology`):
1. Request arrives at `method::handlers`
2. Handler calls `MethodologyService::update_family_methodology`
3. Service validates all methodology IDs exist and are active (queries `method_definitions`)
4. Service validates primary is not in secondary list
5. Service calls `IamService` → which calls `FamilyRepository::set_methodology` (IAM persists)
6. Service publishes `FamilyMethodologyChanged` event
7. Handler returns `MethodologySelectionResponse`

**Why method:: owns the endpoint but iam:: owns the column**: method:: has the domain
knowledge to validate methodology IDs (does it exist? is it active?). `iam::` has no
knowledge of methodology definitions — it just stores the FK. Putting the endpoint in
`iam::` would require `iam::` to import `method::` repository traits, creating a circular
dependency risk. Instead, `method::` calls `iam::` service (which is already a dependency
direction that exists — `method::` depends on `iam::`, not vice versa).

**Student methodology override** (`PATCH /v1/families/students/:id/methodology`, Phase 2):
Same pattern — `method::` validates the methodology ID, then calls
`IamService::update_student` to persist the change on the `iam_students` row.
