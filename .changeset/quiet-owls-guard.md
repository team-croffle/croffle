---
'@croffledev/croffle-cli': patch
'@croffledev/croffle-types': patch
---

Security and tooling maintenance.

- `@croffledev/croffle-cli`: bump `adm-zip` to `^0.6.1` (GHSA-vwc7-r8mq-g2x9). The CLI only creates archives in `pack`; no behaviour change.
- `@croffledev/croffle-types`: add a `typecheck` script and `typescript` / `@types/node` devDependencies so CI covers the published declarations. No public type changes.
