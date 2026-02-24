import { ipcRenderer } from 'electron';
import { PluginSessionAPI } from 'croffle';

export const pluginSessionApi = {
  get: <T = unknown>(pluginId: string, key: string): Promise<T | null> => {
    return ipcRenderer.invoke('sessionStorage:get', { pluginId, key });
  },
  set: <T = unknown>(pluginId: string, key: string, value: T): Promise<void> => {
    return ipcRenderer.invoke('sessionStorage:set', { pluginId, key, value });
  },

  delete: (pluginId: string, key: string): Promise<boolean> => {
    return ipcRenderer.invoke('sessionStorage:delete', { pluginId, key });
  },

  clear: (pluginId: string): Promise<void> => {
    return ipcRenderer.invoke('sessionStorage:clear', { pluginId });
  },

  clearAll: (): Promise<void> => {
    return ipcRenderer.invoke('sessionStorage:clearAll');
  },
} satisfies PluginSessionAPI;
