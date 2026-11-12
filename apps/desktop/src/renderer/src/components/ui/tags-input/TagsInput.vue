<script setup lang="ts">
  import { reactiveOmit } from '@vueuse/core';
  import type { TagsInputRootEmits, TagsInputRootProps } from 'reka-ui';
  import { TagsInputRoot, useForwardPropsEmits } from 'reka-ui';
  import type { HTMLAttributes } from 'vue';

  import { cn } from '@/lib/utils';

  const props = defineProps<TagsInputRootProps & { class?: HTMLAttributes['class'] }>();
  const emits = defineEmits<TagsInputRootEmits>();

  const delegatedProps = reactiveOmit(props, 'class');

  const forwarded = useForwardPropsEmits(delegatedProps, emits);
</script>

<template>
  <TagsInputRoot
    v-bind="forwarded"
    :class="
      cn(
        'border-input bg-background focus-within:border-ring focus-within:ring-ring/50 flex flex-wrap items-center gap-2 rounded-md border px-3 py-1.5 text-sm transition-[color,box-shadow] focus-within:ring-[3px]',
        props.class,
      )
    "
  >
    <slot />
  </TagsInputRoot>
</template>
