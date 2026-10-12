import type { ExtensionConfigurationApi } from '@croffledev/common';
import { ipcRenderer } from 'electron';

const DEFAULT_STORAGE_KEY = '__croffle_configuration__';

/**
 * Normalize a value read from storage.
 * Legacy configuration wrote JSON.stringify(values) into storage.set, which
 * stringified again — so old rows may decode to a JSON string once more.
 */
function coerceConfigurationValue<T>(raw: unknown): T {
  if (raw === null || raw === undefined) {
    return {} as T;
  }

  if (typeof raw === 'string') {
    try {
      return JSON.parse(raw) as T;
    } catch {
      return {} as T;
    }
  }

  if (typeof raw === 'object') {
    return raw as T;
  }

  return {} as T;
}

export const extensionConfigurationApi = {
  get: async <T = Record<string, unknown>>(
    extensionId: string,
    storageKey = DEFAULT_STORAGE_KEY,
  ): Promise<T> => {
    const raw = await ipcRenderer.invoke('extensionStorage:get', { extensionId, key: storageKey });
    return coerceConfigurationValue<T>(raw);
  },
  set: async (
    extensionId: string,
    values: Record<string, unknown>,
    storageKey = DEFAULT_STORAGE_KEY,
  ): Promise<void> => {
    await ipcRenderer.invoke('extensionStorage:set', {
      extensionId,
      key: storageKey,
      value: values,
    });
  },
} satisfies ExtensionConfigurationApi;
