import { ipcMain } from 'electron';
import { PluginStorageService } from '../core/plugin-data/service/PluginStorageService';
import { eventService } from '../core/events/service/EventService';
import { AppEventType } from '../shared/enums';

export const registerPluginStorageIpcHandlers = () => {
  ipcMain.handle('app:storage:get', async (_, { pluginId, key }) => {
    // Add app event emit
    eventService.emit(AppEventType.PLUGIN_STORAGE_GET, pluginId, key);

    return await PluginStorageService.get(pluginId, key);
  });
  ipcMain.handle('app:storage:set', async (_, { pluginId, key, value }) => {
    // Add app event emit
    eventService.emit(AppEventType.PLUGIN_STORAGE_SET, pluginId, key, value);

    await PluginStorageService.set(pluginId, key, value);
  });
};
