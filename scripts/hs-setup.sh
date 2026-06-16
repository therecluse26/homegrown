#!/usr/bin/env bash
# Bootstrap Hyperswitch for local development.
# Creates a merchant account, billing and marketplace business profiles,
# attaches the dummy connector to each, and creates a sub-merchant for the seed creator.
#
# Prerequisites: docker compose up hyperswitch-server (wait for it to be healthy)
#
# Usage:
#   bash scripts/hs-setup.sh
#   HS_URL=http://localhost:8080 bash scripts/hs-setup.sh

set -euo pipefail

HS_URL="${HS_URL:-http://localhost:8280}"
ADMIN_API_KEY="${HS_ADMIN_API_KEY:-test_admin}"
SEED_CREATOR_ID="01900000-0000-7000-8000-000000000202"
MERCHANT_ID="homegrown_ag"

# Database URL used to update mkt_creators.payment_account_id for the seed creator.
# Defaults to agent DB; override with DATABASE_URL env var if needed.
DB_URL="${DATABASE_URL:-postgres://homegrown:homegrown@localhost:5932/homegrown_agent}"

die() { echo "ERROR: $*" >&2; exit 1; }

require_cmd() {
    command -v "$1" >/dev/null 2>&1 || die "$1 is required but not found in PATH"
}

require_cmd curl
require_cmd python3

echo "=== Hyperswitch Local Setup ==="
echo "URL: $HS_URL"
echo ""

# ── Wait for health ──────────────────────────────────────────────────────────
echo "Waiting for Hyperswitch to be ready..."
for i in $(seq 1 60); do
    if curl -sf "$HS_URL/health" 2>/dev/null | grep -qi 'good\|running'; then
        echo "Hyperswitch is ready."
        break
    fi
    if [ "$i" -eq 60 ]; then
        die "Hyperswitch did not become healthy after 60 seconds. Is 'docker compose up hyperswitch-server' running?"
    fi
    sleep 1
done

# ── Helper: API call ─────────────────────────────────────────────────────────
hs_post() {
    local path="$1" api_key="$2" body="$3"
    curl -sf -X POST "$HS_URL$path" \
        -H "Content-Type: application/json" \
        -H "api-key: $api_key" \
        -d "$body"
}

hs_get() {
    local path="$1" api_key="$2"
    curl -sf "$HS_URL$path" -H "api-key: $api_key"
}

extract() {
    python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('$1',''))"
}

# ── Create (or fetch) merchant account ───────────────────────────────────────
echo "Creating merchant account '$MERCHANT_ID'..."
MERCHANT_RESP=$(hs_post "/accounts" "$ADMIN_API_KEY" "{
    \"merchant_id\": \"$MERCHANT_ID\",
    \"merchant_name\": \"Homegrown Academy Dev\",
    \"return_url\": \"http://localhost:5673\"
}" 2>/dev/null || true)

# Create an API key for the merchant (returned only once — idempotent per run)
echo "Creating API key for merchant '$MERCHANT_ID'..."
API_KEY_RESP=$(hs_post "/api_keys/$MERCHANT_ID" "$ADMIN_API_KEY" '{
    "name": "dev-key",
    "description": "Local dev API key",
    "expiration": "never"
}' 2>/dev/null || true)

MERCHANT_API_KEY=$(echo "$API_KEY_RESP" | extract "api_key" 2>/dev/null || true)
[ -n "$MERCHANT_API_KEY" ] || die "Failed to create merchant API key. Is Hyperswitch healthy? Response: $API_KEY_RESP"
echo "  Merchant API key: $MERCHANT_API_KEY"

# ── Create billing business profile ─────────────────────────────────────────
echo "Creating billing business profile..."
BILLING_PROFILE_RESP=$(hs_post "/account/$MERCHANT_ID/business_profile" "$ADMIN_API_KEY" '{
    "profile_name": "billing",
    "return_url": "http://localhost:5673/billing/return",
    "webhook_details": {
        "webhook_url": "http://host.docker.internal:3500/api/billing/webhook",
        "webhook_version": "1.0.0",
        "payment_created_enabled": true,
        "payment_succeeded_enabled": true,
        "payment_failed_enabled": true
    }
}' 2>/dev/null || true)

BILLING_PROFILE_ID=$(echo "$BILLING_PROFILE_RESP" | extract "profile_id" 2>/dev/null || true)
[ -n "$BILLING_PROFILE_ID" ] || die "Failed to create billing profile. Response: $BILLING_PROFILE_RESP"
echo "  Billing profile ID: $BILLING_PROFILE_ID"

# ── Create marketplace business profile ──────────────────────────────────────
echo "Creating marketplace business profile..."
MKT_PROFILE_RESP=$(hs_post "/account/$MERCHANT_ID/business_profile" "$ADMIN_API_KEY" '{
    "profile_name": "marketplace",
    "return_url": "http://localhost:5673/marketplace/return",
    "webhook_details": {
        "webhook_url": "http://host.docker.internal:3500/api/marketplace/webhook",
        "webhook_version": "1.0.0",
        "payment_created_enabled": true,
        "payment_succeeded_enabled": true,
        "payment_failed_enabled": true
    }
}' 2>/dev/null || true)

MKT_PROFILE_ID=$(echo "$MKT_PROFILE_RESP" | extract "profile_id" 2>/dev/null || true)
[ -n "$MKT_PROFILE_ID" ] || die "Failed to create marketplace profile. Response: $MKT_PROFILE_RESP"
echo "  Marketplace profile ID: $MKT_PROFILE_ID"

# ── Attach dummy connector to each profile ───────────────────────────────────
echo "Attaching stripe_test connector to billing profile..."
hs_post "/account/$MERCHANT_ID/connectors" "$ADMIN_API_KEY" "{
    \"connector_type\": \"payment_processor\",
    \"connector_name\": \"stripe_test\",
    \"connector_account_details\": {\"auth_type\": \"HeaderKey\", \"api_key\": \"sk_test_homegrown_local_dev\"},
    \"test_mode\": true,
    \"disabled\": false,
    \"profile_id\": \"$BILLING_PROFILE_ID\",
    \"payment_methods_enabled\": [
        {\"payment_method\": \"card\", \"payment_method_types\": [
            {\"payment_method_type\": \"credit\", \"card_networks\": [\"Visa\", \"Mastercard\"], \"minimum_amount\": 1, \"maximum_amount\": 999999, \"recurring_enabled\": true, \"installment_payment_enabled\": false}
        ]}
    ]
}" > /dev/null 2>&1 || echo "  (billing connector may already exist, skipping)"

echo "Attaching stripe_test connector to marketplace profile..."
hs_post "/account/$MERCHANT_ID/connectors" "$ADMIN_API_KEY" "{
    \"connector_type\": \"payment_processor\",
    \"connector_name\": \"stripe_test\",
    \"connector_account_details\": {\"auth_type\": \"HeaderKey\", \"api_key\": \"sk_test_homegrown_local_dev\"},
    \"test_mode\": true,
    \"disabled\": false,
    \"profile_id\": \"$MKT_PROFILE_ID\",
    \"payment_methods_enabled\": [
        {\"payment_method\": \"card\", \"payment_method_types\": [
            {\"payment_method_type\": \"credit\", \"card_networks\": [\"Visa\", \"Mastercard\"], \"minimum_amount\": 1, \"maximum_amount\": 999999, \"recurring_enabled\": true, \"installment_payment_enabled\": false}
        ]}
    ]
}" > /dev/null 2>&1 || echo "  (marketplace connector may already exist, skipping)"

# ── Create sub-merchant for seed creator ─────────────────────────────────────
SEED_CREATOR_MERCHANT_ID="creator_01900000_0000_7000_8000_000000000202"
echo "Creating sub-merchant for seed creator..."
SUB_MERCHANT_RESP=$(hs_post "/accounts" "$ADMIN_API_KEY" "{
    \"merchant_id\": \"$SEED_CREATOR_MERCHANT_ID\",
    \"merchant_name\": \"Seed Homeschool Press\",
    \"metadata\": {\"creator_id\": \"$SEED_CREATOR_ID\", \"env\": \"dev\"}
}" 2>/dev/null || true)

SUB_MERCHANT_ID=$(echo "$SUB_MERCHANT_RESP" | extract "merchant_id" 2>/dev/null || true)

if [ -n "$SUB_MERCHANT_ID" ]; then
    echo "  Sub-merchant ID: $SUB_MERCHANT_ID"
    echo "  Updating mkt_creators.payment_account_id in database..."
    # Use docker exec if psql not on host
    DB_HOST=$(echo "$DB_URL" | python3 -c "import sys; from urllib.parse import urlparse; u=urlparse(sys.stdin.read().strip()); print(u.hostname)")
    DB_PORT=$(echo "$DB_URL" | python3 -c "import sys; from urllib.parse import urlparse; u=urlparse(sys.stdin.read().strip()); print(u.port or 5432)")
    DB_USER=$(echo "$DB_URL" | python3 -c "import sys; from urllib.parse import urlparse; u=urlparse(sys.stdin.read().strip()); print(u.username)")
    DB_NAME=$(echo "$DB_URL" | python3 -c "import sys; from urllib.parse import urlparse; u=urlparse(sys.stdin.read().strip()); print(u.path.lstrip('/'))")
    SQL="UPDATE mkt_creators SET payment_account_id = '$SUB_MERCHANT_ID' WHERE id = '$SEED_CREATOR_ID';"
    if command -v psql >/dev/null 2>&1; then
        psql "$DB_URL" -c "$SQL" 2>/dev/null && echo "  Database updated." || echo "  WARNING: psql update failed. Update manually."
    elif command -v docker >/dev/null 2>&1; then
        # Find a running postgres container that can reach our app DB
        APP_PG=$(docker ps --format '{{.Names}}' | grep -E 'postgres|pg' | grep -v 'hs-postgres' | head -1 || true)
        if [ -n "$APP_PG" ]; then
            docker exec -i "$APP_PG" psql -U "$DB_USER" "$DB_NAME" -h "$DB_HOST" -p "$DB_PORT" -c "$SQL" \
                2>/dev/null && echo "  Database updated." || echo "  WARNING: docker psql update failed. Update manually."
        else
            echo "  WARNING: No suitable postgres container found. Update manually:"
            echo "    $SQL"
        fi
    else
        echo "  WARNING: psql not available. Update mkt_creators manually:"
        echo "    $SQL"
    fi
else
    echo "  WARNING: Could not create sub-merchant. You can set payment_account_id manually after Hyperswitch is running."
fi

# ── Print env vars ────────────────────────────────────────────────────────────
WEBHOOK_SECRET="dev-hs-webhook-secret"
echo ""
echo "╔══════════════════════════════════════════════════════════════╗"
echo "║  Add these to your .env                                      ║"
echo "╚══════════════════════════════════════════════════════════════╝"
echo ""
echo "HYPERSWITCH_BASE_URL=$HS_URL"
echo "HYPERSWITCH_API_KEY=$MERCHANT_API_KEY"
echo "HYPERSWITCH_WEBHOOK_KEY=$WEBHOOK_SECRET"
echo "HYPERSWITCH_BILLING_PROFILE_ID=$BILLING_PROFILE_ID"
echo "HYPERSWITCH_MKT_PROFILE_ID=$MKT_PROFILE_ID"
echo "HYPERSWITCH_MONTHLY_PRICE_ID=monthly_plan"
echo "HYPERSWITCH_ANNUAL_PRICE_ID=annual_plan"
echo "BILLING_WEBHOOK_SECRET=$WEBHOOK_SECRET"
echo ""
echo "Setup complete."
