<script setup lang="ts">
  import type { NotificationItem } from '@croffledev/common';
  import { storeToRefs } from 'pinia';
  import { computed, onMounted, ref } from 'vue';
  import { useI18n } from 'vue-i18n';

  import { Button } from '@/components/ui/button';
  import { Icon } from '@/components/ui/icon';
  import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
  import { SidebarMenuButton } from '@/components/ui/sidebar';
  import { occurrenceOf, useNotificationStore } from '@/stores/notification-store';
  import { useScheduleStore } from '@/stores/schedule-store';
  import { useUiStore } from '@/stores/ui-store';

  const store = useNotificationStore();
  const { list, unread } = storeToRefs(store);
  const scheduleStore = useScheduleStore();
  const uiStore = useUiStore();
  const { t, locale } = useI18n();
  const open = ref(false);

  onMounted(() => store.init());

  const isEmpty = computed(() => list.value.fired.length === 0 && list.value.upcoming.length === 0);
  const badge = computed(() => (unread.value > 99 ? '99+' : String(unread.value)));

  const formatTime = (ms: number, allDay = false) =>
    new Intl.DateTimeFormat(locale.value === 'ko' ? 'ko-KR' : 'en-US', {
      month: 'short',
      day: 'numeric',
      ...(allDay ? {} : { hour: 'numeric', minute: '2-digit' }),
    }).format(new Date(ms));

  const meta = (item: NotificationItem, key: 'firedMeta' | 'upcomingMeta') =>
    t(`notifications.${key}`, {
      start: formatTime(item.occurrenceStart, item.isAllDay),
      at: formatTime(item.remindAt),
    });

  /** Opens the schedule the reminder belongs to (the occurrence, for a recurring one). */
  const openItem = async (item: NotificationItem) => {
    open.value = false;
    let schedule = scheduleStore.getScheduleById(item.scheduleId);
    if (!schedule) {
      const day = 24 * 60 * 60 * 1000;
      const found = await croffle.calendar.schedules.getAll({
        start: new Date(item.occurrenceStart - day).toISOString(),
        end: new Date(item.occurrenceStart + day).toISOString(),
      });
      schedule = found.find((s) => s.id === item.scheduleId);
    }
    if (!schedule) {
      return;
    }
    // The viewer reads the calendar's store, which holds only the month on screen.
    scheduleStore.upsertSchedule(schedule);
    uiStore.openScheduleModal('view', schedule.id, occurrenceOf(item, schedule));
  };
</script>

<template>
  <Popover v-model:open="open" @update:open="(value: boolean) => value && store.refresh()">
    <PopoverTrigger as-child>
      <SidebarMenuButton
        size="rail"
        class="hover:bg-croffle-hover relative bg-transparent [--sidebar-accent:transparent]"
        :tooltip="$t('sidebar.notifications')"
        :aria-label="
          unread > 0
            ? $t('notifications.bellUnread', { count: unread })
            : $t('sidebar.notifications')
        "
      >
        <Icon icon="lucide:bell" class="text-croffle-text size-5" />
        <span
          v-if="unread > 0"
          class="bg-primary text-primary-foreground absolute top-0.5 right-0.5 flex h-4 min-w-4 items-center justify-center rounded-full px-1 text-2xs leading-none font-semibold"
          aria-hidden="true"
        >
          {{ badge }}
        </span>
      </SidebarMenuButton>
    </PopoverTrigger>

    <PopoverContent side="right" align="end" :side-offset="8" class="w-[340px] p-0">
      <div class="flex items-center justify-between gap-2 border-b py-2 pr-2 pl-4">
        <h2 class="text-sm font-semibold">{{ $t('notifications.title') }}</h2>
        <Button variant="ghost" size="sm" :disabled="unread === 0" @click="store.markAllRead">
          {{ $t('notifications.markAllRead') }}
        </Button>
      </div>

      <p v-if="isEmpty" class="text-muted-foreground px-4 py-8 text-center text-sm">
        {{ $t('notifications.empty') }}
      </p>

      <div v-else class="max-h-96 overflow-y-auto py-1">
        <template v-if="list.fired.length > 0">
          <h3 class="text-muted-foreground px-4 pt-2 pb-1 text-xs font-semibold">
            {{ $t('notifications.fired') }}
          </h3>
          <ul>
            <li v-for="item in list.fired" :key="item.id">
              <button
                type="button"
                class="hover:bg-accent focus-visible:bg-accent flex w-full cursor-pointer items-start gap-2.5 px-4 py-2.5 text-left outline-none"
                @click="openItem(item)"
              >
                <Icon
                  icon="lucide:bell-ring"
                  class="text-muted-foreground mt-0.5 size-4 shrink-0"
                />
                <span class="min-w-0 flex-1">
                  <span
                    class="block truncate text-sm"
                    :class="item.read ? 'font-normal' : 'font-semibold'"
                  >
                    {{ item.title }}
                  </span>
                  <span class="text-muted-foreground block text-xs">
                    {{ meta(item, 'firedMeta') }}
                  </span>
                </span>
              </button>
            </li>
          </ul>
        </template>

        <template v-if="list.upcoming.length > 0">
          <h3 class="text-muted-foreground px-4 pt-2 pb-1 text-xs font-semibold">
            {{ $t('notifications.upcoming') }}
          </h3>
          <ul>
            <li v-for="item in list.upcoming" :key="item.id">
              <button
                type="button"
                class="hover:bg-accent focus-visible:bg-accent flex w-full cursor-pointer items-start gap-2.5 px-4 py-2.5 text-left outline-none"
                @click="openItem(item)"
              >
                <Icon icon="lucide:clock" class="text-muted-foreground mt-0.5 size-4 shrink-0" />
                <span class="min-w-0 flex-1">
                  <span class="block truncate text-sm">{{ item.title }}</span>
                  <span class="text-muted-foreground block text-xs">
                    {{ meta(item, 'upcomingMeta') }}
                  </span>
                </span>
              </button>
            </li>
          </ul>
        </template>
      </div>
    </PopoverContent>
  </Popover>
</template>
