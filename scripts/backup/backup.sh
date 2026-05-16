#!/usr/bin/env bash
# Automated daily pg_dump backup: gzip | openssl-encrypt | S3 upload.
# Sends a Slack or email alert on completion/failure.
# All configuration via environment variables — see .env.backup.example.
set -euo pipefail

# ── Configuration (override via env) ───────────────────────────────────────
PGHOST="${PGHOST:-localhost}"
PGPORT="${PGPORT:-5432}"
PGUSER="${PGUSER:-homegrown}"
PGPASSWORD="${PGPASSWORD:?PGPASSWORD must be set}"
PGDATABASE="${PGDATABASE:-homegrown}"

BACKUP_ENCRYPTION_PASSPHRASE="${BACKUP_ENCRYPTION_PASSPHRASE:?BACKUP_ENCRYPTION_PASSPHRASE must be set}"
BACKUP_S3_BUCKET="${BACKUP_S3_BUCKET:?BACKUP_S3_BUCKET must be set}"
BACKUP_S3_PREFIX="${BACKUP_S3_PREFIX:-homegrown-backups}"
BACKUP_S3_ENDPOINT="${BACKUP_S3_ENDPOINT:-}"  # Set for R2/GCS; leave empty for AWS S3
BACKUP_RETENTION_DAYS="${BACKUP_RETENTION_DAYS:-30}"
BACKUP_TMPDIR="${BACKUP_TMPDIR:-/tmp}"

SLACK_WEBHOOK_URL="${SLACK_WEBHOOK_URL:-}"
ALERT_EMAIL="${ALERT_EMAIL:-}"

# ── Helpers ─────────────────────────────────────────────────────────────────
TIMESTAMP=$(date -u +"%Y%m%dT%H%M%SZ")
BACKUP_NAME="${PGDATABASE}-${TIMESTAMP}.sql.gz.enc"
BACKUP_KEY="${BACKUP_S3_PREFIX}/${BACKUP_NAME}"

export PGPASSWORD

log() { echo "[$(date -u +"%Y-%m-%dT%H:%M:%SZ")] $*" >&2; }

alert() {
  local status="$1" message="$2"
  if [[ -n "$SLACK_WEBHOOK_URL" ]]; then
    curl -fsS -X POST "$SLACK_WEBHOOK_URL" \
      -H 'Content-Type: application/json' \
      -d "{\"text\":\"[homegrown-backup] ${status}: ${message}\"}" || true
  fi
  if [[ -n "$ALERT_EMAIL" ]] && command -v mail &>/dev/null; then
    echo "$message" | mail -s "[homegrown-backup] ${status}" "$ALERT_EMAIL" || true
  fi
}

cleanup() {
  local exit_code=$?
  if [[ $exit_code -ne 0 ]]; then
    log "ERROR: backup failed (exit $exit_code)"
    alert "FAILURE" "Backup of ${PGDATABASE} at ${TIMESTAMP} FAILED. Check server logs."
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

# ── Dependency check ────────────────────────────────────────────────────────
for cmd in pg_dump gzip openssl aws; do
  command -v "$cmd" &>/dev/null || { log "ERROR: required command not found: $cmd"; exit 1; }
done

# ── Backup ───────────────────────────────────────────────────────────────────
log "Starting backup: ${PGDATABASE} → s3://${BACKUP_S3_BUCKET}/${BACKUP_KEY}"

# Stream: pg_dump → gzip → openssl AES-256-CBC encrypt → S3
# Plaintext data never touches disk.
pg_dump \
  -h "$PGHOST" -p "$PGPORT" -U "$PGUSER" \
  --format=plain --no-password \
  "$PGDATABASE" \
  | gzip \
  | openssl enc -aes-256-cbc -pbkdf2 -iter 600000 \
      -pass pass:"$BACKUP_ENCRYPTION_PASSPHRASE" \
  | aws_s3 cp - "s3://${BACKUP_S3_BUCKET}/${BACKUP_KEY}" \
      --storage-class STANDARD_IA \
      --expected-size 0

log "Upload complete: ${BACKUP_KEY}"

# ── Retention: purge backups older than BACKUP_RETENTION_DAYS ───────────────
log "Applying ${BACKUP_RETENTION_DAYS}-day retention policy..."

CUTOFF=$(date -u -d "${BACKUP_RETENTION_DAYS} days ago" +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null \
  || date -u -v "-${BACKUP_RETENTION_DAYS}d" +"%Y-%m-%dT%H:%M:%SZ")  # BSD date fallback

aws_s3 ls "s3://${BACKUP_S3_BUCKET}/${BACKUP_S3_PREFIX}/" \
  | awk '{print $1" "$2" "$4}' \
  | while read -r date time key; do
      file_ts="${date}T${time}Z"
      if [[ "$file_ts" < "$CUTOFF" ]]; then
        log "Deleting expired backup: ${key}"
        aws_s3 rm "s3://${BACKUP_S3_BUCKET}/${BACKUP_S3_PREFIX}/${key}"
      fi
    done

log "Retention sweep complete."

# ── Success alert ─────────────────────────────────────────────────────────────
alert "SUCCESS" "Backup of ${PGDATABASE} completed: ${BACKUP_KEY}"
log "Backup finished successfully."
