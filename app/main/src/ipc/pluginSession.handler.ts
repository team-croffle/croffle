import { ipcMain } from 'electron';
import { PluginSessionService } from '../core/plugin-session/service/PluginSessionService';
import { eventService } from '../core/events/service/EventService';

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
      eventService.emit('sessionStorage:get', { pluginId, key, value });
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
      eventService.emit('sessionStorage:set', { pluginId, key, value });
    } catch (error) {
      console.error('[PluginSession] Set error:', error);
      throw error;
    }
  });

  ipcMain.handle('sessionStorage:delete', async (_, { pluginId, key }) => {
    try {
      validateArgs(pluginId, key);
      const result = PluginSessionService.delete(pluginId, key);
      eventService.emit('sessionStorage:delete', { pluginId, key, result });
      return result;
    } catch (error) {
      console.error('[PluginSession] Delete error:', error);
      throw error;
    }
  });

  ipcMain.handle('sessionStorage:clear', async (_, { pluginId }) => {
    try {
      validateArgs(pluginId);
      PluginSessionService.clear(pluginId);
      eventService.emit('sessionStorage:clear', { pluginId });
    } catch (error) {
      console.error('[PluginSession] Clear error:', error);
      throw error;
    }
  });
};
