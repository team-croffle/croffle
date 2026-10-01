import type { ExtensionInfo } from '../models/extension.js';

export interface ExtensionInfoApi {
  getInstalled(): Promise<ExtensionInfo[]>;
  getEnabled(): Promise<ExtensionInfo[]>;
  getByName(name: string): Promise<ExtensionInfo | null>;
  install(data: Partial<ExtensionInfo>): Promise<ExtensionInfo>;
  installFromLocal(): Promise<ExtensionInfo | null>;
  toggle(name: string, enable: boolean): Promise<ExtensionInfo | null>;
  uninstall(name: string): Promise<boolean>;
}

/** Extension-scoped configuration (persisted via storage) */
export interface ExtensionConfigurationApi {
  get<T = Record<string, unknown>>(extensionId: string, storageKey?: string): Promise<T>;
  set(extensionId: string, values: Record<string, unknown>, storageKey?: string): Promise<void>;
}

export interface ExtensionStorageApi {
  get<T = unknown>(extensionId: string, key: string): Promise<T | null>;
  set(extensionId: string, key: string, value: unknown): Promise<void>;
  delete(extensionId: string, key: string): Promise<boolean>;
  clear(extensionId: string): Promise<void>;
}

export interface ExtensionSessionApi {
  get<T = unknown>(extensionId: string, key: string): Promise<T | null>;
  set<T = unknown>(extensionId: string, key: string, value: T): Promise<void>;
  delete(extensionId: string, key: string): Promise<boolean>;
  clear(extensionId: string): Promise<void>;
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

export interface ExtensionScopedConfigurationApi {
  get<T = Record<string, unknown>>(storageKey?: string): Promise<T>;
  set(values: Record<string, unknown>, storageKey?: string): Promise<void>;
}

export interface ExtensionsApi {
  info: ExtensionInfoApi;
  configuration: ExtensionConfigurationApi;
  storage: ExtensionStorageApi;
  session: ExtensionSessionApi;
}
