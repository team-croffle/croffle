import { ipcMain } from 'electron';
import { pluginService } from '../core/plugin-info/service/PluginInfoService';
import { PluginInfo } from 'croffle';
import { validatePluginName } from '../core/helper/pluginValidator';
import { PluginInfoMapper } from '../core/plugin-info/mapper/PluginInfoMapper';
import { eventService } from '../core/events/service/EventService';
import { AppEventType } from '../shared/enums';

export const registerPluginInfoIpcHandlers = (): void => {
  ipcMain.handle('pluginInfo:getInstalledPlugins', async (): Promise<PluginInfo[]> => {
    const entity = await pluginService.getInstalledPlugins();

    // Add app event emit
    eventService.emit(AppEventType.PLUGIN_INFO_GET_INSTALLED, entity);

    return entity.map(PluginInfoMapper.toInterface);
  });

  ipcMain.handle('pluginInfo:getEnabledPlugins', async (): Promise<PluginInfo[]> => {
    const entity = await pluginService.getEnabledPlugins();

    // Add app event emit
    eventService.emit(AppEventType.PLUGIN_INFO_GET_ENABLED, entity);

    return entity.map(PluginInfoMapper.toInterface);
  });

  ipcMain.handle(
    'pluginInfo:getPluginByName',
    async (_, name: string): Promise<PluginInfo | null> => {
      validatePluginName(name);
      const entity = await pluginService.getPluginByName(name);

      // Add app event emit
      eventService.emit(AppEventType.PLUGIN_INFO_GET_BY_NAME, entity);

      return entity ? PluginInfoMapper.toInterface(entity) : null;
    }
  );

  ipcMain.handle(
    'pluginInfo:installPlugin',
    async (_, pluginData: Partial<PluginInfo>): Promise<PluginInfo> => {
      const entity = await pluginService.installPlugin(pluginData);

      // Add app event emit
      eventService.emit(AppEventType.PLUGIN_INFO_INSTALL, entity);

      return PluginInfoMapper.toInterface(entity);
    }
  );

  ipcMain.handle(
    'pluginInfo:togglePlugin',
    async (_, name: string, enable: boolean): Promise<PluginInfo | null> => {
      const entity = await pluginService.togglePlugin(name, enable);

      // Add app event emit
      eventService.emit(AppEventType.PLUGIN_INFO_TOGGLE, entity);

      return entity ? PluginInfoMapper.toInterface(entity) : null;
    }
  );

  ipcMain.handle('pluginInfo:uninstallPlugin', async (_, name: string): Promise<boolean> => {
    validatePluginName(name);
    const result = await pluginService.uninstallPlugin(name);

    // Add app event emit
    eventService.emit(AppEventType.PLUGIN_INFO_UNINSTALL, result);
    return result;
  });
};
