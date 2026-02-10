import { ipcMain } from 'electron';
import { PluginStorageService } from '../core/plugin-data/service/PluginStorageService';
import { eventService } from '../core/events/service/EventService';
import { AppEventType } from '../shared/enums';

export const registerPluginStorageIpcHandlers = () => {
  ipcMain.handle('app:storage:get', (_, { pluginId, key }) => {
    // Add app event emit
    eventService.emit(AppEventType.PLUGIN_STORAGE_GET, pluginId, key);

    return PluginStorageService.get(pluginId, key);
  });
  ipcMain.handle('app:storage:set', (_, { pluginId, key, value }) => {
    // Add app event emit
    eventService.emit(AppEventType.PLUGIN_STORAGE_SET, pluginId, key, value);

    PluginStorageService.set(pluginId, key, value);
  });
};
