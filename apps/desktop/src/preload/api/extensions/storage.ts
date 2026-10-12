import type { ExtensionStorageApi } from '@croffledev/common';
import { ipcRenderer } from 'electron';

export const extensionStorageApi = {
  get: <T = unknown>(extensionId: string, key: string): Promise<T | null> => {
    return ipcRenderer.invoke('extensionStorage:get', { extensionId, key });
  },
  set: (extensionId: string, key: string, value: unknown): Promise<void> => {
    return ipcRenderer.invoke('extensionStorage:set', { extensionId, key, value });
  },
  delete: (extensionId: string, key: string): Promise<boolean> => {
    return ipcRenderer.invoke('extensionStorage:delete', { extensionId, key });
  },
  clear: (extensionId: string): Promise<void> => {
    return ipcRenderer.invoke('extensionStorage:clear', { extensionId });
  },
} satisfies ExtensionStorageApi;
