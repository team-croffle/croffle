<script setup lang="ts">
  import dayjs from 'dayjs';
  import { storeToRefs } from 'pinia';
  import { computed } from 'vue';
  import { useI18n } from 'vue-i18n';

  import { Badge } from '@/components/ui/badge';
  import { Button } from '@/components/ui/button';
  import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
  import { Icon } from '@/components/ui/icon';
  import { Sidebar, SidebarContent, SidebarFooter, SidebarHeader } from '@/components/ui/sidebar';
  import { SHORTCUT_RIGHT_SIDEBAR } from '@/composables/use-global-shortcuts';
  import { cn } from '@/lib/utils';
  import { useScheduleStore } from '@/stores/schedule-store';
  import { useUiStore } from '@/stores/ui-store';
  import { occurrencesOnDate, type ScheduleOccurrence } from '@/utils/schedule-occurrences';

  import pkg from '../../../../package.json';

  const uiStore = useUiStore();
  const scheduleStore = useScheduleStore();
  const { t, locale } = useI18n();

  const { rightSidebarOpen, selectedDate } = storeToRefs(uiStore);

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

  function getPriorityClass(priority: string) {
    switch (priority) {
      case 'low':
        return 'bg-emerald-500/30 border-emerald-500';
      case 'medium':
        return 'bg-amber-500/30 border-amber-500';
      case 'high':
        return 'bg-rose-500/30 border-rose-500';
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
    class="border-croffle-border bg-croffle-sidebar relative flex h-screen flex-col border-l py-2 [--sidebar-width:20rem] group-data-[collapsible=icon]:w-15"
  >
    <SidebarHeader class="bg-croffle-sidebar shrink-0 px-4 pb-0">
      <div
        class="mb-2 flex h-10 items-center group-data-[collapsible=icon]:justify-center"
        :class="rightSidebarOpen ? 'justify-between' : 'justify-center'"
      >
        <div
          class="space-y-1 overflow-hidden text-left transition-all duration-300 group-data-[collapsible=icon]:hidden"
        >
          <h2 class="text-croffle-text-dark text-lg font-bold whitespace-nowrap">
            {{ $t('rightSidebar.title') }}
          </h2>
          <p class="text-croffle-text text-xs whitespace-nowrap">
            {{ $t('rightSidebar.subtitle') }}
          </p>
        </div>

        <Button
          variant="ghost"
          size="icon"
          class="text-muted-foreground h-7 w-7"
          :aria-label="$t('rightSidebar.toggle')"
          :title="`${$t('rightSidebar.toggle')} (${SHORTCUT_RIGHT_SIDEBAR})`"
          @click="uiStore.toggleRightSidebar"
        >
          <Icon icon="lucide:panel-right" class="h-4 w-4" />
        </Button>
      </div>

      <div class="bg-croffle-border mb-2 h-px w-full group-data-[collapsible=icon]:hidden"></div>
    </SidebarHeader>

    <SidebarContent class="bg-croffle-sidebar no-scrollbar min-h-0 flex-1 overflow-y-auto p-4 pt-0">
      <div class="mt-2 flex justify-center">
        <Button
          class="bg-croffle-primary hover:bg-croffle-hover h-11 w-full rounded-lg border-none font-medium text-white shadow-sm transition-all duration-300 group-data-[collapsible=icon]:h-10 group-data-[collapsible=icon]:w-10 group-data-[collapsible=icon]:rounded-full group-data-[collapsible=icon]:p-0"
          @click="uiStore.openScheduleModal('add')"
        >
          <Icon
            icon="lucide:plus"
            class="h-5 w-5 transition-all"
            :class="rightSidebarOpen ? 'mr-1' : ''"
          />
          <span class="group-data-[collapsible=icon]:hidden">{{ $t('rightSidebar.add') }}</span>
        </Button>
      </div>
      <div
        class="animate-in fade-in flex flex-col gap-4 duration-300 group-data-[collapsible=icon]:hidden"
      >
        <Card
          class="border-croffle-border bg-croffle-sidebar-content gap-0 overflow-hidden rounded-xl border shadow-sm"
        >
          <CardHeader class="space-y-0 px-4 pt-0 pb-2">
            <CardTitle class="text-croffle-text-dark flex items-center gap-2 text-sm font-bold">
              <Icon icon="lucide:calendar" class="h-4 w-4" />
              <span>{{ selectedDateLabel }}</span>
              <Badge
                class="bg-croffle-sidebar text-croffle-text-dark ml-auto h-5 rounded-md px-1.5"
              >
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
                class="flex cursor-pointer items-center gap-2 rounded-md p-2 transition-colors hover:bg-neutral-200 dark:hover:bg-neutral-800"
                @click="handleEditTodo(occurrence)"
              >
                <Icon icon="lucide:square-check" class="text-muted-foreground h-4 w-4 shrink-0" />

                <span class="text-foreground flex-1 truncate text-sm">
                  {{ occurrence.schedule.title }}
                </span>

                <Badge
                  variant="outline"
                  :class="
                    cn(
                      'text-foreground border-sidebar-ring text-2xs h-4 shrink-0 px-1.5 py-0',
                      getPriorityClass(occurrence.schedule.priority),
                    )
                  "
                >
                  {{ getPriorityText(occurrence.schedule.priority) }}
                </Badge>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card
          class="border-croffle-border bg-croffle-sidebar-content overflow-hidden rounded-xl border shadow-sm"
        >
          <CardHeader class="space-y-0 px-4 pt-0 pb-2">
            <CardTitle class="text-croffle-text-dark flex items-center gap-2 text-sm font-bold">
              <Icon icon="lucide:clock" class="h-4 w-4" />
              <span>{{ $t('rightSidebar.upcoming') }}</span>
            </CardTitle>
          </CardHeader>
          <CardContent class="text-croffle-text flex min-h-25 items-center justify-center text-sm">
            {{ $t('rightSidebar.emptyUpcoming') }}
          </CardContent>
        </Card>
      </div>
    </SidebarContent>

    <SidebarFooter class="bg-croffle-sidebar shrink-0 flex-col items-center justify-center pb-4">
      <div
        class="border-croffle-border bg-croffle-sidebar-content-disabled mb-4 flex w-full flex-col items-center justify-center rounded-xl border p-4 shadow-sm group-data-[collapsible=icon]:border-none group-data-[collapsible=icon]:bg-transparent group-data-[collapsible=icon]:p-0 group-data-[collapsible=icon]:shadow-none"
      >
        <Icon
          icon="lucide:home"
          class="text-croffle-primary mb-1 h-6 w-6 group-data-[collapsible=icon]:h-5 group-data-[collapsible=icon]:w-5"
        />
        <div class="text-center group-data-[collapsible=icon]:hidden">
          <h4 class="text-croffle-text-dark text-xs font-bold tracking-wider">CROFFLE</h4>
          <span class="text-croffle-text text-xs">{{ packageVersion }}</span>
        </div>
      </div>
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
