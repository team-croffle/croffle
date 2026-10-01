import type { ExtensionContext } from '@croffledev/croffle-types';
import React from 'react';
import type { Root } from 'react-dom/client';
import { createRoot } from 'react-dom/client';

import MyFeatureView from './MyFeatureView';
import MySettingsTab from './MySettingsTab';

let viewRoot: Root | null = null;
let settingsRoot: Root | null = null;
let detachContextMenu: (() => void) | null = null;

export function activated(context: ExtensionContext) {
  console.log('React Plugin has been activated!');

  // 1. Register Feature View
  context.ui.registerView('react-feature-view', (container: HTMLElement) => {
    viewRoot = createRoot(container);
    viewRoot.render(<MyFeatureView />);
  });

  // 2. Register Settings Tab
  context.ui.registerConfigurationTab('react-settings-tab', {
    label: 'React Tab',
    render: (container: HTMLElement) => {
      settingsRoot = createRoot(container);
      settingsRoot.render(<MySettingsTab />);
    },
  });

  // 3. Context menu: the item is declared in croffle-manifest.json (contributes.contextMenus);
  // attach its action here. The element is the right-clicked day or schedule (or null).
  detachContextMenu = context.ui.onContextMenu('hello', (element: HTMLElement | null) => {
    console.log('Context menu clicked!', element);
    alert('Hello from React Plugin!');
  });
}

export function deactivated() {
  console.log('React Plugin has been deactivated!');

  detachContextMenu?.();
  detachContextMenu = null;

  if (viewRoot) {
    viewRoot.unmount();
    viewRoot = null;
  }

  if (settingsRoot) {
    settingsRoot.unmount();
    settingsRoot = null;
  }
}
