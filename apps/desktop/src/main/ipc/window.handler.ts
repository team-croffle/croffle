import type { IpcMainInvokeEvent } from 'electron';
import { BrowserWindow, ipcMain } from 'electron';

import { logger } from '../logger';
import { windowService } from '../window/window-service';

const validateSender = (event: IpcMainInvokeEvent): BrowserWindow => {
  const window = BrowserWindow.fromWebContents(event.sender);
  const frameUrl = event.senderFrame?.url || '';

  if (!window) {
    throw new Error('[Security] Request from unknown window.');
  }

  const isSafeOrigin =
    frameUrl?.startsWith('file://') || frameUrl === '' || /^http:\/\/localhost:\d+/.test(frameUrl);

  if (!isSafeOrigin) {
    logger.error('Security', `Blocked IPC from unauthorized origin: ${frameUrl}`);
    throw new Error('Unauthorized IPC sender');
  }
  return window;
};

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
