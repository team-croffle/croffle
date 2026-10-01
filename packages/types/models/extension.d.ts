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

/** Manifest `contributes.views` 항목 */
export type ViewManifest = {
  id: string;
  title: string;
  subtitle: string;
  /** JSON에서는 보통 string; 호스트 기본 메뉴는 컴포넌트도 허용 */
  icon?: unknown;
  url: string;
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

export type ExtensionContributes = {
  views?: ViewManifest[];
  contextMenus?: ContextMenuManifest[];
  configuration?: ConfigurationTabManifest[];
};

/** `croffle-manifest.json` on disk */
export type CroffleManifest = {
  id: string;
  name: string;
  version: string;
  author: string;
  description?: string;
  main?: string;
  engines?: {
    /** e.g. ">=1.0.0" — currently only `>=x.y.z` is enforced */
    croffle?: string;
  };
  contributes?: ExtensionContributes;
};

/** Host-loaded extension (manifest + enabled) — also exposed via ExtensionsApi */
export type ExtensionInfo = CroffleManifest & {
  enabled: boolean;
};
