const storage: Map<string, Map<string, any>> = new Map();

const getStore = (pluginId: string): Map<string, any> => {
  if (!storage.has(pluginId)) {
    storage.set(pluginId, new Map());
  }
  return storage.get(pluginId)!;
};

export const PluginSessionService = {
  set: (pluginId: string, key: string, value: any): void => {
    const store = getStore(pluginId);
    store.set(key, value);
    console.info(`[PluginSession] Set: [${pluginId}] ${key}`);
  },

  get: (pluginId: string, key: string): any => {
    const store = getStore(pluginId);
    return store.get(key) ?? null;
  },

  delete: (pluginId: string, key: string): boolean => {
    const store = getStore(pluginId);
    console.info(`[PluginSession] Delete: [${pluginId}] ${key}`);
    return store.delete(key);
  },

  clear: (pluginId: string): void => {
    const store = getStore(pluginId);
    store.clear();
    console.info(`[PluginSession] Cleared: [${pluginId}]`);
  },

  clearAll: (): void => {
    storage.clear();
    console.info(`[PluginSession] Cleared all plugin sessions`);
  },
};
