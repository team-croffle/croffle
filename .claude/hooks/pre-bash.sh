#!/bin/sh
# Claude Code PreToolUse hook for Bash. Exit 2 refuses the command; stderr goes back to Claude.
# - A merge needs a review of the head it is bound to (premerge-review-check.sh).
# - A push runs the harness gates; a push to master also needs a review of what it sends.
# A command that mentions a merge or a push in a form this hook cannot read safely is refused rather
# than guessed: run the merge or the push alone, in the form given in the message.
refuse() { echo "hook: $1" >&2; exit 2; }
command -v jq >/dev/null 2>&1 || refuse "jq is missing; the merge and push checks need it"
input=$(cat)
cmd=$(printf '%s' "$input" | jq -er '.tool_input.command | strings' 2>/dev/null) ||
  refuse "cannot read the Bash command from the hook input"
root=$(git rev-parse --show-toplevel 2>/dev/null) || exit 0
hooks="$root/.claude/hooks"
sp='[[:space:]]'
# Detection reads the command with quotes and backslashes removed, so a quoted word cannot hide a
# merge or a push. Between "git" and "push" (or "gh", "pr" and "merge") only option words — starting
# with "-", with whatever follows them — may stand, so a commit message that mentions "push" is not
# taken for one. The exact-form checks below read the command as written and refuse quotes.
norm=$(printf '%s' "$cmd" | tr -d "'\"\\\\")

# --- merge ------------------------------------------------------------------------------------
if printf '%s\n' "$norm" | grep -qE "(^|[^[:alnum:]_-])gh($sp+-[^;&|]*)?$sp+pr($sp+-[^;&|]*)?$sp+merge($sp|$)"; then
  form="gh pr merge <n> [--rebase|--squash|--merge] [--delete-branch] --match-head-commit <full head SHA>"
  printf '%s\n' "$cmd" | grep -qxE "$sp*gh$sp+pr$sp+merge$sp+[0-9]+($sp+(--rebase|--squash|--merge|--delete-branch|-r|-s|-m|-d|--match-head-commit$sp+[0-9a-f]{40}))*$sp*" ||
    refuse "a merge must be one command of the form: $form (to write text that mentions it, use the Write tool)"
  [ "$(printf '%s\n' "$cmd" | wc -l)" -le 1 ] || refuse "a merge must be one command on one line: $form"
  pr=$(printf '%s' "$cmd" | sed -E "s/^$sp*gh$sp+pr$sp+merge$sp+([0-9]+).*/\1/")
  bound=$(printf '%s' "$cmd" | sed -nE "s/.*--match-head-commit$sp+([0-9a-f]{40}).*/\1/p")
  sh "$hooks/premerge-review-check.sh" pr "$pr" "$bound" || exit 2
fi

# --- push -------------------------------------------------------------------------------------
if printf '%s\n' "$norm" | grep -qE "(^|[^[:alnum:]_-])git($sp+-[^;&|]*)?$sp+push($sp|$)"; then
  form="git push [-u] [<remote> [<refspec>...]] from the repository root"
  printf '%s\n' "$cmd" | grep -qxE "$sp*git($sp+-c$sp+[^[:space:];&|<>\`\$()'\"\\]+)*$sp+push($sp+[^[:space:];&|<>\`\$()'\"\\*]+)*$sp*" ||
    refuse "a push must be one command of the form: $form (no -C, no cd, nothing chained)"
  [ "$(printf '%s\n' "$cmd" | wc -l)" -le 1 ] || refuse "a push must be one command on one line: $form"
  cd "$root" || exit 2
  # The words after "push" that are not options: the remote, then refspecs.
  words=$(printf '%s' "$cmd" | sed -E "s/^.*$sp+push//")
  remote=""
  src=""
  nrefs=0
  for w in $words; do
    case "$w" in
      --all | --mirror | --tags | --prune | --follow-tags)
        refuse "push $w can send master without a check; push named refspecs instead: $form" ;;
      -d | --delete) deleting=1; continue ;;
      -*) continue ;;
    esac
    if [ -z "$remote" ]; then
      remote=$w
      continue
    fi
    nrefs=$((nrefs + 1))
    ref=${w#+}
    case "$ref" in
      :) refuse "a matching push (:) can send master without a check: $form" ;;
      *:*) s=${ref%%:*} d=${ref#*:} ;;
      *) s=$ref d=$ref ;;
    esac
    case "$d" in
      master | refs/heads/master | HEAD:master)
        if [ -n "${deleting:-}" ] || [ -z "$s" ]; then
          refuse "deleting master is a red line (rewriting the default branch)"
        fi
        src=$s ;;
    esac
  done
  # No refspec: git pushes the current branch to its upstream (push.default); master is either.
  if [ -z "$src" ] && [ "$nrefs" -eq 0 ]; then
    up=$(git rev-parse --abbrev-ref --symbolic-full-name '@{u}' 2>/dev/null || true)
    if [ "$(git branch --show-current)" = master ] || [ "${up##*/}" = master ]; then
      src=HEAD
    fi
  fi
  if [ -n "$src" ]; then
    sh "$hooks/premerge-review-check.sh" master "$src" || exit 2
  fi
  sh .agents/harness/run-gates.sh pre-push >/dev/null 2>&1 ||
    refuse "gates failed before push: see .loctx/reports/gates/$(git rev-parse HEAD^{tree}).log"
fi
exit 0
