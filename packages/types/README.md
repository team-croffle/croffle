# @croffledev/croffle-types

<div align="center">
<img src="../../.github/contents/icon.png" width="120" />

# Croffle Types

> Official TypeScript definitions for CROFFLE plugin development

</div>

**@croffledev/croffle-types** provides the essential type definitions and interfaces required to build, extend, and integrate plugins for the [CROFFLE](https://www.google.com/search?q=https://github.com/croffledev/croffle) ecosystem.

By using this package, developers can ensure type safety and leverage IntelliSense when interacting with the Croffle Core API, IPC events, and the plugin lifecycle.

---

## 📦 Installation

This package should be installed as a **development dependency** in your plugin project.

```bash
# Using npm
npm install -D @croffledev/croffle-types

# Using yarn
yarn add -D @croffledev/croffle-types

# Using pnpm
pnpm add -D @croffledev/croffle-types
```

---

## 🛠️ Usage

### Extension entry

An extension's entry module exports `activated(context)` and optionally `deactivated()`. `ExtensionContext` is the public host API (`CroffleAPI`, also reachable as `window.croffle`) plus storage, session and configuration already bound to your extension, and `ui` for adding views, settings tabs and context menu items.

```typescript
import type { ExtensionContext } from '@croffledev/croffle-types';

let detachMenu: (() => void) | null = null;

export function activated(context: ExtensionContext) {
  context.ui.registerView('my-view', (container) => {
    container.textContent = 'Hello from my extension';
  });

  // Declared in croffle-manifest.json → contributes.contextMenus: [{ "id": "hello", ... }]
  detachMenu = context.ui.onContextMenu('hello', (target) => {
    // target: the right-clicked day cell or schedule element, or null
    console.log('clicked', target);
  });
}

export function deactivated() {
  detachMenu?.();
}
```

### Manifest context menus

```json
{
  "contributes": {
    "contextMenus": [{ "id": "hello", "label": "Say hello", "targetView": ["calendar"] }]
  }
}
```

- `id`: unique within the extension; pass it to `onContextMenu`.
- `label`: shown as written.
- `targetView`: where the item appears (`calendar` or one of your view ids); omit for every screen.
- `disabled`: show the item greyed out.

Items appear only after `onContextMenu` attaches an action. `onContextMenu(id, handler, { condition })` hides the item for a right-click when `condition(target)` returns `false`. Requires Croffle 1.2.5 or later.

### What an extension can use

`CroffleAPI` is the whole host surface an extension gets; anything else in the app (window controls, writing settings, installing extensions, file dialogs) is host-only and not exposed.

| Member                                | What it does                                                                                     |
| :------------------------------------ | :----------------------------------------------------------------------------------------------- |
| `calendar.schedules`                  | `getAll`, `create`, `update`, `remove`, `moveOccurrence`                                         |
| `calendar.tags`                       | `getAll`, `getByName`, `create`, `modify`, `remove`                                              |
| `calendar.search`                     | `q(query)`                                                                                       |
| `settings`                            | `getAll`, `getOf` (read-only)                                                                    |
| `extensions.info`                     | `getInstalled`, `getEnabled`, `getByName` (read-only)                                            |
| `event`                               | `emit`, `on` — see below                                                                         |
| `http`, `os`, `enums`                 | HTTP helpers, OS info and the enum objects (`AppEventType`, `CalendarView`, …)                   |
| `storage`, `session`, `configuration` | Context only; already bound to your extension id                                                 |
| `ui`                                  | Context only; `registerView`, `registerConfigurationTab`, `onContextMenu`, `registerContextMenu` |

**Events.** `context.event.emit('ping')` is delivered as `<extensionId>:ping`; a name that already starts with your id is sent as is. Core events (`AppEventType`) and other extensions' prefixes cannot be emitted — they are dropped with a console warning. `on(name)` subscribes to anything, core events included.

### Migrating from 1.x to 2.0

| Removed in 2.0                                                                       | Use instead                                                     |
| :----------------------------------------------------------------------------------- | :-------------------------------------------------------------- |
| `window.*` (`minimize`, `close`, `exitApp`, `checkForUpdates`, `setCloseToTrayMode`) | Host-only; no replacement                                       |
| `settings.update`                                                                    | Host-only; settings change through the app UI                   |
| `extensions.info.install` / `installFromLocal` / `toggle` / `uninstall`              | Host-only                                                       |
| `extensions.storage` / `session` / `configuration` (`extensionId` argument)          | `context.storage` / `context.session` / `context.configuration` |
| `calendar.schedules.exportSchedulesToFile` / `importScheduleFromFile`                | Host-only                                                       |
| `Schedule.reminderMinutes`                                                           | `Schedule.reminders` (`number[] \| null`)                       |
| Helpers that leaked through `croffle.enums`                                          | Only enum objects remain (`AppCloseBehavior` added)             |
| `window.electron`                                                                    | Removed; extensions never needed it                             |

Event names also changed: an `emit('x')` from an extension now reaches subscribers as `<extensionId>:x`.

---

## 📖 Key Definitions

| Type / Interface      | Description                                                                         |
| :-------------------- | :---------------------------------------------------------------------------------- |
| `ExtensionContext`    | What `activated` receives: host API plus extension-bound storage, session, config.  |
| `UiApi`               | `registerView`, `registerConfigurationTab`, `onContextMenu`, `registerContextMenu`. |
| `CroffleManifest`     | Shape of `croffle-manifest.json`, including `contributes`.                          |
| `ContextMenuManifest` | One `contributes.contextMenus` entry.                                               |
| `CroffleAPI`          | The public host API exposed as `window.croffle` (see "What an extension can use").  |

---

## 👩‍💻 For Contributors

If you want to contribute to the type definitions or add support for new Core APIs:

1.  Fork the [Croffle Main Repository](https://www.google.com/search?q=https://github.com/croffledev/croffle).
2.  Navigate to the types workspace (or this standalone repo).
3.  Update `index.d.ts`.
4.  Submit a Pull Request with a clear description of the changes.

---

## 📄 License

This package is distributed under the **MIT License**.

Copyright (c) 2026 **Croffle Dev.** & Croffle Contributors
