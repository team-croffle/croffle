<script setup lang="ts">
  import { AppEventType } from '@croffledev/common';
  import type { CalendarOptions, EventApi } from '@fullcalendar/core';
  import koLocale from '@fullcalendar/core/locales/ko';
  import dayGridPlugin from '@fullcalendar/daygrid';
  import interactionPlugin from '@fullcalendar/interaction';
  import multiMonthPlugin from '@fullcalendar/multimonth';
  import rrulePlugin from '@fullcalendar/rrule';
  import timeGridPlugin from '@fullcalendar/timegrid';
  import FullCalendar from '@fullcalendar/vue3';
  import dayjs from 'dayjs';
  import { storeToRefs } from 'pinia';
  import { onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue';

  import { useCalendarLogic } from '@/composables/use-calendar-logic';
  import { useScheduleDrag } from '@/composables/use-schedule-drag';
  import { useAppSettingsStore } from '@/stores/app-settings-store';
  import { useScheduleStore } from '@/stores/schedule-store';
  import type { SelectedOccurrence } from '@/stores/ui-store';
  import {
    calendarViewToFullCalendar,
    languageToLocale,
    timeFormatToHour12,
    weekStartDayToFirstDay,
  } from '@/utils/calendar-settings';

  const CONTEXT_TARGET_CLASS = 'is-context-menu-target';

  // pinia store 연결
  const scheduleStore = useScheduleStore();
  const appSettingsStore = useAppSettingsStore();
  const { events } = storeToRefs(scheduleStore);
  const { settings } = storeToRefs(appSettingsStore);

  // 캘린더 ref
  const fullCalendarRef = ref<InstanceType<typeof FullCalendar> | null>(null);
  const calendarContainerRef = ref<HTMLElement | null>(null);

  const { startResizeObserver, stopResizeObserver, handleDateDoubleClick, handleEventDoubleClick } =
    useCalendarLogic();

  const { handleEventDrop } = useScheduleDrag();

  /** Occurrence info for recurring / moved events; one-off schedules open as before. */
  function toSelectedOccurrence(
    event: EventApi,
    scheduleId: string,
  ): SelectedOccurrence | undefined {
    const schedule = scheduleStore.getScheduleById(scheduleId);
    const isOverride = event.extendedProps.isOverride === true;
    if (!schedule || !event.start || (!schedule.recurrenceRule?.trim() && !isOverride)) {
      return undefined;
    }
    const duration = new Date(schedule.endDate).getTime() - new Date(schedule.startDate).getTime();
    // FullCalendar's end is exclusive for all-day events; the store keeps the inclusive end.
    const end = event.end
      ? new Date(event.end.getTime() - (event.allDay ? 1000 : 0))
      : new Date(event.start.getTime() + duration);
    return { start: event.start, end, isOverride };
  }

  let unsubscribeSettings: (() => void) | null = null;

  function clearContextTarget() {
    calendarContainerRef.value
      ?.querySelectorAll(`.${CONTEXT_TARGET_CLASS}`)
      .forEach((el) => el.classList.remove(CONTEXT_TARGET_CLASS));
  }

  function handleContextMenu(e: MouseEvent) {
    const target = e.target as HTMLElement;
    // Resolve the day cell before any FullCalendar DOM mutation.
    const dayCell = target.closest('.fc-daygrid-day') as HTMLElement | null;

    clearContextTarget();
    if (dayCell) {
      dayCell.classList.add(CONTEXT_TARGET_CLASS);
    }

    // WHY: Right-click highlight uses a custom CSS class instead of calendarApi.select().
    // Left-click still uses FullCalendar's .fc-highlight, so we must clear that selection
    // or two days look selected. Both select() and unselect() tear down .fc-highlight;
    // if the user right-clicked that overlay, e.target becomes detached and
    // closest('.fc-daygrid-day') fails in the context-menu condition.
    // Defer unselect until after the bubble-phase handler in app.vue captures a stable
    // day/event cell (see resolveContextMenuTarget).
    queueMicrotask(() => {
      fullCalendarRef.value?.getApi()?.unselect();
    });
  }

  function moveWeekNumberToLastDay(weekNumberEl: HTMLElement) {
    // FullCalendar always mounts week numbers in the first day cell of the row.
    // Move the badge to the last day cell so left-aligned dates stay clear.
    const row = weekNumberEl.closest('tr');
    const lastDayFrame = row?.querySelector('td.fc-daygrid-day:last-of-type .fc-daygrid-day-frame');
    if (lastDayFrame && weekNumberEl.parentElement !== lastDayFrame) {
      lastDayFrame.appendChild(weekNumberEl);
    }
  }

  const buildCalendarOptions = (): CalendarOptions => {
    const cal = settings.value?.calendar;
    const lang = settings.value?.general.language;

    return {
      plugins: [dayGridPlugin, timeGridPlugin, multiMonthPlugin, interactionPlugin, rrulePlugin],
      initialView: cal ? calendarViewToFullCalendar(cal.defaultView) : 'dayGridMonth',
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

      events: [],

      // 월·연 보기에서만 드래그로 날짜를 옮긴다 (시간·기간 유지). 주·일 보기는 시각이 바뀌므로 막는다.
      editable: true,
      eventDurationEditable: false,
      views: {
        timeGrid: { editable: false },
      },
      eventDrop: (info) => {
        void handleEventDrop(info);
      },
      selectable: true, // 날짜 선택 가능
      dateClick: (info) => {
        // WHY: Left-click selection is rendered by FullCalendar (.fc-highlight), while
        // right-click uses our custom is-context-menu-target class. If we leave the
        // custom class after a prior right-click, both highlights stay visible and two
        // days look selected. Clear the context-menu target so only the FC selection
        // remains for this left-click.
        clearContextTarget();
        handleDateDoubleClick(info.dateStr);
      }, // 날짜 클릭 핸들러
      eventClick: (info) => {
        const scheduleId =
          (info.event.extendedProps.scheduleId as string | undefined) || info.event.id;
        if (!scheduleId) {
          return;
        }
        handleEventDoubleClick(scheduleId, toSelectedOccurrence(info.event, scheduleId));
      },
      eventDidMount: (info) => {
        const scheduleId =
          (info.event.extendedProps.scheduleId as string | undefined) || info.event.id;
        if (scheduleId) {
          info.el.setAttribute('data-event-id', scheduleId);
          return;
        }
        info.el.removeAttribute('data-event-id');
      },
      weekNumberDidMount: (info) => {
        moveWeekNumberToLastDay(info.el);
      },

      datesSet: (info) => {
        // 월이 변경될 때마다 호출됨
        const start = dayjs(info.start).subtract(1, 'month').startOf('month').toISOString();
        const end = dayjs(info.end).add(1, 'month').endOf('month').toISOString();
        scheduleStore.loadSchedules(start, end);
      },

      windowResizeDelay: 0,
      handleWindowResize: false,
      locale: lang ? languageToLocale(lang) : 'en',
      locales: [koLocale],
      firstDay: cal ? weekStartDayToFirstDay(cal.weekStartDay) : 0,
      weekNumbers: cal?.showWeekNumbers ?? false,
      weekNumberFormat: { week: 'narrow' },
      weekNumberCalculation: 'ISO',
      eventTimeFormat: {
        hour: 'numeric',
        minute: '2-digit',
        meridiem: cal ? timeFormatToHour12(cal.timeFormat) : false,
      },
      slotLabelFormat: {
        hour: 'numeric',
        minute: '2-digit',
        meridiem: cal ? timeFormatToHour12(cal.timeFormat) : false,
      },
    };
  };

  const calendarOptions = reactive<CalendarOptions>(buildCalendarOptions());

  const applyCalendarSettings = () => {
    const api = fullCalendarRef.value?.getApi();
    const cal = settings.value?.calendar;
    const lang = settings.value?.general.language;
    if (!cal) {
      return;
    }

    const patch: Partial<CalendarOptions> = {
      locale: lang ? languageToLocale(lang) : 'en',
      firstDay: weekStartDayToFirstDay(cal.weekStartDay),
      weekNumbers: cal.showWeekNumbers,
      eventTimeFormat: {
        hour: 'numeric',
        minute: '2-digit',
        meridiem: timeFormatToHour12(cal.timeFormat),
      },
      slotLabelFormat: {
        hour: 'numeric',
        minute: '2-digit',
        meridiem: timeFormatToHour12(cal.timeFormat),
      },
    };

    Object.assign(calendarOptions, patch);

    if (api) {
      api.setOption('locale', patch.locale);
      api.setOption('firstDay', patch.firstDay);
      api.setOption('weekNumbers', patch.weekNumbers);
      api.setOption('eventTimeFormat', patch.eventTimeFormat);
      api.setOption('slotLabelFormat', patch.slotLabelFormat);
      api.changeView(calendarViewToFullCalendar(cal.defaultView));
    }
  };

  watch(settings, () => applyCalendarSettings(), { deep: true });

  onMounted(() => {
    startResizeObserver(calendarContainerRef, fullCalendarRef);
    unsubscribeSettings = croffle.event.on(AppEventType.SETTINGS_UPDATE, () => {
      applyCalendarSettings();
    });
    if (settings.value) {
      applyCalendarSettings();
    }
  });

  onBeforeUnmount(() => {
    stopResizeObserver();
    unsubscribeSettings?.();
  });

  // 스토어의 일정 데이터가 변경될 때마다 캘린더에 반영
  watch(
    () => events.value,
    (newEvents) => {
      calendarOptions.events = newEvents;
    },
    { immediate: true, deep: true }, // 초기 로드 시에도 반영, 배열 내부 변경 감지
  );
</script>

<template>
  <div
    ref="calendarContainerRef"
    class="calendar-card flex h-full flex-col"
    @contextmenu="handleContextMenu"
  >
    <FullCalendar ref="fullCalendarRef" :options="calendarOptions" class="h-full w-full flex-1" />
  </div>
</template>

<style scoped>
  /* 캘린더 전체 틀 */
  .calendar-card {
    background-color: var(--card);
    padding: calc(var(--spacing) * 4);
    border: 1px solid var(--croffle-border);
    border-radius: var(--radius-3xl);
    box-shadow: 0 var(--spacing) calc(var(--spacing) * 5) rgba(0, 0, 0, 0.05);
  }

  /* FullCalendar 본체 */
  :deep(.fc) {
    height: 100%;
    width: 100%;
    /*
     * FullCalendar's theme defaults are global (white page background, grey borders).
     * The year (multiMonth) and week/day (timeGrid) views paint with them directly, so map
     * them to the app tokens to follow light / dark.
     */
    --fc-page-bg-color: var(--card);
    --fc-neutral-bg-color: var(--croffle-bg);
    --fc-neutral-text-color: var(--muted-foreground);
    --fc-border-color: var(--croffle-border);
    --fc-more-link-bg-color: var(--croffle-bg);
  }

  /* 연도 보기: 월 제목 */
  :deep(.fc-multimonth-title) {
    color: var(--croffle-text);
    font-size: var(--text-base);
    font-weight: 700;
  }

  /* 헤더(제목+버튼) */
  :deep(.fc .fc-toolbar.fc-header-toolbar) {
    margin-bottom: calc(var(--spacing) * 2.5);
    padding: 0 calc(var(--spacing) * 2.5);
  }

  /* 제목 스타일 */
  :deep(.fc-toolbar-title) {
    font-size: var(--text-2xl);
    font-weight: 700;
    color: var(--croffle-primary);
  }

  /* 버튼 그룹 */
  :deep(.fc-button-group) {
    display: flex;
    gap: calc(var(--spacing) * 2);
  }

  /* 개별 버튼 디자인 */
  :deep(.fc-button) {
    background-color: transparent;
    border: 1px solid var(--croffle-border);
    color: var(--croffle-text);
    font-weight: 500;
    border-radius: var(--radius-lg) !important;
    margin: 0;
    box-shadow: none;
    padding: calc(var(--spacing) * 1.5) calc(var(--spacing) * 3);
    transition: all 0.2s ease;
  }

  :deep(.fc-button:first-child),
  :deep(.fc-button:last-child) {
    padding: calc(var(--spacing) * 1.5) calc(var(--spacing) * 1.5);
  }

  :deep(.fc-button:hover) {
    background-color: var(--croffle-bg);
    color: var(--croffle-primary);
    border-color: var(--croffle-primary);
  }

  :deep(.fc-button:disabled) {
    background-color: var(--croffle-disabled);
    border-color: var(--croffle-border);
    color: var(--croffle-muted);
    opacity: 1;
    cursor: not-allowed;
  }

  /* '오늘' 버튼 등 활성 상태 */
  :deep(.fc-button-primary:not(:disabled).fc-button-active),
  :deep(.fc-button-primary:not(:disabled):active) {
    background-color: var(--croffle-hover);
    color: var(--croffle-primary);
    border-color: var(--croffle-primary);
    box-shadow: inset 0 0 0 1px var(--croffle-primary) !important;
  }

  :deep(.fc-button-primary:focus) {
    box-shadow: none;
  }

  /* 요일 헤더*/
  :deep(.fc-col-header-cell) {
    background-color: var(--croffle-bg);
    padding: calc(var(--spacing) * 2) 0;
    border: none;
    /* 요일 아래쪽 구분선 */
    border-bottom: 1px solid var(--croffle-border);
  }

  /* 요일 텍스트 스타일 */
  :deep(.fc-col-header-cell-cushion) {
    color: var(--croffle-text);
    font-weight: 600;
    text-decoration: none;
  }
  /* 공휴일 컬러 */
  :deep(.fc-day-sun .fc-col-header-cell-cushion),
  :deep(.fc-day-sun .fc-daygrid-day-number) {
    color: var(--croffle-red);
  }
  :deep(.fc-day-sat .fc-col-header-cell-cushion),
  :deep(.fc-day-sat .fc-daygrid-day-number) {
    color: var(--croffle-blue);
  }

  /* 날짜 칸 테두리 (가로선만 표시) */
  :deep(.fc-theme-standard td),
  :deep(.fc-theme-standard th),
  :deep(.fc-scrollgrid) {
    border: none;
    border-bottom: 1px solid var(--croffle-border);
  }

  /* 오늘 날짜 표시 (칸 전체 하이라이트) */
  :deep(.fc .fc-daygrid-day.fc-day-today) {
    background-color: var(--croffle-today-bg);
    border-radius: var(--radius-2xl);
  }

  :deep(.fc-highlight) {
    background-color: var(--croffle-day-select-bg);
    border-radius: var(--radius-2xl);
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
    cursor: default;
    user-select: none;
  }

  :deep(.fc-event.fc-event-draggable) {
    cursor: grab;
  }

  :deep(.fc-event.fc-event-dragging) {
    cursor: grabbing;
  }

  :deep(.fc-event::after) {
    border-radius: 10px;
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

  :deep(.fc-daygrid-day.is-context-menu-target) {
    background-color: var(--croffle-day-select-bg);
    border-radius: var(--radius-2xl);
  }

  /*
   * Week numbers are NOT a separate column in dayGrid — FullCalendar injects
   * .fc-daygrid-week-number as position:absolute inside the first day cell.
   * We relocate it to the last day cell (weekNumberDidMount) and pin it top-right
   * so left-aligned day numbers are never covered.
   */
  :deep(.fc .fc-daygrid-week-number) {
    position: absolute;
    top: 0;
    right: 0;
    left: auto;
    z-index: 5;
    width: auto;
    min-width: 1.25rem;
    padding: 2px 5px;
    font-size: 0.65rem;
    font-weight: 600;
    line-height: 1.2;
    color: var(--muted-foreground);
    background-color: color-mix(in oklab, var(--muted) 55%, transparent);
    text-decoration: none;
    border-radius: 0 0 0 var(--radius-sm);
  }

  :deep(.fc-daygrid-day-frame:has(.fc-daygrid-week-number) .fc-daygrid-day-number) {
    padding-right: 1.75rem;
  }
</style>
