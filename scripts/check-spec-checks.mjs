#!/usr/bin/env node
// Pre-commit check: every check (V-n) added to a spec names who designed it and how it was
// dry-run before the summary was shown. A staged spec.md whose diff adds a "- **V-<n> …" heading
// must have, inside that check, the two lines
//   - 설계: <qa-designer task and answer path>
//   - 사전 실행: <raw output path | 구현 뒤 — <reason>>
// Checks that were already in the spec are not touched. Knowledge:
// .agents/knowledge/2026-11-17_check-wording-and-raw-evidence.md (checks added after approval).
import { execFileSync } from 'node:child_process';

const git = (...args) => execFileSync('git', args, { encoding: 'utf8' });

const specs = git('diff', '--cached', '--name-only', '--diff-filter=AM')
  .split('\n')
  .filter((f) => /^\.agents\/intent\/[^/]+\/spec\.md$/.test(f));

const HEADING = /^- \*\*(V-\d+)\b/;
const problems = [];

for (const file of specs) {
  const added = git('diff', '--cached', '-U0', '--', file)
    .split('\n')
    .filter((l) => l.startsWith('+') && !l.startsWith('+++'))
    .map((l) => l.slice(1));
  const newChecks = added.map((l) => HEADING.exec(l)?.[1]).filter(Boolean);
  if (newChecks.length === 0) {
    continue;
  }
  const lines = git('show', `:${file}`).split('\n');
  for (const id of newChecks) {
    const start = lines.findIndex((l) => HEADING.exec(l)?.[1] === id);
    if (start < 0) {
      continue;
    }
    let end = lines.findIndex((l, i) => i > start && (HEADING.test(l) || /^#{1,3} /.test(l)));
    if (end < 0) {
      end = lines.length;
    }
    const block = lines.slice(start, end);
    const has = (label) => block.some((l) => new RegExp(`^\\s*- ${label}:\\s*\\S`).test(l));
    const missing = ['설계', '사전 실행'].filter((label) => !has(label));
    if (missing.length > 0) {
      problems.push(`${file}: ${id} has no "- ${missing.join(':" / "- ')}:" line`);
    }
  }
}

if (problems.length > 0) {
  process.stderr.write(
    [
      'spec check: a check (V-n) added to a spec must say who designed it and how it was dry-run:',
      ...problems.map((p) => `  ${p}`),
      '  Add inside the check:',
      '    - 설계: <qa-designer task and answer path>',
      '    - 사전 실행: <raw output path | 구현 뒤 — reason>',
      '',
    ].join('\n'),
  );
  process.exit(1);
}
