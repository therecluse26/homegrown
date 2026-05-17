#!/usr/bin/env bash
# check-dev-secrets.sh — fail if any known dev-default secret placeholder is present.
#
# Usage:
#   scripts/check-dev-secrets.sh                     # scans .env.example in repo root
#   scripts/check-dev-secrets.sh path/to/.env        # scans a specific env file
#   scripts/check-dev-secrets.sh --env-vars           # checks live environment variables
#
# Exit codes: 0 = clean, 1 = at least one bad value found.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# ─── Known dev-default values ───────────────────────────────────────────────
# Add new entries here whenever a placeholder is retired from .env.example.
declare -A BAD_DEFAULTS=(
  ["AUTH_WEBHOOK_SECRET"]="dev-webhook-secret-change-in-production"
  ["UNSUBSCRIBE_SECRET"]="dev-unsubscribe-secret-change-in-production"
)

FOUND=0

check_file() {
  local file="$1"
  if [[ ! -f "$file" ]]; then
    echo "check-dev-secrets: file not found: $file" >&2
    exit 1
  fi
  for var in "${!BAD_DEFAULTS[@]}"; do
    bad="${BAD_DEFAULTS[$var]}"
    # Match lines like VAR=bad_value (with optional surrounding whitespace/quotes)
    if grep -qE "^[[:space:]]*${var}[[:space:]]*=[[:space:]]*['\"]?${bad}['\"]?" "$file" 2>/dev/null; then
      echo "FAIL: $file contains known dev-default for $var" >&2
      FOUND=1
    fi
  done
}

check_env_vars() {
  for var in "${!BAD_DEFAULTS[@]}"; do
    bad="${BAD_DEFAULTS[$var]}"
    val="${!var:-}"
    if [[ "$val" == "$bad" ]]; then
      echo "FAIL: live env var $var is set to known dev-default value" >&2
      FOUND=1
    fi
  done
}

MODE="file"
TARGET="$REPO_ROOT/.env.example"

if [[ "${1:-}" == "--env-vars" ]]; then
  MODE="env"
elif [[ -n "${1:-}" ]]; then
  TARGET="$1"
fi

if [[ "$MODE" == "env" ]]; then
  check_env_vars
else
  check_file "$TARGET"
fi

if [[ "$FOUND" -eq 0 ]]; then
  echo "check-dev-secrets: OK — no known dev-default values detected"
  exit 0
else
  echo "check-dev-secrets: FAILED — replace the above values with real secrets before deploying" >&2
  echo "See docs/secrets-checklist.md for the full inventory and rotation guidance." >&2
  exit 1
fi
