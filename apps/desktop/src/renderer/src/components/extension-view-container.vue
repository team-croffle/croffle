<script setup lang="ts">
  import { onBeforeUnmount, onMounted, ref, watch } from 'vue';

  const props = defineProps<{
    renderFn: (container: HTMLElement) => void;
    viewId: string;
  }>();

  const containerRef = ref<HTMLElement | null>(null);

  const renderPlugin = () => {
    if (containerRef.value && props.renderFn) {
      props.renderFn(containerRef.value);
    }
  };

  onMounted(() => {
    renderPlugin();
  });

  watch(
    () => props.viewId,
    () => {
      renderPlugin();
    },
  );

  onBeforeUnmount(() => {
    if (containerRef.value) {
      containerRef.value.innerHTML = '';
    }
  });
</script>

<template>
  <div
    ref="containerRef"
    class="h-full w-full overflow-y-auto rounded-xl border p-6 shadow-sm"
  ></div>
</template>
