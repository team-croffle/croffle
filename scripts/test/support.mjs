// Temporary git repositories for the tests of the repository scripts.
import { execFileSync, spawnSync } from 'node:child_process';
import { mkdirSync, mkdtempSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { dirname, join } from 'node:path';

export const SCRIPTS = join(import.meta.dirname, '..');

const made = [];

/** A new repository with a local identity. `repo.git(...)` runs git in it. */
export function tempRepo() {
  const dir = mkdtempSync(join(tmpdir(), 'croffle-scripts-'));
  made.push(dir);
  const git = (...args) => execFileSync('git', args, { cwd: dir, encoding: 'utf8' });
  git('init', '-q', '-b', 'main');
  git('config', 'user.name', 'test');
  git('config', 'user.email', 'test@example.invalid');
  git('config', 'commit.gpgsign', 'false');
  const write = (path, text) => {
    mkdirSync(dirname(join(dir, path)), { recursive: true });
    writeFileSync(join(dir, path), text);
  };
  /** Runs a repository script (by file name under scripts/) in this repository. */
  const run = (script, ...args) =>
    spawnSync(process.execPath, [join(SCRIPTS, script), ...args], { cwd: dir, encoding: 'utf8' });
  return { dir, git, write, run };
}

export function removeTempRepos() {
  for (const dir of made.splice(0)) {
    rmSync(dir, { recursive: true, force: true });
  }
}
