-- +goose Up
-- Seed the 3-level subject taxonomy. [06-learn §13]
-- Level 0 = Category, Level 1 = Subject, Level 2 = Topic

-- ═══ Mathematics ═══════════════════════════════════════════════════════════════
INSERT INTO learn_subject_taxonomy (id, parent_id, name, slug, level, display_order) VALUES
    (uuidv7(), NULL, 'Mathematics', 'math', 0, 1);

INSERT INTO learn_subject_taxonomy (id, parent_id, name, slug, level, display_order)
SELECT uuidv7(), t.id, s.name, s.slug, s.level, s.display_order
FROM learn_subject_taxonomy t
CROSS JOIN (VALUES
    ('Algebra',           'math.algebra',           1, 1),
    ('Geometry',          'math.geometry',           1, 2),
    ('Calculus',          'math.calculus',            1, 3),
    ('Statistics',        'math.statistics',          1, 4),
    ('Arithmetic',        'math.arithmetic',          1, 5),
    ('Trigonometry',      'math.trigonometry',         1, 6)
) AS s(name, slug, level, display_order)
WHERE t.slug = 'math';

INSERT INTO learn_subject_taxonomy (id, parent_id, name, slug, level, display_order)
SELECT uuidv7(), t.id, s.name, s.slug, s.level, s.display_order
FROM learn_subject_taxonomy t
CROSS JOIN (VALUES
    ('Linear Equations',     'math.algebra.linear-equations',     2, 1),
    ('Quadratic Equations',  'math.algebra.quadratic-equations',  2, 2),
    ('Polynomials',          'math.algebra.polynomials',           2, 3)
) AS s(name, slug, level, display_order)
WHERE t.slug = 'math.algebra';

INSERT INTO learn_subject_taxonomy (id, parent_id, name, slug, level, display_order)
SELECT uuidv7(), t.id, s.name, s.slug, s.level, s.display_order
FROM learn_subject_taxonomy t
CROSS JOIN (VALUES
    ('Triangles',   'math.geometry.triangles',   2, 1),
    ('Circles',     'math.geometry.circles',     2, 2),
    ('Polygons',    'math.geometry.polygons',    2, 3)
) AS s(name, slug, level, display_order)
WHERE t.slug = 'math.geometry';

-- ═══ Language Arts ═════════════════════════════════════════════════════════════
INSERT INTO learn_subject_taxonomy (id, parent_id, name, slug, level, display_order) VALUES
    (uuidv7(), NULL, 'Language Arts', 'language-arts', 0, 2);

INSERT INTO learn_subject_taxonomy (id, parent_id, name, slug, level, display_order)
SELECT uuidv7(), t.id, s.name, s.slug, s.level, s.display_order
FROM learn_subject_taxonomy t
CROSS JOIN (VALUES
    ('Writing',               'language-arts.writing',               1, 1),
    ('Reading Comprehension', 'language-arts.reading-comprehension', 1, 2),
    ('Grammar',               'language-arts.grammar',               1, 3),
    ('Vocabulary',            'language-arts.vocabulary',             1, 4),
    ('Literature',            'language-arts.literature',             1, 5),
    ('Spelling',              'language-arts.spelling',               1, 6)
) AS s(name, slug, level, display_order)
WHERE t.slug = 'language-arts';

INSERT INTO learn_subject_taxonomy (id, parent_id, name, slug, level, display_order)
SELECT uuidv7(), t.id, s.name, s.slug, s.level, s.display_order
FROM learn_subject_taxonomy t
CROSS JOIN (VALUES
    ('Creative Writing', 'language-arts.writing.creative',  2, 1),
    ('Essay Writing',    'language-arts.writing.essay',     2, 2),
    ('Poetry',           'language-arts.writing.poetry',    2, 3)
) AS s(name, slug, level, display_order)
WHERE t.slug = 'language-arts.writing';

-- ═══ Science ═══════════════════════════════════════════════════════════════════
INSERT INTO learn_subject_taxonomy (id, parent_id, name, slug, level, display_order) VALUES
    (uuidv7(), NULL, 'Science', 'science', 0, 3);

INSERT INTO learn_subject_taxonomy (id, parent_id, name, slug, level, display_order)
SELECT uuidv7(), t.id, s.name, s.slug, s.level, s.display_order
FROM learn_subject_taxonomy t
CROSS JOIN (VALUES
    ('Biology',     'science.biology',     1, 1),
    ('Chemistry',   'science.chemistry',   1, 2),
    ('Physics',     'science.physics',     1, 3),
    ('Earth Science','science.earth',      1, 4),
    ('Astronomy',   'science.astronomy',   1, 5)
) AS s(name, slug, level, display_order)
WHERE t.slug = 'science';

INSERT INTO learn_subject_taxonomy (id, parent_id, name, slug, level, display_order)
SELECT uuidv7(), t.id, s.name, s.slug, s.level, s.display_order
FROM learn_subject_taxonomy t
CROSS JOIN (VALUES
    ('Botany',   'science.biology.botany',   2, 1),
    ('Zoology',  'science.biology.zoology',  2, 2),
    ('Anatomy',  'science.biology.anatomy',  2, 3),
    ('Ecology',  'science.biology.ecology',  2, 4)
) AS s(name, slug, level, display_order)
WHERE t.slug = 'science.biology';

-- ═══ History ═══════════════════════════════════════════════════════════════════
INSERT INTO learn_subject_taxonomy (id, parent_id, name, slug, level, display_order) VALUES
    (uuidv7(), NULL, 'History', 'history', 0, 4);

INSERT INTO learn_subject_taxonomy (id, parent_id, name, slug, level, display_order)
SELECT uuidv7(), t.id, s.name, s.slug, s.level, s.display_order
FROM learn_subject_taxonomy t
CROSS JOIN (VALUES
    ('Ancient History',   'history.ancient',   1, 1),
    ('Medieval History',  'history.medieval',  1, 2),
    ('Modern History',    'history.modern',    1, 3),
    ('American History',  'history.american',  1, 4),
    ('World History',     'history.world',     1, 5)
) AS s(name, slug, level, display_order)
WHERE t.slug = 'history';

-- ═══ Social Studies ════════════════════════════════════════════════════════════
INSERT INTO learn_subject_taxonomy (id, parent_id, name, slug, level, display_order) VALUES
    (uuidv7(), NULL, 'Social Studies', 'social-studies', 0, 5);

INSERT INTO learn_subject_taxonomy (id, parent_id, name, slug, level, display_order)
SELECT uuidv7(), t.id, s.name, s.slug, s.level, s.display_order
FROM learn_subject_taxonomy t
CROSS JOIN (VALUES
    ('Geography',    'social-studies.geography',    1, 1),
    ('Economics',    'social-studies.economics',    1, 2),
    ('Government',   'social-studies.government',   1, 3),
    ('Civics',       'social-studies.civics',       1, 4)
) AS s(name, slug, level, display_order)
WHERE t.slug = 'social-studies';

-- ═══ Arts ══════════════════════════════════════════════════════════════════════
INSERT INTO learn_subject_taxonomy (id, parent_id, name, slug, level, display_order) VALUES
    (uuidv7(), NULL, 'Arts', 'arts', 0, 6);

INSERT INTO learn_subject_taxonomy (id, parent_id, name, slug, level, display_order)
SELECT uuidv7(), t.id, s.name, s.slug, s.level, s.display_order
FROM learn_subject_taxonomy t
CROSS JOIN (VALUES
    ('Visual Arts',     'arts.visual',       1, 1),
    ('Music',           'arts.music',        1, 2),
    ('Theater',         'arts.theater',      1, 3),
    ('Dance',           'arts.dance',        1, 4)
) AS s(name, slug, level, display_order)
WHERE t.slug = 'arts';

INSERT INTO learn_subject_taxonomy (id, parent_id, name, slug, level, display_order)
SELECT uuidv7(), t.id, s.name, s.slug, s.level, s.display_order
FROM learn_subject_taxonomy t
CROSS JOIN (VALUES
    ('Drawing',    'arts.visual.drawing',    2, 1),
    ('Painting',   'arts.visual.painting',   2, 2),
    ('Sculpture',  'arts.visual.sculpture',  2, 3)
) AS s(name, slug, level, display_order)
WHERE t.slug = 'arts.visual';

-- ═══ Foreign Languages ════════════════════════════════════════════════════════
INSERT INTO learn_subject_taxonomy (id, parent_id, name, slug, level, display_order) VALUES
    (uuidv7(), NULL, 'Foreign Languages', 'foreign-languages', 0, 7);

INSERT INTO learn_subject_taxonomy (id, parent_id, name, slug, level, display_order)
SELECT uuidv7(), t.id, s.name, s.slug, s.level, s.display_order
FROM learn_subject_taxonomy t
CROSS JOIN (VALUES
    ('Spanish',  'foreign-languages.spanish',  1, 1),
    ('French',   'foreign-languages.french',   1, 2),
    ('Latin',    'foreign-languages.latin',    1, 3),
    ('German',   'foreign-languages.german',   1, 4),
    ('Mandarin', 'foreign-languages.mandarin', 1, 5)
) AS s(name, slug, level, display_order)
WHERE t.slug = 'foreign-languages';

-- ═══ Physical Education ═══════════════════════════════════════════════════════
INSERT INTO learn_subject_taxonomy (id, parent_id, name, slug, level, display_order) VALUES
    (uuidv7(), NULL, 'Physical Education', 'physical-education', 0, 8);

INSERT INTO learn_subject_taxonomy (id, parent_id, name, slug, level, display_order)
SELECT uuidv7(), t.id, s.name, s.slug, s.level, s.display_order
FROM learn_subject_taxonomy t
CROSS JOIN (VALUES
    ('Sports',     'physical-education.sports',     1, 1),
    ('Fitness',    'physical-education.fitness',    1, 2),
    ('Health',     'physical-education.health',     1, 3)
) AS s(name, slug, level, display_order)
WHERE t.slug = 'physical-education';

-- ═══ Life Skills ══════════════════════════════════════════════════════════════
INSERT INTO learn_subject_taxonomy (id, parent_id, name, slug, level, display_order) VALUES
    (uuidv7(), NULL, 'Life Skills', 'life-skills', 0, 9);

INSERT INTO learn_subject_taxonomy (id, parent_id, name, slug, level, display_order)
SELECT uuidv7(), t.id, s.name, s.slug, s.level, s.display_order
FROM learn_subject_taxonomy t
CROSS JOIN (VALUES
    ('Cooking',          'life-skills.cooking',         1, 1),
    ('Financial Literacy','life-skills.financial',      1, 2),
    ('Home Management',  'life-skills.home-management', 1, 3),
    ('First Aid',        'life-skills.first-aid',       1, 4)
) AS s(name, slug, level, display_order)
WHERE t.slug = 'life-skills';

-- ═══ Technology ════════════════════════════════════════════════════════════════
INSERT INTO learn_subject_taxonomy (id, parent_id, name, slug, level, display_order) VALUES
    (uuidv7(), NULL, 'Technology', 'technology', 0, 10);

INSERT INTO learn_subject_taxonomy (id, parent_id, name, slug, level, display_order)
SELECT uuidv7(), t.id, s.name, s.slug, s.level, s.display_order
FROM learn_subject_taxonomy t
CROSS JOIN (VALUES
    ('Computer Science',  'technology.cs',       1, 1),
    ('Digital Literacy',  'technology.digital',  1, 2),
    ('Robotics',          'technology.robotics', 1, 3)
) AS s(name, slug, level, display_order)
WHERE t.slug = 'technology';

-- ═══ Religious Studies ════════════════════════════════════════════════════════
INSERT INTO learn_subject_taxonomy (id, parent_id, name, slug, level, display_order) VALUES
    (uuidv7(), NULL, 'Religious Studies', 'religion', 0, 11);

INSERT INTO learn_subject_taxonomy (id, parent_id, name, slug, level, display_order)
SELECT uuidv7(), t.id, s.name, s.slug, s.level, s.display_order
FROM learn_subject_taxonomy t
CROSS JOIN (VALUES
    ('Bible Study',       'religion.bible',       1, 1),
    ('Theology',          'religion.theology',    1, 2),
    ('Church History',    'religion.history',     1, 3),
    ('Character Education','religion.character',  1, 4)
) AS s(name, slug, level, display_order)
WHERE t.slug = 'religion';

-- +goose Down
DELETE FROM learn_subject_taxonomy;
