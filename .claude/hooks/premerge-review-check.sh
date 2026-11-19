#!/bin/sh
# Refuses a merge (or a direct push to master) whose content was not reviewed.
# Usage: premerge-review-check.sh pr <number> | premerge-review-check.sh master
# The ledger (.loctx/reports/review-ledger.tsv, tab-separated) has one line per review:
#   <pr number or "master">  <reviewed full SHA>  <review file>  <PASS|PASS-with-findings|FAIL>
# The last line for the target counts. Between the reviewed SHA and the head, only audit records
# (.agents/audit/*.md) may change; anything else needs another review first.
# Knowledge: .agents/knowledge/2026-11-15_review-before-merge.md
root=$(git rev-parse --show-toplevel) || exit 2
cd "$root" || exit 2
ledger="${REVIEW_LEDGER:-.loctx/reports/review-ledger.tsv}"
kind=$1
if [ "$kind" = pr ]; then
  target=$2
  [ -n "$target" ] || { echo "premerge: name the PR number (gh pr merge <n>)" >&2; exit 2; }
  head=$(gh pr view "$target" --json headRefOid -q .headRefOid) || { echo "premerge: cannot read PR #$target" >&2; exit 2; }
else
  target=master
  head=$(git rev-parse HEAD)
fi
mkdir -p .loctx/reports/premerge
log=".loctx/reports/premerge/$target-$(date -u +%Y%m%dT%H%M%SZ).log"
line=$(awk -F'\t' -v t="$target" '$1==t' "$ledger" 2>/dev/null | tail -1)
{
  echo "target: $target"
  echo "head: $head"
  echo "ledger: $ledger"
  echo "ledger line: ${line:-(none)}"
} >"$log"
fail() { echo "refused: $1" >>"$log"; echo "exit=2" >>"$log"; echo "premerge: $1 (log $log)" >&2; exit 2; }
[ -n "$line" ] || fail "no review of $target in the ledger; review the head ($head) and add its line"
sha=$(printf '%s' "$line" | cut -f2)
file=$(printf '%s' "$line" | cut -f3)
verdict=$(printf '%s' "$line" | cut -f4)
[ "$verdict" != FAIL ] || fail "the last review ($file) failed"
git cat-file -e "$head^{commit}" 2>/dev/null || git fetch -q origin "$head" 2>/dev/null || git fetch -q origin
git cat-file -e "$sha^{commit}" 2>/dev/null || fail "reviewed SHA $sha is not a commit here"
echo "\$ git diff --name-only $sha $head" >>"$log"
changed=$(git diff --name-only "$sha" "$head") || fail "cannot diff $sha..$head"
printf '%s\n' "$changed" >>"$log"
unreviewed=$(printf '%s\n' "$changed" | grep -v -E '^(\.agents/audit/[^/]+\.md)?$')
[ -z "$unreviewed" ] || fail "changed after the last review ($file, $sha): $(echo $unreviewed)"
echo "passed: reviewed $sha ($file, $verdict); later changes are audit records only" >>"$log"
echo "exit=0" >>"$log"
exit 0
