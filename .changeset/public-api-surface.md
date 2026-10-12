---
'@croffledev/croffle-types': major
---

Narrow the public host API to what an extension may use. `CroffleAPI` (what `window.croffle` and `ExtensionContext` expose) now contains only `os`, `http`, `event`, `calendar` (schedule / tag CRUD, search, `moveOccurrence`), read-only `settings` (`getAll`, `getOf`), read-only `extensions.info` (`getInstalled`, `getEnabled`, `getByName`) and `enums`; `ExtensionContext` adds `ui` and the id-bound `storage`, `session` and `configuration`.

Removed from the package (host-only from now on):

- `WindowApi` and `CroffleAPI.window` (`minimize`, `maximize`, `close`, `exitApp`, `checkForUpdates`)
- `SettingsApi.update` — change settings through the app UI
- `ExtensionInfoApi.install` / `installFromLocal` / `toggle` / `uninstall`
- `ExtensionConfigurationApi`, `ExtensionStorageApi`, `ExtensionSessionApi` (the `extensionId`-keyed forms) and `ExtensionsApi.configuration` / `storage` / `session` — use `context.storage`, `context.session`, `context.configuration`, which are already bound to your extension
- `SchedulesApi.exportSchedulesToFile` / `importScheduleFromFile` (native dialogs)

Also:

- `croffle.enums` exposes only the enum objects (plus the new `AppCloseBehavior`); helper functions that previously leaked from the host's common module are gone.
- Extension events are namespaced: `context.event.emit('ping')` is delivered as `<extensionId>:ping`, core `AppEventType` names cannot be emitted, and `on` is unchanged.
- `window.electron` (a raw `ipcRenderer`) is no longer exposed to page code.
