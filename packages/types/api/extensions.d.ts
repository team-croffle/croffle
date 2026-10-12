import type { ExtensionInfo } from '../models/extension.js';

/** Read-only view of installed extensions. Install / toggle / uninstall are host-only. */
export interface ExtensionInfoApi {
  getInstalled(): Promise<ExtensionInfo[]>;
  getEnabled(): Promise<ExtensionInfo[]>;
  getByName(name: string): Promise<ExtensionInfo | null>;
}

/** Bound to a single extension id (passed via ExtensionContext) */
export interface ExtensionScopedStorageApi {
  get<T = unknown>(key: string): Promise<T | null>;
  set(key: string, value: unknown): Promise<void>;
  delete(key: string): Promise<boolean>;
  clear(): Promise<void>;
}

export interface ExtensionScopedSessionApi {
  get<T = unknown>(key: string): Promise<T | null>;
  set<T = unknown>(key: string, value: T): Promise<void>;
  delete(key: string): Promise<boolean>;
  clear(): Promise<void>;
}

/** Extension-scoped configuration (persisted via storage) */
export interface ExtensionScopedConfigurationApi {
  get<T = Record<string, unknown>>(storageKey?: string): Promise<T>;
  set(values: Record<string, unknown>, storageKey?: string): Promise<void>;
}

export interface ExtensionsApi {
  info: ExtensionInfoApi;
}
