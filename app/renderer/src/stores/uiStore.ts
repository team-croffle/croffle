import { defineStore } from 'pinia';
import { ref } from 'vue';

export const useUiStore = defineStore('ui', () => {
  const leftSidebarOpen = ref(true);
  const rightSidebarOpen = ref(false);
  const selectedDate = ref<string | null>(null);

  // 사이드바 토글 액션
  const toggleLeftSidebar = () => {
    leftSidebarOpen.value = !leftSidebarOpen.value;
  };

  const toggleRightSidebar = () => {
    rightSidebarOpen.value = !rightSidebarOpen.value;
  };

  const openRightSidebarWithDate = (date: string) => {
    selectedDate.value = date;
    rightSidebarOpen.value = true;
  };

  return {
    leftSidebarOpen,
    rightSidebarOpen,
    selectedDate,
    toggleLeftSidebar,
    toggleRightSidebar,
    openRightSidebarWithDate,
  };
});
