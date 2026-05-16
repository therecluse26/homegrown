#!/usr/bin/env bash
# Restore a pg_dump backup from S3 into a target PostgreSQL database.
# Usage: restore.sh <backup-key> [target-database]
#   backup-key: S3 object key, e.g. homegrown-backups/homegrown-20240101T120000Z.sql.gz.enc
#   target-database: optional override; defaults to $PGDATABASE
#
# WARNING: this will DROP and RECREATE the target database. Confirm before use.
set -euo pipefail

BACKUP_KEY="${1:?Usage: restore.sh <backup-s3-key> [target-database]}"
TARGET_DB="${2:-${PGDATABASE:-homegrown}}"

PGHOST="${PGHOST:-localhost}"
PGPORT="${PGPORT:-5432}"
PGUSER="${PGUSER:-homegrown}"
PGPASSWORD="${PGPASSWORD:?PGPASSWORD must be set}"
PGDATABASE="${PGDATABASE:-homegrown}"

BACKUP_ENCRYPTION_PASSPHRASE="${BACKUP_ENCRYPTION_PASSPHRASE:?BACKUP_ENCRYPTION_PASSPHRASE must be set}"
BACKUP_S3_BUCKET="${BACKUP_S3_BUCKET:?BACKUP_S3_BUCKET must be set}"
BACKUP_S3_ENDPOINT="${BACKUP_S3_ENDPOINT:-}"

export PGPASSWORD

log() { echo "[$(date -u +"%Y-%m-%dT%H:%M:%SZ")] $*" >&2; }

aws_s3() {
  if [[ -n "$BACKUP_S3_ENDPOINT" ]]; then
    aws --endpoint-url "$BACKUP_S3_ENDPOINT" s3 "$@"
  else
    aws s3 "$@"
  fi
}

for cmd in pg_dump psql gzip openssl aws; do
  command -v "$cmd" &>/dev/null || { log "ERROR: required command not found: $cmd"; exit 1; }
done

# Warn if restoring into the live production database
if [[ "$TARGET_DB" == "$PGDATABASE" && "${RESTORE_SKIP_CONFIRM:-}" != "1" ]]; then
  echo "WARNING: You are about to DROP and RECREATE database '${TARGET_DB}'."
  echo "         This is DESTRUCTIVE and IRREVERSIBLE."
  read -rp "Type the database name to confirm: " confirm
  [[ "$confirm" == "$TARGET_DB" ]] || { log "Aborted."; exit 1; }
fi

log "Restoring s3://${BACKUP_S3_BUCKET}/${BACKUP_KEY} → ${TARGET_DB}"

# Drop and recreate the target database
psql -h "$PGHOST" -p "$PGPORT" -U "$PGUSER" -d postgres <<SQL
SELECT pg_terminate_backend(pid)
  FROM pg_stat_activity
 WHERE datname = '${TARGET_DB}' AND pid <> pg_backend_pid();
DROP DATABASE IF EXISTS "${TARGET_DB}";
CREATE DATABASE "${TARGET_DB}" OWNER "${PGUSER}";
SQL

log "Database '${TARGET_DB}' recreated. Streaming restore..."

# Stream: S3 download → openssl decrypt → gunzip → psql
aws_s3 cp "s3://${BACKUP_S3_BUCKET}/${BACKUP_KEY}" - \
  | openssl enc -d -aes-256-cbc -pbkdf2 -iter 600000 \
      -pass pass:"$BACKUP_ENCRYPTION_PASSPHRASE" \
  | gzip -d \
  | psql -h "$PGHOST" -p "$PGPORT" -U "$PGUSER" -d "$TARGET_DB" --single-transaction -v ON_ERROR_STOP=1

log "Restore complete: ${TARGET_DB}"
