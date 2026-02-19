import { event } from 'croffle';
import { ipcRenderer, IpcRendererEvent } from 'electron';

type EventApi = typeof event;

export const eventApi = {
  emit: (eventName: string, ...args: unknown[]) => {
    ipcRenderer.send('event:emit', eventName, ...args);
  },

  on: (eventName: string, callback: (...args: unknown[]) => void) => {
    const subscription = (_: IpcRendererEvent, eventType: string, ...args: unknown[]) => {
      if (eventType === eventName) {
        callback(...args);
      }
    };

    ipcRenderer.on('croffle:app:event', subscription);

    return () => {
      ipcRenderer.off('croffle:app:event', subscription);
    };
  },
} satisfies EventApi;
