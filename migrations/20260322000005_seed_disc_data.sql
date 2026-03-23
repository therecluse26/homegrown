-- +goose Up
-- Migration: 20260322000005_seed_disc_data.sql
-- Seeds 51 state guide rows (50 states + DC), Homeschooling 101 content page stubs,
-- and advocacy content page stubs. All state guides seeded as status='draft'. [03-discover §3.2]
-- Content pages seeded as status='draft' — published via admin tooling. [03-discover §3.3]

INSERT INTO disc_state_guides (state_code, state_name, status, requirements)
VALUES
    ('AL', 'Alabama',              'draft', '{}'),
    ('AK', 'Alaska',               'draft', '{}'),
    ('AZ', 'Arizona',              'draft', '{}'),
    ('AR', 'Arkansas',             'draft', '{}'),
    ('CA', 'California',           'draft', '{}'),
    ('CO', 'Colorado',             'draft', '{}'),
    ('CT', 'Connecticut',          'draft', '{}'),
    ('DE', 'Delaware',             'draft', '{}'),
    ('FL', 'Florida',              'draft', '{}'),
    ('GA', 'Georgia',              'draft', '{}'),
    ('HI', 'Hawaii',               'draft', '{}'),
    ('ID', 'Idaho',                'draft', '{}'),
    ('IL', 'Illinois',             'draft', '{}'),
    ('IN', 'Indiana',              'draft', '{}'),
    ('IA', 'Iowa',                 'draft', '{}'),
    ('KS', 'Kansas',               'draft', '{}'),
    ('KY', 'Kentucky',             'draft', '{}'),
    ('LA', 'Louisiana',            'draft', '{}'),
    ('ME', 'Maine',                'draft', '{}'),
    ('MD', 'Maryland',             'draft', '{}'),
    ('MA', 'Massachusetts',        'draft', '{}'),
    ('MI', 'Michigan',             'draft', '{}'),
    ('MN', 'Minnesota',            'draft', '{}'),
    ('MS', 'Mississippi',          'draft', '{}'),
    ('MO', 'Missouri',             'draft', '{}'),
    ('MT', 'Montana',              'draft', '{}'),
    ('NE', 'Nebraska',             'draft', '{}'),
    ('NV', 'Nevada',               'draft', '{}'),
    ('NH', 'New Hampshire',        'draft', '{}'),
    ('NJ', 'New Jersey',           'draft', '{}'),
    ('NM', 'New Mexico',           'draft', '{}'),
    ('NY', 'New York',             'draft', '{}'),
    ('NC', 'North Carolina',       'draft', '{}'),
    ('ND', 'North Dakota',         'draft', '{}'),
    ('OH', 'Ohio',                 'draft', '{}'),
    ('OK', 'Oklahoma',             'draft', '{}'),
    ('OR', 'Oregon',               'draft', '{}'),
    ('PA', 'Pennsylvania',         'draft', '{}'),
    ('RI', 'Rhode Island',         'draft', '{}'),
    ('SC', 'South Carolina',       'draft', '{}'),
    ('SD', 'South Dakota',         'draft', '{}'),
    ('TN', 'Tennessee',            'draft', '{}'),
    ('TX', 'Texas',                'draft', '{}'),
    ('UT', 'Utah',                 'draft', '{}'),
    ('VT', 'Vermont',              'draft', '{}'),
    ('VA', 'Virginia',             'draft', '{}'),
    ('WA', 'Washington',           'draft', '{}'),
    ('WV', 'West Virginia',        'draft', '{}'),
    ('WI', 'Wisconsin',            'draft', '{}'),
    ('WY', 'Wyoming',              'draft', '{}'),
    ('DC', 'District of Columbia', 'draft', '{}');

-- Homeschooling 101 content page stubs (category='101'). [03-discover §3.3]
INSERT INTO disc_content_pages (slug, category, title, status, body)
VALUES
    ('101-getting-started',           '101',      'Getting Started with Homeschooling', 'draft', '{}'),
    ('101-legal-requirements',        '101',      'Understanding Legal Requirements',   'draft', '{}'),
    ('101-choosing-curriculum',       '101',      'Choosing Your Curriculum',           'draft', '{}'),
    ('101-daily-scheduling',          '101',      'Building a Daily Schedule',          'draft', '{}'),
    ('101-record-keeping',            '101',      'Record Keeping and Documentation',   'draft', '{}'),
    ('101-socialization',             '101',      'Socialization and Community',         'draft', '{}'),
    ('101-special-needs',             '101',      'Homeschooling with Special Needs',   'draft', '{}'),
    ('101-transitioning-to-college',  '101',      'Transitioning to College',           'draft', '{}'),
    ('advocacy-homeschool-rights',    'advocacy', 'Know Your Homeschooling Rights',     'draft', '{}'),
    ('advocacy-legislative-updates',  'advocacy', 'Legislative Updates & Action',       'draft', '{}');

-- +goose Down
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

DELETE FROM disc_state_guides
WHERE state_code IN (
    'AL','AK','AZ','AR','CA','CO','CT','DE','FL','GA',
    'HI','ID','IL','IN','IA','KS','KY','LA','ME','MD',
    'MA','MI','MN','MS','MO','MT','NE','NV','NH','NJ',
    'NM','NY','NC','ND','OH','OK','OR','PA','RI','SC',
    'SD','TN','TX','UT','VT','VA','WA','WV','WI','WY','DC'
);
