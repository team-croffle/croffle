// scripts/check-design-commit-msg.mjs: a commit that changes apps/desktop/DESIGN.md carries a
// "Spec-check: <S-n / A-n> — <result>" line, in commit-msg mode and over a range (spec 007 S-4, V-5).
import assert from 'node:assert/strict';
import { writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { after, describe, test } from 'node:test';

import { removeTempRepos, tempRepo } from './support.mjs';

const DESIGN = 'apps/desktop/DESIGN.md';
const GOOD = 'docs: tokens\n\nSpec-check: S-3 — 일치\n';

/** A repository with a committed DESIGN.md and README.md. */
function repoWithDesign() {
  const repo = tempRepo();
  repo.write(DESIGN, '# Design\n');
  repo.write('README.md', 'x\n');
  repo.git('add', '.');
  repo.git('commit', '-q', '-m', 'init\n\nSpec-check: S-1 — 처음');
  return repo;
}

/** Runs commit-msg mode with `message` against what is staged. */
function commitMsg(repo, message) {
  const file = join(repo.dir, 'MSG');
  writeFileSync(file, message);
  return repo.run('check-design-commit-msg.mjs', file);
}

after(removeTempRepos);

describe('check-design-commit-msg', () => {
  test('exits 0 in commit-msg mode when DESIGN.md is not staged', () => {
    const repo = repoWithDesign();
    repo.write('README.md', 'y\n');
    repo.git('add', '.');
    assert.equal(commitMsg(repo, 'docs: readme\n').status, 0);
  });

  test('rejects malformed or commented Spec-check lines in commit-msg mode', () => {
    const repo = repoWithDesign();
    repo.write(DESIGN, '# Design\n\nmore\n');
    repo.git('add', '.');
    for (const message of [
      'docs: x\n\nSpec-check: x\n',
      'docs: x\n\nSpec-check: S-1 --\n',
      'docs: x\n\n# Spec-check: S-3 — 일치\n',
    ]) {
      assert.equal(commitMsg(repo, message).status, 1, message);
    }
    repo.git('config', 'core.commentChar', 'S');
    assert.equal(commitMsg(repo, GOOD).status, 1, 'the line is a comment when core.commentChar=S');
  });

  test('accepts a valid Spec-check line and detects staged rename of DESIGN.md', () => {
    const repo = repoWithDesign();
    repo.write(DESIGN, '# Design\n\nmore\n');
    repo.git('add', '.');
    assert.equal(commitMsg(repo, GOOD).status, 0);
    repo.git('commit', '-q', '-F', join(repo.dir, 'MSG'));
    repo.git('mv', DESIGN, 'apps/desktop/DESIGN-old.md');
    const result = commitMsg(repo, 'chore: move\n');
    assert.equal(result.status, 1);
    assert.match(result.stderr, /this commit changes apps\/desktop\/DESIGN\.md/);
  });

  test('checks commit ranges including amended commits merge commits and root commits', () => {
    // A commit with the line passes.
    const repo = repoWithDesign();
    const base = repo.git('rev-parse', 'HEAD').trim();
    repo.write(DESIGN, '# Design\n\nv2\n');
    repo.git('add', '.');
    repo.git('commit', '-q', '-m', GOOD);
    assert.equal(repo.run('check-design-commit-msg.mjs', '--range', `${base}..HEAD`).status, 0);

    // The same commit amended without the line is named.
    repo.git('commit', '-q', '--amend', '-m', 'docs: tokens');
    let result = repo.run('check-design-commit-msg.mjs', '--range', `${base}..HEAD`);
    assert.equal(result.status, 1);
    assert.match(result.stderr, /docs: tokens/);

    // A two-parent commit that resolves a DESIGN.md conflict, reworded without the line, is named.
    repo.git('reset', '-q', '--hard', base);
    repo.git('checkout', '-q', '-b', 'side');
    repo.write(DESIGN, '# Design\n\nside\n');
    repo.git('commit', '-q', '-am', 'side\n\nSpec-check: S-2 — 일치');
    repo.git('checkout', '-q', 'main');
    repo.write(DESIGN, '# Design\n\nmain\n');
    repo.git('commit', '-q', '-am', 'main\n\nSpec-check: S-2 — 일치');
    try {
      repo.git('merge', '-q', 'side');
    } catch {
      // the conflict is expected
    }
    repo.write(DESIGN, '# Design\n\nboth\n');
    repo.git('add', '.');
    repo.git('commit', '-q', '-m', 'join the two sides');
    result = repo.run('check-design-commit-msg.mjs', '--range', `${base}..HEAD`);
    assert.equal(result.status, 1);
    assert.match(result.stderr, /join the two sides/);

    // A root commit that creates DESIGN.md without the line is named, without an error.
    const fresh = tempRepo();
    fresh.write(DESIGN, '# Design\n');
    fresh.git('add', '.');
    fresh.git('commit', '-q', '-m', 'root without the line');
    result = fresh.run('check-design-commit-msg.mjs', '--range', 'HEAD');
    assert.equal(result.status, 1);
    assert.match(result.stderr, /root without the line/);
    assert.doesNotMatch(result.stderr, /Error|fatal/);
  });
});
