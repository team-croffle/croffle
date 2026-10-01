import type { ExtensionContext } from '@croffledev/croffle-types';
import { createApp } from 'vue';

import FeatureView from './MyFeatureView.vue';
import MySettingsTab from './MySettingsTab.vue';

let viewAppInstance: any = null;
let settingsAppInstance: any = null;
let detachContextMenu: (() => void) | null = null;

export function activated(context: ExtensionContext) {
  console.log('Test Plugin has been activated!');

  // 1. Register Feature View
  context.ui.registerView('test-feature-view', (container: HTMLElement) => {
    viewAppInstance = createApp(FeatureView);
    viewAppInstance.mount(container);
  });

  // 2. Register Settings Tab
  context.ui.registerConfigurationTab('test-settings-tab', {
    label: 'Test Tab',
    render: (container: HTMLElement) => {
      settingsAppInstance = createApp(MySettingsTab);
      settingsAppInstance.mount(container);
    },
  });

  // 3. Context menu: the item is declared in croffle-manifest.json (contributes.contextMenus);
  // attach its action here. The element is the right-clicked day or schedule (or null).
  detachContextMenu = context.ui.onContextMenu('hello', (element: HTMLElement | null) => {
    console.log('Context menu clicked!', element);
    alert('Hello from Test Plugin!');
  });
}

export function deactivated() {
  console.log('Test Plugin has been deactivated!');

  detachContextMenu?.();
  detachContextMenu = null;

  if (viewAppInstance) {
    viewAppInstance.unmount();
    viewAppInstance = null;
  }

  if (settingsAppInstance) {
    settingsAppInstance.unmount();
    settingsAppInstance = null;
  }
}
