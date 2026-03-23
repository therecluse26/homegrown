-- +goose Up
-- Migration: 20260322000003_seed_method_data.sql
-- Seeds methodology definitions, master tool catalog, tool activation mappings,
-- and adds deferred FK constraints from IAM tables. [02-method §3.3]

-- =============================================================================
-- 6 initial methodologies [S§4.5]
-- =============================================================================
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

-- =============================================================================
-- FK migration: add foreign keys from IAM tables to method_definitions
-- (Deferred from IAM migration — method_definitions must exist first)
-- =============================================================================
ALTER TABLE iam_families
    ADD CONSTRAINT fk_iam_families_primary_methodology
    FOREIGN KEY (primary_methodology_slug) REFERENCES method_definitions(slug);

ALTER TABLE iam_students
    ADD CONSTRAINT fk_iam_students_methodology_override
    FOREIGN KEY (methodology_override_slug) REFERENCES method_definitions(slug);

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

-- Interactive learning tools [S§8.1.9-8.1.12]
INSERT INTO method_tools (slug, display_name, description, tier)
VALUES
    ('assessment-engine',   'Assessment Engine',       'Take quizzes and tests online with auto-scoring and parent-scored short answers',      'free'),
    ('content-viewer',      'Content Viewer',          'View PDFs and documents in-platform with page tracking and progress',                 'free'),
    ('video-player',        'Video Player',            'Watch video lessons with adaptive streaming, resume, and completion tracking',          'free'),
    ('lesson-sequences',    'Lesson Sequences',        'Follow structured lesson paths combining readings, videos, quizzes, and activities',   'free');

-- =============================================================================
-- Tool activation mappings [S§4.2]
-- config_overrides carry methodology-specific labels, guidance text, and entry types
-- Direct slug values — no subqueries needed since slug IS the PK. [02-method ADR]
-- =============================================================================

-- Charlotte Mason activations
INSERT INTO method_tool_activations (methodology_slug, tool_slug, config_overrides, sort_order)
VALUES
    ('charlotte-mason', 'activities',
     '{"label": "Lessons", "guidance": "Charlotte Mason recommended short, focused lessons — 15 minutes for young children, up to 45 minutes for older students. Full attention is expected for the duration."}',
     1),
    ('charlotte-mason', 'reading-lists',
     '{"label": "Living Books", "guidance": "Choose ''living books'' — books written by a single author passionate about the subject, with literary quality and narrative power. Avoid twaddle (dumbed-down or overly simplified texts)."}',
     2),
    ('charlotte-mason', 'journaling',
     '{"label": "Narration", "guidance": "After reading a passage, ask the child to tell back what they heard in their own words. Written narration begins around age 10. This is the primary method of retention.", "entry_types": ["narration", "written_narration", "free_journal"]}',
     3),
    ('charlotte-mason', 'nature-journals',
     '{"label": "Nature Journal", "guidance": "Spend regular time outdoors observing nature. Children draw what they see with careful attention to detail, adding notes about species, weather, and season."}',
     4),
    ('charlotte-mason', 'habit-tracking',
     '{"label": "Habit Training", "guidance": "Charlotte Mason considered habit training the foundation of education. Focus on one habit at a time for 4-6 weeks before adding another."}',
     5),
    ('charlotte-mason', 'progress-tracking',
     '{"label": "Progress"}',
     6),
    ('charlotte-mason', 'unit-studies',
     '{"label": "Unit Studies", "guidance": "While not strictly CM, thematic studies complement living books by connecting subjects around historical periods or nature topics."}',
     7),
    ('charlotte-mason', 'field-trip-log',
     '{"label": "Field Trip Log"}',
     8),
    ('charlotte-mason', 'lesson-planner',
     '{"label": "Lesson Planner", "guidance": "Plan short, varied lessons across subjects. CM schedules typically alternate between demanding and lighter subjects."}',
     9);

-- Traditional activations
INSERT INTO method_tool_activations (methodology_slug, tool_slug, config_overrides, sort_order)
VALUES
    ('traditional', 'activities',
     '{"label": "Lessons", "guidance": "Follow your curriculum''s lesson plans. Log each subject session with the textbook/workbook pages covered."}',
     1),
    ('traditional', 'tests-grades',
     '{"label": "Tests & Grades", "guidance": "Record quiz and test scores regularly. Use the grading scale that matches your curriculum or state requirements."}',
     2),
    ('traditional', 'reading-lists',
     '{"label": "Reading Lists"}',
     3),
    ('traditional', 'progress-tracking',
     '{"label": "Progress & Grades"}',
     4),
    ('traditional', 'unit-studies',
     '{"label": "Unit Studies", "guidance": "Supplement textbook work with thematic unit studies to bring subjects to life."}',
     5),
    ('traditional', 'field-trip-log',
     '{"label": "Field Trip Log"}',
     6),
    ('traditional', 'lesson-planner',
     '{"label": "Lesson Planner", "guidance": "Plan your week by subject and textbook chapter. Track pages assigned vs. completed."}',
     7);

-- Classical activations
INSERT INTO method_tool_activations (methodology_slug, tool_slug, config_overrides, sort_order)
VALUES
    ('classical', 'activities',
     '{"label": "Lessons", "guidance": "Adapt lesson structure to the student''s Trivium stage: memory-heavy for Grammar, discussion-based for Logic, presentation-focused for Rhetoric."}',
     1),
    ('classical', 'tests-grades',
     '{"label": "Assessments", "guidance": "Grammar stage: fact recall and recitation. Logic stage: analysis and essay. Rhetoric stage: thesis defense and presentation."}',
     2),
    ('classical', 'reading-lists',
     '{"label": "Great Books", "guidance": "Build lists from the Great Books canon appropriate to the student''s stage. Include primary sources alongside secondary readings."}',
     3),
    ('classical', 'journaling',
     '{"label": "Composition", "guidance": "Grammar stage: copywork and dictation. Logic stage: outlines and essays. Rhetoric stage: research papers and persuasive writing.", "entry_types": ["copywork", "dictation", "essay", "research_paper"]}',
     4),
    ('classical', 'trivium-tracker',
     '{"label": "Trivium Progress", "guidance": "Track each subject through the three stages. Students may be at different stages for different subjects."}',
     5),
    ('classical', 'progress-tracking',
     '{"label": "Academic Progress"}',
     6),
    ('classical', 'unit-studies',
     '{"label": "Unit Studies", "guidance": "Use unit studies to integrate history cycles with literature, writing, and geography."}',
     7),
    ('classical', 'field-trip-log',
     '{"label": "Field Trip Log"}',
     8),
    ('classical', 'lesson-planner',
     '{"label": "Lesson Planner", "guidance": "Plan lessons around the 4-year history cycle and Trivium stage. Include memory work, reading assignments, and writing exercises."}',
     9);

-- Waldorf activations
INSERT INTO method_tool_activations (methodology_slug, tool_slug, config_overrides, sort_order)
VALUES
    ('waldorf', 'activities',
     '{"label": "Main Lessons", "guidance": "Main lesson blocks last 3-4 weeks, diving deep into one subject each morning for ~2 hours. Integrate artistic elements into every lesson."}',
     1),
    ('waldorf', 'journaling',
     '{"label": "Main Lesson Book", "guidance": "The main lesson book is the child''s own textbook — created through beautiful illustrations, summaries, and artistic renderings of the material.", "entry_types": ["main_lesson_page", "illustration", "summary"]}',
     2),
    ('waldorf', 'projects',
     '{"label": "Handwork & Projects", "guidance": "Waldorf handwork progresses through developmental stages: finger knitting → knitting → crocheting → sewing → woodworking."}',
     3),
    ('waldorf', 'rhythm-planner',
     '{"label": "Rhythm Planner", "guidance": "Waldorf education is built on rhythm — daily, weekly, and seasonal. Plan blocks for main lessons, artistic activities, practical work, and free play."}',
     4),
    ('waldorf', 'handwork-tracker',
     '{"label": "Handwork Projects", "guidance": "Track handwork projects with materials, techniques, and photos. Each project builds on skills from previous work."}',
     5),
    ('waldorf', 'progress-tracking',
     '{"label": "Development Progress"}',
     6),
    ('waldorf', 'unit-studies',
     '{"label": "Block Studies", "guidance": "Waldorf main lesson blocks are similar to unit studies — deep, immersive explorations of a single subject area."}',
     7),
    ('waldorf', 'field-trip-log',
     '{"label": "Field Trip Log"}',
     8),
    ('waldorf', 'lesson-planner',
     '{"label": "Block Planner", "guidance": "Plan 3-4 week main lesson blocks with daily rhythm. Include morning circle, main lesson, practice period, and artistic activities."}',
     9);

-- Montessori activations
INSERT INTO method_tool_activations (methodology_slug, tool_slug, config_overrides, sort_order)
VALUES
    ('montessori', 'activities',
     '{"label": "Work", "guidance": "In Montessori, activities are called ''work.'' Children choose their work from the prepared environment. Log the work chosen, materials used, and duration."}',
     1),
    ('montessori', 'projects',
     '{"label": "Investigations", "guidance": "Child-initiated deep dives into topics of interest. Follow the child''s questions and provide resources and materials to explore further."}',
     2),
    ('montessori', 'observation-logs',
     '{"label": "Observation Logs", "guidance": "Observe the child at work: What did they choose? How long did they concentrate? Did they repeat the activity? What social interactions occurred? Use these observations to guide environment preparation."}',
     3),
    ('montessori', 'practical-life',
     '{"label": "Practical Life", "guidance": "Practical life activities are the foundation of Montessori education. Include care of self, care of environment, grace and courtesy, and control of movement."}',
     4),
    ('montessori', 'progress-tracking',
     '{"label": "Development Progress"}',
     5),
    ('montessori', 'unit-studies',
     '{"label": "Cosmic Education", "guidance": "Montessori cosmic education connects all subjects through the Great Lessons, showing children how everything in the universe is interconnected."}',
     6),
    ('montessori', 'field-trip-log',
     '{"label": "Field Trip Log"}',
     7);

-- Unschooling activations
INSERT INTO method_tool_activations (methodology_slug, tool_slug, config_overrides, sort_order)
VALUES
    ('unschooling', 'activities',
     '{"label": "Explorations", "guidance": "Document what your child naturally gravitates toward. Everything counts — cooking, building, gaming, conversations, outings. Learning is living."}',
     1),
    ('unschooling', 'journaling',
     '{"label": "Reflections", "guidance": "Use journaling to reflect on and document your child''s learning journey. This is primarily a parent tool for recognizing learning as it happens.", "entry_types": ["observation", "reflection", "documentation"]}',
     2),
    ('unschooling', 'projects',
     '{"label": "Interest Projects", "guidance": "When a child dives deep into a passion, help them structure it into a project. These emerge naturally from sustained interest, not from assignment."}',
     3),
    ('unschooling', 'interest-led-logs',
     '{"label": "Interest-Led Learning", "guidance": "Document child-initiated activities and the learning within them. The auto-tagging helps map natural learning to academic subjects for compliance purposes."}',
     4),
    ('unschooling', 'progress-tracking',
     '{"label": "Learning Documentation"}',
     5),
    ('unschooling', 'field-trip-log',
     '{"label": "Adventures", "guidance": "Document outings, museum visits, nature explorations, community events — all the ways your family engages with the world."}',
     6);

-- Video Lessons — activated for all methodologies [S§8.1.6]
INSERT INTO method_tool_activations (methodology_slug, tool_slug, config_overrides, sort_order)
SELECT md.slug, 'video-lessons', '{"label": "Video Lessons"}', 99
FROM method_definitions md
WHERE md.is_active = true;

-- Content Viewer — activated for ALL methodologies [S§8.1.10]
INSERT INTO method_tool_activations (methodology_slug, tool_slug, config_overrides, sort_order)
SELECT md.slug, 'content-viewer', '{"label": "Content Viewer"}', 100
FROM method_definitions md
WHERE md.is_active = true;

-- Video Player — activated for ALL methodologies [S§8.1.11]
INSERT INTO method_tool_activations (methodology_slug, tool_slug, config_overrides, sort_order)
SELECT md.slug, 'video-player', '{"label": "Video Player"}', 101
FROM method_definitions md
WHERE md.is_active = true;

-- Assessment Engine — methodology-selective [S§8.1.9]
INSERT INTO method_tool_activations (methodology_slug, tool_slug, config_overrides, sort_order)
VALUES
    ('traditional', 'assessment-engine',
     '{"label": "Online Quizzes", "guidance": "Create and administer quizzes online. Students take tests on the platform with auto-scoring for objective questions. Scores flow directly into grade tracking."}',
     10),
    ('classical', 'assessment-engine',
     '{"label": "Assessments", "guidance": "Grammar stage: fact recall quizzes. Logic stage: analytical questions and matching exercises. Rhetoric stage: short-answer assessments scored by parent."}',
     10),
    ('montessori', 'assessment-engine',
     '{"label": "Knowledge Checks", "guidance": "Montessori primarily uses observation-based assessment. Online quizzes are an optional supplement — use sparingly and focus on self-assessment rather than grading.", "optional": true}',
     10);

-- Lesson Sequences — methodology-selective [S§8.1.12]
INSERT INTO method_tool_activations (methodology_slug, tool_slug, config_overrides, sort_order)
VALUES
    ('charlotte-mason', 'lesson-sequences',
     '{"label": "Lesson Paths", "guidance": "Gentle, living-books-based lesson sequences. Follow the sequence at the child''s pace — short lessons with full attention, then move on."}',
     11),
    ('traditional', 'lesson-sequences',
     '{"label": "Lesson Sequences", "guidance": "Structured lesson paths following curriculum order. Students complete each step before advancing to the next."}',
     11),
    ('classical', 'lesson-sequences',
     '{"label": "Lesson Sequences", "guidance": "Structured lesson paths aligned to Trivium stages. Sequences may include reading, discussion prompts, memory work, and assessments."}',
     11),
    ('waldorf', 'lesson-sequences',
     '{"label": "Block Sequences", "guidance": "Main lesson block sequences following Waldorf rhythmic structure. Each block dives deep into one subject over 3-4 weeks."}',
     11),
    ('montessori', 'lesson-sequences',
     '{"label": "Guided Paths", "guidance": "Suggested work sequences that follow the Montessori scope and sequence. The child may choose to follow or diverge based on their interests."}',
     11);

-- +goose Down
-- Remove FK constraints first (reverse order)
ALTER TABLE iam_students DROP CONSTRAINT IF EXISTS fk_iam_students_methodology_override;
ALTER TABLE iam_families DROP CONSTRAINT IF EXISTS fk_iam_families_primary_methodology;

-- Seed data removed when tables are dropped in 20260322000002
