<script setup lang="ts">
  import { ref, computed } from 'vue';
  import { useI18n } from 'vue-i18n';

  import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
    DialogDescription,
  } from '@/components/ui/dialog';
  import { Icon } from '@/components/ui/icon';

  defineProps<{ open: boolean }>();

  const emit = defineEmits(['update:open']);
  const { t } = useI18n();

  const currentStep = ref(0);

  const steps = computed(() => [
    {
      title: t('help.welcome.title'),
      description: t('help.welcome.description'),
      content: t('help.welcome.content'),
      icon: 'lucide:layout',
    },
    {
      title: t('help.addTodo.title'),
      description: t('help.addTodo.description'),
      content: t('help.addTodo.content'),
      icon: 'lucide:calendar',
    },
    {
      title: t('help.theme.title'),
      description: t('help.theme.description'),
      content: t('help.theme.content'),
      icon: 'lucide:palette',
    },
    {
      title: t('help.ready.title'),
      description: t('help.ready.description'),
      content: t('help.ready.content'),
      icon: 'lucide:party-popper',
    },
  ]);

  const totalSteps = computed(() => steps.value.length);

  const currentStepData = computed(() => {
    return steps.value[currentStep.value] || steps.value[0];
  });

  const nextStep = () => {
    if (currentStep.value < totalSteps.value - 1) {
      currentStep.value++;
    } else {
      handleClose();
    }
  };

  const prevStep = () => {
    if (currentStep.value > 0) {
      currentStep.value--;
    }
  };

  const handleClose = () => {
    emit('update:open', false);

    setTimeout(() => {
      currentStep.value = 0;
    }, 300);
  };
</script>

<template>
  <Dialog :open="open" @update:open="handleClose">
    <DialogContent
      v-if="currentStepData"
      class="bg-croffle-bg overflow-hidden rounded-xl p-0 shadow-lg sm:max-w-md"
    >
      <div class="h-1.5 w-full overflow-hidden bg-neutral-100 dark:bg-neutral-500">
        <div
          class="bg-primary h-full transition-all duration-500 ease-in-out"
          :style="{ width: `${((currentStep + 1) / totalSteps) * 100}%` }"
        ></div>
      </div>

      <div class="p-6">
        <DialogHeader class="flex flex-col items-center text-center">
          <div
            class="bg-primary/10 text-primary mb-4 flex h-16 w-16 items-center justify-center rounded-full"
          >
            <Icon :icon="currentStepData.icon" class="h-8 w-8" />
          </div>

          <DialogTitle class="text-primary text-xl font-bold">
            {{ currentStepData.title }}
          </DialogTitle>

          <DialogDescription class="text-croffle-text pt-1 font-medium">
            {{ currentStepData.description }}
          </DialogDescription>
        </DialogHeader>

        <div
          class="my-6 flex min-h-20 items-center justify-center text-center text-sm leading-relaxed whitespace-pre-line text-neutral-400"
        >
          {{ currentStepData.content }}
        </div>

        <div class="flex items-center justify-between pt-2">
          <button
            v-if="currentStep > 0"
            class="text-croffle-text flex items-center gap-1 text-sm font-medium transition-colors hover:text-neutral-600"
            @click="prevStep"
          >
            <Icon icon="lucide:chevron-left" class="h-4 w-4" />

            {{ $t('help.prev') }}
          </button>

          <div v-else></div>

          <button
            class="bg-primary hover:bg-opacity-90 hover:bg-croffle-hover flex items-center gap-2 rounded-lg px-6 py-2.5 text-sm font-bold text-white shadow-md transition-all"
            @click="nextStep"
          >
            {{ currentStep === totalSteps - 1 ? $t('help.done') : $t('help.next') }}

            <Icon v-if="currentStep < totalSteps - 1" icon="lucide:chevron-right" class="h-4 w-4" />
          </button>
        </div>
      </div>
    </DialogContent>
  </Dialog>
</template>
