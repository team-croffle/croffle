import { AppEventType, SETTINGS_STORED_CHANNEL, type StoredSettings } from '@croffledev/common';
import type { AppSettings } from '@croffledev/croffle-types';
import { BrowserWindow, ipcMain } from 'electron';

import { eventService } from '../event-bus/event-service';
import { settingService } from '../setting/setting-service';
import { validateSettings } from '../utils/settings-validator';
import { validateSender } from './validate-sender';

export const registerSettingsIpcHandlers = (): void => {
  ipcMain.handle('settings:getAll', async (): Promise<AppSettings> => {
    return settingService.get();
  });

  ipcMain.handle('settings:getStored', (event): StoredSettings => {
    validateSender(event);
    return settingService.getStored();
  });

  ipcMain.handle(
    'settings:getOf',
    async (_, key: unknown): Promise<AppSettings[keyof AppSettings]> => {
      if (typeof key !== 'string') {
        throw new Error('[Settings] Key must be a string.');
      }

      return settingService.getOf(key);
    },
  );

  ipcMain.handle(
    'settings:update',
    async (event, partialSettings: Partial<AppSettings>): Promise<StoredSettings> => {
      validateSender(event);
      validateSettings(partialSettings);
      const newSettings = settingService.update(partialSettings);
      const stored = settingService.getStored();
      eventService.emit(AppEventType.SETTINGS_UPDATE, newSettings);
      for (const win of BrowserWindow.getAllWindows()) {
        if (!win.isDestroyed()) {
          win.webContents.send(SETTINGS_STORED_CHANNEL, stored);
        }
      }
      return stored;
    },
  );
};
