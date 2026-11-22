<script setup lang="ts">
  import type { Schedule, UpcomingOccurrence } from '@croffledev/common';
  import dayjs from 'dayjs';
  import { storeToRefs } from 'pinia';
  import { computed } from 'vue';
  import { useI18n } from 'vue-i18n';

  import logoImg from '@/assets/logo-only.png';
  import { Badge } from '@/components/ui/badge';
  import { Button } from '@/components/ui/button';
  import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
  import { Checkbox } from '@/components/ui/checkbox';
  import { Icon } from '@/components/ui/icon';
  import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
  import { Sidebar, SidebarContent, SidebarFooter, SidebarHeader } from '@/components/ui/sidebar';
  import UpdateCheck from '@/components/update-check.vue';
  import { SHORTCUT_RIGHT_SIDEBAR } from '@/composables/use-global-shortcuts';
  import { cn } from '@/lib/utils';
  import { useScheduleStore } from '@/stores/schedule-store';
  import { useTodoStore } from '@/stores/todo-store';
  import { useUiStore } from '@/stores/ui-store';
  import { useUpcomingStore } from '@/stores/upcoming-store';
  import { useUpdateStore } from '@/stores/update-store';
  import { occurrencesOnDate, type ScheduleOccurrence } from '@/utils/schedule-occurrences';

  import pkg from '../../../../package.json';

  const uiStore = useUiStore();
  const scheduleStore = useScheduleStore();
  const upcomingStore = useUpcomingStore();
  upcomingStore.start();
  const todoStore = useTodoStore();
  todoStore.start();
  const { t, locale } = useI18n();

  /** "11월 20일 (금) 오후 3:00", all-day: date only (app language). */
  const upcomingLabel = (occurrence: UpcomingOccurrence) =>
    new Intl.DateTimeFormat(locale.value === 'ko' ? 'ko-KR' : 'en-US', {
      month: 'short',
      day: 'numeric',
      weekday: 'short',
      ...(occurrence.schedule.isAllDay ? {} : { hour: 'numeric', minute: '2-digit' }),
    }).format(occurrence.start);

  const openUpcoming = (occurrence: UpcomingOccurrence) => {
    const detail =
      occurrence.occurrenceStart === null
        ? undefined
        : { start: occurrence.start, end: occurrence.end, isOverride: occurrence.isOverride };
    // The viewer reads the calendar's store, which holds only the month on screen.
    if (!scheduleStore.getScheduleById(occurrence.schedule.id)) {
      scheduleStore.upsertSchedule(occurrence.schedule);
    }
    uiStore.openScheduleModal('view', occurrence.schedule.id, detail);
  };

  /** "11월 25일 (수) 오전 9:00", all-day: date only (app language). */
  const dueLabel = (schedule: Schedule) =>
    new Intl.DateTimeFormat(locale.value === 'ko' ? 'ko-KR' : 'en-US', {
      month: 'short',
      day: 'numeric',
      weekday: 'short',
      ...(schedule.isAllDay ? {} : { hour: 'numeric', minute: '2-digit' }),
    }).format(new Date(schedule.endDate));

  /** Past its deadline: an all-day To-Do is overdue from the next day on. */
  const isOverdue = (schedule: Schedule) => {
    const due = dayjs(schedule.endDate);
    return schedule.isAllDay ? due.endOf('day').isBefore(dayjs()) : due.isBefore(dayjs());
  };

  const openTodo = (schedule: Schedule) => {
    // The viewer reads the calendar's store, which holds only the month on screen.
    if (!scheduleStore.getScheduleById(schedule.id)) {
      scheduleStore.upsertSchedule(schedule);
    }
    uiStore.openScheduleModal('view', schedule.id);
  };

  const { rightSidebarOpen, selectedDate } = storeToRefs(uiStore);
  const { checkStatus } = storeToRefs(useUpdateStore());

  const isSelectedToday = computed(
    () => !selectedDate.value || dayjs(selectedDate.value).isSame(dayjs(), 'day'),
  );

  /** 카드 제목: 오늘이면 "Today", 아니면 선택한 날짜 (앱 언어 로케일) */
  const selectedDateLabel = computed(() => {
    if (isSelectedToday.value || !selectedDate.value) {
      return t('rightSidebar.today');
    }
    return new Intl.DateTimeFormat(locale.value === 'ko' ? 'ko-KR' : 'en-US', {
      month: 'long',
      day: 'numeric',
      weekday: 'short',
    }).format(dayjs(selectedDate.value).toDate());
  });

  const emptyLabel = computed(() =>
    isSelectedToday.value ? t('rightSidebar.emptyToday') : t('rightSidebar.emptySelected'),
  );

  // 선택된 날짜에 걸치는 일정의 발생들. 반복 일정은 그날의 발생(옮긴 발생 포함)으로 전개한다.
  // 시작·끝은 스토어 기준(종료 포함, inclusive)으로 비교한다.
  const selectedSchedules = computed<ScheduleOccurrence[]>(() => {
    if (!selectedDate.value) {
      return [];
    }
    const target = dayjs(selectedDate.value).toDate();
    return scheduleStore.schedules
      .flatMap((schedule) => occurrencesOnDate(schedule, target))
      .toSorted((a, b) => a.start.getTime() - b.start.getTime());
  });

  // 화면에 보여줄 상태값들을 computed로 자동 계산
  // const todayCount = computed(() => selectedSchedules.value.length);
  // const hasTodayEvent = computed(() => todayCount.value > 0);

  const handleEditTodo = (occurrence: ScheduleOccurrence) => {
    const detail =
      occurrence.occurrenceStart === null
        ? undefined
        : { start: occurrence.start, end: occurrence.end, isOverride: occurrence.isOverride };
    uiStore.openScheduleModal('view', occurrence.schedule.id, detail);
  };

  /** Priority: a dot and the word on the status tokens (low success, medium warning, high destructive). */
  function getPriorityClass(priority: string): { badge: string; dot: string } {
    switch (priority) {
      case 'low':
        return { badge: 'border-success bg-success/14', dot: 'bg-success' };
      case 'medium':
        return { badge: 'border-warning bg-warning/14', dot: 'bg-warning' };
      case 'high':
        return { badge: 'border-destructive bg-destructive/14', dot: 'bg-destructive' };
      default:
        return { badge: 'border-warning bg-warning/14', dot: 'bg-warning' };
    }
  }

  function getPriorityText(priority: string) {
    switch (priority) {
      case 'low':
        return t('priority.low');
      case 'medium':
        return t('priority.medium');
      case 'high':
        return t('priority.high');
    }
  }

  const packageVersion = `v${pkg.version}`;
</script>

<template>
  <Sidebar
    side="right"
    collapsible="icon"
    :open="rightSidebarOpen"
    class="bg-croffle-sidebar relative flex h-full flex-col border-l py-2 [--sidebar-width:20rem] group-data-[collapsible=icon]:w-15"
  >
    <SidebarHeader class="bg-croffle-sidebar shrink-0 px-4 pb-0">
      <div
        class="mb-2 flex h-10 items-center group-data-[collapsible=icon]:justify-center"
        :class="rightSidebarOpen ? 'justify-between' : 'justify-center'"
      >
        <div
          class="space-y-1 overflow-hidden text-left transition-all duration-300 group-data-[collapsible=icon]:hidden"
        >
          <h2 class="text-lg font-bold whitespace-nowrap">
            {{ $t('rightSidebar.title') }}
          </h2>
          <p class="text-croffle-text text-xs whitespace-nowrap">
            {{ $t('rightSidebar.subtitle') }}
          </p>
        </div>

        <Button
          variant="ghost"
          size="icon"
          class="text-muted-foreground"
          :aria-label="$t('rightSidebar.toggle')"
          :title="`${$t('rightSidebar.toggle')} (${SHORTCUT_RIGHT_SIDEBAR})`"
          @click="uiStore.toggleRightSidebar"
        >
          <Icon icon="lucide:panel-right" />
        </Button>
      </div>

      <div class="bg-croffle-border mb-2 h-px w-full group-data-[collapsible=icon]:hidden"></div>
    </SidebarHeader>

    <SidebarContent class="bg-croffle-sidebar no-scrollbar min-h-0 flex-1 overflow-y-auto p-4 pt-0">
      <div class="mt-2 flex justify-center">
        <Button
          class="w-full transition-all duration-300 group-data-[collapsible=icon]:size-9 group-data-[collapsible=icon]:rounded-full group-data-[collapsible=icon]:p-0"
          @click="uiStore.openScheduleModal('add')"
        >
          <Icon icon="lucide:plus" />
          <span class="group-data-[collapsible=icon]:hidden">{{ $t('rightSidebar.add') }}</span>
        </Button>
      </div>
      <div
        class="animate-in fade-in flex flex-col gap-4 duration-300 group-data-[collapsible=icon]:hidden"
      >
        <Card class="bg-croffle-sidebar-content gap-0 overflow-hidden">
          <CardHeader class="space-y-0 px-4 pt-0 pb-2">
            <CardTitle class="flex items-center gap-2 text-sm font-bold">
              <Icon icon="lucide:calendar" class="size-4" />
              <span>{{ selectedDateLabel }}</span>
              <Badge variant="secondary" class="ml-auto">
                {{ selectedSchedules.length }}
              </Badge>
            </CardTitle>
          </CardHeader>
          <CardContent
            class="text-croffle-text flex min-h-25 justify-center text-sm"
            :class="selectedSchedules.length === 0 ? 'items-center' : 'items-start'"
          >
            <span v-if="selectedSchedules.length === 0">{{ emptyLabel }}</span>
            <div v-else class="mt-2 flex w-full flex-col gap-1">
              <div
                v-for="occurrence in selectedSchedules"
                :key="`${occurrence.schedule.id}-${occurrence.start.getTime()}`"
                class="flex cursor-pointer items-center gap-2 rounded-md p-2 transition-colors hover:bg-accent"
                @click="handleEditTodo(occurrence)"
              >
                <Checkbox
                  v-if="occurrence.schedule.kind === 'todo'"
                  :model-value="Boolean(occurrence.schedule.completedAt)"
                  :aria-label="$t('schedule.todo.toggleAria', { title: occurrence.schedule.title })"
                  @click.stop
                  @update:model-value="todoStore.setCompleted(occurrence.schedule, $event === true)"
                />
                <Icon v-else icon="lucide:calendar" class="text-muted-foreground size-4 shrink-0" />

                <span
                  class="flex-1 truncate text-sm"
                  :class="
                    occurrence.schedule.completedAt
                      ? 'text-muted-foreground line-through'
                      : 'text-foreground'
                  "
                >
                  {{ occurrence.schedule.title }}
                </span>

                <Badge
                  variant="outline"
                  :class="
                    cn(
                      'text-foreground text-2xs h-4 shrink-0 gap-1 px-1.5 py-0',
                      getPriorityClass(occurrence.schedule.priority).badge,
                    )
                  "
                >
                  <span
                    class="size-1.5 rounded-full"
                    :class="getPriorityClass(occurrence.schedule.priority).dot"
                    aria-hidden="true"
                  />
                  {{ getPriorityText(occurrence.schedule.priority) }}
                </Badge>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card class="bg-croffle-sidebar-content overflow-hidden">
          <CardHeader class="space-y-0 px-4 pt-0 pb-2">
            <CardTitle class="flex items-center gap-2 text-sm font-bold">
              <Icon icon="lucide:square-check" class="size-4" />
              <span>{{ $t('rightSidebar.todos') }}</span>
              <Badge v-if="todoStore.openTodos.length > 0" variant="secondary" class="ml-auto">
                {{ todoStore.openTodos.length }}
              </Badge>
            </CardTitle>
          </CardHeader>
          <CardContent
            v-if="todoStore.openTodos.length === 0"
            class="flex min-h-25 items-center justify-center text-sm"
            :class="todoStore.status === 'error' ? 'text-destructive' : 'text-croffle-text'"
          >
            <template v-if="todoStore.status === 'loading'">
              {{ $t('rightSidebar.todosLoading') }}
            </template>
            <template v-else-if="todoStore.status === 'error'">
              {{ $t('rightSidebar.todosError') }}
            </template>
            <template v-else>{{ $t('rightSidebar.emptyTodos') }}</template>
          </CardContent>
          <CardContent v-else class="px-2 pb-2">
            <ul class="flex flex-col gap-0.5">
              <li v-for="todo in todoStore.openTodos" :key="todo.id">
                <div
                  role="button"
                  tabindex="0"
                  class="hover:bg-croffle-hover focus-visible:ring-ring/50 flex cursor-pointer items-center gap-2 rounded-md px-2 py-1.5 outline-none focus-visible:ring-[3px]"
                  :title="$t('rightSidebar.todoOpenHint')"
                  @dblclick="openTodo(todo)"
                  @keydown.enter="openTodo(todo)"
                >
                  <Checkbox
                    :model-value="false"
                    :aria-label="$t('schedule.todo.toggleAria', { title: todo.title })"
                    @click.stop
                    @dblclick.stop
                    @update:model-value="todoStore.setCompleted(todo, $event === true)"
                  />
                  <span class="min-w-0 flex-1">
                    <span class="text-foreground block truncate text-sm">{{ todo.title }}</span>
                    <span
                      class="block text-xs"
                      :class="isOverdue(todo) ? 'text-destructive' : 'text-muted-foreground'"
                    >
                      {{ dueLabel(todo) }}
                      <template v-if="isOverdue(todo)">
                        · {{ $t('rightSidebar.overdue') }}</template
                      >
                    </span>
                  </span>
                </div>
              </li>
            </ul>
          </CardContent>
        </Card>

        <Card class="bg-croffle-sidebar-content overflow-hidden">
          <CardHeader class="space-y-0 px-4 pt-0 pb-2">
            <CardTitle class="flex items-center gap-2 text-sm font-bold">
              <Icon icon="lucide:clock" class="size-4" />
              <span>{{ $t('rightSidebar.upcoming') }}</span>
            </CardTitle>
          </CardHeader>
          <CardContent
            v-if="upcomingStore.upcoming.length === 0"
            class="text-croffle-text flex min-h-25 items-center justify-center text-sm"
          >
            {{ $t('rightSidebar.emptyUpcoming') }}
          </CardContent>
          <CardContent v-else class="px-2 pb-2">
            <ul class="flex flex-col gap-0.5">
              <li
                v-for="occurrence in upcomingStore.upcoming"
                :key="`${occurrence.schedule.id}:${occurrence.start.getTime()}`"
              >
                <div
                  role="button"
                  tabindex="0"
                  class="hover:bg-croffle-hover focus-visible:ring-ring/50 flex cursor-pointer items-center gap-2 rounded-md px-2 py-1.5 outline-none focus-visible:ring-[3px]"
                  :title="$t('rightSidebar.upcomingOpenHint')"
                  @dblclick="openUpcoming(occurrence)"
                  @keydown.enter="openUpcoming(occurrence)"
                >
                  <span
                    class="size-2 shrink-0 rounded-full"
                    :style="{ backgroundColor: occurrence.schedule.colorLabel }"
                    aria-hidden="true"
                  />
                  <span class="min-w-0 flex-1">
                    <span class="text-foreground block truncate text-sm">
                      {{ occurrence.schedule.title }}
                    </span>
                    <span class="text-muted-foreground block text-xs">
                      {{ upcomingLabel(occurrence) }}
                    </span>
                  </span>
                </div>
              </li>
            </ul>
          </CardContent>
        </Card>
      </div>
    </SidebarContent>

    <SidebarFooter
      class="bg-croffle-sidebar shrink-0 px-4 pt-2 pb-1 group-data-[collapsible=icon]:items-center group-data-[collapsible=icon]:px-0"
    >
      <Popover>
        <PopoverTrigger as-child>
          <Button
            v-if="!rightSidebarOpen"
            variant="ghost"
            size="icon"
            class="text-croffle-primary-strong"
            :aria-label="$t('rightSidebar.versionTile.title')"
          >
            <Icon icon="lucide:info" class="size-5" />
          </Button>
          <button
            v-else
            type="button"
            class="bg-croffle-sidebar-content-disabled hover:bg-croffle-hover focus-visible:ring-ring/50 flex w-full cursor-pointer items-center gap-2.5 rounded-xl border px-3.5 py-3 text-left shadow-sm outline-none focus-visible:ring-[3px]"
            :aria-label="$t('rightSidebar.versionTile.title')"
          >
            <Icon icon="lucide:info" class="text-croffle-primary-strong size-5 shrink-0" />
            <span class="min-w-0 flex-1 truncate text-xs font-bold tracking-wider">
              CROFFLE {{ packageVersion }}
            </span>
            <span
              v-if="checkStatus.kind === 'available'"
              class="bg-info size-2 shrink-0 rounded-full"
              aria-hidden="true"
            />
          </button>
        </PopoverTrigger>
        <PopoverContent side="top" align="end" :side-offset="8" class="w-[340px]">
          <div class="mb-3 flex items-center gap-3">
            <img :src="logoImg" alt="" class="size-10 shrink-0" />
            <div class="min-w-0">
              <p class="text-sm font-semibold">Croffle</p>
              <p class="text-muted-foreground text-xs">
                {{ $t('rightSidebar.versionTile.version', { version: packageVersion }) }}
              </p>
            </div>
          </div>
          <UpdateCheck />
        </PopoverContent>
      </Popover>
    </SidebarFooter>
  </Sidebar>
</template>

<style scoped>
  .no-scrollbar {
    -ms-overflow-style: none;
    scrollbar-width: none;
  }

  .no-scrollbar::-webkit-scrollbar {
    display: none;
  }
</style>
