import type { InternalSettingsApi } from '@croffledev/common';
import type { AppSettings } from '@croffledev/croffle-types';
import { ipcRenderer } from 'electron';

export const settingsApi = {
  getAll: async (): Promise<AppSettings> => {
    return ipcRenderer.invoke('settings:getAll');
  },

  getOf: async (key: string): Promise<AppSettings[keyof AppSettings]> => {
    return ipcRenderer.invoke('settings:getOf', key);
  },

  update: async (partialSettings: Partial<AppSettings>): Promise<AppSettings> => {
    return ipcRenderer.invoke('settings:update', partialSettings);
  },
} satisfies InternalSettingsApi;
