import type { ExtensionInfoApi } from '@croffledev/croffle-types';

import type { ExtensionInfo } from '../models';

/** Public read API plus the host-only install / toggle / uninstall. */
export interface InternalExtensionInfoApi extends ExtensionInfoApi {
  install(data: Partial<ExtensionInfo>): Promise<ExtensionInfo>;
  installFromLocal(): Promise<ExtensionInfo | null>;
  toggle(name: string, enable: boolean): Promise<ExtensionInfo | null>;
  uninstall(name: string): Promise<boolean>;
}

/**
 * extensionId-keyed storage, session and configuration. Host-only: the loader binds
 * the id and hands extensions the scoped variants (`ExtensionScoped*Api`).
 */
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
