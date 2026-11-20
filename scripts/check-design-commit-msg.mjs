#!/usr/bin/env node
// Commit-msg check: a commit that changes apps/desktop/DESIGN.md says which spec items it was
// compared with, on a line "Spec-check: <S-n / A-n … and the result>".
// Knowledge: .agents/knowledge/2026-11-15_spec-before-rule-doc.md
import { execFileSync } from 'node:child_process';
import { readFileSync } from 'node:fs';

const staged = execFileSync('git', ['diff', '--cached', '--name-only'], { encoding: 'utf8' })
  .split('\n')
  .filter(Boolean);
if (!staged.includes('apps/desktop/DESIGN.md')) {
  process.exit(0);
}
const message = readFileSync(process.argv[2], 'utf8')
  .split('\n')
  .filter((l) => !l.startsWith('#'));
if (!message.some((l) => /^Spec-check:\s*\S/.test(l))) {
  process.stderr.write(
    [
      'design check: this commit changes apps/desktop/DESIGN.md.',
      '  Compare the change with the spec items in force and add a line to the message:',
      '    Spec-check: <S-n / A-n compared> — <result>',
      '',
    ].join('\n'),
  );
  process.exit(1);
}
