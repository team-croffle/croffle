<script setup lang="ts">
  import { ref, computed } from 'vue';

  import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
    DialogDescription,
  } from '@/components/ui/dialog';

  import {
    ChevronRight,
    ChevronLeft,
    Calendar,
    Layout,
    Palette,
    PartyPopper,
  } from 'lucide-vue-next';

  // Props 및 Emit 정의

  defineProps<{ open: boolean }>();

  const emit = defineEmits(['update:open']);

  const currentStep = ref(0);

  // 튜토리얼 데이터 정의

  const steps = [
    {
      title: '환영합니다!',

      description: '크로플(CROFFLE)에 오신 것을 환영해요.',

      content: '왼쪽 메뉴를 통해 오늘 할 일과\n전체 달력을 자유롭게 오갈 수 있습니다.',

      icon: Layout,
    },

    {
      title: '할 일 추가하기',

      description: '클릭 한 번으로 일정을 등록하세요.',

      content:
        '달력의 날짜를 클릭하면 즉시 할 일을 적을 수 있습니다.\n드래그해서 날짜를 옮기는 것도 가능해요!',

      icon: Calendar,
    },

    {
      title: '나만의 테마 설정',

      description: '취향에 맞는 색상으로 꾸며보세요.',

      content:
        '설정 메뉴에서 크로플의 메인 색상을 변경하여\n본인만의 작업 환경을 만들 수 있습니다.',

      icon: Palette,
    },

    {
      title: '준비 완료!',

      description: '이제 모든 준비가 끝났습니다.',

      content: '지금 바로 첫 번째 할 일을 등록하고\n크로플과 함께 멋진 하루를 계획해 보세요!',

      icon: PartyPopper,
    },
  ];

  const totalSteps = steps.length;

  const currentStepData = computed(() => {
    return steps[currentStep.value] || steps[0];
  });

  // 로직 함수

  const nextStep = () => {
    if (currentStep.value < totalSteps - 1) {
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
      class="border-croffle-border overflow-hidden rounded-xl bg-white p-0 shadow-lg sm:max-w-md"
    >
      <div class="h-1.5 w-full overflow-hidden bg-gray-100">
        <div
          class="bg-croffle-primary h-full transition-all duration-500 ease-in-out"
          :style="{ width: `${((currentStep + 1) / totalSteps) * 100}%` }"
        ></div>
      </div>

      <div class="p-6">
        <DialogHeader class="flex flex-col items-center text-center">
          <div
            class="bg-croffle-primary/10 text-croffle-primary mb-4 flex h-16 w-16 items-center justify-center rounded-full"
          >
            <component :is="currentStepData.icon" class="h-8 w-8" />
          </div>

          <DialogTitle class="text-croffle-primary text-xl font-bold">
            {{ currentStepData.title }}
          </DialogTitle>

          <DialogDescription class="pt-1 font-medium text-gray-500">
            {{ currentStepData.description }}
          </DialogDescription>
        </DialogHeader>

        <div
          class="my-6 flex min-h-20 items-center justify-center text-center text-sm leading-relaxed whitespace-pre-line text-gray-600"
        >
          {{ currentStepData.content }}
        </div>

        <div class="flex items-center justify-between pt-2">
          <button
            v-if="currentStep > 0"
            class="flex items-center gap-1 text-sm font-medium text-gray-400 transition-colors hover:text-gray-600"
            @click="prevStep"
          >
            <ChevronLeft class="h-4 w-4" />

            이전
          </button>

          <div v-else></div>

          <button
            class="bg-croffle-primary hover:bg-opacity-90 flex items-center gap-2 rounded-lg px-6 py-2.5 text-sm font-bold text-white shadow-md transition-all"
            @click="nextStep"
          >
            {{ currentStep === totalSteps - 1 ? '확인' : '다음' }}

            <ChevronRight v-if="currentStep < totalSteps - 1" class="h-4 w-4" />
          </button>
        </div>
      </div>
    </DialogContent>
  </Dialog>
</template>
