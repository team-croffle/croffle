<script setup lang="ts">
  import { parseNumberInput, type ConfigItemSchema } from '@croffledev/common';
  import { ref } from 'vue';
  import { useI18n } from 'vue-i18n';

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

  const { t } = useI18n();

  /** Text and number fields apply when the field is left or Enter is pressed, not per keystroke. */
  const typed = ref<Record<string, string>>({});
  const fieldErrors = ref<Record<string, string>>({});

  const shown = (key: string) => typed.value[key] ?? String(values.value[key] ?? '');

  const commit = (key: string, schema: ConfigItemSchema) => {
    const raw = typed.value[key];
    if (raw === undefined) {
      return;
    }
    if (schema.type === 'number') {
      const value = parseNumberInput(raw);
      if (value === null) {
        fieldErrors.value = { ...fieldErrors.value, [key]: t('settings.errors.notANumber') };
        return;
      }
      updateValue(key, value);
    } else {
      updateValue(key, raw);
    }
    const { [key]: _done, ...rest } = typed.value;
    typed.value = rest;
    const { [key]: _cleared, ...others } = fieldErrors.value;
    fieldErrors.value = others;
  };
</script>

<template>
  <section class="space-y-4">
    <div v-if="sectionTitle || sectionDescription" class="space-y-1">
      <h4 v-if="sectionTitle" class="text-base font-bold text-foreground">
        {{ sectionTitle }}
      </h4>
      <p v-if="sectionDescription" class="text-muted-foreground text-sm">
        {{ sectionDescription }}
      </p>
    </div>

    <div v-for="(schema, key) in items" :key="key" class="space-y-2">
      <div v-if="schema.type === 'boolean'" class="flex items-center justify-between gap-4">
        <div class="min-w-0 space-y-0.5">
          <Label class="text-foreground text-sm font-medium">{{ schema.label }}</Label>
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

      <div v-else class="flex items-center justify-between gap-4">
        <div class="min-w-0 space-y-0.5">
          <Label :for="`ext-setting-${key}`" class="text-foreground text-sm font-medium">
            {{ schema.label }}
          </Label>
          <p v-if="schema.description" class="text-muted-foreground text-xs">
            {{ schema.description }}
          </p>
        </div>
        <Select
          v-if="schema.type === 'select' && schema.options"
          :model-value="String(values[key] ?? schema.defaultValue)"
          @update:model-value="(v: unknown) => updateValue(key, v)"
        >
          <SelectTrigger :id="`ext-setting-${key}`" class="w-60 shrink-0">
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
        <div v-else class="flex shrink-0 flex-col items-end gap-1">
          <Input
            :id="`ext-setting-${key}`"
            :model-value="shown(key)"
            :type="schema.type === 'number' ? 'number' : 'text'"
            :class="schema.type === 'number' ? 'w-24' : 'w-60'"
            :aria-invalid="fieldErrors[key] ? true : undefined"
            :aria-describedby="fieldErrors[key] ? `ext-setting-${key}-error` : undefined"
            @update:model-value="(v) => (typed[key] = String(v))"
            @blur="commit(key, schema)"
            @keydown.enter="commit(key, schema)"
          />
          <p
            v-if="fieldErrors[key]"
            :id="`ext-setting-${key}-error`"
            class="text-destructive text-xs"
            role="alert"
          >
            {{ fieldErrors[key] }}
          </p>
        </div>
      </div>
    </div>
  </section>
</template>
