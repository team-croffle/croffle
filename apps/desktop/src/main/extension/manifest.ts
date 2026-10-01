import fs from 'node:fs';
import path from 'node:path';

import type { CroffleManifest } from '@croffledev/common';
import { is } from '@electron-toolkit/utils';
import { app } from 'electron';

/**
 * Minimal `>=x.y.z` range check for croffle-manifest `engines.croffle`.
 * Unsupported range syntax fails closed (returns false).
 */
export function satisfiesCroffleEngine(appVersion: string, range: string | undefined): boolean {
  if (!range || !range.trim()) {
    return true;
  }

  const trimmed = range.trim();
  const match = /^>=\s*(\d+)\.(\d+)\.(\d+)$/.exec(trimmed);
  if (!match) {
    return false;
  }

  const need = [Number(match[1]), Number(match[2]), Number(match[3])] as const;
  const haveParts = appVersion
    .split(/[-+]/)[0]
    .split('.')
    .map((p) => Number(p));
  if (haveParts.length < 3 || haveParts.some((n) => Number.isNaN(n))) {
    return false;
  }
  const have = [haveParts[0], haveParts[1], haveParts[2]] as const;

  for (let i = 0; i < 3; i++) {
    if (have[i] > need[i]) {
      return true;
    }
    if (have[i] < need[i]) {
      return false;
    }
  }
  return true;
}

export const MANIFEST_FILENAME = 'croffle-manifest.json';

/**
 * Installed extensions root. Dev runs keep extensions next to the dev DB so they never mix with
 * the packaged app's; every reader and writer goes through here.
 */
export function getExtensionDir(extensionId?: string): string {
  const root = is.dev
    ? path.join(process.cwd(), 'dev/extensions')
    : path.join(app.getPath('userData'), 'extensions');
  return extensionId ? path.join(root, extensionId) : root;
}

/** Best-effort read of installed extension manifest from disk. */
export function readInstalledManifest(extensionId: string): CroffleManifest | null {
  const manifestPath = path.join(getExtensionDir(extensionId), MANIFEST_FILENAME);
  if (!fs.existsSync(manifestPath)) {
    return null;
  }

  try {
    return JSON.parse(fs.readFileSync(manifestPath, 'utf-8')) as CroffleManifest;
  } catch {
    return null;
  }
}

/** True when `target` resolves to `root` itself or somewhere below it. */
export function isInsideDir(root: string, target: string): boolean {
  const relative = path.relative(path.resolve(root), path.resolve(target));
  const escapes = relative === '..' || relative.startsWith(`..${path.sep}`);
  return !escapes && !path.isAbsolute(relative);
}
