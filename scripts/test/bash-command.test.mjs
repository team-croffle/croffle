// The Bash command classifier of the Claude Code hooks (.claude/hooks/bash-command.mjs).
// Cases: the expected-result table of the 1.3.2 retro (spec 007 S-1, V-5).
import assert from 'node:assert/strict';
import { execFileSync } from 'node:child_process';
import { join } from 'node:path';
import { describe, test } from 'node:test';

import { classify } from '../../.claude/hooks/bash-command.mjs';

const CLI = join(import.meta.dirname, '../../.claude/hooks/bash-command.mjs');
const A = '0123456789abcdef0123456789abcdef01234567';
const B = 'fedcba9876543210fedcba9876543210fedcba98';

/** Commands with an exact verdict. */
const EXACT = [
  ['git push -u origin feat/x', 'push origin feat/x'],
  ['git push -q -u origin feat/x', 'push origin feat/x'],
  ['git push origin master', 'push origin master'],
  ['  git push origin feat/x  ', 'push origin feat/x'],
  [`gh pr merge 5 --rebase --delete-branch --match-head-commit ${A}`, `merge 5 ${A}`],
  [`gh pr merge 7 --squash --match-head-commit ${A}`, `merge 7 ${A}`],
  ['gh pr merge 8', 'merge 8'],
  ['gh pr view 146 --json mergedBy,mergeCommit', 'none'],
  ['git merge-base master x', 'none'],
  ['git branch --merged', 'none'],
  ['echo hello', 'none'],
  ['git status; git log', 'none'],
  ['echo "unbalanced', 'none'],
  ['git commitx', 'none'],
  // Known limit: a command assembled from variables is not seen.
  ['x=pu; git ${x}sh origin master', 'none'],
  ['git -C /x commit -F msg.txt', 'commit'],
  ['git add a && git commit -F m.txt', 'commit'],
  ['env A=1 git commit -m y', 'commit'],
  ['git commit --amend --no-edit', 'commit'],
];

/** Commands that must be refused, with a fragment of the reason. */
const REFUSED = [
  [`gh pr merge 146 --match-head-commit ${A} --match-head-commit ${B}`, 'once'],
  ['git push origin HEAD', 'instead of HEAD'],
  ['git push origin refs/heads/HEAD', 'instead of HEAD'],
  ['git push', "'push'"],
  ['git push origin', "'push'"],
  ['git push --all origin', "'push'"],
  ['git push --force origin feat/x', "'push'"],
  ['git push origin :master', "'push'"],
  ['git push --delete origin master', "'push'"],
  ['git push origin master --delete', "'push'"],
  ['git push origin "HEAD:master"', "'push'"],
  ["git push origin 'refs/heads/*:refs/heads/*'", "'push'"],
  ['git push --repo=origin HEAD:master', "'push'"],
  ['git -C "/tmp/a b" push origin HEAD:master', "'push'"],
  ['git -c push.default=matching push origin', "'push'"],
  ['git -c remote.origin.push=HEAD:master push origin', "'push'"],
  ["gh pr 'merge' 141", "'merge'"],
  ['gh pr merge 5 --admin', "'merge'"],
  ['gh -R team/repo pr merge 5', "'merge'"],
  ['cd x && gh pr merge 5 --rebase', "'merge'"],
  ["bash -lc 'git push origin master'", "'push'"],
  ["env bash -c 'git push origin master'", "'push'"],
  ["eval 'gh pr merge 3'", "'merge'"],
  ['env FOO=1 git push origin feat/x', "'push'"],
  ['GIT_DIR=/x git push origin feat/x', "'push'"],
  ['command git push origin feat/y', "'push'"],
  ['sudo -u x git push origin feat/z', "'push'"],
  ['timeout 5 gh pr merge 3', "'merge'"],
  ['> /dev/null git push origin master', "'push'"],
  ['2>/dev/null gh pr merge 146', "'merge'"],
  ['git pu\\\nsh origin master', "'push'"],
  ['git push -q -u origin x 2>&1 | tail -1', "'push'"],
  ['git commit -m "x" && git push origin feat/x', "'push'"],
  ['git push origin feat/x # comment', "'push'"],
  ['git commit -m "docs: git push origin feature"', "'push'"],
  ['gh pr comment 146 --body "please gh pr merge later"', "'merge'"],
  ['echo ";" git push origin master', "'push'"],
  ['echo "unbalanced push', "'push'"],
];

const cli = (input) => execFileSync(process.execPath, [CLI], { input, encoding: 'utf8' });

describe('bash command classifier', () => {
  test('classifies every allowed push merge commit and neutral command in the history table', () => {
    for (const [cmd, verdict] of EXACT) {
      assert.equal(classify(cmd), verdict, cmd);
    }
  });

  test('refuses every non-exact push and merge command in the history table', () => {
    for (const [cmd, fragment] of REFUSED) {
      const verdict = classify(cmd);
      assert.ok(verdict.startsWith('refuse '), `${cmd} → ${verdict}`);
      assert.ok(verdict.includes(fragment), `${cmd} → ${verdict}`);
    }
  });

  test('cli refuses non-json and missing command input on stdin', () => {
    const prefix = 'refuse cannot read the Bash command from the hook input (';
    for (const input of ['not json', '{"tool_input":{}}', '{"tool_input":{"command":12}}', '']) {
      assert.ok(cli(input).startsWith(prefix), JSON.stringify(input));
    }
    assert.equal(cli('{"tool_input":{"command":"echo hi"}}'), 'none\n');
    assert.equal(
      cli('{"tool_input":{"command":"git push origin feat/x"}}'),
      'push origin feat/x\n',
    );
  });
});
