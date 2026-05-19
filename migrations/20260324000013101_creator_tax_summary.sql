-- +goose Up
-- ═══════════════════════════════════════════════════════════════════════════════
-- TABLE: bill_creator_tax_summaries
-- Tracks cumulative creator earnings per calendar year for IRS 1099-K reporting.
-- Threshold: $600/year (60_000 cents). Updated by AggregatePayoutsTask. [HOM-62]
-- ═══════════════════════════════════════════════════════════════════════════════
CREATE TABLE bill_creator_tax_summaries (
    id                   UUID        NOT NULL DEFAULT uuidv7(),
    creator_id           UUID        NOT NULL,
    tax_year             SMALLINT    NOT NULL,
    earnings_cents       BIGINT      NOT NULL DEFAULT 0,
    threshold_reached_at TIMESTAMPTZ,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT pk_bill_creator_tax_summaries    PRIMARY KEY (id),
    CONSTRAINT uq_bill_creator_tax_year         UNIQUE (creator_id, tax_year),
    CONSTRAINT chk_bill_tax_earnings_nonneg     CHECK (earnings_cents >= 0)
);

CREATE INDEX idx_bill_creator_tax_creator_year
    ON bill_creator_tax_summaries(creator_id, tax_year);

-- +goose Down
DROP TABLE IF EXISTS bill_creator_tax_summaries;
