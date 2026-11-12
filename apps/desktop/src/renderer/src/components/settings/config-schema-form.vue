<script setup lang="ts">
  import type { ConfigItemSchema } from '@croffledev/common';

  import { Input } from '@/components/ui/input';
  import { Label } from '@/components/ui/label';
  import {
    Select,
    SelectTrigger,
    SelectValue,
    SelectContent,
    SelectItem,
  } from '@/components/ui/select';
  import { Switch } from '@/components/ui/switch';

  const values = defineModel<Record<string, unknown>>('values', {
    required: true,
  });

  defineProps<{
    items: Record<string, ConfigItemSchema>;
    sectionTitle?: string;
    sectionDescription?: string;
  }>();

  const asBool = (v: unknown): boolean => {
    if (typeof v === 'boolean') {
      return v;
    }
    if (typeof v === 'string') {
      return v === 'true' || v === '1' || v === 'on';
    }
    return !!v;
  };

  const onSwitch = (key: string, v: unknown) => {
    values.value = { ...values.value, [key]: asBool(v) };
  };

  const updateValue = (key: string, v: unknown) => {
    values.value = { ...values.value, [key]: v };
  };
</script>

<template>
  <section class="space-y-4">
    <div v-if="sectionTitle || sectionDescription" class="space-y-1">
      <h4 v-if="sectionTitle" class="text-base font-bold text-neutral-900">
        {{ sectionTitle }}
      </h4>
      <p v-if="sectionDescription" class="text-muted-foreground text-sm">
        {{ sectionDescription }}
      </p>
    </div>

    <div v-for="(schema, key) in items" :key="key" class="space-y-2">
      <div v-if="schema.type === 'boolean'" class="flex items-center justify-between">
        <div class="min-w-0 space-y-0.5">
          <Label class="text-sm font-semibold">{{ schema.label }}</Label>
          <p v-if="schema.description" class="text-muted-foreground text-xs">
            {{ schema.description }}
          </p>
        </div>
        <Switch
          :checked="!!values[key]"
          :model-value="!!values[key]"
          :aria-label="schema.label"
          @update:checked="(v: boolean) => onSwitch(key, v)"
          @update:model-value="(v: unknown) => onSwitch(key, v)"
        />
      </div>

      <template v-else-if="schema.type === 'select' && schema.options">
        <Label :for="`ext-setting-${key}`" class="text-foreground text-sm font-medium">
          {{ schema.label }}
        </Label>
        <Select
          :model-value="String(values[key] ?? schema.defaultValue)"
          @update:model-value="(v: unknown) => updateValue(key, v)"
        >
          <SelectTrigger :id="`ext-setting-${key}`" class="w-full">
            <SelectValue :placeholder="schema.label" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem
              v-for="opt in schema.options"
              :key="String(opt.value)"
              :value="String(opt.value)"
            >
              {{ opt.label }}
            </SelectItem>
          </SelectContent>
        </Select>
        <p v-if="schema.description" class="text-muted-foreground text-xs">
          {{ schema.description }}
        </p>
      </template>

      <template v-else>
        <Label :for="`ext-setting-${key}`" class="text-foreground text-sm font-medium">
          {{ schema.label }}
        </Label>
        <Input
          :id="`ext-setting-${key}`"
          :model-value="String(values[key] ?? '')"
          :type="schema.type === 'number' ? 'number' : 'text'"
          class="h-10"
          @update:model-value="(v) => updateValue(key, schema.type === 'number' ? Number(v) : v)"
        />
        <p v-if="schema.description" class="text-muted-foreground text-xs">
          {{ schema.description }}
        </p>
      </template>
    </div>
  </section>
</template>
