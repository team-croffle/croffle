import { ipcMain } from 'electron';
import { PluginSessionService } from '../core/plugin-session/service/PluginSessionService';
import { eventService } from '../core/events/service/EventService';

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
  const pluginId = data.pluginId;
  const key = data.key;
  const value = data.value;

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
      eventService.emit('sessionStorage:set', { pluginId, key });
    } catch (error) {
      console.error('[PluginSession] Set error:', error);
      throw error;
    }
  });

  ipcMain.handle('sessionStorage:delete', async (_, payload: unknown = {}) => {
    try {
      const { pluginId, key } = validateArgs(payload, true);
      const result = PluginSessionService.delete(pluginId, key as string);
      eventService.emit('sessionStorage:delete', { pluginId, key, result });
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
      eventService.emit('sessionStorage:clear', { pluginId });
    } catch (error) {
      console.error('[PluginSession] Clear error:', error);
      throw error;
    }
  });
};
