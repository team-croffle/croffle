import type { CroffleInternalBridge } from '@croffledev/common';
import type { CroffleAPI } from '@croffledev/croffle-types';

declare global {
  /** contextBridge로 노출된 호스트 API (bare identifier) */
  const croffle: CroffleAPI;

  interface Window {
    croffle: CroffleAPI;
    /** 호스트 전용. `take()`는 한 번만 API를 돌려준다 — renderer host가 `services/host-bridge`에서 가져간다 */
    croffleInternal: CroffleInternalBridge;
  }
}

export {};
