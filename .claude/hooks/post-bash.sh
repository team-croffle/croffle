#!/bin/sh
# Claude Code PostToolUse hook for Bash: after a commit, run the gates on it so every commit has its log
# (.loctx/reports/gates/<tree>.log). Problems go back to Claude (exit 2); the commit stays.
# The gates check HEAD only, so make one commit per command. Every commit of the branch (since
# origin/master) without a clean passing run is named, so that it can be filled with
# `sh .agents/harness/run-gates.sh audit` on that commit.
command -v python3 >/dev/null 2>&1 || { echo "hook: python3 is missing; the post-commit gates did not run" >&2; exit 2; }
root=$(git rev-parse --show-toplevel 2>/dev/null) || exit 0
cd "$root" || exit 0
verdict=$(python3 .claude/hooks/bash_command.py) || { echo "hook: the command check failed to run" >&2; exit 2; }
case "$verdict" in
  commit) ;;
  none | push\ * | merge\ *) exit 0 ;;
  *) echo "hook: the post-commit check could not read the command: $verdict" >&2; exit 2 ;;
esac
sh .agents/harness/run-gates.sh post-commit >/dev/null 2>&1 || {
  echo "hook: gates failed on the new commit: see .loctx/reports/gates/$(git rev-parse HEAD^{tree}).log" >&2
  exit 2
}
# A commit counts as checked when its tree log holds a run on a clean work tree that ended with exit 0.
checked() {
  awk '/^== /{clean=($0 ~ /worktree: clean/)} /^exit: /{if (clean && $2==0) ok=1} END{exit ok?0:1}' "$1" 2>/dev/null
}
missing=""
list=$(git rev-list origin/master..HEAD 2>&1) || {
  echo "hook: cannot list the commits since origin/master ($list); the gate-log check did not run" >&2
  exit 2
}
for c in $list; do
  checked ".loctx/reports/gates/$(git rev-parse "$c^{tree}").log" || missing="$missing $(git rev-parse --short "$c")"
done
if [ -n "$missing" ]; then
  echo "hook: commits without a clean passing gate run:$missing — one commit per command; fill them with run-gates.sh audit" >&2
  exit 2
fi
exit 0
