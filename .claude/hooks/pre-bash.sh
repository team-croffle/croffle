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

# --- merge ------------------------------------------------------------------------------------
if printf '%s\n' "$cmd" | grep -qE "gh($sp+[^;&|]*)?$sp+pr$sp+merge"; then
  form="gh pr merge <n> [--rebase|--squash|--merge] [--delete-branch] --match-head-commit <full head SHA>"
  printf '%s\n' "$cmd" | grep -qxE "$sp*gh$sp+pr$sp+merge$sp+[0-9]+($sp+(--rebase|--squash|--merge|--delete-branch|-r|-s|-m|-d|--match-head-commit$sp+[0-9a-f]{40}))*$sp*" ||
    refuse "a merge must be one command of the form: $form (to write text that mentions it, use the Write tool)"
  [ "$(printf '%s\n' "$cmd" | wc -l)" -le 1 ] || refuse "a merge must be one command on one line: $form"
  pr=$(printf '%s' "$cmd" | sed -E "s/^$sp*gh$sp+pr$sp+merge$sp+([0-9]+).*/\1/")
  bound=$(printf '%s' "$cmd" | sed -nE "s/.*--match-head-commit$sp+([0-9a-f]{40}).*/\1/p")
  sh "$hooks/premerge-review-check.sh" pr "$pr" "$bound" || exit 2
fi

# --- push -------------------------------------------------------------------------------------
gitopts="($sp+(-C|-c)$sp+[^[:space:]]+|$sp+--[a-z-]+(=[^[:space:]]+)?)*"
if printf '%s\n' "$cmd" | grep -qE "git$gitopts$sp+push($sp|$)"; then
  form="git push [-u] [<remote> [<refspec>...]] from the repository root"
  printf '%s\n' "$cmd" | grep -qxE "$sp*git($sp+-c$sp+[^[:space:]]+)*$sp+push($sp+[^[:space:];&|<>\`\$()]+)*$sp*" ||
    refuse "a push must be one command of the form: $form (no -C, no cd, nothing chained)"
  [ "$(printf '%s\n' "$cmd" | wc -l)" -le 1 ] || refuse "a push must be one command on one line: $form"
  cd "$root" || exit 2
  # The words after "push" that are not options: the remote, then refspecs.
  words=$(printf '%s' "$cmd" | sed -E "s/^.*$sp+push//")
  remote=""
  src=""
  for w in $words; do
    case "$w" in
      -*) continue ;;
    esac
    if [ -z "$remote" ]; then
      remote=$w
      continue
    fi
    ref=${w#+}
    case "$ref" in
      *:*) s=${ref%%:*} d=${ref#*:} ;;
      *) s=$ref d=$ref ;;
    esac
    case "$d" in
      master | refs/heads/master) src=${s:-HEAD} ;;
    esac
  done
  if [ -z "$src" ] && [ -z "$(printf '%s' "$words" | awk '{n=0; for(i=1;i<=NF;i++) if ($i !~ /^-/) n++; print (n>1)?"x":""}')" ] &&
    [ "$(git branch --show-current)" = master ]; then
    src=HEAD
  fi
  if [ -n "$src" ]; then
    sh "$hooks/premerge-review-check.sh" master "$src" || exit 2
  fi
  sh .agents/harness/run-gates.sh pre-push >/dev/null 2>&1 ||
    refuse "gates failed before push: see .loctx/reports/gates/$(git rev-parse HEAD^{tree}).log"
fi
exit 0
