#!/bin/sh
# Claude Code PreToolUse hook for Bash: a merge needs a review of its head, a push needs the gates.
# Exit 2 refuses the command; the message on stderr goes back to Claude.
cmd=$(jq -r '.tool_input.command // empty')
root=$(git rev-parse --show-toplevel 2>/dev/null) || exit 0
hooks="$root/.claude/hooks"
if printf '%s' "$cmd" | grep -qE '(^|[;&|(])[[:space:]]*gh[[:space:]]+pr[[:space:]]+merge'; then
  pr=$(printf '%s' "$cmd" | grep -oE 'gh[[:space:]]+pr[[:space:]]+merge[[:space:]]+[0-9]+' | head -1 | grep -oE '[0-9]+$')
  sh "$hooks/premerge-review-check.sh" pr "$pr" || exit 2
fi
if printf '%s' "$cmd" | grep -qE '(^|[;&|(])[[:space:]]*git([[:space:]]+-C[[:space:]]+[^[:space:]]+)?[[:space:]]+push'; then
  if [ "$(git -C "$root" branch --show-current)" = master ]; then
    sh "$hooks/premerge-review-check.sh" master || exit 2
  fi
  sh "$root/.agents/harness/run-gates.sh" pre-push >/dev/null 2>&1 || {
    echo "gates failed before push: see .loctx/reports/gates/$(git -C "$root" rev-parse HEAD^{tree}).log" >&2
    exit 2
  }
fi
exit 0
