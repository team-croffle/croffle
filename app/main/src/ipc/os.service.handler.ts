import { ipcMain } from 'electron';
import { osService } from '../core/native-os/service/nativeOsService';
import type { ClipboardResult } from 'croffle';
import { eventService } from '../core/events/service/EventService';
import { AppEventType } from '../shared/enums';

export const registerOsIpcHandlers = (): void => {
  // 1. 알림
  ipcMain.handle('os:showNotification', (_, title, body) => {
    const result = osService.showNotification(title, body);

    // Add app event emit
    eventService.emit(AppEventType.NATIVE_OS_NOTIFICATION, title, body);

    return result;
  });

  // 2. 클립보드 읽기
  ipcMain.handle('os:getClipboard', (): ClipboardResult => {
    const result = osService.getClipboard();

    // Add app event emit
    eventService.emit(AppEventType.NATIVE_OS_CLIPBOARD_GET, result);

    return result;
  });

  // 3. 클립보드 쓰기
  ipcMain.handle(
    'os:setClipboard',
    (_, data: { type: 'text'; value: string } | { type: 'image'; value: Buffer }): void => {
      osService.setClipboard(data);

      // Add app event emit
      eventService.emit(AppEventType.NATIVE_OS_CLIPBOARD_SET, data);
    }
  );
};
