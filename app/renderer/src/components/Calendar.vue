<script setup lang="ts">
  import { reactive, ref } from 'vue';
  import type { CalendarOptions } from '@fullcalendar/core';
  import ContextMenu from './ui/context-menu/ContextMenu.vue';
  import ContextMenuTrigger from './ui/context-menu/ContextMenuTrigger.vue';
  import ContextMenuContent from './ui/context-menu/ContextMenuContent.vue';
  import ContextMenuItem from './ui/context-menu/ContextMenuItem.vue';
  import FullCalendar from '@fullcalendar/vue3';
  import dayGridPlugin from '@fullcalendar/daygrid';
  import interactionPlugin from '@fullcalendar/interaction';
  import { useScheduleStore } from '@/stores/scheduleStore';
  import { storeToRefs } from 'pinia';
  import { useCalendarLogic } from '@/composables/useCalendarLogic';

  // pinia store 연결
  const scheduleStore = useScheduleStore();
  const { events } = storeToRefs(scheduleStore);

  // 날짜 위치 저장 변수(우클릭 시 컨텍스트 메뉴 위치 지정용)
  const selectedDate = ref<string | null>(null);

  const fullCalendarRef = ref<InstanceType<typeof FullCalendar> | null>(null);
  const calendarContainerRef = ref<HTMLElement | null>(null);

  const { setupCalendarResize, getClickedDate, handleDoubleClick } = useCalendarLogic();

  // 캘린더 리사이징
  setupCalendarResize(calendarContainerRef, fullCalendarRef);

  // 우클릭 핸들러
  const handleContextMenu = (e: MouseEvent) => {
    // 클릭된 요소가 날짜인지 확인
    const date = getClickedDate(e);

    if (date) {
      selectedDate.value = date;
    } else {
      // 날짜 영역 밖은 컨텍스트 메뉴 비활성화
      selectedDate.value = null;
      e.preventDefault();
      e.stopPropagation();
    }
  };

  // fullCalendar 옵션 설정
  const calendarOptions = reactive<CalendarOptions>({
    plugins: [dayGridPlugin, interactionPlugin],
    initialView: 'dayGridMonth',
    initialDate: new Date().toISOString().slice(0, 10),
    headerToolbar: {
      start: 'title',
      center: '',
      end: 'prev,today,next',
    },

    // 제목 형식
    titleFormat: { year: 'numeric', month: 'long' },

    // 날짜 숫자 형식
    dayCellContent: (info) => {
      return info.date.getDate().toString();
    },

    height: '100%',
    expandRows: true,
    fixedWeekCount: true,

    dayMaxEvents: true, // 하루에 표시할 수 있는 최대 이벤트 수

    // 이벤트 시간 숨기기
    displayEventTime: false,

    events: events.value, // pinia store의 events 사용
    editable: true, // 이벤트 드래그 가능
    selectable: true, // 날짜 선택 가능
    dateClick: (info) => handleDoubleClick(info.dateStr), // 날짜 클릭 핸들러

    windowResizeDelay: 0,
    handleWindowResize: false, // 수동으로 크기 조정 처리
    locale: 'ko', // 한국어 설정
  });
</script>

<template>
  <ContextMenu class="h-full">
    <ContextMenuTrigger class="block h-full w-full">
      <div
        ref="calendarContainerRef"
        class="calendar-card flex h-full flex-col"
        @contextmenu="handleContextMenu"
      >
        <FullCalendar
          ref="fullCalendarRef"
          :options="calendarOptions"
          class="h-full w-full flex-1"
        />
      </div>
    </ContextMenuTrigger>
    <ContextMenuContent>
      <ContextMenuItem>일정 추가 ({{ selectedDate }})</ContextMenuItem>
      <ContextMenuItem>일정 수정</ContextMenuItem>
      <ContextMenuItem>일정 삭제</ContextMenuItem>
    </ContextMenuContent>
  </ContextMenu>
</template>

<style scoped>
  /* 캘린더 전체 틀 */
  .calendar-card {
    background-color: var(--card);
    padding: 15px;
    border: 1px solid var(--croffle-border);
    border-radius: 20px;
    box-shadow: 0 4px 20px rgba(0, 0, 0, 0.05);
  }

  /* FullCalendar 본체 */
  :deep(.fc) {
    height: 100%;
    width: 100%;
  }

  /* 헤더(제목+버튼) */
  :deep(.fc-header-toolbar) {
    margin-bottom: 10px !important;
    padding: 0 10px;
  }

  /* 제목 스타일 */
  :deep(.fc-toolbar-title) {
    font-size: 1.5rem;
    font-weight: 700;
    color: var(--croffle-primary);
  }

  /* 버튼 그룹 */
  :deep(.fc-button-group) {
    display: flex !important;
    gap: 8px !important;
  }

  /* 개별 버튼 디자인 */
  :deep(.fc-button) {
    background-color: transparent;
    border: 1px solid var(--croffle-border) !important;
    color: var(--croffle-text);
    font-weight: 500;
    border-radius: 8px !important;
    margin: 0 !important;
    box-shadow: none !important;
    padding: 6px 12px;
    transition: all 0.2s ease;
  }

  :deep(.fc-button:hover) {
    background-color: var(--croffle-bg);
    color: var(--croffle-primary);
    border-color: var(--croffle-primary) !important;
  }

  :deep(.fc-button:disabled) {
    background-color: var(--croffle-disabled) !important;
    border-color: var(--croffle-border) !important;
    color: var(--croffle-muted) !important;
    opacity: 1 !important;
    cursor: not-allowed;
  }

  /* '오늘' 버튼 등 활성 상태 */
  :deep(.fc-button-primary:not(:disabled).fc-button-active),
  :deep(.fc-button-primary:not(:disabled):active) {
    background-color: var(--croffle-hover) !important;
    color: var(--croffle-primary) !important;
    border-color: var(--croffle-primary) !important;
    box-shadow: inset 0 0 0 1px var(--croffle-primary) !important;
  }

  /* 요일 헤더*/
  :deep(.fc-col-header-cell) {
    background-color: var(--croffle-bg);
    padding: 8px 0;
    border: none !important;
    /* 요일 아래쪽 구분선 */
    border-bottom: 1px solid var(--croffle-border) !important;
  }

  /* 요일 텍스트 스타일 */
  :deep(.fc-col-header-cell-cushion) {
    color: var(--croffle-text);
    font-weight: 600;
    text-decoration: none;
  }
  /* 공휴일 컬러 */
  :deep(.fc-day-sun .fc-col-header-cell-cushion) {
    color: var(--croffle-red);
  }
  :deep(.fc-day-sat .fc-col-header-cell-cushion) {
    color: var(--croffle-blue);
  }

  /* 날짜 칸 테두리 (가로선만 표시) */
  :deep(.fc-theme-standard td),
  :deep(.fc-theme-standard th),
  :deep(.fc-scrollgrid) {
    border-left: none !important;
    border-right: none !important;
    border-top: none !important;
    border-bottom: 1px solid var(--croffle-border) !important;
  }

  /* 오늘 날짜 표시 (칸 전체 하이라이트) */
  :deep(.fc-day-today) {
    background-color: var(--croffle-today-bg) !important;
  }

  /* 오늘 날짜 숫자 스타일 */
  :deep(.fc-day-today .fc-daygrid-day-number) {
    background-color: transparent !important;
    color: var(--croffle-primary);
    font-weight: bold;
    border-radius: 0;
  }

  /* 일반 날짜 숫자 스타일 */
  :deep(.fc-daygrid-day-number) {
    font-size: 0.8rem;
    font-weight: normal;
    color: var(--muted-foreground);
    text-decoration: none;
    padding: 8px;
    width: 100%;
    text-align: left;
  }

  /* 이벤트 스타일 */
  :deep(.fc-event) {
    border: none;
    border-radius: 10px;
    box-shadow: none;
    padding: 1px 4px;
    margin-top: 1px !important;
    margin-bottom: 1px !important;
    color: var(--foreground);
    display: flex;
    align-items: center;
  }

  :deep(.fc-event-main) {
    overflow: hidden;
  }

  /* 이벤트 제목 (긴 제목 처리) */
  :deep(.fc-event-title) {
    display: block;
    white-space: nowrap;
    overflow: hidden; /* 넘치면 숨김 */
    text-overflow: ellipsis; /* ... 으로 표시 */
    font-weight: 500;
    line-height: 1.2;
  }

  /* 더보기 링크 (+2 more) 디자인 */
  :deep(.fc-more-link) {
    color: var(--croffle-muted);
    font-size: 0.75rem;
    font-weight: 600;
    text-decoration: none;
    display: block;
    margin-top: 2px;
    padding-left: 4px;
  }

  :deep(.fc-more-link:hover) {
    color: var(--croffle-primary);
  }
</style>
