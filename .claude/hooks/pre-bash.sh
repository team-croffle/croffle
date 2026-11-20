#!/bin/sh
# Claude Code PreToolUse hook for Bash. Exit 2 refuses the command; stderr goes back to Claude.
# - A merge needs a review of the head it is bound to (premerge-review-check.sh).
# - A push runs the harness gates; a push of master also needs a review of what it sends.
# bash_command.py splits the command the way the shell does and allows only exact forms of a merge
# and a push; anything else that is one is refused rather than guessed.
refuse() { echo "hook: $1" >&2; exit 2; }
command -v python3 >/dev/null 2>&1 || refuse "python3 is missing; the merge and push checks need it"
root=$(git rev-parse --show-toplevel 2>/dev/null) || exit 0
hooks="$root/.claude/hooks"
verdict=$(python3 "$hooks/bash_command.py") || refuse "the command check failed to run"
set -f
set -- $verdict
case "$1" in
  none | commit) exit 0 ;;
  refuse) shift; refuse "$*" ;;
  merge)
    sh "$hooks/premerge-review-check.sh" pr "$2" "${3:-}" || exit 2 ;;
  push)
    cd "$root" || exit 2
    if [ "$3" = master ] || [ "$3" = refs/heads/master ]; then
      sh "$hooks/premerge-review-check.sh" master "$3" || exit 2
    fi
    # Commits that change DESIGN.md name the spec items they were compared with (amended and
    # rebased commits too, which the commit-msg hook does not see).
    if [ -f scripts/check-design-commit-msg.mjs ]; then
      node scripts/check-design-commit-msg.mjs --range "origin/master..$3" || exit 2
    fi
    sh .agents/harness/run-gates.sh pre-push >/dev/null 2>&1 ||
      refuse "gates failed before push: see .loctx/reports/gates/$(git rev-parse HEAD^{tree}).log" ;;
  *) refuse "unexpected answer from the command check: $verdict" ;;
esac
exit 0
