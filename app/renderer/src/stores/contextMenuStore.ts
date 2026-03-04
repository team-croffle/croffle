import { defineStore } from 'pinia';
import { ref } from 'vue';

export interface ContextMenuItem {
  id: string; // 메뉴의 고유 식별자 ex) add-schedule, delete-schedule
  label: string; // 메뉴에 표시될 텍스트
  action: () => void; // 메뉴 클릭 시 실행될 함수
  disabled?: boolean; // 메뉴 활성화 여부 (선택 사항)
}

export const useContextMenuStore = defineStore('contextMenu', () => {
  // 현재 열려 있는 컨텍스트 메뉴의 아이템 목록
  const items = ref<ContextMenuItem[]>([]);

  const setMenu = (newItems: ContextMenuItem[]) => {
    items.value = newItems;
  };

  const clearMenu = () => {
    items.value = [];
  };

  // 기존 메뉴는 유지하면서 항목만 추가할 때 사용 (추후 플러그인 확장용)
  const addItems = (newItems: ContextMenuItem[]) => {
    items.value = [...items.value, ...newItems];
  };

  return {
    items,
    setMenu,
    clearMenu,
    addItems,
  };
});
