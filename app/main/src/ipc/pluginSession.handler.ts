import { ipcMain } from 'electron';
import { PluginSessionService } from '../core/plugin-session/service/PluginSessionService';

export const registerPluginSessionIpcHandlers = () => {
  ipcMain.handle('sessionStorage:get', async (_, { pluginId, key }) => {
    try {
      return PluginSessionService.get(pluginId, key);
    } catch (error) {
      console.error('[PluginSession] Get error:', error);
      return null;
    }
  });

  ipcMain.handle('sessionStorage:set', async (_, { pluginId, key, value }) => {
    try {
      PluginSessionService.set(pluginId, key, value);
    } catch (error) {
      console.error('[PluginSession] Set error:', error);
      throw error;
    }
  });

  ipcMain.handle('sessionStorage:delete', async (_, { pluginId, key }) => {
    try {
      return PluginSessionService.delete(pluginId, key);
    } catch (error) {
      console.error('[PluginSession] Delete error:', error);
      return false;
    }
  });

  ipcMain.handle('sessionStorage:clear', async (_, { pluginId }) => {
    try {
      PluginSessionService.clear(pluginId);
    } catch (error) {
      console.error('[PluginSession] Clear error:', error);
      throw error;
    }
  });
};
