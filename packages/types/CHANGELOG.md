# @croffledev/croffle-types

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
