import type { CroffleManifest, ExtensionContributes, ExtensionInfo } from '@croffledev/common';
import { sanitizeContextMenus } from '@croffledev/common';

import type { ExtensionInfoRow, NewExtensionInfo } from '../database/schema';
import { logger } from '../logger';

// The mapper runs on every extension list request; warn about a bad manifest entry only once.
const reportedProblems = new Set<string>();

function toContributes(
  extensionId: string,
  manifest: CroffleManifest | null,
): ExtensionContributes {
  const contributes = manifest?.contributes ?? {};
  if (contributes.contextMenus === undefined) {
    return contributes;
  }

  const { items, dropped } = sanitizeContextMenus(contributes.contextMenus);
  for (const problem of dropped) {
    const key = `${extensionId}:${problem}`;
    if (!reportedProblems.has(key)) {
      reportedProblems.add(key);
      logger.warn('ExtensionManifest', `${extensionId}: ignoring ${problem}`);
    }
  }
  return { ...contributes, contextMenus: items };
}

export const extensionInfoMapper = {
  toInterface(entity: ExtensionInfoRow, manifest: CroffleManifest | null = null): ExtensionInfo {
    return {
      id: entity.id,
      name: manifest?.name ?? entity.name,
      version: manifest?.version ?? entity.version,
      author: manifest?.author ?? entity.author,
      description: manifest?.description ?? entity.description ?? '',
      main: manifest?.main ?? entity.main ?? undefined,
      engines: manifest?.engines,
      contributes: toContributes(entity.id, manifest),
      enabled: entity.enabled,
    };
  },

  toEntity(api: ExtensionInfo): Omit<NewExtensionInfo, 'installedAt' | 'updatedAt'> {
    return {
      id: api.id,
      name: api.name,
      version: api.version,
      author: api.author,
      description: api.description ?? null,
      enabled: api.enabled,
      main: api.main ?? null,
    };
  },
};
