# Domain Spec 10 — Billing & Subscriptions (billing::)

## §1 Overview

The Billing & Subscriptions domain is the **subscription lifecycle and tier-gating engine** — it
manages family subscription state, premium tier enforcement via domain events, COPPA micro-charge
verification, and creator payout aggregation. It wraps the self-hosted Hyperswitch instance (shared
with `mkt::`) using a dedicated billing profile for subscription flows, keeping marketplace payments
and subscription billing isolated at the orchestration layer. Hyperswitch is the authoritative
source for subscription state; the local database mirrors it via webhooks. `[S§15, V§9, V§10]`

| Attribute | Value |
|-----------|-------|
| **Module path** | `src/billing/` |
| **DB prefix** | `bill_` `[ARCH §5.1]` |
| **Complexity class** | Simple (no `domain/` subdirectory) — Hyperswitch delegation; subscription state machine lives in the payment processor `[ARCH §4.5]` |
| **CQRS** | No — read and write paths are straightforward; no separated query model needed |
| **External adapter** | `src/billing/adapters/hyperswitch.rs` (Hyperswitch — processor-agnostic subscription + payment orchestration) `[ARCH §2.9]` |
| **Key constraint** | One subscription per family `[S§15.3]`; every user-data query family-scoped via `FamilyScope` `[CODING §2.4, §2.5]`; Hyperswitch is authoritative for subscription state — local table is a mirror |

**What billing:: owns**: Subscription lifecycle (create, update, cancel, reactivate, pause, resume),
tier-gating domain events (`SubscriptionCreated`, `SubscriptionChanged`, `SubscriptionCancelled`),
COPPA micro-charge verification adapter, Hyperswitch customer mapping, transaction history
(subscription payments + COPPA charges), payment method management (attach, list, detach via
Hyperswitch SetupIntents), subscription pricing estimates, creator payout aggregation (Phase 2),
Hyperswitch webhook processing for subscription events.

**What billing:: does NOT own**: Marketplace payments and split payments (owned by `mkt::` —
`src/mkt/adapters/payment.rs`) `[07-mkt §7]`, the `subscription_tier` column on `iam_families`
(owned by `iam::` — updated via event subscription) `[01-iam §3.2]`, revenue split calculation
(owned by `mkt::`) `[07-mkt §15]`, creator KYC and sub-merchant onboarding (owned by `mkt::`)
`[07-mkt §11]`, notification delivery (owned by `notify::`), feature gating logic per domain
(each domain uses `RequirePremium` extractor from `00-core`), user accounts and family
membership (owned by `iam::`).

**What billing:: delegates**: Notification delivery → `notify::` (via domain events). User/family
email lookup → `iam::IamService`. Subscription state machine → Hyperswitch (self-hosted).
Background job scheduling → sidekiq-rs `[ARCH §12]`. Creator payout execution → `mkt::` Hyperswitch
adapter (payouts go to sub-merchant accounts managed by `mkt::`) `[07-mkt §7]`.

---

## §2 Requirements Traceability

Every SPEC.md §15 requirement maps to a section in this document. Cross-references from
other spec sections are included where the billing domain is involved.

| Requirement | SPEC Reference | Domain Spec Section |
|-------------|---------------|---------------------|
| Free tier: social, basic learning, marketplace access, methodology, discovery, onboarding, data export | `[S§15.1]` | §10 (tier system — free is default) |
| Premium tier: compliance reporting, portfolio/transcript, advanced analytics, AI, enhanced storage, advanced mastery paths, methodology-specific tools | `[S§15.2]` | §10 (premium features), §16 (events → `iam::` tier sync) |
| Family-level billing (one subscription per family) | `[S§15.3]` | §3.2 (`bill_subscriptions` — `UNIQUE(family_id)`) |
| Monthly + annual billing cycles | `[S§15.3]` | §3.1 (`billing_interval` CHECK), §10 |
| Annual discount | `[S§15.3]` | §10, §11 (pricing via Hyperswitch `item_price_id`) |
| Immediate upgrade with proration | `[S§15.3]` | §11 (Hyperswitch `CreditOption::Prorate`) |
| End-of-period downgrade | `[S§15.3]` | §9 (`cancel_at_period_end`), §11 |
| Downgrade behavior: data preserved, premium read-only, reports downloadable, AI disabled | `[S§15.3]` | §9 (downgrade — `SubscriptionCancelled` event), §16 (`iam::` sets tier=free) |
| Advance renewal notice | `[S§15.3]` | §14 (webhook `invoice.upcoming` → Phase 3 via `notify::`) |
| Third-party payment processor | `[S§15.4]` | §7 (Hyperswitch adapter, Stripe as connector) |
| Sales tax collection/remittance | `[S§15.4]` | §7 (Stripe Tax via Hyperswitch) `[ARCH §2.9]` |
| COPPA micro-charge verification | `[S§1.4]` | §13 (COPPA micro-charge flow) |
| Family deletion lifecycle | `[S§3.4]` | §5 (`handle_family_deletion_scheduled`), §16 |
| Primary parent transfer | `[S§3.4]` | §5 (`handle_primary_parent_transferred`), §16 |
| Subscription notification types | `[S§13.1]` | §16 (events → `notify::`) |
| Premium subscription pricing (~$10-15/month, ~20% annual discount) | `[S§20.2]` | §10 (configurable via Hyperswitch price IDs, not hardcoded) |

> **Coverage note on `[S§15.4]` (marketplace transactions)**: SPEC.md §15.4 covers marketplace
> payment processing, sales tax, and 1099-K. Marketplace payments are owned by `mkt::` (split
> payments, creator sub-merchants, payouts) `[07-mkt §7, §11, §15]`. This billing spec covers
> only subscription payments and COPPA micro-charges. Both `mkt::` and `billing::` use the same
> Hyperswitch instance with different business profiles `[07-mkt §18.5]`.

---

## §3 Database Schema

The billing domain stores subscription state (mirrored from Hyperswitch), transaction history,
Hyperswitch customer mappings, and creator payout records. All user-data tables are family-scoped
via `family_id` foreign key. `[ARCH §5.1, ARCH §5.2]`

> **Refinement note**: ARCHITECTURE.md §5.1 sketches `bill_subscriptions`, `bill_transactions`,
> and `bill_payouts`. This spec adds: (1) `bill_hyperswitch_customers` for customer ID mapping,
> (2) comprehensive CHECK constraints for subscription status, billing interval, transaction
> type, transaction status, and payout status, (3) `hyperswitch_subscription_id` on subscriptions
> for webhook reconciliation, (4) `metadata` JSONB on transactions for idempotency via event_id.

### §3.1 Enums

Implemented as `CHECK` constraints (not PostgreSQL ENUM types) per `[CODING §4.1]`:

```sql
-- Subscription status, billing interval, transaction type, transaction status,
-- and payout status are all enforced via CHECK constraints on their respective
-- columns rather than as PostgreSQL enum types. This avoids ALTER TYPE
-- limitations when adding new values in future migrations. [ARCH §5.2]
--
-- Subscription status values: active, trialing, past_due, canceled, paused, incomplete
-- Billing interval values: monthly, annual
-- Transaction type values: subscription_payment, subscription_refund, coppa_charge, coppa_refund
-- Transaction status values: succeeded, pending, failed, refunded
-- Payout status values: pending, processing, completed, failed
```

**Subscription status state machine** (Hyperswitch-authoritative, mirrored locally via webhooks):

```
[none] ──► incomplete  (payment pending on first subscription)
               │
               ├──► active      (payment succeeded)
               │
               └──► canceled    (payment failed / expired)

active ──► past_due    (renewal payment failed)
               │
               ├──► active      (retry payment succeeded)
               │
               └──► canceled    (all retries exhausted)

active ──► paused      (family requested pause — Phase 2)
               │
               └──► active      (family resumed)

active ──► canceled    (cancel_at_period_end reached end of term)

trialing ──► active    (trial ended, payment succeeded — Phase 2)
```

### §3.2 Tables

```sql
-- =============================================================================
-- Migration: YYYYMMDD_000001_create_bill_tables.rs
-- =============================================================================

-- ═══════════════════════════════════════════════════════════════════════════════
-- TABLE 1: bill_hyperswitch_customers — Family → Hyperswitch Customer mapping
-- ═══════════════════════════════════════════════════════════════════════════════
-- Maps each family to a Hyperswitch customer record. Created during the first
-- billing interaction (subscription creation or COPPA micro-charge). One
-- customer per family.
-- ═══════════════════════════════════════════════════════════════════════════════
CREATE TABLE bill_hyperswitch_customers (
    family_id                 UUID PRIMARY KEY REFERENCES iam_families(id) ON DELETE CASCADE,
    hyperswitch_customer_id   TEXT NOT NULL UNIQUE,     -- Hyperswitch customer ID
    default_payment_method_id TEXT,                     -- Hyperswitch payment method ID
    created_at                TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at                TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Lookup by Hyperswitch customer ID (webhook processing)
CREATE UNIQUE INDEX idx_bill_hs_customers_hs_id
    ON bill_hyperswitch_customers(hyperswitch_customer_id);

-- ═══════════════════════════════════════════════════════════════════════════════
-- TABLE 2: bill_subscriptions — Subscription state mirror [S§15.3]
-- ═══════════════════════════════════════════════════════════════════════════════
-- Mirrors Hyperswitch subscription state via webhooks. Hyperswitch is the
-- authoritative source — this table is a read-optimized projection. The system
-- can always recover by re-syncing from Hyperswitch. One subscription per
-- family (enforced by UNIQUE constraint on family_id).
-- ═══════════════════════════════════════════════════════════════════════════════
CREATE TABLE bill_subscriptions (
    id                        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id                 UUID NOT NULL UNIQUE REFERENCES iam_families(id) ON DELETE CASCADE,
    hyperswitch_subscription_id TEXT NOT NULL UNIQUE,   -- Hyperswitch subscription ID
    hyperswitch_customer_id   TEXT NOT NULL,            -- denormalized for webhook fast-path
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
    canceled_at               TIMESTAMPTZ,             -- when cancellation was requested
    amount_cents              INTEGER NOT NULL,         -- current price in cents
    currency                  TEXT NOT NULL DEFAULT 'usd',
    hyperswitch_price_id      TEXT NOT NULL,            -- Hyperswitch item_price_id
    created_at                TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at                TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Lookup by Hyperswitch subscription ID (webhook processing)
CREATE INDEX idx_bill_subscriptions_hs_sub_id
    ON bill_subscriptions(hyperswitch_subscription_id);

-- Lookup by Hyperswitch customer ID (webhook fast-path)
CREATE INDEX idx_bill_subscriptions_hs_cust_id
    ON bill_subscriptions(hyperswitch_customer_id);

-- Active subscriptions (for tier enforcement queries)
CREATE INDEX idx_bill_subscriptions_active
    ON bill_subscriptions(status)
    WHERE status = 'active';

-- ═══════════════════════════════════════════════════════════════════════════════
-- TABLE 3: bill_transactions — Payment + COPPA history [S§15.3, S§1.4]
-- ═══════════════════════════════════════════════════════════════════════════════
-- Records every financial transaction: subscription payments, refunds, COPPA
-- micro-charges, and COPPA refunds. Family-scoped for privacy. Hyperswitch
-- payment/invoice IDs link back to the payment processor for reconciliation.
-- ═══════════════════════════════════════════════════════════════════════════════
CREATE TABLE bill_transactions (
    id                        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
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
    hyperswitch_payment_id    TEXT,                     -- Hyperswitch payment ID (NULL for pending)
    hyperswitch_invoice_id    TEXT,                     -- Hyperswitch invoice ID (subscription payments only)
    description               TEXT,                     -- human-readable description
    metadata                  JSONB NOT NULL DEFAULT '{}',  -- contains event_id for webhook idempotency
    created_at                TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Transaction history for a family, newest first
CREATE INDEX idx_bill_transactions_family_created
    ON bill_transactions(family_id, created_at DESC);

-- Idempotency: prevent duplicate transactions for the same Hyperswitch event
CREATE UNIQUE INDEX idx_bill_transactions_idempotency
    ON bill_transactions(hyperswitch_payment_id, transaction_type)
    WHERE hyperswitch_payment_id IS NOT NULL;

-- Lookup by Hyperswitch payment ID (webhook reconciliation)
CREATE INDEX idx_bill_transactions_hs_payment
    ON bill_transactions(hyperswitch_payment_id)
    WHERE hyperswitch_payment_id IS NOT NULL;

-- ═══════════════════════════════════════════════════════════════════════════════
-- TABLE 4: bill_payouts — Creator payout aggregation (Phase 2) [S§9.6]
-- ═══════════════════════════════════════════════════════════════════════════════
-- Aggregated monthly creator payouts. Earnings computed from
-- mkt_purchases.creator_payout_cents minus paid payouts minus refunds.
-- Payout execution via mkt:: Hyperswitch adapter (payouts go to sub-merchant
-- accounts). Creator-scoped (not family-scoped).
-- ═══════════════════════════════════════════════════════════════════════════════
CREATE TABLE bill_payouts (
    id                        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
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
    hyperswitch_payout_id     TEXT,                     -- Hyperswitch payout ID (set after execution)
    processed_at              TIMESTAMPTZ,              -- when payout was executed
    created_at                TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at                TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Creator payout history
CREATE INDEX idx_bill_payouts_creator
    ON bill_payouts(creator_id, created_at DESC);

-- Pending payouts for batch processing
CREATE INDEX idx_bill_payouts_pending
    ON bill_payouts(status)
    WHERE status = 'pending';

-- One payout per creator per period (idempotency)
CREATE UNIQUE INDEX idx_bill_payouts_creator_period
    ON bill_payouts(creator_id, period_start, period_end);
```

### §3.3 RLS / Family-Scoping

Application-layer enforcement via `FamilyScope` extractor `[CODING §2.4, §2.5, 00-core §8]`:

```sql
-- Application-layer enforcement: every query on bill_subscriptions,
-- bill_transactions, and bill_hyperswitch_customers MUST include
-- WHERE family_id = $family_scope.family_id
-- Repository methods accept FamilyScope as first parameter to enforce this.
-- No direct table access without family_id filter.

-- bill_payouts is creator-scoped. Access is via creator_id, which maps
-- to parent_id → family_id through mkt_creators.parent_id → iam_parents.
-- Repository methods verify creator ownership before returning payout data.
```

---

## §4 API Endpoints

**Prefix**: `/v1/billing`

**Auth**: All endpoints require `AuthContext` (`FamilyScope`) unless noted otherwise.
`[00-core §7.2, §8]`

### §4.1 Phase 1 (4 endpoints)

#### GET /v1/billing/subscription — Current Subscription

Return the current family's subscription status and tier. Returns `{ tier: "free", status: null }`
if no subscription exists (the default state for all families).

- **Auth**: `AuthContext` + `FamilyScope`
- **Response**: `200 OK` → `SubscriptionResponse`
```rust
{
    "tier": "free",                   // "free" | "premium"
    "status": null,                   // null if no subscription, else subscription status
    "billing_interval": null,         // "monthly" | "annual" | null
    "current_period_end": null,       // ISO 8601 | null
    "cancel_at_period_end": false,
    "amount_cents": null,             // current price | null
    "currency": null
}
```
- **Error codes**: `401` (unauthenticated)

#### POST /v1/billing/coppa-verify — COPPA Micro-Charge Verification

Process a COPPA parental consent micro-charge ($0.50 charge + immediate refund). Called by
`iam::` service during the COPPA consent flow. `[S§1.4]`

- **Auth**: `AuthContext` + `FamilyScope`
- **Body**: `CoppaVerificationCommand`
```rust
{
    "payment_method_id": "pm_xxx"     // Hyperswitch payment method ID
}
```
- **Response**: `200 OK` → `CoppaVerificationResult`
```rust
{
    "verified": true,
    "charge_id": "pay_xxx",
    "refund_id": "ref_xxx"
}
```
- **Side effects**:
  - Creates Hyperswitch customer if not exists (via `bill_hyperswitch_customers`)
  - Charges $0.50 via Hyperswitch one-time payment (not subscription)
  - Immediately refunds the charge
  - Creates two `bill_transactions` rows: `coppa_charge` + `coppa_refund`
- **Error codes**:
  - `422` — payment declined (`PaymentDeclined`)
  - `502` — payment adapter unavailable (`PaymentAdapterUnavailable`)

#### POST /hooks/hyperswitch/billing — Webhook Receiver

Receive and process Hyperswitch webhook events for subscription lifecycle. `[ARCH §4.6]`

- **Auth**: Hyperswitch webhook signature verification (HMAC) — no user auth
- **Body**: Raw Hyperswitch webhook payload
- **Response**: `200 OK` (always — to prevent Hyperswitch retry storms)
- **Side effects**:
  - Verifies webhook signature via adapter
  - Parses event type and dispatches to appropriate handler
  - Idempotency: checks `metadata.event_id` against `bill_transactions` to prevent duplicates
  - Updates `bill_subscriptions` status mirror
  - Publishes domain events (`SubscriptionCreated`, `SubscriptionChanged`, `SubscriptionCancelled`)
- **Error handling**: Logs errors internally, always returns `200` to Hyperswitch

#### GET /v1/billing/transactions — Transaction History

List financial transactions for the current family (subscription payments, COPPA charges).

- **Auth**: `AuthContext` + `FamilyScope`
- **Query**: `TransactionListParams { cursor?, limit? (default 20, max 100) }`
- **Response**: `200 OK` → `TransactionListResponse`
```rust
{
    "transactions": [
        {
            "id": "uuid",
            "transaction_type": "subscription_payment",
            "status": "succeeded",
            "amount_cents": 1499,
            "currency": "usd",
            "description": "Premium subscription — monthly",
            "created_at": "2026-03-21T..."
        }
    ],
    "next_cursor": "..."
}
```
- **Pagination**: Cursor-based on `(created_at, id)` for stable ordering
- **Error codes**: `401` (unauthenticated), `400` (invalid params)

### §4.2 Phase 2 (~10 endpoints)

#### POST /v1/billing/subscription — Create Premium Subscription

Create a new premium subscription via Hyperswitch.

- **Auth**: `AuthContext` + `FamilyScope` + `RequireRole(PrimaryParent)`
- **Body**: `CreateSubscriptionCommand`
```rust
{
    "billing_interval": "monthly",     // "monthly" | "annual"
    "payment_method_id": "pm_xxx"      // Hyperswitch payment method ID
}
```
- **Response**: `201 Created` → `SubscriptionResponse`
- **Side effects**:
  - Creates Hyperswitch customer if not exists
  - Creates Hyperswitch subscription with appropriate `item_price_id`
  - Creates `bill_subscriptions` row with status `incomplete` (updated to `active` via webhook)
  - Publishes `SubscriptionCreated` event (on webhook confirmation)
- **Error codes**:
  - `409` — subscription already exists (`SubscriptionAlreadyExists`)
  - `422` — payment declined (`PaymentDeclined`)
  - `502` — adapter unavailable (`PaymentAdapterUnavailable`)

#### PATCH /v1/billing/subscription — Update Subscription

Update billing interval (monthly ↔ annual) with proration.

- **Auth**: `AuthContext` + `FamilyScope` + `RequireRole(PrimaryParent)`
- **Body**: `UpdateSubscriptionCommand`
```rust
{
    "billing_interval": "annual"       // new billing interval
}
```
- **Response**: `200 OK` → `SubscriptionResponse`
- **Side effects**:
  - Calls Hyperswitch `update_subscription` with proration
  - Updates local mirror via webhook
  - Publishes `SubscriptionChanged` event (on webhook confirmation)
- **Error codes**:
  - `404` — no active subscription (`SubscriptionNotFound`)
  - `409` — subscription not active (`SubscriptionNotActive`)
  - `422` — payment declined (`PaymentDeclined`)

#### DELETE /v1/billing/subscription — Cancel Subscription

Cancel subscription at end of current billing period. `[S§15.3]`

- **Auth**: `AuthContext` + `FamilyScope` + `RequireRole(PrimaryParent)`
- **Response**: `200 OK` → `SubscriptionResponse` (with `cancel_at_period_end: true`)
- **Side effects**:
  - Calls Hyperswitch `cancel_subscription` with `CancelOption::EndOfTerm`
  - Sets `cancel_at_period_end = true` and `canceled_at = now()` on local mirror
  - Publishes `SubscriptionCancelled` event at period end (via webhook)
- **Error codes**:
  - `404` — no subscription (`SubscriptionNotFound`)
  - `409` — already canceled or not active (`CannotReactivate`)

#### POST /v1/billing/subscription/reactivate — Reactivate Subscription

Reverse a pending cancellation before the period ends.

- **Auth**: `AuthContext` + `FamilyScope` + `RequireRole(PrimaryParent)`
- **Response**: `200 OK` → `SubscriptionResponse` (with `cancel_at_period_end: false`)
- **Side effects**:
  - Calls Hyperswitch to remove cancellation schedule
  - Clears `cancel_at_period_end` and `canceled_at` on local mirror
- **Error codes**:
  - `404` — no subscription (`SubscriptionNotFound`)
  - `409` — subscription is not pending cancellation (`CannotReactivate`)

#### POST /v1/billing/subscription/estimate — Preview Pricing

Preview pricing for a new subscription or plan change (before committing).

- **Auth**: `AuthContext` + `FamilyScope`
- **Body**: `EstimateSubscriptionQuery`
```rust
{
    "billing_interval": "annual"       // desired interval
}
```
- **Response**: `200 OK` → `EstimateResponse`
```rust
{
    "amount_cents": 11999,
    "currency": "usd",
    "billing_interval": "annual",
    "proration_credits_cents": 750,    // credit from current period (if upgrading)
    "total_due_today_cents": 11249,
    "next_billing_date": "2027-03-21"
}
```

#### POST /v1/billing/payment-methods — Attach Payment Method

Attach a payment method via Hyperswitch SetupIntent flow.

- **Auth**: `AuthContext` + `FamilyScope` + `RequireRole(PrimaryParent)`
- **Response**: `201 Created` → `PaymentMethodResponse`

#### GET /v1/billing/payment-methods — List Payment Methods

List attached payment methods for the family's Hyperswitch customer.

- **Auth**: `AuthContext` + `FamilyScope`
- **Response**: `200 OK` → `Vec<PaymentMethodResponse>`

#### DELETE /v1/billing/payment-methods/:id — Detach Payment Method

Remove a payment method. Cannot remove the last method if an active subscription exists.

- **Auth**: `AuthContext` + `FamilyScope` + `RequireRole(PrimaryParent)`
- **Response**: `204 No Content`
- **Error codes**:
  - `404` — payment method not found (`PaymentMethodNotFound`)
  - `409` — cannot remove last method with active subscription (`CannotRemoveLastPaymentMethod`)

#### GET /v1/billing/invoices — Subscription Invoices

List Hyperswitch invoices for the family's subscription.

- **Auth**: `AuthContext` + `FamilyScope`
- **Query**: `InvoiceListParams { cursor?, limit? }`
- **Response**: `200 OK` → `InvoiceListResponse`

#### GET /v1/billing/payouts — Creator Payout History

List payout records for the authenticated creator. Creator-scoped, not family-scoped.

- **Auth**: `AuthContext` + `RequireRole(Creator)`
- **Query**: `PayoutListParams { cursor?, limit? }`
- **Response**: `200 OK` → `PayoutListResponse`

---

## §5 Service Interface

The `BillingService` trait defines all use cases exposed to handlers and event handler
structs. No CQRS separation needed — this is a simple domain with straightforward
read/write paths. `[CODING §8.2]`

```rust
// src/billing/ports.rs

use crate::shared::types::{FamilyId, FamilyScope};
use crate::shared::error::AppError;

#[async_trait]
pub trait BillingService: Send + Sync {
    // ─── Queries (read, no side effects) ────────────────────────────────

    /// Get current subscription for a family. Returns None if free tier.
    async fn get_subscription(
        &self,
        scope: FamilyScope,
    ) -> Result<Option<SubscriptionResponse>, AppError>;

    /// List transaction history for a family.
    async fn list_transactions(
        &self,
        params: TransactionListParams,
        scope: FamilyScope,
    ) -> Result<TransactionListResponse, AppError>;

    /// List Hyperswitch invoices for a family's subscription. (Phase 2)
    async fn list_invoices(
        &self,
        params: InvoiceListParams,
        scope: FamilyScope,
    ) -> Result<InvoiceListResponse, AppError>;

    /// List attached payment methods for a family. (Phase 2)
    async fn list_payment_methods(
        &self,
        scope: FamilyScope,
    ) -> Result<Vec<PaymentMethodResponse>, AppError>;

    /// Preview pricing for a subscription or plan change. (Phase 2)
    async fn estimate_subscription(
        &self,
        query: EstimateSubscriptionQuery,
        scope: FamilyScope,
    ) -> Result<EstimateResponse, AppError>;

    /// List creator payout history. (Phase 2)
    async fn list_payouts(
        &self,
        params: PayoutListParams,
        creator_id: Uuid,
    ) -> Result<PayoutListResponse, AppError>;

    // ─── Commands (write, has side effects) ─────────────────────────────

    /// Create a new premium subscription via Hyperswitch. (Phase 2)
    async fn create_subscription(
        &self,
        cmd: CreateSubscriptionCommand,
        scope: FamilyScope,
    ) -> Result<SubscriptionResponse, AppError>;

    /// Update subscription (billing interval change) with proration. (Phase 2)
    async fn update_subscription(
        &self,
        cmd: UpdateSubscriptionCommand,
        scope: FamilyScope,
    ) -> Result<SubscriptionResponse, AppError>;

    /// Cancel subscription at end of current billing period. (Phase 2)
    async fn cancel_subscription(
        &self,
        scope: FamilyScope,
    ) -> Result<SubscriptionResponse, AppError>;

    /// Reverse a pending cancellation. (Phase 2)
    async fn reactivate_subscription(
        &self,
        scope: FamilyScope,
    ) -> Result<SubscriptionResponse, AppError>;

    /// Attach a payment method via Hyperswitch SetupIntent. (Phase 2)
    async fn attach_payment_method(
        &self,
        cmd: AttachPaymentMethodCommand,
        scope: FamilyScope,
    ) -> Result<PaymentMethodResponse, AppError>;

    /// Detach a payment method. (Phase 2)
    async fn detach_payment_method(
        &self,
        payment_method_id: &str,
        scope: FamilyScope,
    ) -> Result<(), AppError>;

    /// Process COPPA micro-charge verification ($0.50 charge + immediate refund).
    /// Called by iam:: during COPPA consent flow. [S§1.4]
    async fn process_coppa_verification(
        &self,
        cmd: CoppaVerificationCommand,
        scope: FamilyScope,
    ) -> Result<CoppaVerificationResult, AppError>;

    // ─── Event handlers ─────────────────────────────────────────────────
    // Each method is called by its corresponding DomainEventHandler struct
    // in event_handlers.rs. Failures are logged but do not propagate to
    // the source domain. [ARCH §4.6]

    /// Handle family deletion: cancel subscription in Hyperswitch.
    /// Consumed from iam::FamilyDeletionScheduled. [01-iam §13.3]
    async fn handle_family_deletion_scheduled(
        &self,
        event: &FamilyDeletionScheduled,
    ) -> Result<(), AppError>;

    /// Handle primary parent transfer: update Hyperswitch customer email.
    /// Consumed from iam::PrimaryParentTransferred. [01-iam §13.3]
    async fn handle_primary_parent_transferred(
        &self,
        event: &PrimaryParentTransferred,
    ) -> Result<(), AppError>;

    /// Handle marketplace purchase: record creator earnings. (Phase 2)
    /// Consumed from mkt::PurchaseCompleted. [07-mkt §18.3]
    async fn handle_purchase_completed(
        &self,
        event: &PurchaseCompleted,
    ) -> Result<(), AppError>;

    /// Handle marketplace refund: deduct from creator earnings. (Phase 2)
    /// Consumed from mkt::PurchaseRefunded. [07-mkt §18.3]
    async fn handle_purchase_refunded(
        &self,
        event: &PurchaseRefunded,
    ) -> Result<(), AppError>;

    // ─── Webhook processing ─────────────────────────────────────────────

    /// Process a verified Hyperswitch webhook payload.
    async fn process_hyperswitch_webhook(
        &self,
        payload: &[u8],
        signature: &str,
    ) -> Result<(), AppError>;
}
```

### `BillingServiceImpl`

```rust
// src/billing/service.rs

pub struct BillingServiceImpl {
    subscription_repo: Arc<dyn SubscriptionRepository>,
    transaction_repo: Arc<dyn TransactionRepository>,
    customer_repo: Arc<dyn CustomerRepository>,
    payout_repo: Arc<dyn PayoutRepository>,          // Phase 2
    adapter: Arc<dyn SubscriptionPaymentAdapter>,
    iam_service: Arc<dyn IamService>,                // Email lookup for Hyperswitch customer
    events: Arc<EventBus>,
    config: BillingConfig,
}
```

---

## §6 Repository Interfaces

All billing repositories are family-scoped via `FamilyScope` parameter (except
`PayoutRepository`, which is creator-scoped). `[CODING §8.2]`

```rust
// src/billing/ports.rs (continued)

// ─── SubscriptionRepository ───────────────────────────────────────────
// Family-scoped. One subscription per family. [S§15.3]
#[async_trait]
pub trait SubscriptionRepository: Send + Sync {
    /// Create a new subscription record.
    async fn create(
        &self,
        input: CreateSubscriptionRow,
    ) -> Result<BillSubscription, AppError>;

    /// Find subscription by family ID.
    async fn find_by_family(
        &self,
        scope: FamilyScope,
    ) -> Result<Option<BillSubscription>, AppError>;

    /// Find subscription by Hyperswitch subscription ID (webhook processing).
    async fn find_by_hyperswitch_id(
        &self,
        hyperswitch_subscription_id: &str,
    ) -> Result<Option<BillSubscription>, AppError>;

    /// Update subscription status and fields (from webhook data).
    async fn update(
        &self,
        subscription_id: Uuid,
        updates: SubscriptionUpdate,
    ) -> Result<BillSubscription, AppError>;

    /// Delete subscription record (family deletion cascade).
    async fn delete_by_family(
        &self,
        family_id: FamilyId,
    ) -> Result<(), AppError>;
}

// ─── TransactionRepository ────────────────────────────────────────────
// Family-scoped. Immutable records — insert only, no updates.
#[async_trait]
pub trait TransactionRepository: Send + Sync {
    /// Create a new transaction record.
    async fn create(
        &self,
        input: CreateTransactionRow,
    ) -> Result<BillTransaction, AppError>;

    /// List transactions for a family, paginated by created_at DESC.
    async fn list_by_family(
        &self,
        scope: FamilyScope,
        params: &TransactionListParams,
    ) -> Result<Vec<BillTransaction>, AppError>;

    /// Check if a transaction with this Hyperswitch payment ID and type
    /// already exists (idempotency check for webhook processing).
    async fn exists_by_payment_id(
        &self,
        hyperswitch_payment_id: &str,
        transaction_type: &str,
    ) -> Result<bool, AppError>;
}

// ─── CustomerRepository ───────────────────────────────────────────────
// Family-scoped (family_id is PK).
#[async_trait]
pub trait CustomerRepository: Send + Sync {
    /// Create or update a Hyperswitch customer mapping.
    async fn upsert(
        &self,
        family_id: FamilyId,
        input: UpsertCustomerRow,
    ) -> Result<BillHyperswitchCustomer, AppError>;

    /// Find customer by family ID.
    async fn find_by_family(
        &self,
        family_id: FamilyId,
    ) -> Result<Option<BillHyperswitchCustomer>, AppError>;

    /// Find customer by Hyperswitch customer ID (webhook processing).
    async fn find_by_hyperswitch_id(
        &self,
        hyperswitch_customer_id: &str,
    ) -> Result<Option<BillHyperswitchCustomer>, AppError>;
}

// ─── PayoutRepository (Phase 2) ──────────────────────────────────────
// Creator-scoped — accessed via creator_id.
#[async_trait]
pub trait PayoutRepository: Send + Sync {
    /// Create a new payout record.
    async fn create(
        &self,
        input: CreatePayoutRow,
    ) -> Result<BillPayout, AppError>;

    /// List payouts for a creator, paginated by created_at DESC.
    async fn list_by_creator(
        &self,
        creator_id: Uuid,
        params: &PayoutListParams,
    ) -> Result<Vec<BillPayout>, AppError>;

    /// Update payout status (processing → completed/failed).
    async fn update_status(
        &self,
        payout_id: Uuid,
        status: &str,
        hyperswitch_payout_id: Option<&str>,
    ) -> Result<BillPayout, AppError>;

    /// Find pending payouts ready for processing.
    async fn find_pending(
        &self,
        limit: u32,
    ) -> Result<Vec<BillPayout>, AppError>;
}
```

---

## §7 Adapter Interface (SubscriptionPaymentAdapter)

Single adapter wrapping Hyperswitch subscription + payment APIs. Uses the billing-specific
Hyperswitch business profile (separate from `mkt::`'s marketplace profile). Both adapters
talk to the same Hyperswitch instance. `[07-mkt §18.5]`

> **Relationship to mkt::PaymentAdapter**: `mkt::` has its own `PaymentAdapter` trait `[07-mkt §7]`
> for marketplace-specific operations (split payments, sub-merchant onboarding, marketplace
> refunds). `billing::` has its own `SubscriptionPaymentAdapter` for subscription-specific
> operations (subscription CRUD, SetupIntents, invoices). They are separate traits because
> the operations are fundamentally different — marketplace payments are one-time split payments,
> while billing payments are recurring subscriptions. Both wrap the same Hyperswitch REST API
> but use different endpoints and different business profiles.

```rust
// src/billing/ports.rs (continued)

/// Processor-agnostic subscription + payment adapter backed by Hyperswitch.
/// Implementations: HyperswitchSubscriptionAdapter (production),
///                  MockSubscriptionAdapter (tests).
///
/// Uses the billing-specific Hyperswitch business profile, separate from
/// the marketplace profile used by mkt::PaymentAdapter. [07-mkt §18.5]
#[async_trait]
pub trait SubscriptionPaymentAdapter: Send + Sync {
    // ─── Customer Management ────────────────────────────────────────────

    /// Create a Hyperswitch customer for a family.
    /// Returns the Hyperswitch customer ID.
    async fn create_customer(
        &self,
        email: &str,
        name: &str,
        metadata: HashMap<String, String>,
    ) -> Result<String, BillingError>;

    /// Update a Hyperswitch customer (e.g., email change on parent transfer).
    async fn update_customer(
        &self,
        customer_id: &str,
        email: &str,
        name: &str,
    ) -> Result<(), BillingError>;

    // ─── Subscriptions ──────────────────────────────────────────────────
    // Maps to Hyperswitch subscription API:
    //   POST   /subscriptions
    //   PATCH  /subscriptions/{id}
    //   POST   /subscriptions/{id}/cancel
    //   POST   /subscriptions/{id}/pause
    //   POST   /subscriptions/{id}/resume
    //   GET    /subscriptions/{id}/estimate

    /// Create a new subscription in Hyperswitch.
    async fn create_subscription(
        &self,
        customer_id: &str,
        price_id: &str,
        payment_method_id: &str,
        metadata: HashMap<String, String>,
    ) -> Result<HyperswitchSubscription, BillingError>;

    /// Update a subscription (e.g., change billing interval / price).
    /// Hyperswitch handles proration automatically.
    async fn update_subscription(
        &self,
        subscription_id: &str,
        new_price_id: &str,
    ) -> Result<HyperswitchSubscription, BillingError>;

    /// Cancel a subscription at end of current period.
    async fn cancel_subscription(
        &self,
        subscription_id: &str,
    ) -> Result<HyperswitchSubscription, BillingError>;

    /// Pause a subscription (Phase 2).
    async fn pause_subscription(
        &self,
        subscription_id: &str,
    ) -> Result<HyperswitchSubscription, BillingError>;

    /// Resume a paused subscription (Phase 2).
    async fn resume_subscription(
        &self,
        subscription_id: &str,
    ) -> Result<HyperswitchSubscription, BillingError>;

    /// Preview pricing for a new subscription or plan change.
    async fn estimate_subscription(
        &self,
        customer_id: &str,
        price_id: &str,
        current_subscription_id: Option<&str>,
    ) -> Result<HyperswitchEstimate, BillingError>;

    // ─── Payment Methods ────────────────────────────────────────────────

    /// Create a SetupIntent for attaching a payment method.
    async fn create_setup_intent(
        &self,
        customer_id: &str,
    ) -> Result<SetupIntentResponse, BillingError>;

    /// List payment methods for a customer.
    async fn list_payment_methods(
        &self,
        customer_id: &str,
    ) -> Result<Vec<HyperswitchPaymentMethod>, BillingError>;

    /// Detach a payment method from a customer.
    async fn detach_payment_method(
        &self,
        payment_method_id: &str,
    ) -> Result<(), BillingError>;

    // ─── One-Time Payments (COPPA) ──────────────────────────────────────

    /// Process a COPPA micro-charge: charge $0.50 and immediately refund.
    /// Uses Hyperswitch one-time payment (not subscription).
    /// Returns (payment_id, refund_id).
    async fn process_micro_charge(
        &self,
        customer_id: &str,
        payment_method_id: &str,
        amount_cents: i64,
        description: &str,
        metadata: HashMap<String, String>,
    ) -> Result<(String, String), BillingError>;

    // ─── Invoices ───────────────────────────────────────────────────────

    /// List invoices for a customer's subscription.
    async fn list_invoices(
        &self,
        customer_id: &str,
        limit: u32,
    ) -> Result<Vec<HyperswitchInvoice>, BillingError>;

    // ─── Payouts (Phase 2) ──────────────────────────────────────────────

    /// Create a payout to a creator's sub-merchant account.
    /// Reuses mkt:: Hyperswitch profile for payouts to creator accounts.
    async fn create_payout(
        &self,
        payment_account_id: &str,
        amount_cents: i64,
        currency: &str,
        metadata: HashMap<String, String>,
    ) -> Result<HyperswitchPayout, BillingError>;

    // ─── Webhooks ───────────────────────────────────────────────────────

    /// Verify webhook signature (HMAC).
    async fn verify_webhook(
        &self,
        payload: &[u8],
        signature: &str,
    ) -> Result<bool, BillingError>;

    /// Parse a verified webhook payload into a billing event.
    async fn parse_webhook_event(
        &self,
        payload: &[u8],
    ) -> Result<BillingWebhookEvent, BillingError>;
}

// ─── Adapter Supporting Types ───────────────────────────────────────────

pub struct HyperswitchSubscription {
    pub id: String,
    pub customer_id: String,
    pub status: String,
    pub current_period_start: DateTime<Utc>,
    pub current_period_end: DateTime<Utc>,
    pub cancel_at_period_end: bool,
    pub price_id: String,
    pub amount_cents: i64,
    pub currency: String,
}

pub struct HyperswitchEstimate {
    pub amount_cents: i64,
    pub currency: String,
    pub proration_credits_cents: i64,
    pub total_due_today_cents: i64,
    pub next_billing_date: NaiveDate,
}

pub struct SetupIntentResponse {
    pub client_secret: String,         // for frontend Hyperswitch.js confirmation
}

pub struct HyperswitchPaymentMethod {
    pub id: String,
    pub method_type: String,           // "card", "bank_debit", etc.
    pub last_four: Option<String>,
    pub brand: Option<String>,         // "visa", "mastercard", etc.
    pub exp_month: Option<u8>,
    pub exp_year: Option<u16>,
    pub is_default: bool,
}

pub struct HyperswitchInvoice {
    pub id: String,
    pub amount_cents: i64,
    pub currency: String,
    pub status: String,
    pub period_start: DateTime<Utc>,
    pub period_end: DateTime<Utc>,
    pub paid_at: Option<DateTime<Utc>>,
    pub pdf_url: Option<String>,
}

pub struct HyperswitchPayout {
    pub id: String,
    pub amount_cents: i64,
    pub status: String,
}

pub enum BillingWebhookEvent {
    SubscriptionCreated {
        subscription: HyperswitchSubscription,
    },
    SubscriptionUpdated {
        subscription: HyperswitchSubscription,
    },
    SubscriptionDeleted {
        subscription_id: String,
    },
    InvoicePaid {
        invoice_id: String,
        subscription_id: String,
        amount_cents: i64,
        payment_id: String,
    },
    PaymentFailed {
        payment_id: String,
        subscription_id: Option<String>,
        reason: String,
    },
}
```

---

## §8 Models (DTOs)

### §8.1 Request Types

```rust
// src/billing/models.rs

use serde::Deserialize;

/// Body for POST /v1/billing/subscription (Phase 2)
#[derive(Debug, Deserialize)]
pub struct CreateSubscriptionCommand {
    pub billing_interval: String,      // "monthly" | "annual"
    pub payment_method_id: String,     // Hyperswitch payment method ID
}

/// Body for PATCH /v1/billing/subscription (Phase 2)
#[derive(Debug, Deserialize)]
pub struct UpdateSubscriptionCommand {
    pub billing_interval: String,      // new billing interval
}

/// Body for POST /v1/billing/coppa-verify
#[derive(Debug, Deserialize)]
pub struct CoppaVerificationCommand {
    pub payment_method_id: String,     // Hyperswitch payment method ID
}

/// Body for POST /v1/billing/payment-methods (Phase 2)
#[derive(Debug, Deserialize)]
pub struct AttachPaymentMethodCommand {
    pub setup_intent_client_secret: String,
}

/// Query for POST /v1/billing/subscription/estimate (Phase 2)
#[derive(Debug, Deserialize)]
pub struct EstimateSubscriptionQuery {
    pub billing_interval: String,      // desired interval
}

/// Query parameters for GET /v1/billing/transactions
#[derive(Debug, Deserialize)]
pub struct TransactionListParams {
    pub cursor: Option<String>,
    pub limit: Option<u8>,             // Default 20, max 100
}

/// Query parameters for GET /v1/billing/invoices (Phase 2)
#[derive(Debug, Deserialize)]
pub struct InvoiceListParams {
    pub cursor: Option<String>,
    pub limit: Option<u8>,
}

/// Query parameters for GET /v1/billing/payouts (Phase 2)
#[derive(Debug, Deserialize)]
pub struct PayoutListParams {
    pub cursor: Option<String>,
    pub limit: Option<u8>,
}
```

### §8.2 Response Types

```rust
use serde::Serialize;

/// Subscription status response.
#[derive(Debug, Serialize)]
pub struct SubscriptionResponse {
    pub tier: String,                  // "free" | "premium"
    pub status: Option<String>,        // null if free tier (no subscription)
    pub billing_interval: Option<String>,
    pub current_period_end: Option<DateTime<Utc>>,
    pub cancel_at_period_end: bool,
    pub amount_cents: Option<i64>,
    pub currency: Option<String>,
}

/// A single financial transaction.
#[derive(Debug, Serialize)]
pub struct TransactionResponse {
    pub id: Uuid,
    pub transaction_type: String,
    pub status: String,
    pub amount_cents: i64,
    pub currency: String,
    pub description: Option<String>,
    pub created_at: DateTime<Utc>,
}

/// Paginated transaction list.
#[derive(Debug, Serialize)]
pub struct TransactionListResponse {
    pub transactions: Vec<TransactionResponse>,
    pub next_cursor: Option<String>,
}

/// Subscription invoice (Phase 2).
#[derive(Debug, Serialize)]
pub struct InvoiceResponse {
    pub id: String,
    pub amount_cents: i64,
    pub currency: String,
    pub status: String,
    pub period_start: DateTime<Utc>,
    pub period_end: DateTime<Utc>,
    pub paid_at: Option<DateTime<Utc>>,
    pub pdf_url: Option<String>,
}

/// Paginated invoice list (Phase 2).
#[derive(Debug, Serialize)]
pub struct InvoiceListResponse {
    pub invoices: Vec<InvoiceResponse>,
    pub next_cursor: Option<String>,
}

/// Attached payment method (Phase 2).
#[derive(Debug, Serialize)]
pub struct PaymentMethodResponse {
    pub id: String,
    pub method_type: String,
    pub last_four: Option<String>,
    pub brand: Option<String>,
    pub exp_month: Option<u8>,
    pub exp_year: Option<u16>,
    pub is_default: bool,
}

/// Pricing estimate for subscription changes (Phase 2).
#[derive(Debug, Serialize)]
pub struct EstimateResponse {
    pub amount_cents: i64,
    pub currency: String,
    pub billing_interval: String,
    pub proration_credits_cents: i64,
    pub total_due_today_cents: i64,
    pub next_billing_date: NaiveDate,
}

/// COPPA micro-charge verification result.
#[derive(Debug, Serialize)]
pub struct CoppaVerificationResult {
    pub verified: bool,
    pub charge_id: String,
    pub refund_id: String,
}

/// Creator payout record (Phase 2).
#[derive(Debug, Serialize)]
pub struct PayoutResponse {
    pub id: Uuid,
    pub status: String,
    pub amount_cents: i64,
    pub currency: String,
    pub period_start: DateTime<Utc>,
    pub period_end: DateTime<Utc>,
    pub purchase_count: i32,
    pub refund_deduction_cents: i64,
    pub processed_at: Option<DateTime<Utc>>,
    pub created_at: DateTime<Utc>,
}

/// Paginated payout list (Phase 2).
#[derive(Debug, Serialize)]
pub struct PayoutListResponse {
    pub payouts: Vec<PayoutResponse>,
    pub next_cursor: Option<String>,
}
```

### §8.3 Config

```rust
/// Runtime configuration for billing domain.
/// All sensitive values loaded from environment variables.
pub struct BillingConfig {
    /// Hyperswitch API key (billing profile).
    pub hyperswitch_api_key: String,
    /// Hyperswitch billing profile ID (separate from mkt:: profile).
    pub hyperswitch_profile_id: String,
    /// Hyperswitch base URL (same instance as mkt::).
    pub hyperswitch_base_url: String,
    /// Hyperswitch price ID for monthly premium plan.
    pub monthly_price_id: String,
    /// Hyperswitch price ID for annual premium plan.
    pub annual_price_id: String,
    /// COPPA micro-charge amount in cents (default: 50 = $0.50).
    pub coppa_charge_cents: i64,
    /// Webhook signing secret for signature verification.
    pub webhook_signing_secret: String,
}
```

---

## §9 Subscription Lifecycle & State Machine

### State Transitions

Hyperswitch is authoritative for subscription state. The local `bill_subscriptions` table
mirrors Hyperswitch via webhook events. Each webhook event maps to a local status update and
optionally publishes a domain event.

```
                                  ┌──────────────┐
                                  │   [none]     │  ← All families start here (free tier)
                                  └──────┬───────┘
                                         │ POST /v1/billing/subscription
                                         ▼
                                  ┌──────────────┐
                                  │  incomplete   │  ← Payment pending
                                  └──────┬───────┘
                                         │ invoice.paid webhook
                                         ▼
                                  ┌──────────────┐
                              ┌──▶│   active      │◀─┐
                              │   └──┬───┬───┬───┘   │
                              │      │   │   │        │
                  resume      │      │   │   │        │ retry succeeds
                              │      │   │   │        │
                   ┌──────────┘      │   │   │   ┌────┴─────┐
                   │                 │   │   │   │ past_due  │ ← payment.failed webhook
                   │                 │   │   │   └───────────┘
              ┌────┴────┐            │   │   │
              │ paused  │◀───────────┘   │   └────────┐
              └─────────┘   pause        │            │ cancel_at_period_end
                                         │            │ + period_end reached
                                         │            ▼
                                         │     ┌────────────┐
                                         └────▶│  canceled   │
                                   immediate   └────────────┘
                                   cancel
```

### Webhook Event → Status Mapping

| Hyperswitch Webhook Event | Local Status Update | Domain Event Published |
|--------------------------|--------------------|-----------------------|
| `subscription.created` | Set status = `incomplete` | — |
| `subscription.updated` (status → active) | Set status = `active`, update period | `SubscriptionCreated` (first activation) or `SubscriptionChanged` |
| `subscription.updated` (cancel_at_period_end) | Set `cancel_at_period_end = true` | — (pending cancellation, not yet effective) |
| `subscription.deleted` | Set status = `canceled` | `SubscriptionCancelled` |
| `invoice.paid` | Create `bill_transactions` row | `SubscriptionChanged` (renewal) |
| `payment.failed` | Set status = `past_due` | — (internal — notify:: can subscribe if needed) |

### Downgrade Behavior `[S§15.3]`

When `SubscriptionCancelled` fires at the end of the billing period:
1. `billing::` publishes `SubscriptionCancelled { family_id, effective_at }`
2. `iam::` event handler sets `iam_families.subscription_tier = 'free'` `[01-iam §13.3, line 1431]`
3. Per `[S§15.3]`: data is preserved, premium tools become read-only, already-generated
   reports stay downloadable, AI recommendations are disabled
4. Feature gating is per-domain via `RequirePremium` extractor — no code in `billing::`

---

## §10 Pricing & Tier System

### Two Tiers

| Tier | Price | Features |
|------|-------|----------|
| **Free** (default) | $0 | Social, basic learning tools, marketplace access, methodology education, discovery, onboarding, data export `[S§15.1]` |
| **Premium** | ~$10-15/month, ~$96-144/year (~20-25% annual discount) | Compliance reporting, portfolio/transcript generation, advanced analytics, AI recommendations, enhanced storage, advanced mastery paths, methodology-specific tools `[S§15.2, S§20.2]` |

### Pricing Configuration

Prices are configured as Hyperswitch `item_price_id` values — **not hardcoded in application
code**. This allows price changes without deployments.

```rust
// BillingConfig (from environment)
pub monthly_price_id: String,    // e.g., "price_monthly_premium_v1"
pub annual_price_id: String,     // e.g., "price_annual_premium_v1"
```

The service resolves `billing_interval` to the appropriate `price_id`:
```rust
fn resolve_price_id(&self, interval: &str) -> Result<&str, BillingError> {
    match interval {
        "monthly" => Ok(&self.config.monthly_price_id),
        "annual" => Ok(&self.config.annual_price_id),
        _ => Err(BillingError::InvalidBillingInterval),
    }
}
```

### Feature Gating

`billing::` does NOT enforce feature gates. It only keeps `subscription_tier` in sync via
domain events. Feature checks are per-domain using the `RequirePremium` extractor defined in
`00-core`. Each domain decides which endpoints require premium. `[00-core §13]`

---

## §11 Upgrade/Downgrade & Proration

### Monthly → Annual (Upgrade within Same Tier)

1. Family calls `PATCH /v1/billing/subscription` with `billing_interval: "annual"`
2. `billing::` calls `adapter.update_subscription(sub_id, annual_price_id)`
3. Hyperswitch applies `CreditOption::Prorate` — credits remaining monthly period, charges
   prorated annual amount
4. Webhook confirms update → local mirror updated → `SubscriptionChanged` event

### Annual → Monthly (Downgrade within Same Tier)

1. Same flow as above with `billing_interval: "monthly"`
2. Hyperswitch prorates credit from remaining annual period

### Premium → Free (Downgrade Tier)

1. Family calls `DELETE /v1/billing/subscription`
2. `billing::` calls `adapter.cancel_subscription(sub_id)` with `CancelOption::EndOfTerm`
3. Subscription remains active until `current_period_end`
4. At period end, Hyperswitch fires `subscription.deleted` webhook
5. `billing::` publishes `SubscriptionCancelled` → `iam::` sets tier to `free`

### Pricing Preview

The `POST /v1/billing/subscription/estimate` endpoint calls
`adapter.estimate_subscription()` to preview exactly what the family will be charged,
including proration credits. This should be called before any plan change to show the user
the impact.

---

## §12 Creator Payout Aggregation (Phase 2)

### Earnings Calculation

Creator earnings are computed from `mkt_purchases`:

```sql
-- Unpaid earnings = total creator_payout_cents - refund deductions - already-paid payouts
SELECT
    SUM(CASE WHEN refunded_at IS NULL THEN creator_payout_cents ELSE 0 END)
    - COALESCE(SUM(refund_amount_cents), 0)
    - (SELECT COALESCE(SUM(amount_cents), 0) FROM bill_payouts
       WHERE creator_id = $1 AND status IN ('completed', 'processing'))
    AS unpaid_earnings_cents
FROM mkt_purchases
WHERE creator_id = $1;
```

### AggregatePayoutsJob

Runs monthly (1st of each month at 6:00 AM UTC). For each creator with unpaid earnings above
the minimum threshold:

1. Calculate unpaid earnings since last payout
2. Deduct any refunds in the period
3. If amount ≥ minimum payout threshold → create `bill_payouts` row with status `pending`
4. Payout execution is handled by a separate `ExecutePayoutsJob` that processes pending rows

### Payout Execution

Payouts go to creator sub-merchant accounts managed by `mkt::`. The payout call uses `mkt::`'s
Hyperswitch profile (since creator sub-merchant accounts are registered under `mkt::`'s profile).

```
AggregatePayoutsJob → creates bill_payouts (pending)
ExecutePayoutsJob   → calls adapter.create_payout() → updates bill_payouts (processing → completed/failed)
```

---

## §13 COPPA Micro-Charge Verification

### Flow `[S§1.4]`

COPPA parental consent verification via credit card micro-charge. Called by `iam::` service
during the COPPA consent flow `[01-iam §10.2]`.

```
1. Parent enters payment details in frontend
2. Frontend calls POST /v1/billing/coppa-verify with payment_method_id
3. billing::service:
   a. Get or create Hyperswitch customer for the family
   b. Call adapter.process_micro_charge($0.50)
      → Hyperswitch charges $0.50 via one-time payment (not subscription)
      → Hyperswitch immediately refunds $0.50
   c. Create bill_transactions row: { type: coppa_charge, status: succeeded }
   d. Create bill_transactions row: { type: coppa_refund, status: succeeded }
   e. Return CoppaVerificationResult { verified: true, charge_id, refund_id }
4. iam:: receives success → updates coppa_consent_status to 'verified'
```

### Error Handling

| Failure | Response | Recovery |
|---------|----------|----------|
| Card declined | `422 PaymentDeclined` | Parent tries different card |
| Charge succeeds but refund fails | Log error, return success (charge was verified) | Refund retried via background job |
| Hyperswitch unreachable | `502 PaymentAdapterUnavailable` | Parent retries later |

### Configuration

```rust
pub coppa_charge_cents: i64,       // default: 50 ($0.50)
```

The COPPA charge amount is configurable via `BillingConfig` but defaults to $0.50 as required
by FTC guidelines for credit card verification as a COPPA consent mechanism.

---

## §14 Webhook Processing & Idempotency

### Webhook Endpoint

`POST /hooks/hyperswitch/billing` receives all subscription-related webhook events from
Hyperswitch. This is a separate endpoint from `mkt::`'s webhook receiver
(`POST /hooks/hyperswitch/marketplace`) — Hyperswitch routes events to the appropriate
endpoint based on the business profile.

### Processing Flow

```
1. Receive raw payload + signature header
2. adapter.verify_webhook(payload, signature)
   → Reject with 200 (log warning) if signature invalid — do NOT return 4xx to avoid retries
3. adapter.parse_webhook_event(payload)
4. Check idempotency:
   → Extract event_id from payload metadata
   → If transaction_repo.exists_by_payment_id(payment_id, type) → skip (already processed)
5. Match event type → handler:
   → SubscriptionCreated  → upsert bill_subscriptions, publish event
   → SubscriptionUpdated  → update bill_subscriptions, publish event
   → SubscriptionDeleted  → update bill_subscriptions status=canceled, publish SubscriptionCancelled
   → InvoicePaid          → create bill_transactions row
   → PaymentFailed        → update bill_subscriptions status=past_due, log
6. Always return 200 OK
```

### Idempotency

Webhook idempotency is enforced via the unique index on `bill_transactions(hyperswitch_payment_id, transaction_type)`. If a webhook is delivered multiple times (Hyperswitch retry), the duplicate transaction insert fails and the event is safely skipped.

For subscription status updates (no transaction), idempotency is enforced by comparing the
incoming status against the current local status — if unchanged, the update is a no-op.

### Handled Webhook Events

| Hyperswitch Event | Action |
|-------------------|--------|
| `subscription.created` | Create/update `bill_subscriptions` row |
| `subscription.updated` | Update `bill_subscriptions` fields (status, period, cancel_at_period_end) |
| `subscription.deleted` | Set `bill_subscriptions.status = 'canceled'`, publish `SubscriptionCancelled` |
| `invoice.paid` | Create `bill_transactions` row (type: `subscription_payment`) |
| `invoice.upcoming` | Phase 3: publish event for `notify::` to send advance renewal notice `[S§15.3]` |
| `payment.failed` | Update `bill_subscriptions.status = 'past_due'` |

---

## §15 Error Types

All billing errors use `thiserror` and map to HTTP status codes via `AppError`. Internal
details are logged but never exposed in API responses. `[CODING §5.2, S§18]`

```rust
// src/billing/errors.rs

use thiserror::Error;

#[derive(Debug, Error)]
pub enum BillingError {
    // ─── Subscription Errors ────────────────────────────────────────────

    #[error("Subscription not found")]
    SubscriptionNotFound,

    #[error("Subscription already exists for this family")]
    SubscriptionAlreadyExists,

    #[error("Cannot reactivate subscription in current state")]
    CannotReactivate,

    #[error("Subscription is not active")]
    SubscriptionNotActive,

    #[error("Invalid billing interval")]
    InvalidBillingInterval,

    // ─── Payment Method Errors ──────────────────────────────────────────

    #[error("Payment method not found")]
    PaymentMethodNotFound,

    #[error("Cannot remove last payment method with active subscription")]
    CannotRemoveLastPaymentMethod,

    // ─── Payment Errors ─────────────────────────────────────────────────

    #[error("Payment was declined")]
    PaymentDeclined,

    #[error("COPPA verification failed")]
    CoppaVerificationFailed,

    // ─── Adapter Errors ─────────────────────────────────────────────────

    #[error("Payment adapter unavailable")]
    PaymentAdapterUnavailable,

    #[error("Invalid webhook signature")]
    InvalidWebhookSignature,

    // ─── Infrastructure ─────────────────────────────────────────────────

    #[error("Database error")]
    DbError(#[from] sea_orm::DbErr),   // internal — NOT exposed in API

    #[error("Adapter error")]
    AdapterError(String),              // internal — NOT exposed in API
}
```

### Error-to-HTTP Mapping

| Error Variant | HTTP Status | Response Code | User-Facing Message |
|---------------|-------------|---------------|---------------------|
| `SubscriptionNotFound` | `404 Not Found` | `subscription_not_found` | "Subscription not found" |
| `SubscriptionAlreadyExists` | `409 Conflict` | `subscription_exists` | "A subscription already exists for this family" |
| `CannotReactivate` | `409 Conflict` | `cannot_reactivate` | "Subscription cannot be reactivated in its current state" |
| `SubscriptionNotActive` | `409 Conflict` | `subscription_not_active` | "Subscription is not currently active" |
| `InvalidBillingInterval` | `422 Unprocessable` | `invalid_billing_interval` | "Invalid billing interval — must be 'monthly' or 'annual'" |
| `PaymentMethodNotFound` | `404 Not Found` | `payment_method_not_found` | "Payment method not found" |
| `CannotRemoveLastPaymentMethod` | `409 Conflict` | `cannot_remove_last_payment_method` | "Cannot remove the only payment method while a subscription is active" |
| `PaymentDeclined` | `422 Unprocessable` | `payment_declined` | "Payment was declined — please try a different payment method" |
| `CoppaVerificationFailed` | `422 Unprocessable` | `coppa_verification_failed` | "Parental verification failed — please try again" |
| `PaymentAdapterUnavailable` | `502 Bad Gateway` | `payment_adapter_unavailable` | "Payment service is temporarily unavailable" |
| `InvalidWebhookSignature` | `200 OK` (logged) | — | — (never exposed — webhook always returns 200) |
| `DbError` | `500 Internal` | `internal_error` | "An unexpected error occurred" |
| `AdapterError` | `500 Internal` | `internal_error` | "An unexpected error occurred" |

> **Security note**: `InvalidWebhookSignature` returns `200 OK` (not `401`) to prevent
> Hyperswitch from retrying invalid webhooks. The failure is logged with `tracing::warn!`
> for monitoring.

---

## §16 Cross-Domain Interactions

### §16.1 billing:: Provides (Consumed by Other Domains)

| Export | Consumers | Mechanism |
|--------|-----------|-----------|
| `BillingService::process_coppa_verification()` | `iam::` | `Arc<dyn BillingService>` via AppState — COPPA consent flow `[01-iam §10.2]` |
| `SubscriptionCreated` event | `iam::`, `notify::` | Domain event — tier sync, welcome email |
| `SubscriptionChanged` event | `iam::`, `notify::` | Domain event — tier sync (if needed), plan change notification |
| `SubscriptionCancelled` event | `iam::`, `notify::` | Domain event — set tier=free, cancellation notification |
| `PayoutCompleted` event (Phase 2) | `notify::` | Domain event — payout confirmation notification |

### §16.2 billing:: Consumes

| Dependency | Source | Purpose |
|-----------|--------|---------|
| `AuthContext` | `iam::` middleware | User identity on every request `[00-core §7.2]` |
| `FamilyScope` | `iam::` middleware | Family-scoped data access `[00-core §8]` |
| `IamService::get_family_primary_email()` | `iam::` | Email for Hyperswitch customer creation |
| `SubscriptionPaymentAdapter` | Hyperswitch (self-hosted) | Subscription + payment orchestration `[ARCH §2.9]` |
| `FamilyDeletionScheduled` event | `iam::` | Cancel subscription on family deletion `[01-iam §13.3]` |
| `PrimaryParentTransferred` event | `iam::` | Update Hyperswitch customer email `[01-iam §13.3]` |
| `PurchaseCompleted` event (Phase 2) | `mkt::` | Record creator earnings for payout aggregation `[07-mkt §18.3]` |
| `PurchaseRefunded` event (Phase 2) | `mkt::` | Deduct refund from creator earnings `[07-mkt §18.3]` |

### §16.3 Events billing:: Publishes

Defined in `src/billing/events.rs`. `[CODING §8.4]`

```rust
// src/billing/events.rs

use crate::shared::types::FamilyId;
use chrono::{DateTime, Utc};

/// Published when a family's subscription becomes active for the first time.
/// Consumed by iam:: (set tier=premium) and notify:: (welcome email).
#[derive(Clone, Debug)]
pub struct SubscriptionCreated {
    pub family_id: FamilyId,
    pub tier: String,                   // "premium"
    pub billing_interval: String,       // "monthly" | "annual"
    pub current_period_end: DateTime<Utc>,
}
impl DomainEvent for SubscriptionCreated {}

/// Published when a subscription is modified (interval change, renewal, reactivation).
/// Consumed by iam:: (update tier if changed) and notify:: (plan change notification).
#[derive(Clone, Debug)]
pub struct SubscriptionChanged {
    pub family_id: FamilyId,
    pub tier: String,
    pub billing_interval: String,
    pub current_period_end: DateTime<Utc>,
    pub change_type: String,           // "interval_change" | "renewal" | "reactivation"
}
impl DomainEvent for SubscriptionChanged {}

/// Published when a subscription is fully canceled (end of term reached).
/// Consumed by iam:: (set tier=free) and notify:: (cancellation confirmation).
///
/// IMPORTANT: This event fires at the END of the billing period, not when
/// the family requests cancellation. Between cancel request and period end,
/// the subscription remains active. [S§15.3]
#[derive(Clone, Debug)]
pub struct SubscriptionCancelled {
    pub family_id: FamilyId,
    pub effective_at: DateTime<Utc>,   // when the cancellation took effect
}
impl DomainEvent for SubscriptionCancelled {}

/// Published when a creator payout is completed. (Phase 2)
/// Consumed by notify:: (payout confirmation notification).
#[derive(Clone, Debug)]
pub struct PayoutCompleted {
    pub creator_id: Uuid,
    pub payout_id: Uuid,
    pub amount_cents: i64,
    pub currency: String,
}
impl DomainEvent for PayoutCompleted {}
```

### §16.4 Events billing:: Subscribes To

| Event | Source | Effect |
|-------|--------|--------|
| `FamilyDeletionScheduled { family_id, delete_after }` | `iam::` | Cancel subscription immediately in Hyperswitch (no end-of-term wait). Delete local records. `[01-iam §13.3]` |
| `PrimaryParentTransferred { family_id, old_primary, new_primary }` | `iam::` | Update Hyperswitch customer email to new primary parent's email. `[01-iam §13.3]` |
| `PurchaseCompleted { family_id, purchase_id, listing_id, content_metadata }` | `mkt::` (Phase 2) | No-op in billing:: Phase 1. Phase 2: record creator earnings for payout aggregation. `[07-mkt §18.3]` |
| `PurchaseRefunded { purchase_id, listing_id, family_id, refund_amount_cents }` | `mkt::` (Phase 2) | No-op in billing:: Phase 1. Phase 2: deduct refund from creator's unpaid earnings. `[07-mkt §18.3]` |

```rust
// src/billing/event_handlers.rs

use crate::iam::events::{FamilyDeletionScheduled, PrimaryParentTransferred};
use crate::mkt::events::{PurchaseCompleted, PurchaseRefunded};

// ─── iam:: events ─────────────────────────────────────────────────────

pub struct FamilyDeletionScheduledHandler {
    billing_service: Arc<dyn BillingService>,
}

#[async_trait]
impl DomainEventHandler<FamilyDeletionScheduled> for FamilyDeletionScheduledHandler {
    async fn handle(&self, event: &FamilyDeletionScheduled) -> Result<(), AppError> {
        self.billing_service.handle_family_deletion_scheduled(event).await
    }
}

pub struct PrimaryParentTransferredHandler {
    billing_service: Arc<dyn BillingService>,
}

#[async_trait]
impl DomainEventHandler<PrimaryParentTransferred> for PrimaryParentTransferredHandler {
    async fn handle(&self, event: &PrimaryParentTransferred) -> Result<(), AppError> {
        self.billing_service.handle_primary_parent_transferred(event).await
    }
}

// ─── mkt:: events (Phase 2) ──────────────────────────────────────────

pub struct PurchaseCompletedHandler {
    billing_service: Arc<dyn BillingService>,
}

#[async_trait]
impl DomainEventHandler<PurchaseCompleted> for PurchaseCompletedHandler {
    async fn handle(&self, event: &PurchaseCompleted) -> Result<(), AppError> {
        self.billing_service.handle_purchase_completed(event).await
    }
}

pub struct PurchaseRefundedHandler {
    billing_service: Arc<dyn BillingService>,
}

#[async_trait]
impl DomainEventHandler<PurchaseRefunded> for PurchaseRefundedHandler {
    async fn handle(&self, event: &PurchaseRefunded) -> Result<(), AppError> {
        self.billing_service.handle_purchase_refunded(event).await
    }
}
```

### §16.5 iam_families.subscription_tier Sync

The `subscription_tier` column on `iam_families` `[01-iam §3.2, line 110-111]` is owned by
`iam::` but kept in sync by `billing::` events:

| billing:: Event | iam:: Action |
|----------------|-------------|
| `SubscriptionCreated` | Set `subscription_tier = 'premium'` |
| `SubscriptionChanged` | Update `subscription_tier` if tier changed |
| `SubscriptionCancelled` | Set `subscription_tier = 'free'` |

This respects bounded context ownership: `iam::` owns the column and decides how to react
to billing events. `billing::` publishes facts; `iam::` interprets them. `[01-iam §13.3, line 1431]`

---

## §17 Phase Scope

### Phase 1 — Foundation `[S§19, ARCH §15.9]`

**In scope**:
- 4 database tables (`bill_hyperswitch_customers`, `bill_subscriptions`,
  `bill_transactions`, `bill_payouts`)
- 4 API endpoints (get subscription, COPPA verify, webhook receiver, transaction history)
- Hyperswitch customer creation (for COPPA micro-charge)
- COPPA micro-charge adapter ($0.50 charge + refund)
- Webhook skeleton (signature verification, event parsing, idempotency)
- Free tier default for all families
- 2 event handlers (`FamilyDeletionScheduled`, `PrimaryParentTransferred`)
- Domain event definitions (`SubscriptionCreated`, `SubscriptionChanged`,
  `SubscriptionCancelled`) — published via webhook processing but no premium subscriptions
  to trigger them yet
- **4 endpoints, 4 tables, 2 event handlers**

**Out of scope for Phase 1**: Premium subscriptions, payment method management, upgrade/downgrade
flows, invoices, creator payouts, subscription estimate, `PurchaseCompleted` / `PurchaseRefunded`
handlers.

### Phase 2 — Premium Subscriptions & Payouts `[S§19]`

**In scope**:
- Premium subscription CRUD (create, update, cancel, reactivate)
- Payment method management (attach, list, detach via SetupIntent)
- Subscription estimate / pricing preview
- Invoice listing
- Creator payout aggregation (`AggregatePayoutsJob`, `ExecutePayoutsJob`)
- Creator payout history endpoint
- Full webhook lifecycle (all subscription events)
- `PurchaseCompleted` and `PurchaseRefunded` event handlers
- Pause/resume subscription
- **~10 additional endpoints, 2 additional event handlers, 2 background jobs**

### Phase 3 — Metering & Tax `[S§19]`

**In scope**:
- Storage quota enforcement for `media::` (per-tier limits)
- Usage metering (if usage-based pricing tiers added)
- Tax dashboard / tax reporting
- Advance renewal notification (`invoice.upcoming` webhook → `notify::`)
- Subscription analytics dashboard

---

## §18 Verification Checklist

Each item is a testable assertion. Implementation is not complete until all assertions pass.

### Subscription CRUD

1. `GET /v1/billing/subscription` returns `{ tier: "free", status: null }` for families with
   no subscription
2. `GET /v1/billing/subscription` returns full subscription details for families with an active
   subscription
3. `POST /v1/billing/subscription` creates a Hyperswitch subscription and local mirror
   (Phase 2)
4. `POST /v1/billing/subscription` returns `409` if family already has a subscription (Phase 2)
5. `PATCH /v1/billing/subscription` updates billing interval with proration (Phase 2)
6. `DELETE /v1/billing/subscription` sets `cancel_at_period_end = true` (does not immediately
   cancel) (Phase 2)
7. `POST /v1/billing/subscription/reactivate` clears pending cancellation (Phase 2)
8. `POST /v1/billing/subscription/reactivate` returns `409` if no pending cancellation (Phase 2)
9. Only `PrimaryParent` role can create, update, cancel, or reactivate subscriptions (Phase 2)

### COPPA Flow

10. `POST /v1/billing/coppa-verify` charges $0.50 and immediately refunds
11. COPPA verification creates two `bill_transactions` rows (charge + refund)
12. COPPA verification creates Hyperswitch customer if not exists
13. Failed COPPA charge returns `422 PaymentDeclined`
14. Hyperswitch unavailable during COPPA returns `502 PaymentAdapterUnavailable`

### Webhook Processing

15. Webhook with valid signature is processed successfully
16. Webhook with invalid signature returns `200` (not 4xx) and is logged
17. Duplicate webhook (same `event_id`) is idempotent — no duplicate transactions created
18. `subscription.updated` (active) publishes `SubscriptionCreated` on first activation
19. `subscription.deleted` publishes `SubscriptionCancelled` event
20. `invoice.paid` creates a `bill_transactions` row with type `subscription_payment`

### Event Publishing & Subscription

21. `SubscriptionCreated` event is consumed by `iam::` to set `subscription_tier = 'premium'`
22. `SubscriptionCancelled` event is consumed by `iam::` to set `subscription_tier = 'free'`
23. `FamilyDeletionScheduled` event triggers immediate subscription cancellation in Hyperswitch
24. `PrimaryParentTransferred` event updates Hyperswitch customer email

### Family-Scoping

25. Every query on `bill_subscriptions`, `bill_transactions`, and `bill_hyperswitch_customers`
    is family-scoped via `FamilyScope` `[CODING §2.4]`
26. `UNIQUE(family_id)` on `bill_subscriptions` enforces one subscription per family
27. `bill_payouts` is creator-scoped — creator ownership is verified before returning data

### Payment Methods (Phase 2)

28. `POST /v1/billing/payment-methods` creates a SetupIntent for client-side confirmation
29. `GET /v1/billing/payment-methods` returns all attached methods for the family
30. `DELETE /v1/billing/payment-methods/:id` removes a payment method
31. Cannot remove last payment method with active subscription → `409`

### Error Handling

32. Zero `.unwrap()` / `.expect()` in production code `[CODING §2.2]`
33. All errors use `BillingError` with `thiserror` `[CODING §2.2, §5.2]`
34. Internal error details (adapter errors, DB errors) are logged but never exposed in API
    responses `[CODING §2.2, §5.2]`
35. All `BillingError` variants map to documented HTTP status codes

---

## §19 Module Structure

```
src/billing/
├── mod.rs                    # Re-exports, domain-level doc comments
├── handlers.rs               # 4 Phase 1 Axum route handlers (thin layer only)
│                             #   get_subscription, coppa_verify,
│                             #   hyperswitch_webhook, list_transactions
├── service.rs                # BillingServiceImpl — subscription lifecycle,
│                             #   COPPA verification, webhook processing,
│                             #   event handling
├── repository.rs             # PgSubscriptionRepo, PgTransactionRepo,
│                             #   PgCustomerRepo, PgPayoutRepo (Phase 2)
│                             #   All user-data queries family-scoped via FamilyScope
├── models.rs                 # Request/response types (serde + OpenAPI derives),
│                             #   internal types (BillingConfig, etc.)
├── ports.rs                  # BillingService trait, all repository traits,
│                             #   SubscriptionPaymentAdapter trait
├── errors.rs                 # BillingError thiserror enum
├── events.rs                 # SubscriptionCreated, SubscriptionChanged,
│                             #   SubscriptionCancelled, PayoutCompleted
│                             #   [ARCH §4.6]
├── event_handlers.rs         # 4 DomainEventHandler structs (one per
│                             #   subscribed event type) [ARCH §4.6]
├── jobs.rs                   # AggregatePayoutsJob (Phase 2),
│                             #   ExecutePayoutsJob (Phase 2) [ARCH §12]
├── adapters/
│   ├── mod.rs                # Adapter trait re-exports
│   └── hyperswitch.rs        # HyperswitchSubscriptionAdapter — wraps Hyperswitch
│                             #   subscription + payment REST API [ARCH §2.9]
└── entities/                 # SeaORM-generated — never hand-edit [CODING §6.3]
```

> **Complexity class**: Simple (no `domain/` subdirectory). `billing::` delegates subscription
> state management to Hyperswitch — no complex domain invariants that warrant aggregate models.
> `[ARCH §4.5]`
