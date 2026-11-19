#!/bin/sh
# Claude Code PostToolUse hook for Bash: after a commit, run the gates on it so every commit has its log
# (.loctx/reports/gates/<tree>.log). Problems go back to Claude (exit 2); the commit stays.
# The gates check HEAD only, so make one commit per command. Every commit of the branch (since
# origin/master) without a clean passing run is named, so that it can be filled with
# `sh .agents/harness/run-gates.sh audit` on that commit.
command -v jq >/dev/null 2>&1 || { echo "hook: jq is missing; the post-commit gates did not run" >&2; exit 2; }
cmd=$(jq -er '.tool_input.command | strings' 2>/dev/null) ||
  { echo "hook: cannot read the Bash command; the post-commit gates did not run" >&2; exit 2; }
printf '%s\n' "$cmd" | grep -qE '(^|[;&|(])[[:space:]]*git([[:space:]]+(-C|-c)[[:space:]]+[^[:space:]]+)*[[:space:]]+commit' || exit 0
root=$(git rev-parse --show-toplevel 2>/dev/null) || exit 0
cd "$root" || exit 0
sh .agents/harness/run-gates.sh post-commit >/dev/null 2>&1 || {
  echo "hook: gates failed on the new commit: see .loctx/reports/gates/$(git rev-parse HEAD^{tree}).log" >&2
  exit 2
}
# A commit counts as checked when its tree log holds a run on a clean work tree that ended with exit 0.
checked() {
  awk '/^== /{clean=($0 ~ /worktree: clean/)} /^exit: /{if (clean && $2==0) ok=1} END{exit ok?0:1}' "$1" 2>/dev/null
}
missing=""
for c in $(git rev-list origin/master..HEAD 2>/dev/null); do
  checked ".loctx/reports/gates/$(git rev-parse "$c^{tree}").log" || missing="$missing $(git rev-parse --short "$c")"
done
if [ -n "$missing" ]; then
  echo "hook: commits without a clean passing gate run:$missing — one commit per command; fill them with run-gates.sh audit" >&2
  exit 2
fi
exit 0
