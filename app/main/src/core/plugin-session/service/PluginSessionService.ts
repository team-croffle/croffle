const storage: Map<string, Map<string, unknown>> = new Map();

const getStore = (pluginId: string, autoCreate = false): Map<string, unknown> | undefined => {
  if (!storage.has(pluginId)) {
    if (autoCreate) {
      storage.set(pluginId, new Map());
    } else {
      return undefined;
    }
  }
  return storage.get(pluginId)!;
};

export const PluginSessionService = {
  set: <T = unknown>(pluginId: string, key: string, value: T): void => {
    const store = getStore(pluginId, true)!;
    store.set(key, value);
    console.info(`[PluginSession] Set: [${pluginId}] ${key}`);
  },

  get: <T = unknown>(pluginId: string, key: string): T | null => {
    const store = getStore(pluginId);
    if (!store) return null;
    return (store.get(key) as T) ?? null;
  },

  delete: (pluginId: string, key: string): boolean => {
    const store = getStore(pluginId, false);
    if (!store) return false;
    const result = store.delete(key);
    if (result) {
      console.info(`[PluginSession] Delete: [${pluginId}] ${key}`);
    }
    return result;
  },

  clear: (pluginId: string): void => {
    if (storage.has(pluginId)) {
      storage.delete(pluginId);
      console.info(`[PluginSession] Cleared: [${pluginId}]`);
    }
  },

  clearAll: (): void => {
    storage.clear();
    console.info(`[PluginSession] Cleared all plugin sessions`);
  },
};
