#!/usr/bin/env node
// A commit that changes apps/desktop/DESIGN.md says which spec items it was compared with, on a
// line "Spec-check: <S-n / A-n …> — <result>". Knowledge: .agents/knowledge/2026-11-15_spec-before-rule-doc.md
//
//   node scripts/check-design-commit-msg.mjs <message file>   husky commit-msg: the staged change
//   node scripts/check-design-commit-msg.mjs --range <a..b>    every commit in the range (run by the
//                                                              Claude Code hook before a push, so an
//                                                              amended or rebased commit is caught too)
// It guards against the PM forgetting the line, not against a deliberate way around the check.
import { execFileSync } from 'node:child_process';
import { readFileSync } from 'node:fs';

const DESIGN = 'apps/desktop/DESIGN.md';
const LINE = /^Spec-check:\s*.*\b[SA]-\d+\b.*\s[—–-]+\s*.*[\p{L}\p{N}]/u;
const FORM = 'Spec-check: <S-n / A-n compared> — <result>';
const git = (...args) => execFileSync('git', args, { encoding: 'utf8' });

function commentChar() {
  try {
    const c = git('config', 'core.commentChar').trim();
    return c && c !== 'auto' ? c : '#';
  } catch {
    return '#';
  }
}

const ok = (text) =>
  text
    .split('\n')
    .filter((l) => !l.startsWith(commentChar()))
    .some((l) => LINE.test(l));

if (process.argv[2] === '--range') {
  // A merge commit counts by its change against the first parent; a rename counts as delete + add.
  const changed = (c) =>
    git(
      'diff-tree',
      '--root',
      '--diff-merges=first-parent',
      '--no-commit-id',
      '--name-only',
      '--no-renames',
      '-r',
      c,
      '--',
      DESIGN,
    ).trim() !== '';
  const bad = git('rev-list', process.argv[3])
    .split('\n')
    .filter(Boolean)
    .filter(changed)
    .filter((c) => !ok(git('log', '-1', '--format=%B', c)))
    .map((c) => git('log', '-1', '--format=%h %s', c).trim());
  if (bad.length > 0) {
    process.stderr.write(
      [
        `design check: these commits change ${DESIGN} without a "${FORM}" line:`,
        ...bad.map((b) => `  ${b}`),
        '  Reword them (git rebase -i is not available here: use git commit --amend -F <file> on the last one, or ask the user).',
        '',
      ].join('\n'),
    );
    process.exit(1);
  }
  process.exit(0);
}

const staged = git('diff', '--cached', '--name-only', '--no-renames').split('\n').filter(Boolean);
if (!staged.includes(DESIGN)) {
  process.exit(0);
}
if (!ok(readFileSync(process.argv[2], 'utf8'))) {
  process.stderr.write(
    [
      `design check: this commit changes ${DESIGN}.`,
      '  Compare the change with the spec items in force and add a line to the message:',
      `    ${FORM}`,
      '',
    ].join('\n'),
  );
  process.exit(1);
}
