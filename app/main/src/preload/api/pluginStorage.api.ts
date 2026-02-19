import { pluginStorage } from 'croffle';
import { ipcRenderer } from 'electron';

type PluginStorageAPI = typeof pluginStorage;

export const pluginStorageApi = {
  get: (pluginId: string, key: string) => {
    return ipcRenderer.invoke('app:storage:get', { pluginId, key });
  },
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  set: (pluginId: string, key: string, value: any) => {
    return ipcRenderer.invoke('app:storage:set', { pluginId, key, value });
  },
} satisfies PluginStorageAPI;
