<script setup lang="ts">
  import type { ConfigurationSectionContribution, ExtensionInfo } from '@croffledev/common';
  import { ref, onMounted, onUnmounted } from 'vue';

  import {
    ContextMenu,
    ContextMenuTrigger,
    ContextMenuContent,
    ContextMenuItem,
  } from '@/components/ui/context-menu';
  import { Icon } from '@/components/ui/icon';
  import { SidebarProvider, SidebarInset } from '@/components/ui/sidebar';
  import { Toaster } from '@/components/ui/sonner';
  import { translateOrRaw } from '@/i18n';

  import ConfirmModal from './components/confirm-modal.vue';
  import LeftSidebar from './components/left-sidebar.vue';
  import RightSidebar from './components/right-sidebar.vue';
  import ScheduleModal from './components/schedule-modal.vue';
  import SettingsModal from './components/settings-modal.vue';
  import Button from './components/ui/button/Button.vue';
  import { Separator } from './components/ui/separator/index.ts';
  import UpdateModal from './components/update-modal.vue';
  import { defaultMenus } from './data/default-context-menus.ts';
  import { extensionLoader } from './services/extension-loader.ts';
  import { useAppSettingsStore } from './stores/app-settings-store.ts';
  import { useContextMenuStore } from './stores/context-menu-store.ts';
  import { useSettingsStore } from './stores/settings-store.ts';
  import { useThemeStore } from './stores/theme-store.ts';
  import { useUiStore } from './stores/ui-store.ts';
  import { useUpdateStore } from './stores/update-store.ts';
  import { useViewStore } from './stores/view-store.ts';
  // import { mockPluginsList } from './test/testPluginMenu';

  const uiStore = useUiStore();
  const contextMenuStore = useContextMenuStore();
  const viewStore = useViewStore();
  const settingsStore = useSettingsStore();
  const themeStore = useThemeStore();
  const appSettingsStore = useAppSettingsStore();
  const updateStore = useUpdateStore();

  // 설정 모달 상태
  const isSettingsOpen = ref(false);

  const isDev = import.meta.env.DEV;
  if (isDev) {
    // oxlint-disable-next-line no-console
    console.log('isDev', isDev);
  }

  // Electron 윈도우 제어 함수
  const minimizeWindow = async () => {
    croffle.window.minimize();
  };
  const maximizeWindow = async () => {
    croffle.window.maximize();
  };
  const closeWindow = async () => {
    croffle.window.close();
  };

  const handleRegisterView = (event: Event) => {
    const customEvent = event as CustomEvent<{
      extensionId: string;
      viewId: string;
      renderFn: (container: HTMLElement) => void;
    }>;
    const { viewId, renderFn } = customEvent.detail;

    viewStore.registerView(viewId, renderFn);
  };

  const handleRegisterSettingsTab = (event: Event) => {
    const customEvent = event as CustomEvent<{
      extensionId: string;
      extensionName: string;
      tabId: string;
      label: string;
      icon?: unknown;
      order?: number;
      render?: (container: HTMLElement) => void;
      sections?: ConfigurationSectionContribution[];
    }>;
    const { extensionId, extensionName, tabId, label, icon, order, render, sections } =
      customEvent.detail;

    // NOTE: the section means the configuration section using the pre-defined schema.
    // and the render means the custom configuration section.
    // render has the permission for rendering on the one of the settings tabs.
    if (render && sections && isDev) {
      // oxlint-disable-next-line no-console
      console.warn(
        `[Plugin ${extensionName}] registerConfigurationTab: render와 sections는 동시에 사용할 수 없습니다. render가 우선됩니다.`,
      );
    }

    settingsStore.registerTab({
      id: tabId,
      label,
      icon,
      order,
      extensionId,
      extensionName,
      render,
      sections: render ? undefined : sections,
    });
  };

  const handleRegisterContextMenu = (event: Event) => {
    const customEvent = event as CustomEvent<{
      extensionId: string;
      target: string;
      command: string;
      label: string;
      callback: (element: HTMLElement | null) => void;
    }>;
    const { extensionId, target, command, label, callback } = customEvent.detail;

    contextMenuStore.registerMenu({
      id: `${extensionId}-${command}`,
      targetView: [target],
      label,
      action: callback,
      extensionId,
    });
  };

  const handlePluginUnloaded = (event: Event) => {
    const customEvent = event as CustomEvent<{ extensionId: string }>;
    const { extensionId } = customEvent.detail;
    settingsStore.unregisterExtensionConfigurationTabs(extensionId);
    viewStore.unregisterExtensionMenus(extensionId);
    contextMenuStore.unregisterExtensionMenus(extensionId);
  };

  const handleExtensionLoaded = (event: Event) => {
    const customEvent = event as CustomEvent<{
      extension: ExtensionInfo;
    }>;
    const { extension } = customEvent.detail;

    if (isDev) {
      // oxlint-disable-next-line no-console
      console.log(`[app.vue] handleExtensionLoaded: ${extension.id}`, extension.contributes);
    }
    if (extension.contributes?.views) {
      const views = extension.contributes.views.map((v) => ({
        ...v,
        extensionName: extension.name,
        extensionId: extension.id,
      }));

      if (isDev) {
        // oxlint-disable-next-line no-console
        console.log(`[app.vue] registerMenus views:`, views);
      }

      viewStore.registerMenus(views);
    }
    if (extension.contributes?.configuration) {
      settingsStore.registerManifestTabs(
        extension.id,
        extension.name,
        extension.contributes.configuration,
      );
    }
  };

  const registerDefaultContextMenu = () => {
    contextMenuStore.registerMenus(defaultMenus);
  };

  const setPluginMenus = async () => {
    // 이벤트로 플러그인 호출 동작 매핑
    window.addEventListener('extension:register-view', handleRegisterView);
    window.addEventListener('extension:register-configuration-tab', handleRegisterSettingsTab);
    window.addEventListener('extension:register-context-menu', handleRegisterContextMenu);
    window.addEventListener('extension:loaded', handleExtensionLoaded);
    window.addEventListener('extension:unloaded', handlePluginUnloaded);

    // 최초 로드 시 활성화된 플러그인들의 매니페스트는 extensionLoader.init()에서 extension:loaded 이벤트를 쏴주므로
    // 여기서 직접 registerManifestTabs를 순회해서 호출할 필요가 없어짐.
    // extensionLoader.init() 내부에서 플러그인을 로딩할 때 extension:loaded를 쏘면 자동으로 handlePluginLoaded가 처리함.
  };

  // Prefer a stable FullCalendar cell/event node. Raw e.target is often .fc-highlight
  // (or a child), which select()/unselect() remove from the DOM — closest() then fails
  // on the detached node and the menu shows "no actions".
  function resolveFrom(el: HTMLElement): HTMLElement | null {
    return (
      (el.closest('.fc-event') as HTMLElement | null) ||
      (el.closest('.fc-daygrid-day') as HTMLElement | null)
    );
  }

  function resolveContextMenuTarget(e: MouseEvent): HTMLElement | null {
    const raw = e.target as HTMLElement | null;
    if (!raw) {
      return null;
    }

    if (raw.isConnected) {
      return resolveFrom(raw) ?? raw;
    }

    // Fallback when calendar already mutated selection DOM during this contextmenu.
    for (const node of document.elementsFromPoint(e.clientX, e.clientY)) {
      if (!(node instanceof HTMLElement)) {
        continue;
      }
      const stable = resolveFrom(node);
      if (stable) {
        return stable;
      }
    }

    return raw;
  }

  const handleContextMenuEvent = (e: MouseEvent) => {
    contextMenuStore.setActiveElement(resolveContextMenuTarget(e));
  };

  const handleMenuOpenChange = (isOpen: boolean) => {
    if (!isOpen) {
      contextMenuStore.setActiveElement(null); // 우클릭 메뉴가 닫힐 때 activeElement 초기화
    }
  };

  onMounted(async () => {
    registerDefaultContextMenu();
    await appSettingsStore.initialize();
    updateStore.init();
    await setPluginMenus();
    await extensionLoader.init();
  });

  onUnmounted(() => {
    window.removeEventListener('extension:register-view', handleRegisterView);
    window.removeEventListener('extension:register-configuration-tab', handleRegisterSettingsTab);
    window.removeEventListener('extension:register-context-menu', handleRegisterContextMenu);
    window.removeEventListener('extension:loaded', handleExtensionLoaded);
    window.removeEventListener('extension:unloaded', handlePluginUnloaded);
    appSettingsStore.dispose();
    updateStore.dispose();
  });
</script>

<template>
  <div class="bg-croffle-bg flex h-screen flex-col overflow-hidden font-sans text-neutral-800">
    <!-- 커스텀 타이틀 바 -->
    <div
      class="drag-region border-croffle-border bg-croffle-bg z-50 flex h-8 shrink-0 justify-between border-b"
    >
      <div class="flex h-full items-center">
        <div class="relative flex h-full w-12 items-center justify-center">
          <Button
            variant="ghost"
            size="icon"
            class="no-drag h-7 w-7 text-neutral-500"
            @click="uiStore.toggleLeftSidebar()"
          >
            <Icon icon="lucide:panel-left" class="h-4 w-4" />
          </Button>

          <div class="absolute right-0 h-4 w-px bg-neutral-300"></div>
        </div>

        <span class="font-logo ml-4 text-xs font-bold text-neutral-600">Croffle</span>
      </div>

      <div class="no-drag flex h-full items-center">
        <button
          class="flex h-6 w-6 items-center justify-center rounded-full text-neutral-500 transition-colors hover:bg-neutral-200 hover:text-neutral-900 dark:hover:bg-neutral-500"
          aria-label="Toggle theme"
          @click="themeStore.changeTheme"
        >
          <Icon v-if="themeStore.isDark" icon="lucide:sun" class="h-4 w-4" />
          <Icon v-else icon="lucide:moon" class="h-4 w-4" />
        </button>
        <Separator orientation="vertical" class="mr-6 ml-4 bg-neutral-300 dark:bg-neutral-700" />
        <button
          class="flex h-full w-12 items-center justify-center text-neutral-500 transition-colors hover:bg-neutral-200 hover:text-neutral-900 dark:hover:bg-neutral-500"
          aria-label="Minimize window"
          @click="minimizeWindow"
        >
          <Icon icon="lucide:minus" class="h-4 w-4" />
        </button>

        <button
          class="flex h-full w-12 items-center justify-center text-neutral-500 transition-colors hover:bg-neutral-200 hover:text-neutral-900 dark:hover:bg-neutral-500"
          aria-label="Maximize window"
          @click="maximizeWindow"
        >
          <Icon icon="lucide:square" class="h-3 w-3" />
        </button>

        <button
          class="flex h-full w-12 items-center justify-center text-neutral-500 transition-colors hover:bg-red-600 hover:text-red-100 dark:hover:bg-red-700"
          aria-label="Close window"
          @click="closeWindow"
        >
          <Icon icon="lucide:x" class="h-4 w-4" />
        </button>
      </div>
    </div>

    <!-- 메인 콘텐츠 영역 -->
    <div class="relative min-h-0 flex-1">
      <!-- 사이드바 및 캘린더 -->
      <SidebarProvider class="h-full min-h-full w-full">
        <LeftSidebar @open-settings="isSettingsOpen = true" />
        <SidebarInset class="bg-croffle-bg flex h-full flex-col">
          <ContextMenu @update:open="handleMenuOpenChange">
            <ContextMenuTrigger as-child>
              <!-- 메인 영역 -->
              <div class="flex-1 overflow-hidden p-4" @contextmenu="handleContextMenuEvent">
                <router-view />
              </div>
            </ContextMenuTrigger>
            <ContextMenuContent v-if="contextMenuStore.currentItems.length > 0">
              <ContextMenuItem
                v-for="item in contextMenuStore.currentItems"
                :key="item.id"
                :disabled="item.disabled"
                @click="item.action(contextMenuStore.activeElement)"
              >
                {{ translateOrRaw(item.label) }}
              </ContextMenuItem>
            </ContextMenuContent>
            <ContextMenuContent v-else>
              <ContextMenuItem disabled>{{ $t('contextMenu.empty') }}</ContextMenuItem>
            </ContextMenuContent>
          </ContextMenu>
        </SidebarInset>
        <RightSidebar />
        <ScheduleModal />
        <ConfirmModal />
      </SidebarProvider>
    </div>

    <!-- 설정 모달 -->
    <SettingsModal :open="isSettingsOpen" @update:open="isSettingsOpen = $event" />
    <UpdateModal />

    <Toaster />
  </div>
</template>

<style scoped>
  /* 드래그 영역 설정 */
  .drag-region {
    -webkit-app-region: drag;
  }
  /* 드래그 영역 제외 */
  .no-drag {
    -webkit-app-region: no-drag;
  }
</style>
