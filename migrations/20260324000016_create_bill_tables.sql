-- +goose Up

-- ═══════════════════════════════════════════════════════════════════════════════
-- TABLE 1: bill_hyperswitch_customers — Family → Hyperswitch Customer mapping
-- ═══════════════════════════════════════════════════════════════════════════════
CREATE TABLE bill_hyperswitch_customers (
    family_id                 UUID PRIMARY KEY REFERENCES iam_families(id) ON DELETE CASCADE,
    hyperswitch_customer_id   TEXT NOT NULL UNIQUE,
    default_payment_method_id TEXT,
    created_at                TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at                TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_bill_hs_customers_hs_id
    ON bill_hyperswitch_customers(hyperswitch_customer_id);

-- ═══════════════════════════════════════════════════════════════════════════════
-- TABLE 2: bill_subscriptions — Subscription state mirror [S§15.3]
-- ═══════════════════════════════════════════════════════════════════════════════
CREATE TABLE bill_subscriptions (
    id                        UUID PRIMARY KEY DEFAULT uuidv7(),
    family_id                 UUID NOT NULL UNIQUE REFERENCES iam_families(id) ON DELETE CASCADE,
    hyperswitch_subscription_id TEXT NOT NULL UNIQUE,
    hyperswitch_customer_id   TEXT NOT NULL,
    tier                      TEXT NOT NULL DEFAULT 'premium'
                              CHECK (tier IN ('free', 'premium')),
    status                    TEXT NOT NULL DEFAULT 'incomplete'
                              CHECK (status IN (
                                  'active', 'trialing', 'past_due',
                                  'canceled', 'paused', 'incomplete'
                              )),
    billing_interval          TEXT NOT NULL
                              CHECK (billing_interval IN ('monthly', 'annual')),
    current_period_start      TIMESTAMPTZ NOT NULL,
    current_period_end        TIMESTAMPTZ NOT NULL,
    cancel_at_period_end      BOOLEAN NOT NULL DEFAULT false,
    canceled_at               TIMESTAMPTZ,
    amount_cents              INTEGER NOT NULL,
    currency                  TEXT NOT NULL DEFAULT 'usd',
    hyperswitch_price_id      TEXT NOT NULL,
    created_at                TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at                TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_bill_subscriptions_hs_sub_id
    ON bill_subscriptions(hyperswitch_subscription_id);

CREATE INDEX idx_bill_subscriptions_hs_cust_id
    ON bill_subscriptions(hyperswitch_customer_id);

CREATE INDEX idx_bill_subscriptions_active
    ON bill_subscriptions(status)
    WHERE status = 'active';

-- ═══════════════════════════════════════════════════════════════════════════════
-- TABLE 3: bill_transactions — Payment + COPPA history [S§15.3, S§1.4]
-- ═══════════════════════════════════════════════════════════════════════════════
CREATE TABLE bill_transactions (
    id                        UUID PRIMARY KEY DEFAULT uuidv7(),
    family_id                 UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    transaction_type          TEXT NOT NULL
                              CHECK (transaction_type IN (
                                  'subscription_payment', 'subscription_refund',
                                  'coppa_charge', 'coppa_refund'
                              )),
    status                    TEXT NOT NULL DEFAULT 'pending'
                              CHECK (status IN (
                                  'succeeded', 'pending', 'failed', 'refunded'
                              )),
    amount_cents              INTEGER NOT NULL,
    currency                  TEXT NOT NULL DEFAULT 'usd',
    hyperswitch_payment_id    TEXT,
    hyperswitch_invoice_id    TEXT,
    description               TEXT,
    metadata                  JSONB NOT NULL DEFAULT '{}',
    created_at                TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_bill_transactions_family_created
    ON bill_transactions(family_id, created_at DESC);

CREATE UNIQUE INDEX idx_bill_transactions_idempotency
    ON bill_transactions(hyperswitch_payment_id, transaction_type)
    WHERE hyperswitch_payment_id IS NOT NULL;

CREATE INDEX idx_bill_transactions_hs_payment
    ON bill_transactions(hyperswitch_payment_id)
    WHERE hyperswitch_payment_id IS NOT NULL;

-- ═══════════════════════════════════════════════════════════════════════════════
-- TABLE 4: bill_payouts — Creator payout aggregation (Phase 2) [S§9.6]
-- ═══════════════════════════════════════════════════════════════════════════════
CREATE TABLE bill_payouts (
    id                        UUID PRIMARY KEY DEFAULT uuidv7(),
    creator_id                UUID NOT NULL REFERENCES mkt_creators(id),
    status                    TEXT NOT NULL DEFAULT 'pending'
                              CHECK (status IN (
                                  'pending', 'processing', 'completed', 'failed'
                              )),
    amount_cents              INTEGER NOT NULL,
    currency                  TEXT NOT NULL DEFAULT 'usd',
    period_start              TIMESTAMPTZ NOT NULL,
    period_end                TIMESTAMPTZ NOT NULL,
    purchase_count            INTEGER NOT NULL DEFAULT 0,
    refund_deduction_cents    INTEGER NOT NULL DEFAULT 0,
    hyperswitch_payout_id     TEXT,
    processed_at              TIMESTAMPTZ,
    created_at                TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at                TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_bill_payouts_creator
    ON bill_payouts(creator_id, created_at DESC);

CREATE INDEX idx_bill_payouts_pending
    ON bill_payouts(status)
    WHERE status = 'pending';

CREATE UNIQUE INDEX idx_bill_payouts_creator_period
    ON bill_payouts(creator_id, period_start, period_end);

-- +goose Down
DROP TABLE IF EXISTS bill_payouts;
DROP TABLE IF EXISTS bill_transactions;
DROP TABLE IF EXISTS bill_subscriptions;
DROP TABLE IF EXISTS bill_hyperswitch_customers;
