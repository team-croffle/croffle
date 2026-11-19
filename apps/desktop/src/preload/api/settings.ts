import {
  SETTINGS_STORED_CHANNEL,
  type InternalSettingsApi,
  type StoredSettings,
} from '@croffledev/common';
import type { AppSettings } from '@croffledev/croffle-types';
import { ipcRenderer, type IpcRendererEvent } from 'electron';

export const settingsApi = {
  getAll: async (): Promise<AppSettings> => {
    return ipcRenderer.invoke('settings:getAll');
  },

  getOf: async (key: string): Promise<AppSettings[keyof AppSettings]> => {
    return ipcRenderer.invoke('settings:getOf', key);
  },

  update: async (partialSettings: Partial<AppSettings>): Promise<StoredSettings> => {
    return ipcRenderer.invoke('settings:update', partialSettings);
  },

  getStored: async (): Promise<StoredSettings> => {
    return ipcRenderer.invoke('settings:getStored');
  },

  onStored: (callback: (stored: StoredSettings) => void) => {
    const subscription = (_event: IpcRendererEvent, stored: StoredSettings) => callback(stored);
    ipcRenderer.on(SETTINGS_STORED_CHANNEL, subscription);
    return () => {
      ipcRenderer.off(SETTINGS_STORED_CHANNEL, subscription);
    };
  },
} satisfies InternalSettingsApi;
