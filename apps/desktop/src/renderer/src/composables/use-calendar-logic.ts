import type FullCalendarComponent from '@fullcalendar/vue3';
import type { Ref } from 'vue';

import { useUiStore } from '@/stores/ui-store';

export function useCalendarLogic() {
  const uiStore = useUiStore();

  let resizeObserver: ResizeObserver | null = null;
  let animationFrameId: number | null = null;

  // 캘린더 리사이징 최적화 로직
  const startResizeObserver = (
    containerRef: Ref<HTMLElement | null>,
    calendarRef: Ref<InstanceType<typeof FullCalendarComponent> | null>,
  ) => {
    if (!containerRef.value || !calendarRef.value) {
      return;
    }

    resizeObserver = new ResizeObserver(() => {
      if (animationFrameId !== null) {
        return;
      }
      animationFrameId = requestAnimationFrame(() => {
        calendarRef.value?.getApi().updateSize();
        animationFrameId = null;
      });
    });
    resizeObserver.observe(containerRef.value);
  };

  const stopResizeObserver = () => {
    if (resizeObserver) {
      resizeObserver.disconnect();
      resizeObserver = null;
    }
    if (animationFrameId !== null) {
      cancelAnimationFrame(animationFrameId);
      animationFrameId = null;
    }
  };

  // 더블클릭 핸들러
  let clickCount = 0;
  let clickTimer: ReturnType<typeof setTimeout> | null = null;
  let lastClickedKey: string | null = null; // 추가

  const handleDoubleClick = (clickKey: string, onDoubleClick: () => void) => {
    // 이전 클릭과 다른 날짜를 누르면 카운트 초기화
    if (lastClickedKey !== clickKey) {
      if (clickTimer) {
        clearTimeout(clickTimer);
        clickTimer = null;
      }
      clickCount = 0;
      lastClickedKey = clickKey;
    }

    clickCount++;

    if (clickCount === 1) {
      clickTimer = setTimeout(() => {
        clickCount = 0;
        lastClickedKey = null;
        clickTimer = null;
      }, 300);
    } else if (clickCount === 2) {
      if (clickTimer) {
        clearTimeout(clickTimer);
        clickTimer = null;
      }
      clickCount = 0;
      lastClickedKey = null;
      onDoubleClick();
    }
  };

  // 날짜를 더블 클릭했을 때
  const handleDateDoubleClick = (dateStr: string) => {
    handleDoubleClick(`date:${dateStr}`, () => {
      uiStore.openRightSidebarWithDate(dateStr);
    });
  };

  // 이벤트를 더블 클릭했을 때
  const handleEventDoubleClick = (eventId: string) => {
    handleDoubleClick(`event:${eventId}`, () => {
      uiStore.openScheduleModal('view', eventId);
    });
  };

  return {
    startResizeObserver,
    stopResizeObserver,
    handleDateDoubleClick,
    handleEventDoubleClick,
  };
}
