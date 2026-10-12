import type { CroffleInternalAPI } from '@croffledev/common';
import type { CroffleAPI } from '@croffledev/croffle-types';

declare global {
  /** contextBridge로 노출된 호스트 API (bare identifier) */
  const croffle: CroffleAPI;
  /** 호스트 전용 API. 확장 컨텍스트에는 절대 넘기지 않는다 */
  const croffleInternal: CroffleInternalAPI;

  interface Window {
    croffle: CroffleAPI;
    croffleInternal: CroffleInternalAPI;
  }
}

export {};
