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
 * Re-exported from the public package until 2.0 drops them there.
 */
export type {
  ExtensionConfigurationApi,
  ExtensionSessionApi,
  ExtensionStorageApi,
} from '@croffledev/croffle-types';
