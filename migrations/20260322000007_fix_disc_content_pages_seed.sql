-- +goose Up
-- Fix: replace wrong content page slugs from migration 005 with spec-defined slugs.
-- Fix: replace wrong index on disc_content_pages with spec-correct partial index.
-- [03-discover §3.1, §3.2]

-- Remove wrong slugs seeded in migration 005.
DELETE FROM disc_content_pages
WHERE slug IN (
    '101-getting-started',
    '101-legal-requirements',
    '101-choosing-curriculum',
    '101-daily-scheduling',
    '101-record-keeping',
    '101-socialization',
    '101-special-needs',
    '101-transitioning-to-college',
    'advocacy-homeschool-rights',
    'advocacy-legislative-updates'
);

-- Insert spec-defined slugs with correct display_order. [03-discover §3.2]
-- These slugs become Astro URL paths: /101/:slug and /why/:slug.
INSERT INTO disc_content_pages (slug, category, title, status, display_order) VALUES
    ('socialization',          '101',      'What About Socialization?',                    'draft', 1),
    ('cost',                   '101',      'How Much Does Homeschooling Cost?',             'draft', 2),
    ('time-commitment',        '101',      'How Much Time Does Homeschooling Take?',        'draft', 3),
    ('dual-income',            '101',      'Can Dual-Income Families Homeschool?',          'draft', 4),
    ('special-needs',          '101',      'Homeschooling Children with Special Needs',     'draft', 5),
    ('transitioning-mid-year', '101',      'Transitioning from Public School Mid-Year',     'draft', 6),
    ('skeptical-relatives',    '101',      'How to Talk to Skeptical Relatives',            'draft', 7),
    ('getting-started',        '101',      'How to Start Homeschooling: A Complete Guide',  'draft', 8),
    ('case-for-homeschooling', 'advocacy', 'The Case for Homeschooling',                   'draft', 1),
    ('why-families-choose',    'advocacy', 'Why Families Choose Homeschooling',             'draft', 2);

-- Fix index: drop wrong index from migration 004, create spec-correct partial index.
-- Old: ON (category, status) — misses display_order, no WHERE filter
-- New: ON (category, display_order) WHERE status = 'published' [03-discover §3.1]
DROP INDEX IF EXISTS idx_disc_content_pages_category;
CREATE INDEX idx_disc_content_pages_category
    ON disc_content_pages(category, display_order)
    WHERE status = 'published';

-- +goose Down
-- Restore old index.
DROP INDEX IF EXISTS idx_disc_content_pages_category;
CREATE INDEX idx_disc_content_pages_category
    ON disc_content_pages(category, status);

-- Restore old slugs. At this point migration 006 is still applied, so body column
-- does NOT exist — insert without it. display_order defaults to 0.
DELETE FROM disc_content_pages
WHERE slug IN (
    'socialization', 'cost', 'time-commitment', 'dual-income', 'special-needs',
    'transitioning-mid-year', 'skeptical-relatives', 'getting-started',
    'case-for-homeschooling', 'why-families-choose'
);

INSERT INTO disc_content_pages (slug, category, title, status) VALUES
    ('101-getting-started',          '101',      'Getting Started with Homeschooling', 'draft'),
    ('101-legal-requirements',       '101',      'Understanding Legal Requirements',   'draft'),
    ('101-choosing-curriculum',      '101',      'Choosing Your Curriculum',           'draft'),
    ('101-daily-scheduling',         '101',      'Building a Daily Schedule',          'draft'),
    ('101-record-keeping',           '101',      'Record Keeping and Documentation',   'draft'),
    ('101-socialization',            '101',      'Socialization and Community',        'draft'),
    ('101-special-needs',            '101',      'Homeschooling with Special Needs',   'draft'),
    ('101-transitioning-to-college', '101',      'Transitioning to College',           'draft'),
    ('advocacy-homeschool-rights',   'advocacy', 'Know Your Homeschooling Rights',     'draft'),
    ('advocacy-legislative-updates', 'advocacy', 'Legislative Updates & Action',       'draft');
