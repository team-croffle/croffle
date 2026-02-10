import { ipcMain } from 'electron';
import { eventService } from '../core/events/service/EventService';

export const registerEventIpcHandlers = (): void => {
  // register event ipc handlers

  // emit event to all windows
  ipcMain.on('event:emit', (_event, eventName: string, ...args: unknown[]) => {
    eventService.emit(eventName, ...args);
  });
};
