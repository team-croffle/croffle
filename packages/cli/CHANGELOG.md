# @croffledev/croffle-cli

## 1.1.3

### Patch Changes

- 123f87c: Security and tooling maintenance.

  - `@croffledev/croffle-cli`: bump `adm-zip` to `^0.6.1` (GHSA-vwc7-r8mq-g2x9). The CLI only creates archives in `pack`; no behaviour change.
  - `@croffledev/croffle-types`: add a `typecheck` script and `typescript` / `@types/node` devDependencies so CI covers the published declarations. No public type changes.

## 1.1.2

### Patch Changes

- c5a0fcd: Address Dependabot/npm advisories in CLI templates and shared type package deps.

  - Plugin templates (vanilla / Vue / React): upgrade Vite to `^8.2.1` (and matching plugins), TypeScript to `6.0.3`, and pin `@croffledev/croffle-types` to `^1.4.1` instead of `latest`
  - `@croffledev/croffle-types`: bump `@types/node` to `^26.2.0`

  ***

## 1.1.1

### Patch Changes

- dd87681: update deps

## 1.1.0

### Minor Changes

- 64c01d2: Scaffold and pack with croffle-manifest.json.

  Templates include contributes.views and schema-based configuration examples.

## 1.0.1

### Patch Changes

- 250dac6: integrated monorepo
