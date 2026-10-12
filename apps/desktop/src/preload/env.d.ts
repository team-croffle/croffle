import type { CroffleInternalAPI } from '@croffledev/common';
import type { CroffleAPI } from '@croffledev/croffle-types';

declare global {
  /** contextBridge로 노출된 호스트 API (bare identifier) */
  const croffle: CroffleAPI;

  interface Window {
    croffle: CroffleAPI;
    /** 호스트 전용. renderer는 `services/host-bridge`로 캡처한 뒤 전역에서 지운다 */
    croffleInternal: CroffleInternalAPI;
  }
}

export {};
