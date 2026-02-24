const storage: Map<string, Map<string, any>> = new Map();

const getStore = (pluginId: string): Map<string, any> => {
  if (!storage.has(pluginId)) {
    storage.set(pluginId, new Map());
  }
  return storage.get(pluginId)!;
};

export const PluginSessionService = {
  set: <T = unknown>(pluginId: string, key: string, value: T): void => {
    const store = getStore(pluginId);
    store.set(key, value);
    console.info(`[PluginSession] Set: [${pluginId}] ${key}`);
  },

  get: <T = unknown>(pluginId: string, key: string): T | null => {
    const store = getStore(pluginId);
    return store.get(key) ?? null;
  },

  delete: (pluginId: string, key: string): boolean => {
    const store = getStore(pluginId);
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
