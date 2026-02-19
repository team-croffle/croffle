import EventEmitter from 'events';
import { BrowserWindow } from 'electron';
import Logger from 'electron-log';

class EventService extends EventEmitter {
  constructor() {
    super();
  }

  /**
   * Emit and event, and broadcast to all renderer processes (windows)
   * @param eventName Event name
   * @param args Arguments
   * @returns true if the event was emitted, false otherwise
   */
  public emit(eventName: string, ...args: unknown[]): boolean {
    const result = super.emit(eventName, ...args);

    // send ipc message to all windows which aren't destroyed (means active windows)
    BrowserWindow.getAllWindows().forEach((win) => {
      if (!win.isDestroyed()) {
        // send event through a single ipc channel with 'croffle:app:event'
        try {
          win.webContents.send('croffle:app:event', eventName, ...args);
        } catch (err) {
          Logger.info(
            `[EventService] Failed to send event ${eventName} to window ${win.id}: ${err}`
          );
        }
      }
    });

    return result;
  }

  /**
   * Register an event listener
   * @param eventName Event name
   * @param listener Listener function
   * @returns this
   */
  public on(eventName: string, listener: (...args: unknown[]) => void): this {
    super.on(eventName, listener);
    return this;
  }
}

// singleton instance
export const eventService = new EventService();
