import { BrowserWindow, ipcMain, IpcMainInvokeEvent } from 'electron';
import { windowService } from '../core/window/service/WindowService';
import { eventService } from '../core/events/service/EventService';
import { AppEventType } from '../shared/enums';

const validateSender = (event: IpcMainInvokeEvent): BrowserWindow => {
  const window = BrowserWindow.fromWebContents(event.sender);
  const frameUrl = event.senderFrame?.url || '';

  if (!window) {
    throw new Error('[Security] Request from unknown window.');
  }

  const isSafeOrigin =
    frameUrl?.startsWith('file://') || frameUrl === '' || /^http:\/\/localhost:\d+/.test(frameUrl);

  if (!isSafeOrigin) {
    console.error(`[Security] Blocked IPC from unauthorized origin: ${frameUrl}`);
    throw new Error('Unauthorized IPC sender');
  }
  return window;
};

export function registerWindowIpcHandlers() {
  ipcMain.handle('window:minimize', (event) => {
    const window = validateSender(event);
    window.minimize();

    // Add app event emit
    eventService.emit(AppEventType.WINDOW_MINIMIZE, window);
  });

  ipcMain.handle('window:maximize', (event) => {
    const window = validateSender(event);
    if (window.isMaximized()) {
      window.unmaximize();

      // Add app event emit
      eventService.emit(AppEventType.WINDOW_RESTORE, window);
    } else {
      window.maximize();

      // Add app event emit
      eventService.emit(AppEventType.WINDOW_MAXIMIZE, window);
    }
  });

  ipcMain.handle('window:close', (event) => {
    const window = validateSender(event);
    window.close();

    // Add app event emit
    eventService.emit(AppEventType.WINDOW_CLOSE, window);
  });

  ipcMain.handle('window:exitApp', (event) => {
    validateSender(event);
    windowService.exitApp();
  });

  ipcMain.handle('window:checkForUpdates', async (event) => {
    validateSender(event);
    await windowService.checkForUpdates();

    // Add app event emit
    eventService.emit(AppEventType.WINDOW_CHECK_FOR_UPDATES, event);
  });

  ipcMain.handle('window:setCloseToTrayMode', (event, enabled: boolean) => {
    validateSender(event);
    windowService.setCloseToTrayMode(enabled);
  });
}
