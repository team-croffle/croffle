import { ipcMain } from 'electron';

import { notificationService } from '../notification/notification-service';
import { validateSender } from './validate-sender';

/** Host-only: the bell's notification list (`croffleInternal.notifications`). */
export function registerNotificationsIpcHandlers() {
  ipcMain.handle('notifications:list', (event) => {
    validateSender(event);
    return notificationService.list();
  });

  ipcMain.handle('notifications:markAllRead', (event) => {
    validateSender(event);
    notificationService.markAllRead();
  });
}
