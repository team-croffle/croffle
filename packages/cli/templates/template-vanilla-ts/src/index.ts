import type { ExtensionContext } from '@croffledev/croffle-types';

import './style.css'; // We will create this

let detachContextMenu: (() => void) | null = null;

export function activated(context: ExtensionContext) {
  console.log('Vanilla Plugin has been activated!');

  // 1. Register Feature View
  context.ui.registerView('vanilla-feature-view', (container: HTMLElement) => {
    container.innerHTML = `
      <div class="vanilla-container">
        <h1 class="vanilla-title">Vanilla JS Feature View</h1>
        <div class="vanilla-card">
          <p class="vanilla-text">This view is rendered using pure Vanilla JS/TS without any framework.</p>
          <button id="vanilla-btn" class="vanilla-btn">Clicked 0 times</button>
        </div>
      </div>
    `;

    let count = 0;
    const btn = container.querySelector('#vanilla-btn');
    if (btn) {
      btn.addEventListener('click', () => {
        count++;
        btn.textContent = `Clicked ${count} times`;
      });
    }
  });

  // 2. Register Settings Tab
  context.ui.registerConfigurationTab('vanilla-settings-tab', {
    label: 'Vanilla Tab',
    render: (container: HTMLElement) => {
      container.innerHTML = `
        <div class="vanilla-container">
          <h2 class="vanilla-title">Hello from Vanilla Plugin!</h2>
          <p class="vanilla-text">This is a custom settings tab rendered using pure JS.</p>
          <button id="vanilla-tab-btn" class="vanilla-btn">Count is: 0</button>
        </div>
      `;

      let count = 0;
      const btn = container.querySelector('#vanilla-tab-btn');
      if (btn) {
        btn.addEventListener('click', () => {
          count++;
          btn.textContent = `Count is: ${count}`;
        });
      }
    },
  });

  // 3. Context menu: the item is declared in croffle-manifest.json (contributes.contextMenus);
  // attach its action here. The element is the right-clicked day or schedule (or null).
  detachContextMenu = context.ui.onContextMenu('hello', (element: HTMLElement | null) => {
    console.log('Context menu clicked!', element);
    alert('Hello from Vanilla Plugin!');
  });
}

export function deactivated() {
  console.log('Vanilla Plugin has been deactivated!');

  detachContextMenu?.();
  detachContextMenu = null;
}
