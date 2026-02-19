import { ipcMain } from 'electron';
import { eventService } from '../core/events/service/EventService';

export const registerEventIpcHandlers = (): void => {
  // register event ipc handlers

  // emit event to all windows
  ipcMain.on('event:emit', (event, eventName: string, ...args: unknown[]) => {
    const url = event.senderFrame?.url ?? '';

    if (!url.startsWith('file://') && !url.startsWith('app://')) {
      return;
    }

    if (typeof eventName !== 'string') {
      return;
    }

    eventService.emit(eventName, ...args);
  });
};
