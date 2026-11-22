// scripts/check-spec-checks.mjs (pre-commit): a check V-n added to a spec carries its
// "- 설계:" and "- 사전 실행:" lines (spec 007 S-4, V-5).
import assert from 'node:assert/strict';
import { after, describe, test } from 'node:test';

import { removeTempRepos, tempRepo } from './support.mjs';

const SPEC = '.agents/intent/001_x/spec.md';
const LINES = ['  - 설계: qa-designer task t.md, answer a.md', '  - 사전 실행: dryrun/v.txt'];

const spec = (...checks) =>
  ['## 검증 기준', ...checks.flat(), '', '## 조정 기록', '—', ''].join('\n');
const check = (id, withLines, body = '본문') => [
  `- **${id} 제목** — 실행`,
  `  - 기대: ${body}`,
  ...(withLines ? LINES : []),
];

/** A repository whose committed spec holds V-1 (with both lines). */
function repoWithSpec() {
  const repo = tempRepo();
  repo.write(SPEC, spec(check('V-1', true)));
  repo.git('add', '.');
  repo.git('commit', '-q', '-m', 'spec');
  return repo;
}

const runCheck = (repo) => repo.run('check-spec-checks.mjs');

after(removeTempRepos);

describe('check-spec-checks', () => {
  test('rejects a newly added V-n heading when design or dry-run line is missing', () => {
    const repo = repoWithSpec();
    repo.write(SPEC, spec(check('V-1', true), check('V-2', false)));
    repo.git('add', '.');
    const result = runCheck(repo);
    assert.equal(result.status, 1);
    assert.match(result.stderr, /V-2 has no "- 설계:" \/ "- 사전 실행:" line/);
  });

  test('accepts a newly added V-n heading when both design and dry-run lines are present', () => {
    const repo = repoWithSpec();
    repo.write(SPEC, spec(check('V-1', true), check('V-2', true)));
    repo.git('add', '.');
    assert.equal(runCheck(repo).status, 0);
  });

  test('ignores edits to the body of an existing V-n check', () => {
    const repo = tempRepo();
    repo.write(SPEC, spec(check('V-1', false)));
    repo.git('add', '.');
    repo.git('commit', '-q', '-m', 'an old spec from before the rule');
    repo.write(SPEC, spec(check('V-1', false, '고친 본문')));
    repo.git('add', '.');
    assert.equal(runCheck(repo).status, 0);
  });

  test('requires both lines on every V-n in a newly added spec file', () => {
    const repo = tempRepo();
    repo.write('README.md', 'x\n');
    repo.git('add', '.');
    repo.git('commit', '-q', '-m', 'init');
    repo.write(SPEC, spec(check('V-1', true), check('V-2', false), check('V-3', false)));
    repo.git('add', '.');
    const result = runCheck(repo);
    assert.equal(result.status, 1);
    assert.match(result.stderr, /V-2 has no/);
    assert.match(result.stderr, /V-3 has no/);
    assert.doesNotMatch(result.stderr, /V-1 has no/);
  });

  test('requires both lines when a V-n heading is moved or renumbered in the diff', () => {
    const repo = tempRepo();
    repo.write(SPEC, spec(check('V-1', false)));
    repo.git('add', '.');
    repo.git('commit', '-q', '-m', 'an old spec from before the rule');
    repo.write(SPEC, spec(check('V-3', false)));
    repo.git('add', '.');
    const result = runCheck(repo);
    assert.equal(result.status, 1);
    assert.match(result.stderr, /V-3 has no/);
  });
});
