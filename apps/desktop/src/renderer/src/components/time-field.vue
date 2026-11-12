<script setup lang="ts">
  import { computed } from 'vue';

  import { Icon } from '@/components/ui/icon';
  import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
  } from '@/components/ui/select';
  import { cn } from '@/lib/utils';

  const props = withDefaults(
    defineProps<{
      id?: string;
      class?: string;
    }>(),
    {},
  );

  const model = defineModel<string>({ required: true });

  const HOUR_OPTIONS = Array.from({ length: 24 }, (_, i) => String(i).padStart(2, '0'));
  const MINUTE_OPTIONS = Array.from({ length: 60 }, (_, i) => String(i).padStart(2, '0'));

  const hour = computed({
    get: () => {
      const [h = '09'] = model.value.split(':');
      return HOUR_OPTIONS.includes(h) ? h : '09';
    },
    set: (nextHour: string) => {
      model.value = `${nextHour}:${minute.value}`;
    },
  });

  const minute = computed({
    get: () => {
      const [, m = '00'] = model.value.split(':');
      return MINUTE_OPTIONS.includes(m) ? m : '00';
    },
    set: (nextMinute: string) => {
      const normalized = MINUTE_OPTIONS.includes(nextMinute) ? nextMinute : '00';
      model.value = `${hour.value}:${normalized}`;
    },
  });

  const onHourChange = (value: unknown) => {
    if (typeof value === 'string') {
      hour.value = value;
    }
  };

  const onMinuteChange = (value: unknown) => {
    if (typeof value === 'string') {
      minute.value = value;
    }
  };
</script>

<template>
  <div
    :id="props.id"
    :class="cn('bg-background flex h-10 items-center gap-1 rounded-md border px-2', props.class)"
  >
    <Icon icon="lucide:clock" class="text-muted-foreground ml-1 size-4 shrink-0" />
    <Select :model-value="hour" @update:model-value="onHourChange">
      <SelectTrigger
        class="border-0 shadow-none focus-visible:ring-0 data-[size=default]:h-8 data-[size=default]:px-2"
        :aria-label="`${props.id ?? 'time'}-hour`"
      >
        <SelectValue :placeholder="hour" />
      </SelectTrigger>
      <SelectContent class="max-h-56 min-w-18">
        <SelectItem v-for="option in HOUR_OPTIONS" :key="option" :value="option">
          {{ option }}
        </SelectItem>
      </SelectContent>
    </Select>
    <span class="text-muted-foreground text-sm font-medium">:</span>
    <Select :model-value="minute" @update:model-value="onMinuteChange">
      <SelectTrigger
        class="border-0 shadow-none focus-visible:ring-0 data-[size=default]:h-8 data-[size=default]:px-2"
        :aria-label="`${props.id ?? 'time'}-minute`"
      >
        <SelectValue :placeholder="minute" />
      </SelectTrigger>
      <SelectContent class="max-h-56 min-w-18">
        <SelectItem v-for="option in MINUTE_OPTIONS" :key="option" :value="option">
          {{ option }}
        </SelectItem>
      </SelectContent>
    </Select>
  </div>
</template>
