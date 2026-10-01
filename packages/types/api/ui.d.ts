import type { ConfigurationSectionContribution } from '../models/extension';

export type RegisterConfigurationTabOptions = {
  label: string;
  icon?: unknown;
  order?: number;
  render?: (container: HTMLElement) => void;
  sections?: ConfigurationSectionContribution[];
};

export type ContextMenuHandlerOptions = {
  condition?: (target: HTMLElement | null) => boolean;
};

export interface UiApi {
  registerView(viewId: string, renderFn: (container: HTMLElement) => void): void;
  /**
   * 코드만으로 컨텍스트 메뉴 항목을 추가합니다. manifest에 선언한 항목은 `onContextMenu`를 쓰세요.
   * @param target 메뉴가 뜨는 화면 (`calendar` 또는 확장 viewId)
   * @param callback 우클릭한 요소(날짜 칸, 일정 등)를 받습니다. 없으면 `null`.
   */
  registerContextMenu(
    target: string,
    command: string,
    label: string,
    callback: (target: HTMLElement | null) => void,
  ): void;
  /**
   * manifest `contributes.contextMenus`에 선언한 항목에 동작을 붙입니다.
   * 동작이 붙기 전까지 항목은 메뉴에 나타나지 않습니다.
   * @param id manifest 항목의 `id`
   * @param handler 우클릭한 요소를 받습니다. 없으면 `null`.
   * @param options.condition `false`를 돌려주면 그 우클릭에서는 항목을 숨깁니다.
   * @returns 동작을 떼어 내는 함수 (`deactivated`에서 호출)
   */
  onContextMenu(
    id: string,
    handler: (target: HTMLElement | null) => void,
    options?: ContextMenuHandlerOptions,
  ): () => void;
  /**
   * 앱 설정 모달에 extension configuration 탭을 추가합니다.
   * @param tabId 고유 탭 ID (`${extensionId}:${tabId}` 형태로 저장됨)
   */
  registerConfigurationTab(tabId: string, options: RegisterConfigurationTabOptions): void;
}
