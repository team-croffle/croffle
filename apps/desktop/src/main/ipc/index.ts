import { logger } from '../logger';
import { registerEventIpcHandlers } from './event.handler';
import { registerExtensionInfoIpcHandlers } from './extension-info.handler';
import { registerExtensionSessionIpcHandlers } from './extension-session.handler';
import { registerExtensionStorageIpcHandlers } from './extension-storage.handler';
import { registerHttpIpcHandlers } from './http.handler';
import { registerOsIpcHandlers } from './os.handler';
import { registerScheduleIpcHandlers } from './schedule.handler';
import { registerSearchIpcHandlers } from './search.handler';
import { registerSettingsIpcHandlers } from './settings.handler';
import { registerSyncIpcHandlers } from './sync.handler';
import { registerTagIpcHandlers } from './tag.handler';
import { registerWindowIpcHandlers } from './window.handler';

export function registerAllIpcHandlers() {
  logger.debug('IPC', 'Registering Window IPC Handlers');
  // Window IPC Handlers
  registerWindowIpcHandlers();

  logger.debug('IPC', 'Registering Schedule IPC Handlers');
  // Schedule IPC Handlers
  registerScheduleIpcHandlers();

  logger.debug('IPC', 'Registering Tag IPC Handlers');
  // Tag IPC Handlers
  registerTagIpcHandlers();

  logger.debug('IPC', 'Registering Plugin Manager IPC Handlers');
  // Plugin Manager IPC Handlers
  registerExtensionInfoIpcHandlers();

  logger.debug('IPC', 'Registering Search IPC Handlers');
  // Search IPC Handlers
  registerSearchIpcHandlers();

  // Application Management IPC Handlers

  logger.debug('IPC', 'Registering Settings IPC Handlers');
  // Settings IPC Handlers
  registerSettingsIpcHandlers();

  // Schdule Import/Export IPC Handlers

  logger.debug('IPC', 'Registering OS Service IPC Handlers');
  // OS Service IPC Handlers
  registerOsIpcHandlers();

  logger.debug('IPC', 'Registering Plugin Storage IPC Handlers');
  // Plugin Storage IPC Handlers
  registerExtensionStorageIpcHandlers();

  logger.debug('IPC', 'Registering HTTP IPC Handlers');
  // HTTP IPC Handlers
  registerHttpIpcHandlers();

  logger.debug('IPC', 'Registering Event IPC Handlers');
  // Event IPC Handlers
  registerEventIpcHandlers();

  logger.debug('IPC', 'Registering Plugin Session IPC Handlers');
  // Plugin Session IPC Handlers
  registerExtensionSessionIpcHandlers();

  registerSyncIpcHandlers();

  logger.info('IPC', 'All IPC handlers registered successfully');
}
