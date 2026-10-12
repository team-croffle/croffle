# @croffledev/croffle-types

## 2.0.0

### Major Changes

- f9a01be: Narrow the public host API to what an extension may use. `CroffleAPI` (what `window.croffle` and `ExtensionContext` expose) now contains only `os`, `http`, `event`, `calendar` (schedule / tag CRUD, search, `moveOccurrence`), read-only `settings` (`getAll`, `getOf`), read-only `extensions.info` (`getInstalled`, `getEnabled`, `getByName`) and `enums`; `ExtensionContext` adds `ui` and the id-bound `storage`, `session` and `configuration`.

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

- 77a7df0: Remove the deprecated `Schedule.reminderMinutes` mirror. Read and write `Schedule.reminders` (`number[] | null`) instead: `null` follows the app-wide default, `[]` means no reminder, and `[m]` replaces the old single value `m`.
- 9377e00: Remove the deprecated `WindowApi.setCloseToTrayMode`. The close-button behavior is the `general.closeBehavior` setting (`ask` | `tray` | `quit`); write it with `settings.update({ general: { ...general, closeBehavior } })`.

## 1.8.1

### Patch Changes

- 694ce15: Fix type resolution for projects using `moduleResolution: "nodenext"` (such as the `croffle-cli` templates): relative imports in the declaration files now carry `.js` extensions, so `ExtensionContext` and every other export resolve instead of reporting "no exported member".

## 1.8.0

### Minor Changes

- 07a762e: Add `UiApi.onContextMenu(id, handler, options?)` to attach an action to a menu item declared in the manifest's `contributes.contextMenus`; it returns a function that detaches it. `registerContextMenu`'s callback now receives the right-clicked element (`HTMLElement | null`), matching what the app has always passed. `ContextMenuManifest` fields are documented.

## 1.7.0

### Minor Changes

- 8c5a08b: Add recurring-schedule exceptions: `ScheduleException`, optional `Schedule.exceptions` and `SchedulesApi.moveOccurrence(id, occurrenceStart, newStart)`. Extensions that expand `recurrenceRule` themselves should skip each exception's `occurrenceStart` and use its `startDate`/`endDate` unless `cancelled`.

## 1.6.0

### Minor Changes

- d3a086a: Add `general.closeBehavior` to `AppSettings` and deprecate `window.setCloseToTrayMode`

  The close button behavior is now a user setting instead of a hard-coded "hide to tray".

  - `AppSettings.general.closeBehavior: AppCloseBehavior` — `ask` (default; the host prompts once and stores the answer), `tray`, or `quit`.
  - `AppCloseBehavior` enum.
  - `AppEventType.WINDOW_CLOSE_REQUESTED` — emitted when a close is requested while the setting is `ask`.
  - `WindowApi.setCloseToTrayMode` still works but now writes the setting (`true` → `tray`, `false` → `quit`). It is deprecated and will be removed in 1.3; use `settings.update` instead.

## 1.5.0

### Minor Changes

- 078559a: Add `reminders` to `Schedule` and deprecate `reminderMinutes`

  A schedule can now carry several reminder offsets instead of a single one.

  - `reminders: number[] | null` — offsets in minutes before the start, ascending. `null` follows the app-wide default; `[]` means no reminder.
  - `reminderMinutes` still reports the first offset so extensions written against 1.2.0 keep working. It is deprecated and will be removed in 1.3; read `reminders` instead.

## 1.4.4

### Patch Changes

- 123f87c: Security and tooling maintenance.

  - `@croffledev/croffle-cli`: bump `adm-zip` to `^0.6.1` (GHSA-vwc7-r8mq-g2x9). The CLI only creates archives in `pack`; no behaviour change.
  - `@croffledev/croffle-types`: add a `typecheck` script and `typescript` / `@types/node` devDependencies so CI covers the published declarations. No public type changes.

## 1.4.3

### Patch Changes

- 15858b4: Remove AppSettingStartupBehavior, general.startupBehavior, and SETTINGS_STARTUP_NAVIGATE.

  Login presentation is now controlled only by startMinimized (and the
  --croffle-start-hidden launch arg). Host settings no longer offer
  last-session or calendar-home startup routing.

- 21b2ceb: Remove unused behavior

## 1.4.2

### Patch Changes

- c5a0fcd: Address Dependabot/npm advisories in CLI templates and shared type package deps.

  - Plugin templates (vanilla / Vue / React): upgrade Vite to `^8.2.1` (and matching plugins), TypeScript to `6.0.3`, and pin `@croffledev/croffle-types` to `^1.4.1` instead of `latest`
  - `@croffledev/croffle-types`: bump `@types/node` to `^26.2.0`

  ***

## 1.4.1

### Patch Changes

- 076d5aa: Add an attribute to Schedule for features to develeoped later

## 1.4.0

### Minor Changes

- 9e6331c: Add priority column in schedule

## 1.3.0

### Minor Changes

- f7dcae4: Schedule dates use Date instead of ISO strings, and host-only UI types are no longer exported from croffle-types.

## 1.2.0

### Minor Changes

- 64c01d2: Introduce CroffleManifest with contributes and extension-scoped storage APIs.

  Replace features with contributes, trim AppEventType to domain mutations, and bind storage/session/configuration on ExtensionContext.

## 1.1.3

### Patch Changes

- 250dac6: integrated monorepo
