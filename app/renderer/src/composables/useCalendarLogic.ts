import { useUiStore } from '@/stores/uiStore';
import type FullCalendarComponent from '@fullcalendar/vue3';
import { type Ref } from 'vue';

export function useCalendarLogic() {
  const uiStore = useUiStore();

  let resizeObserver: ResizeObserver | null = null;
  let animationFrameId: number | null = null;

  // 캘린더 리사이징 최적화 로직
  const startResizeObserver = (
    containerRef: Ref<HTMLElement | null>,
    calendarRef: Ref<InstanceType<typeof FullCalendarComponent> | null>
  ) => {
    if (!containerRef.value || !calendarRef.value) return;

    resizeObserver = new ResizeObserver(() => {
      if (animationFrameId !== null) return;
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

  const getClickedDate = (e: MouseEvent): string | null => {
    // 클릭된 요소가 날짜인지 확인
    const target = e.target as HTMLElement;
    const dayCell = target.closest('.fc-daygrid-day');

    return dayCell ? dayCell.getAttribute('data-date') : null;
  };

  // 더블클릭 핸들러
  let clickCount = 0;
  let clickTimer: ReturnType<typeof setTimeout> | null = null;
  let lastClickedDate: string | null = null; // 추가

  const handleDoubleClick = (dateStr: string) => {
    // 이전 클릭과 다른 날짜를 누르면 카운트 초기화
    if (lastClickedDate !== dateStr) {
      if (clickTimer) {
        clearTimeout(clickTimer);
        clickTimer = null;
      }
      clickCount = 0;
      lastClickedDate = dateStr;
    }

    clickCount++;

    if (clickCount === 1) {
      clickTimer = setTimeout(() => {
        clickCount = 0;
        lastClickedDate = null;
        clickTimer = null;
      }, 300);
    } else if (clickCount === 2) {
      if (clickTimer) {
        clearTimeout(clickTimer);
        clickTimer = null;
      }
      clickCount = 0;
      lastClickedDate = null;

      uiStore.openRightSidebarWithDate(dateStr);
    }
  };
  return { getClickedDate, startResizeObserver, stopResizeObserver, handleDoubleClick };
}
