import { ipcMain } from 'electron';
import { PluginSessionService } from '../core/plugin-session/service/PluginSessionService';
import { eventService } from '../core/events/service/EventService';
import { AppEventType } from '../shared/enums';

const validateArgs = (
  payload: unknown,
  requireKey: boolean = true
): { pluginId: string; key?: string; value?: unknown } => {
  // payload가 비어있거나 객체가 아니면 에러
  if (!payload || typeof payload !== 'object') {
    throw new Error('Invalid payload');
  }

  // 객체임이 확인되었으므로 안전하게 Record로 캐스팅
  const data = payload as Record<string, unknown>;
  const { pluginId, key, value } = data;

  if (typeof pluginId !== 'string' || pluginId.trim() === '') {
    throw new Error('Invalid pluginId');
  }
  if (requireKey && (typeof key !== 'string' || key.trim() === '')) {
    throw new Error('Invalid key');
  }

  return {
    pluginId,
    key: key as string | undefined,
    value,
  };
};

export const registerPluginSessionIpcHandlers = () => {
  ipcMain.handle('sessionStorage:get', async (_, payload: unknown = {}) => {
    try {
      const { pluginId, key } = validateArgs(payload, true);
      const value = PluginSessionService.get(pluginId, key as string);
      eventService.emit(AppEventType.PLUGIN_SESSION_STORAGE_GET, { pluginId, key });
      return value;
    } catch (error) {
      console.error('[PluginSession] Get error:', error);
      throw error;
    }
  });

  ipcMain.handle('sessionStorage:set', async (_, payload: unknown = {}) => {
    try {
      const { pluginId, key, value } = validateArgs(payload, true);
      PluginSessionService.set(pluginId, key as string, value);
      eventService.emit(AppEventType.PLUGIN_SESSION_STORAGE_SET, { pluginId, key });
    } catch (error) {
      console.error('[PluginSession] Set error:', error);
      throw error;
    }
  });

  ipcMain.handle('sessionStorage:delete', async (_, payload: unknown = {}) => {
    try {
      const { pluginId, key } = validateArgs(payload, true);
      const result = PluginSessionService.delete(pluginId, key as string);
      eventService.emit(AppEventType.PLUGIN_SESSION_STORAGE_DELETE, { pluginId, key, result });
      return result;
    } catch (error) {
      console.error('[PluginSession] Delete error:', error);
      throw error;
    }
  });

  ipcMain.handle('sessionStorage:clear', async (_, payload: unknown = {}) => {
    try {
      const { pluginId } = validateArgs(payload, false);
      PluginSessionService.clear(pluginId);
      eventService.emit(AppEventType.PLUGIN_SESSION_STORAGE_CLEAR, { pluginId });
    } catch (error) {
      console.error('[PluginSession] Clear error:', error);
      throw error;
    }
  });

  ipcMain.handle('sessionStorage:clearAll', async () => {
    try {
      PluginSessionService.clearAll();
      eventService.emit(AppEventType.PLUGIN_SESSION_STORAGE_CLEAR_ALL, {});
    } catch (error) {
      console.error('[PluginSession] ClearAll error:', error);
      throw error;
    }
  });
};
