import type { ExtensionInfo, InternalExtensionInfoApi } from '@croffledev/common';
import { ipcRenderer } from 'electron';

export const extensionInfoApi = {
  getInstalled: async (): Promise<ExtensionInfo[]> => {
    return ipcRenderer.invoke('extensionInfo:getInstalledExtensions');
  },

  getEnabled: async (): Promise<ExtensionInfo[]> => {
    return ipcRenderer.invoke('extensionInfo:getEnabledExtensions');
  },

  getByName: async (name: string): Promise<ExtensionInfo | null> => {
    return ipcRenderer.invoke('extensionInfo:getExtensionByName', name);
  },

  install: async (pluginData: Partial<ExtensionInfo>): Promise<ExtensionInfo> => {
    return ipcRenderer.invoke('extensionInfo:installExtension', pluginData);
  },

  installFromLocal: async (): Promise<ExtensionInfo | null> => {
    return ipcRenderer.invoke('extensionInfo:installFromLocal');
  },

  toggle: async (name: string, enable: boolean): Promise<ExtensionInfo | null> => {
    return ipcRenderer.invoke('extensionInfo:toggleExtension', name, enable);
  },

  uninstall: async (name: string): Promise<boolean> => {
    return ipcRenderer.invoke('extensionInfo:uninstallExtension', name);
  },
} satisfies InternalExtensionInfoApi;
