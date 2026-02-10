import { ipcMain } from 'electron';
import { eventService } from '../core/events/service/EventService';

export const registerEventIpcHandlers = (): void => {
  // register event ipc handlers

  // emit event to all windows
  ipcMain.on('event:emit', (_event, eventName: string, ...args: unknown[]) => {
    eventService.emit(eventName, ...args);
  });

  // listen to event from all windows
  ipcMain.on('event:on', (_event, eventName: string, callback: (payload: unknown) => void) => {
    eventService.on(eventName, callback);
  });
};
