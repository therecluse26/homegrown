-- +goose Up
-- =============================================================================
-- Migration: 20260326_000024_create_plan_tables.sql
-- Domain: 17-plan (Planning & Scheduling)
-- =============================================================================

-- Schedule items: family-created calendar entries [17-planning §3.1]
-- These are plan::-owned data, NOT duplicates of learning activities or events
CREATE TABLE plan_schedule_items (
    id              UUID PRIMARY KEY DEFAULT uuidv7(),
    family_id       UUID NOT NULL REFERENCES iam_families(id),
    -- Who this item is for (NULL = whole family)
    student_id      UUID REFERENCES iam_students(id),
    -- Schedule details
    title           VARCHAR(200) NOT NULL,
    description     TEXT,
    -- Time block
    start_date      DATE NOT NULL,
    start_time      TIME,                  -- NULL = all-day item
    end_time        TIME,
    duration_minutes INT,                  -- computed or explicit
    -- Categorization
    category        VARCHAR(30) NOT NULL DEFAULT 'custom'
                    CHECK (category IN (
                        'lesson', 'reading', 'activity', 'assessment',
                        'field_trip', 'co_op', 'break', 'custom'
                    )),
    -- Subject (optional, from learn:: taxonomy)
    subject_id      UUID,                  -- references learn_subjects(id)
    -- Color for calendar display
    color           VARCHAR(7),            -- hex color, e.g., "#3B82F6"
    -- Completion status
    is_completed    BOOLEAN NOT NULL DEFAULT false,
    completed_at    TIMESTAMPTZ,
    -- Link to other domain entities (optional)
    linked_activity_id UUID,               -- if this schedule item was logged as an activity
    linked_event_id    UUID,               -- if created from a social event
    -- Recurrence (Phase 2 — stored as RRULE string)
    recurrence_rule VARCHAR(255),          -- e.g., "FREQ=WEEKLY;BYDAY=MO,WE,FR"
    recurrence_end  DATE,
    -- Metadata
    notes           TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_plan_schedule_items_family_date
    ON plan_schedule_items(family_id, start_date);
CREATE INDEX idx_plan_schedule_items_student
    ON plan_schedule_items(student_id, start_date)
    WHERE student_id IS NOT NULL;
CREATE INDEX idx_plan_schedule_items_linked_activity
    ON plan_schedule_items(linked_activity_id)
    WHERE linked_activity_id IS NOT NULL;

-- Schedule templates (Phase 2): reusable weekly schedule patterns [17-planning §3.1]
CREATE TABLE plan_schedule_templates (
    id              UUID PRIMARY KEY DEFAULT uuidv7(),
    family_id       UUID NOT NULL REFERENCES iam_families(id),
    name            VARCHAR(100) NOT NULL,
    description     TEXT,
    -- Template items stored as JSONB array
    -- Each item: { day_of_week, start_time, end_time, title, category, subject_id, color }
    items           JSONB NOT NULL DEFAULT '[]'::JSONB,
    is_active       BOOLEAN NOT NULL DEFAULT false,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_plan_schedule_templates_family
    ON plan_schedule_templates(family_id);

-- Row-Level Security [17-planning §3.2]
ALTER TABLE plan_schedule_items ENABLE ROW LEVEL SECURITY;
CREATE POLICY plan_schedule_items_family_scope ON plan_schedule_items
    USING (family_id = current_setting('app.current_family_id')::UUID);

ALTER TABLE plan_schedule_templates ENABLE ROW LEVEL SECURITY;
CREATE POLICY plan_schedule_templates_family_scope ON plan_schedule_templates
    USING (family_id = current_setting('app.current_family_id')::UUID);

-- +goose Down
DROP POLICY IF EXISTS plan_schedule_templates_family_scope ON plan_schedule_templates;
ALTER TABLE plan_schedule_templates DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS plan_schedule_items_family_scope ON plan_schedule_items;
ALTER TABLE plan_schedule_items DISABLE ROW LEVEL SECURITY;

DROP INDEX IF EXISTS idx_plan_schedule_templates_family;
DROP TABLE IF EXISTS plan_schedule_templates;

DROP INDEX IF EXISTS idx_plan_schedule_items_linked_activity;
DROP INDEX IF EXISTS idx_plan_schedule_items_student;
DROP INDEX IF EXISTS idx_plan_schedule_items_family_date;
DROP TABLE IF EXISTS plan_schedule_items;
