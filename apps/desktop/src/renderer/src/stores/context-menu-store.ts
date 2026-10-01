import type { ContextMenuManifest, FeatureContextMenu } from '@croffledev/common';
import { defineStore } from 'pinia';
import { computed, ref, shallowRef } from 'vue';
import { useRoute } from 'vue-router';

type MenuTarget = HTMLElement | null;

/** Item an extension declared in its manifest; shown once code attaches an action. */
type DeclaredMenu = ContextMenuManifest & {
  key: string;
  extensionId: string;
  extensionName: string;
};

type MenuBinding = {
  handler: (target: MenuTarget) => void;
  condition?: (target: MenuTarget) => boolean;
};

const menuKey = (extensionId: string, menuId: string) => `${extensionId}-${menuId}`;

const groupName = (group: FeatureContextMenu[]) =>
  group[0].extensionName ?? group[0].extensionId ?? '';

export const useContextMenuStore = defineStore('contextMenu', () => {
  // 현재 열려 있는 컨텍스트 메뉴의 아이템 목록
  const menuRegistry = ref<FeatureContextMenu[]>([]);
  const declaredMenus = ref<DeclaredMenu[]>([]);
  // A binding may arrive before its declaration: activated() runs before extension:loaded.
  const bindings = ref<Record<string, MenuBinding>>({});

  const route = useRoute();
  const activeElement = shallowRef<HTMLElement | null>(null);

  const currentItems = computed(() => {
    let currentTarget = 'calendar'; // default;

    if (route.path.startsWith('/extension/') && route.params.viewId) {
      currentTarget = route.params.viewId as string;
    } else if (route.name) {
      currentTarget = route.name as string;
    }

    const boundDeclared: FeatureContextMenu[] = declaredMenus.value.flatMap((menu) => {
      const binding = bindings.value[menu.key];
      if (!binding) {
        return [];
      }
      const { key, ...item } = menu;
      return [{ ...item, id: key, action: binding.handler, condition: binding.condition }];
    });

    return [...menuRegistry.value, ...boundDeclared].filter((item) => {
      if (!item.targetView || item.targetView.length === 0) {
        return true;
      }

      const viewMatch = item.targetView.includes(currentTarget);
      // targetViews가 없거나 global이 있으면 전역 메뉴로 취급
      if (!viewMatch) {
        return false;
      }

      if (item.condition) {
        return item.condition(activeElement.value);
      }

      return true;
    });
  });

  /**
   * Host items first, then one group per extension (by name). Within an extension, code-registered
   * items come first, then manifest items in declared order.
   */
  const currentGroups = computed(() => {
    const host = currentItems.value.filter((item) => !item.extensionId);
    const byExtension = new Map<string, FeatureContextMenu[]>();
    for (const item of currentItems.value) {
      if (item.extensionId) {
        byExtension.set(item.extensionId, [...(byExtension.get(item.extensionId) ?? []), item]);
      }
    }
    const extensionGroups = [...byExtension.values()].toSorted((a, b) =>
      groupName(a).localeCompare(groupName(b)),
    );
    return [host, ...extensionGroups].filter((group) => group.length > 0);
  });

  const declareExtensionMenus = (
    extensionId: string,
    extensionName: string,
    menus: ContextMenuManifest[],
  ) => {
    declaredMenus.value = [
      ...declaredMenus.value.filter((m) => m.extensionId !== extensionId),
      ...menus.map((m) => ({ ...m, key: menuKey(extensionId, m.id), extensionId, extensionName })),
    ];
  };

  const bindExtensionMenu = (extensionId: string, menuId: string, binding: MenuBinding) => {
    bindings.value = { ...bindings.value, [menuKey(extensionId, menuId)]: binding };
  };

  /** Detach only if the handler is still the bound one, so an old detach cannot drop a rebind. */
  const unbindExtensionMenu = (
    extensionId: string,
    menuId: string,
    handler: MenuBinding['handler'],
  ) => {
    const key = menuKey(extensionId, menuId);
    if (bindings.value[key]?.handler !== handler) {
      return;
    }
    bindings.value = Object.fromEntries(
      Object.entries(bindings.value).filter(([bindingKey]) => bindingKey !== key),
    );
  };

  const registerMenu = (menu: FeatureContextMenu) => {
    const existingIndex = menuRegistry.value.findIndex((m) => m.id === menu.id);
    if (existingIndex !== -1) {
      menuRegistry.value[existingIndex] = menu;
    } else {
      menuRegistry.value.push(menu);
    }
  };

  const registerMenus = (menus: FeatureContextMenu[]) => {
    menus.forEach((m) => registerMenu(m));
  };

  const unregisterMenu = (menuId: string) => {
    menuRegistry.value = menuRegistry.value.filter((m) => m.id !== menuId);
  };

  const unregisterMenus = (...menuId: string[]) => {
    menuRegistry.value = menuRegistry.value.filter((m) => !menuId.includes(m.id));
  };

  const unregisterExtensionMenus = (extensionId: string) => {
    menuRegistry.value = menuRegistry.value.filter((m) => m.extensionId !== extensionId);
    declaredMenus.value = declaredMenus.value.filter((m) => m.extensionId !== extensionId);
    const prefix = menuKey(extensionId, '');
    bindings.value = Object.fromEntries(
      Object.entries(bindings.value).filter(([key]) => !key.startsWith(prefix)),
    );
  };

  const setActiveElement = (el: HTMLElement | null) => {
    activeElement.value = el;
  };

  return {
    menuRegistry,
    currentItems,
    currentGroups,
    activeElement,
    declareExtensionMenus,
    bindExtensionMenu,
    unbindExtensionMenu,
    registerMenu,
    registerMenus,
    unregisterMenu,
    unregisterMenus,
    unregisterExtensionMenus,
    setActiveElement,
  };
});
