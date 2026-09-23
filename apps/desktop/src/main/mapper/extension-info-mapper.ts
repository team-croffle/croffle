import type { CroffleManifest, ExtensionInfo } from '@croffledev/common';

import type { ExtensionInfoRow, NewExtensionInfo } from '../database/schema';

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
      contributes: manifest?.contributes ?? {},
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
