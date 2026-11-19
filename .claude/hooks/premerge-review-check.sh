#!/bin/sh
# Refuses a merge (or a push to master) whose content was not reviewed.
# Usage: premerge-review-check.sh pr <number> <head SHA the merge is bound to>
#        premerge-review-check.sh master <commit that goes to master>
# The ledger (.loctx/reports/review-ledger.tsv) has one tab-separated line per review:
#   <pr number or "master">  <reviewed full SHA>  <review file>  <PASS|PASS-with-findings|FAIL>
# The last line for the target counts. Between the reviewed SHA and the head, only audit records
# (.agents/audit/*.md) may change; anything else needs another review first.
# Knowledge: .agents/knowledge/2026-11-15_review-before-merge.md
root=$(git rev-parse --show-toplevel) || exit 2
cd "$root" || exit 2
ledger="${REVIEW_LEDGER:-.loctx/reports/review-ledger.tsv}"
kind=$1
target=$2
bound=$3
mkdir -p .loctx/reports/premerge
if [ "$kind" = pr ]; then
  log=".loctx/reports/premerge/pr$target-$(date -u +%Y%m%dT%H%M%SZ).log"
else
  bound=$2
  target=master
  log=".loctx/reports/premerge/master-$(date -u +%Y%m%dT%H%M%SZ).log"
fi
fail() {
  { echo "refused: $1"; echo "exit=2"; } >>"$log"
  echo "premerge: $1 (log $log)" >&2
  exit 2
}
: >"$log"
echo "target: $target" >>"$log"
if [ "$kind" = pr ]; then
  printf '%s' "$target" | grep -qE '^[0-9]+$' || fail "name one PR number: gh pr merge <n> --match-head-commit <sha>"
  printf '%s' "$bound" | grep -qE '^[0-9a-f]{40}$' ||
    fail "bind the merge to the reviewed head: add --match-head-commit <full head SHA>"
  head=$(gh pr view "$target" --json headRefOid -q .headRefOid) || fail "cannot read PR #$target"
  [ "$head" = "$bound" ] || fail "PR #$target head is $head, not the --match-head-commit $bound"
else
  head=$(git rev-parse --verify --quiet "$bound^{commit}") || fail "cannot resolve $bound"
fi
echo "head: $head" >>"$log"
echo "ledger: $ledger" >>"$log"
[ -f "$ledger" ] || fail "no review ledger at $ledger"
line=$(awk -F'\t' -v t="$target" '$1==t' "$ledger" | tail -1)
echo "ledger line: ${line:-(none)}" >>"$log"
[ -n "$line" ] || fail "no review of $target in the ledger; review the head ($head) and add its line"
fields=$(printf '%s' "$line" | awk -F'\t' '{print NF}')
[ "$fields" = 4 ] || fail "ledger line for $target has $fields fields, not 4"
sha=$(printf '%s' "$line" | cut -f2)
file=$(printf '%s' "$line" | cut -f3)
verdict=$(printf '%s' "$line" | cut -f4)
printf '%s' "$sha" | grep -qE '^[0-9a-f]{40}$' || fail "ledger SHA '$sha' is not a full SHA"
[ -n "$file" ] && [ -f "$file" ] || fail "review file '$file' does not exist"
case "$verdict" in
  PASS | PASS-with-findings) ;;
  *) fail "last review of $target ($file) is '$verdict', not PASS or PASS-with-findings" ;;
esac
git cat-file -e "$head^{commit}" 2>/dev/null || git fetch -q origin "$head" 2>/dev/null || git fetch -q origin
git cat-file -e "$sha^{commit}" 2>/dev/null || fail "reviewed SHA $sha is not a commit here"
git cat-file -e "$head^{commit}" 2>/dev/null || fail "head $head is not a commit here"
echo "\$ git diff --name-only $sha $head" >>"$log"
changed=$(git diff --name-only "$sha" "$head") || fail "cannot diff $sha..$head"
printf '%s\n' "$changed" >>"$log"
unreviewed=$(printf '%s\n' "$changed" | grep -v -E '^(\.agents/audit/[^/]+\.md)?$')
[ -z "$unreviewed" ] || fail "changed after the last review ($file, $sha): $(echo $unreviewed)"
echo "passed: reviewed $sha ($file, $verdict); later changes are audit records only" >>"$log"
echo "exit=0" >>"$log"
exit 0
