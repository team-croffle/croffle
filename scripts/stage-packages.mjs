#!/usr/bin/env node
// Stages the publishable packages on npm instead of publishing them (run by Publish Packages
// after the Changesets version PR is merged). A staged version is not installable until a
// maintainer approves it on npmjs.com or with `npm stage approve <id>` (2FA).
//
// For each package:
//   - the version is live on npm  → nothing (reported)
//   - the version is already staged → nothing (waiting for approval)
//   - otherwise                     → `pnpm pack` (resolves workspace: ranges),
//                                     `npm stage publish <tarball> --provenance`, and the git
//                                     tag `<name>@<version>` on this commit
// The tag goes on the commit that was staged, when it is staged — this run is the merge of the
// version PR, so HEAD is that source. A later run cannot tag an older commit: GitHub refuses a
// GITHUB_TOKEN ref whose workflow files differ from the default branch. If a staged version is
// rejected on npm, delete its tag by hand.
//
// DRY_RUN=1 passes --dry-run to npm and creates no tags. Needs npm >= 11.15.0.

import { execFileSync } from 'node:child_process';
import { appendFileSync, mkdtempSync, readdirSync, readFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';

const PACKAGES = ['packages/types', 'packages/cli'];
const dryRun = process.env.DRY_RUN === '1';
const summary = [];
/** Tag problems: reported in full, but they do not fail the run. */
const warnings = [];

function run(cmd, args, options = {}) {
  return execFileSync(cmd, args, {
    encoding: 'utf8',
    stdio: ['ignore', 'pipe', 'pipe'],
    ...options,
  });
}

function tryRun(cmd, args, options) {
  try {
    return { ok: true, out: run(cmd, args, options) };
  } catch (error) {
    return { ok: false, out: `${error.stdout ?? ''}${error.stderr ?? ''}` };
  }
}

function isLive(name, version) {
  const result = tryRun('npm', ['view', `${name}@${version}`, 'version']);
  return result.ok && result.out.trim() === version;
}

function isStaged(name, version) {
  const result = tryRun('npm', ['stage', 'list', name, '--json']);
  if (!result.ok) {
    return false;
  }
  try {
    const listed = JSON.parse(result.out);
    const entries = Array.isArray(listed) ? listed : Object.values(listed).flat();
    return entries.some(
      (entry) => entry && (entry.version === version || entry.spec?.endsWith(`@${version}`)),
    );
  } catch {
    return result.out.includes(`${name}@${version}`);
  }
}

function ensureTag(tag) {
  if (tryRun('git', ['rev-parse', '-q', '--verify', `refs/tags/${tag}`]).ok) {
    return 'tag exists';
  }
  if (tryRun('git', ['ls-remote', '--exit-code', '--tags', 'origin', `refs/tags/${tag}`]).ok) {
    return 'tag exists on origin';
  }
  if (dryRun) {
    return 'would create tag';
  }
  // Tagging is a by-product: a refused tag must not hide whether staging worked. Report the
  // git output in full — the reason a push was refused is not on its last line.
  const created = tryRun('git', ['tag', tag, 'HEAD']);
  if (!created.ok) {
    warnings.push({ tag, step: 'git tag', output: created.out });
    return 'tag not created (see warnings)';
  }
  const pushed = tryRun('git', ['push', 'origin', `refs/tags/${tag}`]);
  if (!pushed.ok) {
    tryRun('git', ['tag', '-d', tag]);
    warnings.push({ tag, step: 'git push', output: pushed.out });
    return 'tag not pushed (see warnings)';
  }
  return 'tag created';
}

function stage(dir, name, version) {
  const out = mkdtempSync(join(tmpdir(), 'croffle-stage-'));
  run('pnpm', ['pack', '--pack-destination', out], { cwd: dir });
  const tarball = readdirSync(out).find((f) => f.endsWith('.tgz'));
  if (!tarball) {
    throw new Error(`pnpm pack produced no tarball for ${name}`);
  }
  const args = ['stage', 'publish', join(out, tarball), '--provenance'];
  // A pre-release must not take `latest`; the staged tag cannot be changed later.
  if (version.includes('-')) {
    args.push('--tag', 'next');
  }
  if (dryRun) {
    args.push('--dry-run');
  }
  const result = run('npm', args).trim().split('\n').slice(-3).join(' ');
  return dryRun ? result : `${result}; ${ensureTag(`${name}@${version}`)}`;
}

const npmVersion = run('npm', ['--version']).trim();
const [major, minor] = npmVersion.split('.').map(Number);
if (major < 11 || (major === 11 && minor < 15)) {
  console.error(`npm ${npmVersion} cannot stage; 11.15.0 or newer is required.`);
  process.exit(1);
}

let failed = false;
for (const dir of PACKAGES) {
  const {
    name,
    version,
    private: isPrivate,
  } = JSON.parse(readFileSync(join(dir, 'package.json'), 'utf8'));
  if (isPrivate) {
    continue;
  }
  const tag = `${name}@${version}`;
  try {
    if (isLive(name, version)) {
      summary.push(`- \`${tag}\`: live on npm`);
    } else if (isStaged(name, version)) {
      summary.push(`- \`${tag}\`: already staged, waiting for approval`);
    } else {
      summary.push(
        `- \`${tag}\`: staged${dryRun ? ' (dry run)' : ''} — ${stage(dir, name, version)}`,
      );
    }
  } catch (error) {
    failed = true;
    summary.push(`- \`${tag}\`: failed — see below`);
    warnings.push({ tag, step: 'stage', output: String(error.stderr ?? error.message) });
  }
}

const report = [
  '### npm staging',
  '',
  ...summary,
  '',
  'Approve staged versions on npmjs.com (Staged Packages) or with `npm stage approve <id>`.',
  ...warnings.flatMap((w) => ['', `#### ${w.tag} — ${w.step}`, '', '```', w.output.trim(), '```']),
].join('\n');
console.log(report);
if (process.env.GITHUB_STEP_SUMMARY) {
  appendFileSync(process.env.GITHUB_STEP_SUMMARY, `${report}\n`);
}
process.exit(failed ? 1 : 0);
