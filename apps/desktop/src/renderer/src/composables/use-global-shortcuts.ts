import { useEventListener } from '@vueuse/core';

import { useUiStore } from '@/stores/ui-store';

/** Keyboard shortcuts shown in tooltips. Kept here so labels and handlers stay in sync. */
export const SHORTCUT_LEFT_SIDEBAR = 'Alt+B';
export const SHORTCUT_RIGHT_SIDEBAR = 'Alt+Shift+B';

/**
 * App-wide keyboard shortcuts (renderer only; they work while the window has focus).
 *
 * - `Alt+B` toggles the left sidebar, `Alt+Shift+B` the right one.
 *
 * `event.code` is used instead of `event.key` so the shortcut works under any keyboard
 * layout or IME. Alt combinations do not type anything on Windows/Linux; on macOS `Alt+B`
 * would insert `∫`, so the event is always cancelled. Inputs are not excluded on purpose:
 * the shortcut has no meaning in a text field, and toggling a sidebar is harmless.
 *
 * Note: shadcn's SidebarProvider also listens for Ctrl/Cmd+B, but our sidebars are driven
 * by the ui-store `open` props, so that shortcut has no visible effect and is left alone.
 */
export function useGlobalShortcuts(): void {
  const uiStore = useUiStore();

  useEventListener(window, 'keydown', (event: KeyboardEvent) => {
    if (event.repeat || !event.altKey || event.ctrlKey || event.metaKey) {
      return;
    }
    if (event.code !== 'KeyB') {
      return;
    }
    event.preventDefault();
    if (event.shiftKey) {
      uiStore.toggleRightSidebar();
    } else {
      uiStore.toggleLeftSidebar();
    }
  });
}
