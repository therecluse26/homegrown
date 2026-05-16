#!/usr/bin/env bash
# Automated restore drill: downloads the latest backup, restores it into a
# temporary database, runs data-integrity checks, and reports pass/fail.
# Safe: never touches the live database.
# Usage: restore-drill.sh [backup-key]
#   If backup-key is omitted, uses the most recent object in BACKUP_S3_PREFIX.
set -euo pipefail

PGHOST="${PGHOST:-localhost}"
PGPORT="${PGPORT:-5432}"
PGUSER="${PGUSER:-homegrown}"
PGPASSWORD="${PGPASSWORD:?PGPASSWORD must be set}"
PGDATABASE="${PGDATABASE:-homegrown}"

BACKUP_ENCRYPTION_PASSPHRASE="${BACKUP_ENCRYPTION_PASSPHRASE:?BACKUP_ENCRYPTION_PASSPHRASE must be set}"
BACKUP_S3_BUCKET="${BACKUP_S3_BUCKET:?BACKUP_S3_BUCKET must be set}"
BACKUP_S3_PREFIX="${BACKUP_S3_PREFIX:-homegrown-backups}"
BACKUP_S3_ENDPOINT="${BACKUP_S3_ENDPOINT:-}"

SLACK_WEBHOOK_URL="${SLACK_WEBHOOK_URL:-}"
ALERT_EMAIL="${ALERT_EMAIL:-}"

DRILL_DB="${DRILL_DB:-homegrown_restore_drill}"
BACKUP_KEY="${1:-}"

export PGPASSWORD

log()  { echo "[$(date -u +"%Y-%m-%dT%H:%M:%SZ")] $*" >&2; }
pass() { echo "[PASS] $*"; }
fail() { echo "[FAIL] $*"; DRILL_FAILED=1; }

DRILL_FAILED=0
TIMESTAMP=$(date -u +"%Y%m%dT%H%M%SZ")

alert() {
  local status="$1" message="$2"
  if [[ -n "$SLACK_WEBHOOK_URL" ]]; then
    curl -fsS -X POST "$SLACK_WEBHOOK_URL" \
      -H 'Content-Type: application/json' \
      -d "{\"text\":\"[homegrown-restore-drill] ${status}: ${message}\"}" || true
  fi
  if [[ -n "$ALERT_EMAIL" ]] && command -v mail &>/dev/null; then
    echo "$message" | mail -s "[homegrown-restore-drill] ${status}" "$ALERT_EMAIL" || true
  fi
}

cleanup() {
  local exit_code=$?
  log "Dropping drill database '${DRILL_DB}'..."
  PGPASSWORD="$PGPASSWORD" psql \
    -h "$PGHOST" -p "$PGPORT" -U "$PGUSER" -d postgres \
    -c "DROP DATABASE IF EXISTS \"${DRILL_DB}\";" 2>/dev/null || true
  if [[ $DRILL_FAILED -ne 0 || $exit_code -ne 0 ]]; then
    alert "FAILURE" "Restore drill at ${TIMESTAMP} FAILED. Manual investigation required."
    exit 1
  fi
}
trap cleanup EXIT

aws_s3() {
  if [[ -n "$BACKUP_S3_ENDPOINT" ]]; then
    aws --endpoint-url "$BACKUP_S3_ENDPOINT" s3 "$@"
  else
    aws s3 "$@"
  fi
}

psql_drill() {
  PGPASSWORD="$PGPASSWORD" psql \
    -h "$PGHOST" -p "$PGPORT" -U "$PGUSER" -d "$DRILL_DB" \
    -tAq "$@"
}

for cmd in pg_dump psql gzip openssl aws; do
  command -v "$cmd" &>/dev/null || { log "ERROR: required command not found: $cmd"; exit 1; }
done

# ── Resolve backup key ───────────────────────────────────────────────────────
if [[ -z "$BACKUP_KEY" ]]; then
  log "No backup key specified; finding most recent object in s3://${BACKUP_S3_BUCKET}/${BACKUP_S3_PREFIX}/"
  BACKUP_KEY=$(aws_s3 ls "s3://${BACKUP_S3_BUCKET}/${BACKUP_S3_PREFIX}/" \
    | sort | tail -1 | awk '{print $4}')
  [[ -n "$BACKUP_KEY" ]] || { log "ERROR: no backups found in bucket"; exit 1; }
  BACKUP_KEY="${BACKUP_S3_PREFIX}/${BACKUP_KEY}"
fi

log "=== RESTORE DRILL: ${BACKUP_KEY} → ${DRILL_DB} ==="

# ── Create drill database ────────────────────────────────────────────────────
PGPASSWORD="$PGPASSWORD" psql \
  -h "$PGHOST" -p "$PGPORT" -U "$PGUSER" -d postgres <<SQL
SELECT pg_terminate_backend(pid)
  FROM pg_stat_activity
 WHERE datname = '${DRILL_DB}' AND pid <> pg_backend_pid();
DROP DATABASE IF EXISTS "${DRILL_DB}";
CREATE DATABASE "${DRILL_DB}" OWNER "${PGUSER}";
SQL

# ── Restore from backup ──────────────────────────────────────────────────────
log "Streaming restore from S3..."
aws_s3 cp "s3://${BACKUP_S3_BUCKET}/${BACKUP_KEY}" - \
  | openssl enc -d -aes-256-cbc -pbkdf2 -iter 600000 \
      -pass pass:"$BACKUP_ENCRYPTION_PASSPHRASE" \
  | gzip -d \
  | PGPASSWORD="$PGPASSWORD" psql \
      -h "$PGHOST" -p "$PGPORT" -U "$PGUSER" -d "$DRILL_DB" \
      --single-transaction -v ON_ERROR_STOP=1 -q
log "Restore stream complete."

# ── Data integrity checks ────────────────────────────────────────────────────
log "Running integrity checks..."

# 1. Core tables are non-empty
for table in families students methodologies; do
  count=$(psql_drill -c "SELECT COUNT(*) FROM ${table};")
  if [[ "$count" -gt 0 ]]; then
    pass "Table '${table}' has ${count} rows"
  else
    fail "Table '${table}' is EMPTY — data missing from backup"
  fi
done

# 2. Foreign key integrity: no orphaned students
orphaned=$(psql_drill -c "
  SELECT COUNT(*) FROM students s
   WHERE NOT EXISTS (SELECT 1 FROM families f WHERE f.id = s.family_id);
")
if [[ "$orphaned" -eq 0 ]]; then
  pass "No orphaned students (family_id FK intact)"
else
  fail "Found ${orphaned} orphaned students — FK integrity broken"
fi

# 3. Schema sanity: goose_db_version table exists and has rows
migration_count=$(psql_drill -c "SELECT COUNT(*) FROM goose_db_version;" 2>/dev/null || echo 0)
if [[ "$migration_count" -gt 0 ]]; then
  pass "goose_db_version has ${migration_count} migration records"
else
  fail "goose_db_version missing or empty — schema may be incomplete"
fi

# 4. No NULL ids in families (primary key sanity)
null_ids=$(psql_drill -c "SELECT COUNT(*) FROM families WHERE id IS NULL;")
if [[ "$null_ids" -eq 0 ]]; then
  pass "families.id: no NULL primary keys"
else
  fail "families.id: found ${null_ids} NULL primary key rows"
fi

# 5. Cross-check row counts against live database
log "Cross-checking row counts against live '${PGDATABASE}' database..."
for table in families students; do
  live_count=$(PGPASSWORD="$PGPASSWORD" psql \
    -h "$PGHOST" -p "$PGPORT" -U "$PGUSER" -d "$PGDATABASE" \
    -tAq -c "SELECT COUNT(*) FROM ${table};")
  drill_count=$(psql_drill -c "SELECT COUNT(*) FROM ${table};")
  if [[ "$live_count" -eq "$drill_count" ]]; then
    pass "Row count match: ${table} (${drill_count} rows)"
  else
    fail "Row count MISMATCH: ${table} live=${live_count} drill=${drill_count}"
  fi
done

# ── Summary ──────────────────────────────────────────────────────────────────
if [[ "$DRILL_FAILED" -eq 0 ]]; then
  log "=== RESTORE DRILL PASSED ==="
  alert "SUCCESS" "Restore drill passed at ${TIMESTAMP}. Backup key: ${BACKUP_KEY}"
else
  log "=== RESTORE DRILL FAILED — see output above ==="
  # cleanup trap sends the failure alert
fi
