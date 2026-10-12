import type { ExtensionSessionApi } from '@croffledev/common';
import { ipcRenderer } from 'electron';

export const extensionSessionApi = {
  get: <T = unknown>(extensionId: string, key: string): Promise<T | null> => {
    return ipcRenderer.invoke('extensionSession:get', { extensionId, key });
  },
  set: <T = unknown>(extensionId: string, key: string, value: T): Promise<void> => {
    return ipcRenderer.invoke('extensionSession:set', { extensionId, key, value });
  },

  delete: (extensionId: string, key: string): Promise<boolean> => {
    return ipcRenderer.invoke('extensionSession:delete', { extensionId, key });
  },

  clear: (extensionId: string): Promise<void> => {
    return ipcRenderer.invoke('extensionSession:clear', { extensionId });
  },
} satisfies ExtensionSessionApi;
