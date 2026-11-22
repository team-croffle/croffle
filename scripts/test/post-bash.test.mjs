// The PostToolUse hook (.claude/hooks/post-bash.sh) skips the gates when the commit was not made in
// this repository: its HEAD is older than 120 seconds (spec 007 S-3, V-5).
import assert from 'node:assert/strict';
import { spawnSync, execFileSync } from 'node:child_process';
import { copyFileSync, mkdirSync, mkdtempSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { after, describe, test } from 'node:test';

const HOOKS = join(import.meta.dirname, '../../.claude/hooks');
const dirs = [];

/** A repository with the hook files and one commit dated `secondsAgo` before now. */
function repoWithCommit(secondsAgo) {
  const dir = mkdtempSync(join(tmpdir(), 'post-bash-'));
  dirs.push(dir);
  const git = (...args) =>
    execFileSync('git', args, {
      cwd: dir,
      encoding: 'utf8',
      env: {
        ...process.env,
        GIT_AUTHOR_DATE: `@${Math.floor(Date.now() / 1000) - secondsAgo} +0000`,
        GIT_COMMITTER_DATE: `@${Math.floor(Date.now() / 1000) - secondsAgo} +0000`,
      },
    });
  git('init', '-q');
  git('config', 'user.name', 'test');
  git('config', 'user.email', 'test@example.invalid');
  git('config', 'commit.gpgsign', 'false');
  mkdirSync(join(dir, '.claude/hooks'), { recursive: true });
  for (const file of ['post-bash.sh', 'bash-command.mjs']) {
    copyFileSync(join(HOOKS, file), join(dir, '.claude/hooks', file));
  }
  git('add', '.');
  git('commit', '-q', '-m', 'first');
  return dir;
}

/** Runs the hook in `dir` for a `git commit` command. There is no run-gates.sh in the repository,
 * so reaching the gate step fails with "gates failed". */
function runHook(dir) {
  return spawnSync('sh', ['.claude/hooks/post-bash.sh'], {
    cwd: dir,
    input: JSON.stringify({ tool_input: { command: 'git commit -F msg.txt' } }),
    encoding: 'utf8',
  });
}

after(() => {
  for (const dir of dirs) {
    rmSync(dir, { recursive: true, force: true });
  }
});

describe('post-bash hook', () => {
  test('skips gates and exits 0 when HEAD commit is older than 120 seconds', () => {
    const result = runHook(repoWithCommit(600));
    assert.equal(result.status, 0, result.stderr);
    assert.equal(result.stderr, '');
  });

  test('runs the gate step when HEAD commit is fresh', () => {
    const result = runHook(repoWithCommit(0));
    assert.equal(result.status, 2);
    assert.match(result.stderr, /gates failed on the new commit/);
  });
});
