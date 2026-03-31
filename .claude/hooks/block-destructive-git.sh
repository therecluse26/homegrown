#!/usr/bin/env bash
# Hook: Block destructive git commands (PreToolUse → Bash)
# Allows read-only git operations; blocks anything that modifies repo state.
set -euo pipefail

INPUT=$(cat)
COMMAND=$(echo "$INPUT" | jq -r '.tool_input.command // empty')

# No command or not a git command → allow
[[ -z "$COMMAND" ]] && exit 0
echo "$COMMAND" | grep -qE '(^|\s|&&|\|\||;|`)git\s' || exit 0

# Allowlist: read-only git operations (exit 0 = allow)
if echo "$COMMAND" | grep -qP '(^|\s|&&|\|\||;|`)git\s+(status|log|diff|show|branch(?!\s+-[dD])|fetch|stash\s+(list|show|create)|rev-parse|ls-tree|ls-files|grep|shortlog|describe|name-rev|cat-file|for-each-ref|remote(?!\s+(add|remove|rename|set-url))|config\s+--get|blame|reflog|worktree\s+list|symbolic-ref|merge-base)(\s|$|"|'"'"')'; then
    exit 0
fi

# Blocklist: destructive/state-modifying git operations
if echo "$COMMAND" | grep -qP '(^|\s|&&|\|\||;|`)git\s+(commit|push|rebase|reset|checkout|restore|clean|branch\s+-[dD]|stash\s+(drop|clear|pop|apply)|merge|add|tag|cherry-pick|revert|am|format-patch|pull|remote\s+(add|remove|rename|set-url)|submodule|init|clone|mv|rm|bisect|switch|worktree\s+(add|remove|prune))(\s|$|"|'"'"')'; then
    echo '{"decision":"block","reason":"Destructive git command blocked by hook. Ask the user to run this command manually."}'
    exit 0
fi

# Unknown git subcommand → allow (be permissive for read operations)
exit 0
