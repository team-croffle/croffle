import { useUiStore } from '@/stores/uiStore';
import type FullCalendarComponent from '@fullcalendar/vue3';
import { onBeforeUnmount, onMounted, type Ref } from 'vue';

export function useCalendarLogic() {
  const uiStore = useUiStore();

  // 캘린더 리사이징 최적화 로직
  const setupCalendarResize = (
    containerRef: Ref<HTMLElement | null>,
    calendarRef: Ref<InstanceType<typeof FullCalendarComponent> | null>
  ) => {
    let resizeObserver: ResizeObserver | null = null;

    onMounted(() => {
      if (containerRef.value && calendarRef.value) {
        let animationFrameId: number | null = null;
        resizeObserver = new ResizeObserver(() => {
          if (animationFrameId !== null) return;
          animationFrameId = requestAnimationFrame(() => {
            calendarRef.value?.getApi().updateSize();
            animationFrameId = null;
          });
        });
        resizeObserver.observe(containerRef.value);
      }
    });

    onBeforeUnmount(() => {
      resizeObserver?.disconnect();
    });
  };

  const getClickedDate = (e: MouseEvent): string | null => {
    // 클릭된 요소가 날짜인지 확인
    const target = e.target as HTMLElement;
    const dayCell = target.closest('.fc-daygrid-day');

    if (dayCell) {
      return dayCell.getAttribute('data-date');
    }

    return null;
  };

  // 더블클릭 핸들러
  let clickCount = 0;
  let clickTimer: ReturnType<typeof setTimeout> | null = null;

  const handleDoubleClick = (dateStr: string) => {
    clickCount++;

    if (clickCount === 1) {
      // 단일 클릭: 아무 동작도 하지 않고 0.3초 동안 다음 클릭을 기다리기만 함
      clickTimer = setTimeout(() => {
        clickCount = 0; // 0.3초 안에 안 누르면 그냥 무시
      }, 300);
    } else if (clickCount === 2) {
      // 0.3초 안에 두 번 눌렀을 때만 작동
      if (clickTimer) clearTimeout(clickTimer);
      clickCount = 0;

      uiStore.openRightSidebarWithDate(dateStr);
    }
  };
  return { getClickedDate, setupCalendarResize, handleDoubleClick };
}
