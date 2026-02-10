import { event } from 'croffle';
import { ipcRenderer, IpcRendererEvent } from 'electron';

type EventApi = typeof event;

export const eventApi = {
  emit: (eventName: string, payload?: unknown) => {
    ipcRenderer.send('event:emit', eventName, payload);
  },

  on: (eventName: string, callback: (payload: unknown) => void) => {
    const subscription = (_: IpcRendererEvent, eventType: string, payload: unknown) => {
      if (eventType === eventName) {
        callback(payload);
      }
    };

    ipcRenderer.on('croffle:app:event', subscription);

    return () => {
      ipcRenderer.off('croffle:app:event', subscription);
    };
  },
} satisfies EventApi;
