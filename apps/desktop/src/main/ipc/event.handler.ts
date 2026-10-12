import { is } from '@electron-toolkit/utils';
import { ipcMain } from 'electron';

import { eventService } from '../event-bus/event-service';
import { logger } from '../logger';

/** Same rule as window.handler: packaged app origins, plus the Vite dev server in dev. */
const isTrustedSender = (url: string): boolean =>
  url.startsWith('file://') ||
  url.startsWith('app://') ||
  (is.dev && /^http:\/\/localhost:\d+/.test(url));

export const registerEventIpcHandlers = (): void => {
  // emit event to all windows
  ipcMain.on('event:emit', (event, eventName: string, ...args: unknown[]) => {
    const url = event.senderFrame?.url ?? '';

    if (!isTrustedSender(url)) {
      logger.warn('Security', `Ignored event:emit from unauthorized origin: ${url}`);
      return;
    }

    if (typeof eventName !== 'string') {
      return;
    }

    eventService.emit(eventName, ...args);
  });
};
