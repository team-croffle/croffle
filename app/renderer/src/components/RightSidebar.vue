<script setup lang="ts">
  import { Calendar, Clock, Plus, Home, PanelRight } from 'lucide-vue-next';
  import { Button } from '@/components/ui/button';
  import { Badge } from '@/components/ui/badge';
  import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
  import { Sidebar, SidebarContent, SidebarFooter, SidebarHeader } from '@/components/ui/sidebar';
  import { useUiStore } from '@/stores/uiStore';
  import { storeToRefs } from 'pinia';
  import { computed } from 'vue';
  import { useScheduleStore } from '@/stores/scheduleStore';

  const emit = defineEmits(['click-add-schedule']);

  const uiStore = useUiStore();
  const scheduleStore = useScheduleStore();

  const { rightSidebarOpen, selectedDate } = storeToRefs(uiStore);

  // 선택된 날짜에 해당하는 일정만 스토어에서 가져옴
  const selectedSchedules = computed(() => {
    if (!selectedDate.value) return [];

    return scheduleStore.schedules.filter((schedule) =>
      // 예: '2026-02-17'로 시작하는 일정 찾기
      schedule.startDate.startsWith(selectedDate.value as string)
    );
  });

  // 화면에 보여줄 상태값들을 computed로 자동 계산
  const todayCount = computed(() => selectedSchedules.value.length);
  const hasTodayEvent = computed(() => todayCount.value > 0);
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
          <h2 class="text-croffle-text-dark text-lg font-bold whitespace-nowrap">일정 관리</h2>
          <p class="text-croffle-text text-xs whitespace-nowrap">오늘의 일정과 계획</p>
        </div>

        <Button
          variant="ghost"
          size="icon"
          class="text-muted-foreground h-7 w-7"
          @click="uiStore.toggleRightSidebar"
        >
          <PanelRight class="h-4 w-4" />
        </Button>
      </div>

      <div class="bg-croffle-border mb-2 h-px w-full group-data-[collapsible=icon]:hidden"></div>
    </SidebarHeader>

    <SidebarContent class="bg-croffle-sidebar no-scrollbar min-h-0 flex-1 overflow-y-auto p-4 pt-0">
      <div class="mt-2 flex justify-center">
        <Button
          class="bg-croffle-primary hover:bg-croffle-hover h-11 w-full rounded-lg border-none font-medium text-white shadow-sm transition-all duration-300 group-data-[collapsible=icon]:h-10 group-data-[collapsible=icon]:w-10 group-data-[collapsible=icon]:rounded-full group-data-[collapsible=icon]:p-0"
          @click="emit('click-add-schedule')"
        >
          <Plus class="h-5 w-5 transition-all" :class="rightSidebarOpen ? 'mr-1' : ''" />
          <span class="group-data-[collapsible=icon]:hidden">새 일정 추가</span>
        </Button>
      </div>
      <div
        class="animate-in fade-in flex flex-col gap-4 duration-300 group-data-[collapsible=icon]:hidden"
      >
        <Card class="border-croffle-border overflow-hidden rounded-xl border bg-white/80 shadow-sm">
          <CardHeader class="space-y-0 px-4 pt-0 pb-2">
            <CardTitle class="text-croffle-text-dark flex items-center gap-2 text-sm font-bold">
              <Calendar class="h-4 w-4" />
              <span>오늘의 일정</span>
              <Badge
                class="bg-croffle-sidebar text-croffle-text-dark ml-auto h-5 rounded-md px-1.5"
              >
                {{ todayCount || 0 }}
              </Badge>
            </CardTitle>
          </CardHeader>
          <CardContent class="text-croffle-text flex min-h-25 items-center justify-center text-sm">
            <span v-if="!hasTodayEvent">오늘 일정이 없습니다</span>
            <span v-else>일정 리스트</span>
          </CardContent>
        </Card>

        <Card class="border-croffle-border overflow-hidden rounded-xl border bg-white/80 shadow-sm">
          <CardHeader class="space-y-0 px-4 pt-0 pb-2">
            <CardTitle class="text-croffle-text-dark flex items-center gap-2 text-sm font-bold">
              <Clock class="h-4 w-4" />
              <span>다가오는 일정</span>
            </CardTitle>
          </CardHeader>
          <CardContent class="text-croffle-text flex min-h-25 items-center justify-center text-sm">
            다가오는 일정이 없습니다
          </CardContent>
        </Card>
      </div>
    </SidebarContent>

    <SidebarFooter class="bg-croffle-sidebar shrink-0 flex-col items-center justify-center pb-4">
      <div
        class="border-croffle-border mb-4 flex w-full flex-col items-center justify-center rounded-xl border bg-white/50 p-4 shadow-sm group-data-[collapsible=icon]:border-none group-data-[collapsible=icon]:bg-transparent group-data-[collapsible=icon]:p-0 group-data-[collapsible=icon]:shadow-none"
      >
        <Home
          class="text-croffle-primary mb-1 h-6 w-6 group-data-[collapsible=icon]:h-5 group-data-[collapsible=icon]:w-5"
        />
        <div class="text-center group-data-[collapsible=icon]:hidden">
          <h4 class="text-croffle-text-dark text-xs font-bold tracking-wider">CROFFLE</h4>
          <span class="text-croffle-text text-xs">v 버전 추가해야함</span>
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
