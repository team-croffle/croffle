import { ipcMain } from 'electron';

import { windowService } from '../window/window-service';
import { validateSender } from './validate-sender';

export function registerWindowIpcHandlers() {
  ipcMain.handle('window:minimize', (event) => {
    const window = validateSender(event);
    window.minimize();
  });

  ipcMain.handle('window:maximize', (event) => {
    const window = validateSender(event);
    if (window.isMaximized()) {
      window.unmaximize();
    } else {
      window.maximize();
    }
  });

  ipcMain.handle('window:close', (event) => {
    const window = validateSender(event);
    window.close();
  });

  ipcMain.handle('window:exitApp', (event) => {
    validateSender(event);
    windowService.exitApp();
  });

  ipcMain.handle('window:checkForUpdates', async (event) => {
    validateSender(event);
    await windowService.checkForUpdates();
  });
}
