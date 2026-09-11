<script setup lang="ts">
  import {
    type Schedule,
    RECURRENCE_PRESET_OPTIONS,
    WEEKDAY_OPTIONS,
    buildRRule,
    createDefaultRecurrenceFormState,
    parseRRule,
    weekdayCodeFromDate,
    type RecurrenceEndMode,
    type RecurrenceFormState,
    type RecurrencePreset,
    type WeekdayCode,
  } from '@croffledev/common';
  import { CalendarDate, getLocalTimeZone } from '@internationalized/date';
  import dayjs from 'dayjs';
  import { storeToRefs } from 'pinia';
  import { computed, reactive, ref, shallowRef, toRaw, watch } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { toast } from 'vue-sonner';

  import { Button } from '@/components/ui/button';
  import { Calendar } from '@/components/ui/calendar';
  import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
  } from '@/components/ui/dialog';
  import {
    Field,
    FieldDescription,
    FieldGroup,
    FieldLabel,
    FieldLegend,
    FieldSeparator,
    FieldSet,
  } from '@/components/ui/field';
  import { Icon } from '@/components/ui/icon';
  import { Input } from '@/components/ui/input';
  import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
  import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
  } from '@/components/ui/select';
  import { Switch } from '@/components/ui/switch';
  import { Textarea } from '@/components/ui/textarea';
  import { cn } from '@/lib/utils';
  import { useScheduleStore } from '@/stores/schedule-store';
  import { useUiStore } from '@/stores/ui-store';

  import TimeField from './time-field.vue';

  const uiStore = useUiStore();
  const scheduleStore = useScheduleStore();
  const { t, locale } = useI18n();
  const { isScheduleModalOpen, scheduleModalMode, selectedScheduleId } = storeToRefs(uiStore);

  const DEFAULT_START_TIME = '09:00';
  const DEFAULT_END_TIME = '10:00';

  const COLOR_PRESETS = [
    '#DCA780',
    '#F87171',
    '#FBBF24',
    '#34D399',
    '#60A5FA',
    '#A78BFA',
    '#F472B6',
  ];

  const toCalendarDate = (value: string | Date) => {
    const d = value instanceof Date ? value : new Date(value);
    return new CalendarDate(d.getFullYear(), d.getMonth() + 1, d.getDate());
  };

  const formatCalendarDate = (calendarDate: CalendarDate | undefined) => {
    if (!calendarDate) {
      return t('schedule.pickDate');
    }
    const jsDate = calendarDate.toDate(getLocalTimeZone());
    return new Intl.DateTimeFormat(locale.value === 'ko' ? 'ko-KR' : 'en-US', {
      month: 'long',
      day: 'numeric',
      weekday: 'short',
    }).format(jsDate);
  };

  const formatTimeFromDate = (value: string | Date) => {
    const date = value instanceof Date ? value : new Date(value);
    const hours = String(date.getHours()).padStart(2, '0');
    const minutes = String(date.getMinutes()).padStart(2, '0');
    return `${hours}:${minutes}`;
  };

  const combineDateAndTime = (date: CalendarDate, time: string) => {
    const [hours = 0, minutes = 0] = time.split(':').map((part) => Number(part));
    const jsDate = date.toDate(getLocalTimeZone());
    jsDate.setHours(hours, minutes, 0, 0);
    return jsDate;
  };

  /** 로컬 기준 오늘 (YYYY-MM-DD). toISOString()은 UTC라 자정~09:00(KST)에 어제로 잡힌다. */
  const todayLocal = () => dayjs().format('YYYY-MM-DD');

  /**
   * 일정 추가 시 초기 시각.
   * 대상 날짜가 오늘이면 현재 시각 / +1h, 아니면 09:00 / 10:00.
   * +1h가 자정을 넘으면 종료 날짜를 다음 날로 넘긴다.
   */
  const resolveInitialTimes = (
    dateStr: string,
  ): { start: string; end: string; endDate: string } => {
    if (dateStr !== todayLocal()) {
      return { start: DEFAULT_START_TIME, end: DEFAULT_END_TIME, endDate: dateStr };
    }
    const now = dayjs();
    const end = now.add(1, 'hour');
    return {
      start: now.format('HH:mm'),
      end: end.format('HH:mm'),
      endDate: end.format('YYYY-MM-DD'),
    };
  };

  const form = reactive({
    title: '',
    description: '',
    location: '',
    priority: 'medium' as 'low' | 'medium' | 'high',
    isAllDay: false,
    colorLabel: '#DCA780',
  });

  const recurrence = reactive<RecurrenceFormState>(createDefaultRecurrenceFormState());
  const isUntilCalendarOpen = ref(false);
  const untilDate = shallowRef<CalendarDate | undefined>(undefined);

  const showInterval = computed(
    () => recurrence.preset === 'every-n-days' || recurrence.preset === 'every-n-weeks',
  );
  const showWeekdays = computed(
    () =>
      recurrence.preset === 'weekly' ||
      recurrence.preset === 'every-n-weeks' ||
      recurrence.preset === 'weekdays',
  );
  const weekdaysEditable = computed(
    () => recurrence.preset === 'weekly' || recurrence.preset === 'every-n-weeks',
  );

  // CalendarDate 객체 내부의 #private 필드가 Vue Proxy와 충돌하는 것을 막기 위해
  // 반드시 ref가 아닌 shallowRef를 사용해야 함
  const startDate = shallowRef<CalendarDate | undefined>(undefined);
  const endDate = shallowRef<CalendarDate | undefined>(undefined);
  const startTime = ref(DEFAULT_START_TIME);
  const endTime = ref(DEFAULT_END_TIME);
  const isStartCalendarOpen = ref(false);
  const isEndCalendarOpen = ref(false);

  const applyRecurrenceState = (state: RecurrenceFormState) => {
    recurrence.preset = state.preset;
    recurrence.interval = state.interval;
    recurrence.byWeekday = [...state.byWeekday];
    recurrence.endMode = state.endMode;
    recurrence.until = state.until;
    recurrence.count = state.count;
    recurrence.rawRule = state.rawRule;
    untilDate.value = state.until ? toCalendarDate(state.until) : undefined;
  };

  const resetRecurrence = () => {
    applyRecurrenceState(createDefaultRecurrenceFormState());
  };

  const resetForm = () => {
    form.title = '';
    form.description = '';
    form.location = '';
    form.priority = 'medium';
    form.isAllDay = false;
    form.colorLabel = '#DCA780';
    startDate.value = undefined;
    endDate.value = undefined;
    startTime.value = DEFAULT_START_TIME;
    endTime.value = DEFAULT_END_TIME;
    resetRecurrence();
  };

  const fillFormFromSchedule = (schedule: Schedule) => {
    const cloned = structuredClone(toRaw(schedule));

    form.title = cloned.title ?? '';
    form.description = cloned.description ?? '';
    form.location = cloned.location ?? '';
    form.isAllDay = cloned.isAllDay ?? false;
    form.colorLabel = cloned.colorLabel ?? '#DCA780';
    form.priority = cloned.priority ?? 'medium';
    startDate.value = cloned.startDate ? toCalendarDate(cloned.startDate) : undefined;
    endDate.value = cloned.endDate ? toCalendarDate(cloned.endDate) : startDate.value;
    startTime.value = cloned.startDate ? formatTimeFromDate(cloned.startDate) : DEFAULT_START_TIME;
    endTime.value = cloned.endDate ? formatTimeFromDate(cloned.endDate) : DEFAULT_END_TIME;
    applyRecurrenceState(parseRRule(cloned.recurrenceRule));
  };

  watch(
    () => ({
      open: isScheduleModalOpen.value,
      mode: scheduleModalMode.value,
      scheduleId: selectedScheduleId.value,
    }),
    ({ open, mode, scheduleId }) => {
      if (!open) {
        return;
      }

      if (mode === 'add') {
        resetForm();
        const target = uiStore.selectedDate ?? todayLocal();
        const initial = resolveInitialTimes(target);
        startDate.value = toCalendarDate(target);
        endDate.value = toCalendarDate(initial.endDate);
        startTime.value = initial.start;
        endTime.value = initial.end;
        return;
      }

      if (!scheduleId) {
        resetForm();
        uiStore.closeScheduleModal();
        return;
      }

      const schedule = scheduleStore.getScheduleById(scheduleId);
      if (!schedule) {
        resetForm();
        uiStore.closeScheduleModal();
        return;
      }

      fillFormFromSchedule(schedule);
    },
    { immediate: true },
  );

  const handleSave = async () => {
    if (!form.title.trim() || !startDate.value || !endDate.value) {
      return;
    }

    let start: Date;
    let end: Date;

    if (form.isAllDay) {
      start = startDate.value.toDate(getLocalTimeZone());
      start.setHours(0, 0, 0, 0);
      end = endDate.value.toDate(getLocalTimeZone());
      end.setHours(23, 59, 59, 999);
    } else {
      start = combineDateAndTime(startDate.value, startTime.value);
      end = combineDateAndTime(endDate.value, endTime.value);
    }

    if (end < start) {
      toast.error(t('schedule.invalidRange'));
      return;
    }

    if (recurrence.endMode === 'until' && untilDate.value) {
      recurrence.until = untilDate.value.toString();
    }

    if (
      (recurrence.preset === 'weekly' || recurrence.preset === 'every-n-weeks') &&
      recurrence.byWeekday.length === 0
    ) {
      recurrence.byWeekday = [weekdayCodeFromDate(start)];
    }

    let recurrenceRule: string | undefined;
    try {
      recurrenceRule = buildRRule(recurrence, start);
    } catch {
      toast.error(t('schedule.invalidRule'));
      return;
    }

    const payload: Partial<Schedule> = {
      title: form.title.trim(),
      description: form.description.trim(),
      location: form.location.trim(),
      startDate: start,
      endDate: end,
      isAllDay: form.isAllDay,
      recurrenceRule,
      colorLabel: form.colorLabel || '#DCA780',
      priority: form.priority,
      tags: [],
    };

    try {
      if (scheduleModalMode.value === 'add') {
        await scheduleStore.createSchedule(payload);
      } else {
        if (!selectedScheduleId.value) {
          return;
        }
        await scheduleStore.updateScheduleById(selectedScheduleId.value, payload);
      }

      uiStore.closeScheduleModal();
    } catch (error) {
      toast.error(t('schedule.saveFailed', { error: JSON.stringify(error) }));
    }
  };

  const handleDelete = async () => {
    if (scheduleModalMode.value !== 'edit' || !selectedScheduleId.value) {
      return;
    }

    try {
      const isSuccess = await scheduleStore.removeScheduleById(selectedScheduleId.value);
      if (isSuccess) {
        uiStore.closeScheduleModal();
      }
    } catch (error) {
      toast.error(t('schedule.deleteFailed', { error: JSON.stringify(error) }));
    }
  };

  const onAllDayChange = (value: unknown) => {
    form.isAllDay = Boolean(value);
  };

  const onRecurrencePresetChange = (value: unknown) => {
    if (typeof value !== 'string') {
      return;
    }
    const preset = value as RecurrencePreset;
    recurrence.preset = preset;
    if (preset === 'weekdays') {
      recurrence.byWeekday = ['MO', 'TU', 'WE', 'TH', 'FR'];
    } else if (
      (preset === 'weekly' || preset === 'every-n-weeks') &&
      recurrence.byWeekday.length === 0 &&
      startDate.value
    ) {
      recurrence.byWeekday = [weekdayCodeFromDate(startDate.value.toDate(getLocalTimeZone()))];
    } else if (preset === 'none' || preset === 'daily' || preset === 'monthly') {
      recurrence.byWeekday = [];
    }
    if (preset !== 'custom') {
      recurrence.rawRule = '';
    }
  };

  const onRecurrenceEndModeChange = (value: unknown) => {
    if (typeof value !== 'string') {
      return;
    }
    recurrence.endMode = value as RecurrenceEndMode;
    if (recurrence.endMode === 'until' && !untilDate.value && startDate.value) {
      untilDate.value = startDate.value;
      recurrence.until = startDate.value.toString();
    }
  };

  const toggleWeekday = (day: WeekdayCode) => {
    if (!weekdaysEditable.value) {
      return;
    }
    if (recurrence.byWeekday.includes(day)) {
      recurrence.byWeekday = recurrence.byWeekday.filter((d) => d !== day);
      return;
    }
    recurrence.byWeekday = [...recurrence.byWeekday, day];
  };

  const RECURRENCE_PRESET_KEYS: Record<RecurrencePreset, string> = {
    none: 'recurrence.none',
    daily: 'recurrence.daily',
    weekdays: 'recurrence.weekdays',
    weekly: 'recurrence.weekly',
    monthly: 'recurrence.monthly',
    'every-n-days': 'recurrence.everyNDays',
    'every-n-weeks': 'recurrence.everyNWeeks',
    custom: 'recurrence.custom',
  };

  const priorityOptions = computed(() => [
    {
      value: 'low' as const,
      label: t('priority.low'),
      active: 'border-emerald-500/40 bg-emerald-500/10 text-emerald-700',
      iconClass: 'text-emerald-600',
    },
    {
      value: 'medium' as const,
      label: t('priority.medium'),
      active: 'border-amber-500/40 bg-amber-500/10 text-amber-700',
      iconClass: 'text-amber-600',
    },
    {
      value: 'high' as const,
      label: t('priority.high'),
      active: 'border-rose-500/40 bg-rose-500/10 text-rose-700',
      iconClass: 'text-rose-600',
    },
  ]);
</script>

<template>
  <Dialog
    :open="isScheduleModalOpen"
    @update:open="
      (val) => {
        if (!val) uiStore.closeScheduleModal();
      }
    "
  >
    <DialogContent
      class="border-croffle-border bg-background gap-0 overflow-hidden p-0 shadow-2xl sm:max-w-lg"
    >
      <DialogHeader class="space-y-3 border-b px-6 pt-6 pb-5 text-left">
        <div class="flex items-start gap-3 pr-6">
          <div
            class="bg-croffle-primary/10 text-croffle-primary flex size-10 shrink-0 items-center justify-center rounded-xl"
          >
            <Icon
              :icon="
                scheduleModalMode === 'edit' ? 'lucide:calendar-check-2' : 'lucide:calendar-plus'
              "
              class="size-5"
            />
          </div>
          <div class="min-w-0 space-y-1">
            <DialogTitle class="text-foreground text-lg font-semibold tracking-tight">
              {{
                scheduleModalMode === 'edit' ? $t('schedule.editTitle') : $t('schedule.createTitle')
              }}
            </DialogTitle>
            <DialogDescription class="text-muted-foreground text-sm">
              {{
                scheduleModalMode === 'edit'
                  ? $t('schedule.editDescription')
                  : $t('schedule.createDescription')
              }}
            </DialogDescription>
          </div>
        </div>
      </DialogHeader>

      <div class="max-h-[min(62vh,560px)] overflow-y-auto px-6 py-5">
        <form class="contents" @submit.prevent="handleSave">
          <FieldGroup class="gap-6">
            <FieldSet class="gap-4">
              <FieldLegend
                variant="label"
                class="text-muted-foreground mb-0 text-xs tracking-wide uppercase"
              >
                {{ $t('schedule.basics') }}
              </FieldLegend>

              <Field>
                <FieldLabel for="schedule-title">
                  {{ $t('schedule.title') }} <span class="text-destructive">*</span>
                </FieldLabel>
                <Input
                  id="schedule-title"
                  v-model="form.title"
                  :placeholder="$t('schedule.titlePlaceholder')"
                  class="border-croffle-border focus-visible:ring-croffle-primary/30 h-10"
                />
              </Field>

              <Field>
                <FieldLabel for="schedule-description">{{ $t('schedule.description') }}</FieldLabel>
                <Textarea
                  id="schedule-description"
                  v-model="form.description"
                  :placeholder="$t('schedule.descriptionPlaceholder')"
                  rows="3"
                  class="border-croffle-border focus-visible:ring-croffle-primary/30 min-h-20 resize-none"
                />
              </Field>

              <Field>
                <FieldLabel for="schedule-location">{{ $t('schedule.location') }}</FieldLabel>
                <div class="relative">
                  <Icon
                    icon="lucide:map-pin"
                    class="text-muted-foreground pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2"
                  />
                  <Input
                    id="schedule-location"
                    v-model="form.location"
                    :placeholder="$t('schedule.locationPlaceholder')"
                    class="border-croffle-border focus-visible:ring-croffle-primary/30 h-10 pl-9"
                  />
                </div>
              </Field>
            </FieldSet>

            <FieldSeparator />

            <FieldSet class="gap-4">
              <FieldLegend
                variant="label"
                class="text-muted-foreground mb-0 text-xs tracking-wide uppercase"
              >
                {{ $t('schedule.dateTime') }}
              </FieldLegend>

              <Field
                orientation="horizontal"
                class="border-croffle-border bg-muted/30 items-center justify-between rounded-xl border px-3.5 py-3"
              >
                <div class="space-y-0.5">
                  <FieldLabel for="schedule-all-day" class="text-sm font-medium">
                    {{ $t('schedule.allDay') }}
                  </FieldLabel>
                  <FieldDescription class="text-xs">
                    {{ $t('schedule.allDayHint') }}
                  </FieldDescription>
                </div>
                <Switch
                  id="schedule-all-day"
                  :checked="form.isAllDay"
                  @update:checked="onAllDayChange"
                  @update:model-value="onAllDayChange"
                />
              </Field>

              <div class="grid grid-cols-2 gap-3">
                <Field>
                  <FieldLabel class="text-muted-foreground text-xs">
                    {{ $t('schedule.startDate') }}
                  </FieldLabel>
                  <Popover v-model:open="isStartCalendarOpen">
                    <PopoverTrigger as-child>
                      <Button
                        type="button"
                        variant="outline"
                        :class="
                          cn(
                            'border-croffle-border hover:bg-muted/50 h-10 w-full justify-between px-3 font-normal',
                            !startDate && 'text-muted-foreground',
                          )
                        "
                      >
                        <span class="flex min-w-0 items-center gap-2">
                          <Icon
                            icon="lucide:calendar"
                            class="text-muted-foreground size-4 shrink-0"
                          />
                          <span class="truncate">{{ formatCalendarDate(startDate) }}</span>
                        </span>
                        <Icon
                          icon="lucide:chevron-down"
                          class="text-muted-foreground size-3.5 opacity-60"
                        />
                      </Button>
                    </PopoverTrigger>
                    <PopoverContent class="border-croffle-border z-50 w-auto p-0">
                      <Calendar
                        v-model="startDate"
                        mode="single"
                        class="rounded-md border-0"
                        @update:model-value="isStartCalendarOpen = false"
                      />
                    </PopoverContent>
                  </Popover>
                </Field>

                <Field>
                  <FieldLabel class="text-muted-foreground text-xs">
                    {{ $t('schedule.endDate') }}
                  </FieldLabel>
                  <Popover v-model:open="isEndCalendarOpen">
                    <PopoverTrigger as-child>
                      <Button
                        type="button"
                        variant="outline"
                        :class="
                          cn(
                            'border-croffle-border hover:bg-muted/50 h-10 w-full justify-between px-3 font-normal',
                            !endDate && 'text-muted-foreground',
                          )
                        "
                      >
                        <span class="flex min-w-0 items-center gap-2">
                          <Icon
                            icon="lucide:calendar"
                            class="text-muted-foreground size-4 shrink-0"
                          />
                          <span class="truncate">{{ formatCalendarDate(endDate) }}</span>
                        </span>
                        <Icon
                          icon="lucide:chevron-down"
                          class="text-muted-foreground size-3.5 opacity-60"
                        />
                      </Button>
                    </PopoverTrigger>
                    <PopoverContent class="border-croffle-border z-50 w-auto p-0">
                      <Calendar
                        v-model="endDate"
                        mode="single"
                        class="rounded-md border-0"
                        @update:model-value="isEndCalendarOpen = false"
                      />
                    </PopoverContent>
                  </Popover>
                </Field>
              </div>

              <div
                v-if="!form.isAllDay"
                class="grid grid-cols-2 gap-3 duration-200 animate-in fade-in-0 slide-in-from-top-1"
              >
                <Field>
                  <FieldLabel for="schedule-start-time" class="text-muted-foreground text-xs">
                    {{ $t('schedule.startTime') }}
                  </FieldLabel>
                  <TimeField id="schedule-start-time" v-model="startTime" />
                </Field>
                <Field>
                  <FieldLabel for="schedule-end-time" class="text-muted-foreground text-xs">
                    {{ $t('schedule.endTime') }}
                  </FieldLabel>
                  <TimeField id="schedule-end-time" v-model="endTime" />
                </Field>
              </div>
            </FieldSet>

            <FieldSeparator />

            <FieldSet class="gap-4">
              <FieldLegend
                variant="label"
                class="text-muted-foreground mb-0 text-xs tracking-wide uppercase"
              >
                {{ $t('schedule.options') }}
              </FieldLegend>

              <Field>
                <FieldLabel>{{ $t('schedule.priority') }}</FieldLabel>
                <div
                  class="border-croffle-border bg-muted/20 grid grid-cols-3 gap-1 rounded-xl border p-1"
                  role="radiogroup"
                  :aria-label="$t('schedule.priority')"
                >
                  <button
                    v-for="option in priorityOptions"
                    :key="option.value"
                    type="button"
                    role="radio"
                    :aria-checked="form.priority === option.value"
                    class="flex cursor-pointer flex-col items-center gap-1.5 rounded-lg px-2 py-2.5 text-xs font-medium transition-all"
                    :class="
                      form.priority === option.value
                        ? cn('bg-background shadow-sm ring-1 ring-black/5', option.active)
                        : 'text-muted-foreground hover:bg-background/70 hover:text-foreground'
                    "
                    @click="form.priority = option.value"
                  >
                    {{ option.label }}
                  </button>
                </div>
              </Field>

              <Field>
                <FieldLabel>{{ $t('schedule.color') }}</FieldLabel>
                <div class="flex flex-wrap items-center gap-2">
                  <button
                    v-for="color in COLOR_PRESETS"
                    :key="color"
                    type="button"
                    class="size-8 cursor-pointer rounded-full border-2 transition-transform hover:scale-105"
                    :class="
                      form.colorLabel.toLowerCase() === color.toLowerCase()
                        ? 'border-foreground scale-105'
                        : 'border-transparent'
                    "
                    :style="{ backgroundColor: color }"
                    :aria-label="$t('schedule.colorAria', { color })"
                    @click="form.colorLabel = color"
                  />
                  <label
                    class="border-croffle-border hover:bg-muted/50 relative flex size-8 cursor-pointer items-center justify-center overflow-hidden rounded-full border border-dashed"
                    :title="$t('schedule.customColor')"
                  >
                    <Icon icon="lucide:palette" class="text-muted-foreground size-3.5" />
                    <input
                      v-model="form.colorLabel"
                      type="color"
                      class="absolute inset-0 cursor-pointer opacity-0"
                      :aria-label="$t('schedule.customColorAria')"
                    />
                  </label>
                </div>
              </Field>

              <Field>
                <FieldLabel for="schedule-recurrence">{{ $t('schedule.recurrence') }}</FieldLabel>
                <Select
                  :model-value="recurrence.preset"
                  @update:model-value="onRecurrencePresetChange"
                >
                  <SelectTrigger id="schedule-recurrence" class="border-croffle-border h-10 w-full">
                    <SelectValue :placeholder="$t('schedule.recurrencePlaceholder')" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem
                      v-for="preset in RECURRENCE_PRESET_OPTIONS"
                      :key="preset"
                      :value="preset"
                    >
                      {{ $t(RECURRENCE_PRESET_KEYS[preset]) }}
                    </SelectItem>
                  </SelectContent>
                </Select>
                <FieldDescription>
                  {{ $t('schedule.recurrenceHint') }}
                </FieldDescription>
              </Field>

              <Field v-if="showInterval">
                <FieldLabel for="schedule-recurrence-interval">
                  {{ $t('schedule.interval') }}
                </FieldLabel>
                <div class="flex items-center gap-2">
                  <Input
                    id="schedule-recurrence-interval"
                    v-model.number="recurrence.interval"
                    type="number"
                    min="1"
                    class="border-croffle-border h-10 w-24"
                  />
                  <span class="text-muted-foreground text-sm">
                    {{
                      recurrence.preset === 'every-n-weeks'
                        ? $t('schedule.everyWeeks')
                        : $t('schedule.everyDays')
                    }}
                  </span>
                </div>
              </Field>

              <Field v-if="showWeekdays">
                <FieldLabel>{{ $t('schedule.weekdays') }}</FieldLabel>
                <div class="flex flex-wrap gap-1.5">
                  <button
                    v-for="day in WEEKDAY_OPTIONS"
                    :key="day"
                    type="button"
                    class="h-8 min-w-8 rounded-md border px-2 text-xs font-medium transition-colors"
                    :class="
                      recurrence.byWeekday.includes(day)
                        ? 'border-croffle-primary bg-croffle-primary/10 text-croffle-primary'
                        : 'border-croffle-border text-muted-foreground hover:bg-muted/50'
                    "
                    :disabled="!weekdaysEditable"
                    @click="toggleWeekday(day)"
                  >
                    {{ $t(`recurrence.weekday.${day}`) }}
                  </button>
                </div>
              </Field>

              <template v-if="recurrence.preset !== 'none'">
                <Field>
                  <FieldLabel for="schedule-recurrence-end">
                    {{ $t('schedule.recurrenceEnd') }}
                  </FieldLabel>
                  <Select
                    :model-value="recurrence.endMode"
                    @update:model-value="onRecurrenceEndModeChange"
                  >
                    <SelectTrigger
                      id="schedule-recurrence-end"
                      class="border-croffle-border h-10 w-full"
                    >
                      <SelectValue :placeholder="$t('schedule.endConditionPlaceholder')" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="never">{{ $t('schedule.endNever') }}</SelectItem>
                      <SelectItem value="until">{{ $t('schedule.endUntil') }}</SelectItem>
                      <SelectItem value="count">{{ $t('schedule.endCount') }}</SelectItem>
                    </SelectContent>
                  </Select>
                </Field>

                <Field v-if="recurrence.endMode === 'until'">
                  <FieldLabel>{{ $t('schedule.untilDate') }}</FieldLabel>
                  <Popover v-model:open="isUntilCalendarOpen">
                    <PopoverTrigger as-child>
                      <Button
                        type="button"
                        variant="outline"
                        class="border-croffle-border hover:bg-muted/50 h-10 w-full justify-between px-3 font-normal"
                      >
                        <span class="flex min-w-0 items-center gap-2">
                          <Icon
                            icon="lucide:calendar"
                            class="text-muted-foreground size-4 shrink-0"
                          />
                          <span class="truncate">{{ formatCalendarDate(untilDate) }}</span>
                        </span>
                        <Icon
                          icon="lucide:chevron-down"
                          class="text-muted-foreground size-3.5 opacity-60"
                        />
                      </Button>
                    </PopoverTrigger>
                    <PopoverContent class="border-croffle-border z-50 w-auto p-0">
                      <Calendar
                        v-model="untilDate"
                        mode="single"
                        class="rounded-md border-0"
                        @update:model-value="
                          (value) => {
                            isUntilCalendarOpen = false;
                            if (value) recurrence.until = value.toString();
                          }
                        "
                      />
                    </PopoverContent>
                  </Popover>
                </Field>

                <Field v-if="recurrence.endMode === 'count'">
                  <FieldLabel for="schedule-recurrence-count">
                    {{ $t('schedule.count') }}
                  </FieldLabel>
                  <div class="flex items-center gap-2">
                    <Input
                      id="schedule-recurrence-count"
                      v-model.number="recurrence.count"
                      type="number"
                      min="1"
                      class="border-croffle-border h-10 w-24"
                    />
                    <span class="text-muted-foreground text-sm">
                      {{ $t('schedule.countSuffix') }}
                    </span>
                  </div>
                </Field>
              </template>

              <Field v-if="recurrence.preset === 'custom'">
                <FieldLabel for="schedule-recurrence-raw">
                  {{ $t('schedule.customRrule') }}
                </FieldLabel>
                <Input
                  id="schedule-recurrence-raw"
                  v-model="recurrence.rawRule"
                  placeholder="FREQ=WEEKLY;BYDAY=FR"
                  class="border-croffle-border focus-visible:ring-croffle-primary/30 h-10 font-mono text-xs"
                />
              </Field>
            </FieldSet>
          </FieldGroup>
        </form>
      </div>

      <DialogFooter
        class="border-croffle-border bg-muted/20 flex-row items-center gap-2 border-t px-6 py-4 sm:justify-between"
      >
        <div class="min-w-0">
          <Button
            v-if="scheduleModalMode === 'edit'"
            type="button"
            variant="ghost"
            class="text-destructive hover:bg-destructive/10 hover:text-destructive h-9 px-3"
            @click="handleDelete"
          >
            <Icon icon="lucide:trash-2" class="mr-1.5 size-4" />
            {{ $t('common.delete') }}
          </Button>
        </div>

        <div class="flex shrink-0 gap-2">
          <Button
            type="button"
            variant="outline"
            class="border-croffle-border h-9"
            @click="uiStore.closeScheduleModal()"
          >
            {{ $t('common.cancel') }}
          </Button>
          <Button
            type="button"
            :disabled="!form.title.trim() || !startDate || !endDate"
            class="bg-croffle-primary hover:bg-croffle-hover h-9 text-white"
            @click="handleSave"
          >
            <Icon
              :icon="scheduleModalMode === 'edit' ? 'lucide:check' : 'lucide:plus'"
              class="mr-1.5 size-4"
            />
            {{ scheduleModalMode === 'edit' ? $t('common.save') : $t('common.add') }}
          </Button>
        </div>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
