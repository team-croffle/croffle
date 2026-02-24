import { ipcMain } from 'electron';
import { PluginSessionService } from '../core/plugin-session/service/PluginSessionService';
import { eventService } from '../core/events/service/EventService';
import { AppEventType, event } from 'croffle';

const validateArgs = (pluginId: unknown, key?: unknown) => {
  if (typeof pluginId !== 'string' || pluginId.trim() === '') {
    throw new Error('Invalid pluginId');
  }
  if (key !== undefined && (typeof key !== 'string' || key.trim() === '')) {
    throw new Error('Invalid key');
  }
};

export const registerPluginSessionIpcHandlers = () => {
  ipcMain.handle('sessionStorage:get', async (_, { pluginId, key }) => {
    try {
      validateArgs(pluginId, key);
      const value = PluginSessionService.get(pluginId, key);
      eventService.emit(AppEventType.PLUGIN_SESSION_STORAGE_GET, { pluginId, key, value });
      return value;
    } catch (error) {
      console.error('[PluginSession] Get error:', error);
      throw error;
    }
  });

  ipcMain.handle('sessionStorage:set', async (_, { pluginId, key, value }) => {
    try {
      validateArgs(pluginId, key);
      PluginSessionService.set(pluginId, key, value);
      eventService.emit(AppEventType.PLUGIN_SESSION_STORAGE_SET, { pluginId, key, value });
    } catch (error) {
      console.error('[PluginSession] Set error:', error);
      throw error;
    }
  });

  ipcMain.handle('sessionStorage:delete', async (_, { pluginId, key }) => {
    try {
      validateArgs(pluginId, key);
      const result = PluginSessionService.delete(pluginId, key);
      eventService.emit(AppEventType.PLUGIN_SESSION_STORAGE_DELETE, { pluginId, key, result });
      return result;
    } catch (error) {
      console.error('[PluginSession] Delete error:', error);
      return false;
    }
  });

  ipcMain.handle('sessionStorage:clear', async (_, { pluginId }) => {
    try {
      validateArgs(pluginId);
      PluginSessionService.clear(pluginId);
      eventService.emit(AppEventType.PLUGIN_SESSION_STORAGE_CLEAR, { pluginId });
    } catch (error) {
      console.error('[PluginSession] Clear error:', error);
      throw error;
    }
  });
};
