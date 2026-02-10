import { registerWindowIpcHandlers } from './window.handler';
import { registerScheduleIpcHandlers } from './schedule.handler';
import { registerTagIpcHandlers } from './tag.handler';
import { registerPluginInfoIpcHandlers } from './pluginInfo.handler';
import { registerSettingsIpcHandlers } from './settings.handler';
import { registerOsIpcHandlers } from './os.service.handler';
import { registerPluginStorageIpcHandlers } from './pluginStorage.handler';
import { registerHttpIpcHandlers } from './http.handler';
import { registerEventIpcHandlers } from './event.handler';

export function registerAllIpcHandlers() {
  // Window IPC Handlers
  registerWindowIpcHandlers();

  // Schedule IPC Handlers
  registerScheduleIpcHandlers();

  // Tag IPC Handlers
  registerTagIpcHandlers();

  // Plugin Manager IPC Handlers
  registerPluginInfoIpcHandlers();

  // Search IPC Handlers

  // Application Management IPC Handlers

  // Settings IPC Handlers
  registerSettingsIpcHandlers();

  // Schdule Import/Export IPC Handlers

  // OS Service IPC Handlers
  registerOsIpcHandlers();

  // Plugin Storage IPC Handlers
  registerPluginStorageIpcHandlers();

  // HTTP IPC Handlers
  registerHttpIpcHandlers();

  // Event IPC Handlers
  registerEventIpcHandlers();
}
