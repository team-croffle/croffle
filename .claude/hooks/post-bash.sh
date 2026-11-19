#!/bin/sh
# Claude Code PostToolUse hook for Bash: after a commit, run the gates on it so every commit has its log
# (.loctx/reports/gates/<tree>.log). A failure is reported back to Claude; the commit stays.
# The gates check HEAD only: make one commit per command. Earlier commits of the same command that have
# no log are named in the message so that they can be filled (`run-gates.sh audit` on each).
cmd=$(jq -r '.tool_input.command // empty')
printf '%s' "$cmd" | grep -qE '(^|[;&|(])[[:space:]]*git([[:space:]]+-C[[:space:]]+[^[:space:]]+)?[[:space:]]+commit' || exit 0
root=$(git rev-parse --show-toplevel 2>/dev/null) || exit 0
cd "$root" || exit 0
sh .agents/harness/run-gates.sh post-commit >/dev/null 2>&1 || {
  echo "gates failed on the new commit: see .loctx/reports/gates/$(git rev-parse HEAD^{tree}).log" >&2
  exit 2
}
missing=""
for c in $(git rev-list --max-count=5 HEAD~1 2>/dev/null); do
  [ -f ".loctx/reports/gates/$(git rev-parse "$c^{tree}").log" ] && break
  git merge-base --is-ancestor "$c" origin/master 2>/dev/null && break
  missing="$missing $(git rev-parse --short "$c")"
done
if [ -n "$missing" ]; then
  echo "commits without a gate log:$missing — one commit per command; fill them with run-gates.sh audit" >&2
  exit 2
fi
exit 0
