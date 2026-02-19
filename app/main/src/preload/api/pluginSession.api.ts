import { ipcRenderer } from 'electron';

export interface PluginSessionAPI {
  get: (pluginId: string, key: string) => Promise<any>;
  set: (pluginId: string, key: string, value: any) => Promise<void>;
  delete: (pluginId: string, key: string) => Promise<boolean>;
  clear: (pluginId: string) => Promise<void>;
}

export const pluginSessionApi: PluginSessionAPI = {
  get: (pluginId: string, key: string) => {
    return ipcRenderer.invoke('sessionStorage:get', { pluginId, key });
  },
  set: (pluginId: string, key: string, value: any) => {
    return ipcRenderer.invoke('sessionStorage:set', { pluginId, key, value });
  },

  delete: (pluginId: string, key: string) => {
    return ipcRenderer.invoke('sessionStorage:delete', { pluginId, key });
  },

  clear: (pluginId: string) => {
    return ipcRenderer.invoke('sessionStorage:clear', { pluginId });
  },
};
