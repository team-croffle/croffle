#!/bin/sh
# Claude Code PostToolUse hook for Bash: after a commit, run the gates on it so every commit has its log
# (.loctx/reports/gates/<tree>.log). A failure is reported back to Claude; the commit stays.
cmd=$(jq -r '.tool_input.command // empty')
printf '%s' "$cmd" | grep -qE '(^|[;&|(])[[:space:]]*git([[:space:]]+-C[[:space:]]+[^[:space:]]+)?[[:space:]]+commit' || exit 0
root=$(git rev-parse --show-toplevel 2>/dev/null) || exit 0
sh "$root/.agents/harness/run-gates.sh" post-commit >/dev/null 2>&1 || {
  echo "gates failed on the new commit: see .loctx/reports/gates/$(git -C "$root" rev-parse HEAD^{tree}).log" >&2
  exit 2
}
exit 0
