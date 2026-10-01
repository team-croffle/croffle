export type ConfigItemType = 'string' | 'number' | 'boolean' | 'select' | 'path';

export type ConfigItemSchema<T = unknown> = {
  type: ConfigItemType;
  label: string;
  description?: string;
  defaultValue: T;
  options?: {
    label: string;
    value: T extends string | number ? T : never;
  }[];
};

export type ConfigurationSectionContribution = {
  id: string;
  title?: string;
  description?: string;
  items: Record<string, ConfigItemSchema>;
};

/** Manifest `contributes.configuration` 항목 (런타임 extensionId/render 제외) */
export type ConfigurationTabManifest = {
  id: string;
  label: string;
  icon?: unknown;
  order?: number;
  sections?: ConfigurationSectionContribution[];
};

/** Host runtime: settings modal tab (includes render hook). */
export type ConfigurationTabContribution = {
  id: string;
  label: string;
  icon?: unknown;
  extensionId: string;
  extensionName?: string;
  order?: number;
  render?: (container: HTMLElement) => void;
  sections?: ConfigurationSectionContribution[];
};

/** Manifest `contributes.views` 항목 */
export type ViewManifest = {
  id: string;
  title: string;
  subtitle: string;
  icon?: unknown;
  url: string;
};

/** Host runtime: sidebar / registered view. */
export type FeatureView = ViewManifest & {
  active?: boolean;
  extensionName?: string;
  extensionId?: string;
};

/**
 * Manifest `contributes.contextMenus` 항목. 동작은 코드에서 `context.ui.onContextMenu(id, …)`로 붙입니다.
 */
export type ContextMenuManifest = {
  /** 확장 안에서 고유. `onContextMenu`의 id */
  id: string;
  /** 메뉴에 그대로 표시되는 문구 */
  label: string;
  /** 메뉴가 뜨는 화면 (`calendar` 또는 확장 viewId). 생략하면 모든 화면 */
  targetView?: string[];
  /** `true`면 항목을 비활성으로 표시 */
  disabled?: boolean;
};

/** Host runtime: context menu with callbacks. */
export type FeatureContextMenu = ContextMenuManifest & {
  action: (targetElement: HTMLElement | null) => void;
  condition?: (targetElement: HTMLElement | null) => boolean;
  extensionId?: string;
  extensionName?: string;
};

export type ExtensionContributes = {
  views?: ViewManifest[];
  contextMenus?: ContextMenuManifest[];
  configuration?: ConfigurationTabManifest[];
};

export type ExtensionEngines = {
  croffle?: string;
};

/** `croffle-manifest.json` on disk */
export type CroffleManifest = {
  id: string;
  name: string;
  version: string;
  author: string;
  description?: string;
  main?: string;
  engines?: ExtensionEngines;
  contributes?: ExtensionContributes;
};

/** Host-loaded extension (manifest + enabled). */
export type ExtensionInfo = CroffleManifest & {
  enabled: boolean;
};
