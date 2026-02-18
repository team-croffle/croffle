<script setup lang="ts">
  import { ref, computed } from 'vue';

  import { DEFAULT_MENU_ITEMS } from '@/data/dummyMenu';

  import {
    Sidebar,
    SidebarContent,
    SidebarFooter,
    SidebarHeader,
    SidebarMenu,
    SidebarMenuButton,
    SidebarMenuItem,
    SidebarGroup,
    SidebarGroupContent,
  } from '@/components/ui/sidebar';

  import { Bell, CircleHelp, Settings } from 'lucide-vue-next';

  import logoImg from '@/assets/Logo2Only.png';
  import { storeToRefs } from 'pinia';
  import { useUiStore } from '@/stores/uiStore';
  import HelpModal from './HelpModal.vue';

  const uiStore = useUiStore();
  const { leftSidebarOpen } = storeToRefs(uiStore);

  const menuItems = computed(() => DEFAULT_MENU_ITEMS);

  const isHelpModalOpen = ref(false);
</script>

<template>
  <Sidebar
    side="left"
    :open="leftSidebarOpen"
    collapsible="icon"
    class="border-croffle-border bg-croffle-sidebar border-r pt-8"
  >
    <SidebarHeader
      class="border-croffle-border bg-croffle-sidebar relative flex flex-col border-b transition-all duration-200"
      :class="[leftSidebarOpen ? 'p-4' : 'items-center py-4']"
    >
      <div
        class="flex w-full shrink-0 items-center gap-3"
        :class="{ 'flex-col justify-center': !leftSidebarOpen }"
      >
        <div class="flex shrink-0 items-center justify-center">
          <img
            :src="logoImg"
            alt="Croffle Logo"
            class="object-contain transition-all duration-200"
            :class="leftSidebarOpen ? 'h-12 w-12' : 'h-8 w-8'"
          />
        </div>

        <div v-if="leftSidebarOpen" class="flex flex-col gap-0.5">
          <span class="font-logo text-croffle-primary text-2xl leading-none font-bold"
            >CROFFLE</span
          >

          <span class="text-croffle-text text-xs leading-none">할일 달력</span>
        </div>
      </div>
    </SidebarHeader>

    <div
      v-if="leftSidebarOpen"
      class="bg-croffle-sidebar text-croffle-text w-full pt-3 pr-0 pb-2 pl-4 text-left text-xs font-semibold tracking-wider uppercase"
    >
      메인 메뉴
    </div>

    <SidebarContent class="bg-croffle-sidebar">
      <SidebarGroup>
        <SidebarGroupContent>
          <SidebarMenu>
            <SidebarMenuItem v-for="item in menuItems" :key="item.title">
              <SidebarMenuButton
                as-child
                size="lg"
                class="hover:bg-croffle-hover rounded-lg bg-transparent ring-0 transition-all duration-200 outline-none"
                :class="[
                  { 'bg-croffle-primary hover:bg-croffle-primary': item.active },
                  leftSidebarOpen ? 'mr-2 ml-0' : 'mx-0 justify-center',
                ]"
                :tooltip="item.title"
              >
                <a
                  :href="item.url"
                  class="text-croffle-text flex w-full items-center py-2.5"
                  :class="[leftSidebarOpen ? 'gap-3 px-4' : 'justify-center px-0']"
                >
                  <component
                    :is="item.icon"
                    class="text-croffle-text h-5 w-5 shrink-0"
                    :class="{ 'text-white': item.active }"
                  />

                  <div v-if="leftSidebarOpen" class="flex flex-col gap-0.5">
                    <span
                      class="text-croffle-text text-sm leading-tight font-medium"
                      :class="{ 'text-white': item.active }"
                    >
                      {{ item.title }}
                    </span>

                    <span
                      class="text-croffle-text text-xs leading-none"
                      :class="{ 'text-white/80': item.active }"
                    >
                      {{ item.subtitle }}
                    </span>
                  </div>
                </a>
              </SidebarMenuButton>
            </SidebarMenuItem>
          </SidebarMenu>
        </SidebarGroupContent>
      </SidebarGroup>
    </SidebarContent>

    <SidebarFooter class="border-croffle-border bg-croffle-sidebar border-t p-3">
      <div class="flex items-center justify-around gap-2" :class="{ 'flex-col': !leftSidebarOpen }">
        <SidebarMenuButton
          size="sm"
          class="hover:bg-croffle-hover flex aspect-square h-9 w-9 items-center justify-center border-none bg-transparent shadow-none ring-0 ring-offset-0 transition-colors outline-none [--sidebar-accent:transparent] focus:ring-0 focus-visible:ring-0"
          tooltip="알림"
        >
          <Bell class="text-croffle-text h-5 w-5" />
        </SidebarMenuButton>

        <SidebarMenuButton
          size="sm"
          class="hover:bg-croffle-hover flex aspect-square h-9 w-9 items-center justify-center border-none bg-transparent shadow-none ring-0 ring-offset-0 transition-colors outline-none [--sidebar-accent:transparent] focus:ring-0 focus-visible:ring-0"
          tooltip="설정"
        >
          <Settings class="text-croffle-text h-5 w-5" />
        </SidebarMenuButton>

        <SidebarMenuButton
          size="sm"
          class="hover:bg-croffle-hover flex aspect-square h-9 w-9 items-center justify-center border-none bg-transparent shadow-none ring-0 ring-offset-0 transition-colors outline-none [--sidebar-accent:transparent] focus:ring-0 focus-visible:ring-0"
          tooltip="도움말"
          @click="isHelpModalOpen = true"
        >
          <CircleHelp class="text-croffle-text h-5 w-5" />
        </SidebarMenuButton>
      </div>
    </SidebarFooter>
  </Sidebar>

  <HelpModal v-model:open="isHelpModalOpen" />
</template>
