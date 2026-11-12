<script setup lang="ts">
  import type { FeatureView } from '@croffledev/common';
  import { storeToRefs } from 'pinia';
  import { ref, computed } from 'vue';
  import { useRoute } from 'vue-router';

  import logoImg from '@/assets/logo-only.png';
  import { Icon } from '@/components/ui/icon';
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
  import { DEFAULT_MENU_ITEMS } from '@/data/default-menus';
  import { translateOrRaw } from '@/i18n';
  import { useUiStore } from '@/stores/ui-store';
  import { useViewStore } from '@/stores/view-store';

  import HelpModal from './help-modal.vue';
  // import { testMockPlugin } from '@/test/testPluginMenu';

  const uiStore = useUiStore();
  const { leftSidebarOpen } = storeToRefs(uiStore);

  const isHelpModalOpen = ref(false);

  const route = useRoute();
  const viewStore = useViewStore();

  // 메뉴 아이템을 computed로 정의하여, store된 state를 반영하도록 함.
  const menuItems = computed<FeatureView[]>(() => {
    const baseMenus: FeatureView[] = DEFAULT_MENU_ITEMS.map((item) => ({
      ...item,
      // calendar 메뉴 활성화 체크
      active: route.path.includes(item.url),
    }));

    const additionalMenus: FeatureView[] = viewStore.menus.map((menu) => ({
      ...menu,
      icon: menu.icon || 'lucide:puzzle',
      // 현재 URL의 viewId 파라미터가 메뉴의 id와 같으면 활성화
      active: route.params.viewId === `${menu.id}`,
      url: `/extension/${menu.id}`,
    }));

    return [...baseMenus, ...additionalMenus];
  });
  const emit = defineEmits<{
    (e: 'open-settings'): void;
  }>();
</script>

<template>
  <Sidebar
    side="left"
    :open="leftSidebarOpen"
    collapsible="icon"
    class="bg-croffle-sidebar border-r pt-8"
  >
    <SidebarHeader
      class="bg-croffle-sidebar relative flex flex-col border-b transition-all duration-200"
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
            :class="leftSidebarOpen ? 'size-12' : 'size-8'"
          />
        </div>

        <div v-if="leftSidebarOpen" class="flex flex-col gap-0.5">
          <span class="font-logo text-primary text-2xl leading-none font-bold">CROFFLE</span>

          <span class="text-croffle-text text-xs leading-none">{{ $t('sidebar.tagline') }}</span>
        </div>
      </div>
    </SidebarHeader>

    <div
      v-if="leftSidebarOpen"
      class="bg-croffle-sidebar text-croffle-text w-full pt-3 pr-0 pb-2 pl-4 text-left text-xs font-semibold tracking-wider uppercase"
    >
      {{ $t('sidebar.mainMenu') }}
    </div>

    <SidebarContent class="bg-croffle-sidebar">
      <SidebarGroup>
        <SidebarGroupContent>
          <SidebarMenu>
            <!-- <SidebarMenuItem v-for="item in menuItems" :key="item.title"> -->
            <SidebarMenuItem v-for="item in menuItems" :key="item.id">
              <SidebarMenuButton
                as-child
                size="lg"
                class="hover:bg-croffle-hover bg-transparent ring-0 transition-all duration-200 outline-none"
                :class="[
                  { 'bg-primary hover:bg-primary': item.active },
                  leftSidebarOpen ? 'mr-2 ml-0' : 'mx-0 justify-center',
                ]"
                :tooltip="translateOrRaw(item.title)"
              >
                <router-link
                  :to="item.url"
                  class="text-croffle-text flex w-full items-center py-2.5"
                  :class="[leftSidebarOpen ? 'gap-3 px-4' : 'justify-center px-0']"
                >
                  <Icon
                    v-if="typeof item.icon === 'string'"
                    :icon="item.icon"
                    class="text-croffle-text size-5 shrink-0"
                    :class="{ 'text-primary-foreground': item.active }"
                  />
                  <component
                    v-else
                    :is="item.icon"
                    class="text-croffle-text size-5 shrink-0"
                    :class="{ 'text-primary-foreground': item.active }"
                  />

                  <div v-if="leftSidebarOpen" class="flex flex-col gap-0.5">
                    <span
                      class="text-croffle-text text-sm leading-tight font-medium"
                      :class="{ 'text-primary-foreground': item.active }"
                    >
                      {{ translateOrRaw(item.title) }}
                    </span>

                    <span
                      class="text-croffle-text text-xs leading-none"
                      :class="{ 'text-primary-foreground/80': item.active }"
                    >
                      {{ translateOrRaw(item.subtitle) }}
                    </span>
                  </div>
                </router-link>
              </SidebarMenuButton>
            </SidebarMenuItem>
          </SidebarMenu>
        </SidebarGroupContent>
      </SidebarGroup>
    </SidebarContent>

    <SidebarFooter class="bg-croffle-sidebar border-t p-3">
      <div class="flex items-center justify-around gap-2" :class="{ 'flex-col': !leftSidebarOpen }">
        <SidebarMenuButton
          size="sm"
          class="hover:bg-croffle-hover flex aspect-square size-9 items-center justify-center border-none bg-transparent shadow-none ring-0 ring-offset-0 transition-colors outline-none [--sidebar-accent:transparent] focus:ring-0 focus-visible:ring-0"
          :tooltip="$t('sidebar.notifications')"
        >
          <Icon icon="lucide:bell" class="text-croffle-text size-5" />
        </SidebarMenuButton>

        <SidebarMenuButton
          size="sm"
          class="hover:bg-croffle-hover flex aspect-square size-9 items-center justify-center border-none bg-transparent shadow-none ring-0 ring-offset-0 transition-colors outline-none [--sidebar-accent:transparent] focus:ring-0 focus-visible:ring-0"
          :tooltip="$t('sidebar.settings')"
          @click="emit('open-settings')"
        >
          <Icon icon="lucide:settings" class="text-croffle-text size-5" />
        </SidebarMenuButton>

        <SidebarMenuButton
          size="sm"
          class="hover:bg-croffle-hover flex aspect-square size-9 items-center justify-center border-none bg-transparent shadow-none ring-0 ring-offset-0 transition-colors outline-none [--sidebar-accent:transparent] focus:ring-0 focus-visible:ring-0"
          :tooltip="$t('sidebar.help')"
          @click="isHelpModalOpen = true"
        >
          <Icon icon="lucide:circle-help" class="text-croffle-text size-5" />
        </SidebarMenuButton>
      </div>
    </SidebarFooter>
  </Sidebar>

  <HelpModal v-model:open="isHelpModalOpen" />
</template>
