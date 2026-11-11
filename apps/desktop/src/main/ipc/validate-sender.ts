import type { IpcMainInvokeEvent } from 'electron';
import { BrowserWindow } from 'electron';

import { logger } from '../logger';

/** Rejects IPC that does not come from the app's own window (file:// or the dev server). */
export const validateSender = (event: IpcMainInvokeEvent): BrowserWindow => {
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
